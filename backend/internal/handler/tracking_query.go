package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
)

// 快递100 承运商代码映射
var kuaidi100Company = map[string]string{
	"CHINA_POST": "chinapost", "EMS": "ems", "SF": "shunfeng",
	"YT": "yuantong", "ZTO": "zhongtong", "STO": "shentong",
}

// 进程内复用一个客户端：每次查询都 new Client 会丢掉已建立的 TLS 连接。
var kuaidiClient = &http.Client{Timeout: 10 * time.Second}

type kuaidi100Resp struct {
	Message string `json:"message"`
	Nu      string `json:"nu"`
	State   string `json:"state"`
	Data    []struct {
		Time    string `json:"time"`
		Context string `json:"context"`
		Fcity   string `json:"fcity"`
	} `json:"data"`
}

// getSettingValue 读一个系统设置项。
// 走内存快照（见 settings_cache.go），不再每个键一次单行 SELECT。
func getSettingValue(db *gorm.DB, key string) string {
	return settingsSnapshot(db)[key]
}

// parseTrackingDetail 解析卡片上存储的轨迹 JSON（数组）
func parseTrackingDetail(raw string) []gin.H {
	details := []gin.H{}
	if raw == "" {
		return details
	}
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		return []gin.H{}
	}
	return details
}

// kuaidi100Configured 是否已配置快递100 自动查询
func kuaidi100Configured(db *gorm.DB) bool {
	return getSettingValue(db, "kuaidi100_customer") != "" && getSettingValue(db, "kuaidi100_key") != ""
}

// fetchKuaidi100 调用快递100 实时查询；未配置凭据时返回 details=nil, err=nil
func fetchKuaidi100(db *gorm.DB, num, carrier string) ([]gin.H, string, error) {
	customer := getSettingValue(db, "kuaidi100_customer")
	key := getSettingValue(db, "kuaidi100_key")
	if customer == "" || key == "" {
		return nil, "", nil
	}
	company := kuaidi100Company[carrier]
	if company == "" {
		company = "chinapost"
	}
	param, _ := json.Marshal(map[string]string{"num": num, "company": company})
	buf := make([]byte, 0, len(param)+len(key)+len(customer))
	buf = append(buf, param...)
	buf = append(buf, key...)
	buf = append(buf, customer...)
	sum := md5.Sum(buf)
	sign := strings.ToUpper(hex.EncodeToString(sum[:]))

	resp, err := kuaidiClient.PostForm("https://poll.kuaidi100.com/poll/query.do", url.Values{
		"customer": {customer}, "sign": {sign}, "param": {string(param)},
	})
	if err != nil {
		return nil, "", fmt.Errorf("连接快递100失败: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取快递100响应失败")
	}
	var kr kuaidi100Resp
	if err := json.Unmarshal(body, &kr); err != nil {
		return nil, "", fmt.Errorf("快递100响应解析失败")
	}
	if kr.Message != "ok" {
		return nil, "", fmt.Errorf("快递100: %s", kr.Message)
	}
	details := []gin.H{}
	for _, d := range kr.Data {
		details = append(details, gin.H{"time": d.Time, "status": d.Context, "location": d.Fcity})
	}
	status := ""
	if len(kr.Data) > 0 {
		status = kr.Data[0].Context
	}
	return details, status, nil
}

// trackingCacheTTL 快递轨迹的内存缓存窗口。
//
// 原来这里写成 10 * time.Minute，但 TrackingUpdatedAt 落库是东八区墙钟字符串，
// 用 time.Parse 当 UTC 解析会晚 8 小时 → time.Since 恒为负 → 缓存永不失效，
// 只有等真实 8 小时过去才会再查一次。现在解析走 parseCST，窗口才是真的 10 分钟。
const trackingCacheTTL = 10 * time.Minute

// ensureTrackingFresh 轨迹为空或距上次更新超过 trackingCacheTTL 时尝试实时刷新（需已配置快递100）
func ensureTrackingFresh(db *gorm.DB, card *model.CardRecord) string {
	if !kuaidi100Configured(db) {
		return ""
	}
	if !card.CardSent || card.TrackingNumber == "" {
		return ""
	}
	if t, err := parseCST(card.TrackingUpdatedAt); err == nil && time.Since(t) < trackingCacheTTL {
		return "" // 缓存未过期
	}
	details, status, err := fetchKuaidi100(db, card.TrackingNumber, card.TrackingCarrier)
	if err != nil {
		return err.Error()
	}
	if details == nil {
		return ""
	}
	raw, _ := json.Marshal(details)
	card.TrackingDetail = string(raw)
	card.TrackingStatus = status
	card.TrackingUpdatedAt = cstNow().Format(cstLayout)
	db.Save(card)
	return ""
}
