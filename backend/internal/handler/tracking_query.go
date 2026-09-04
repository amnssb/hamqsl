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

type kuaidi100Resp struct {
	Message string `json:"message"`
	Nu      string `json:"nu"`
	State   string `json:"state"`
	Data []struct {
		Time    string `json:"time"`
		Context string `json:"context"`
		Fcity   string `json:"fcity"`
	} `json:"data"`
}

func getSettingValue(db *gorm.DB, key string) string {
	var s model.SystemSetting
	if err := db.Where("key = ?", key).First(&s).Error; err != nil {
		return ""
	}
	return s.Value
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

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm("https://poll.kuaidi100.com/poll/query.do", url.Values{
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

// ensureTrackingFresh 轨迹为空或距上次更新超过 10 分钟时尝试实时刷新（需已配置快递100）
func ensureTrackingFresh(db *gorm.DB, card *model.CardRecord) string {
	if !kuaidi100Configured(db) {
		return ""
	}
	if !card.CardSent || card.TrackingNumber == "" {
		return ""
	}
	if t, err := time.Parse("2006-01-02 15:04:05", card.TrackingUpdatedAt); err == nil && time.Since(t) < 10*time.Minute {
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
	card.TrackingUpdatedAt = cstNow().Format("2006-01-02 15:04:05")
	db.Save(card)
	return ""
}

