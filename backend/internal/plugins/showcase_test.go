package plugins

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qsl-management/internal/model"
)

func TestBuildSnapshotOmitsPrivateFields(t *testing.T) {
	db := openMem(t)
	db.Create(&model.StationProfile{
		CallSign:  "BH1TST",
		Name:      "测试台",
		NameEn:    "Test Station",
		Telephone: "SECRET-TEL-9",
		Address:   "SECRET-ADDR-9",
		Email:     "secret@example.com",
	})
	db.Create(&model.QsoRecord{
		RecordCode: "Q1", Date: "2026-01-01", Time: "08:00", CallSign: "OLD1",
		Band: "40m", Mode: "CW", Freq: "7.010", RstSent: "599", RstRcvd: "589",
		QTH: "SECRET-QTH-9", Remarks: "SECRET-NOTE-9",
	})
	db.Create(&model.QsoRecord{
		RecordCode: "Q2", Date: "2026-11-01", Time: "12:30", CallSign: "NEW1",
		Band: "20m", Mode: "SSB", Freq: "14.200",
	})
	db.Create(&model.CardRecord{CardCode: "C1", CallSign: "NEW1", CardSent: true, ReceiptConfirmed: true})
	db.Create(&model.CardRecord{CardCode: "C2", CallSign: "OLD1", CardSent: false})
	db.Create(&model.ReceiveRecord{ReceiveCode: "R1", CallSign: "NEW1", ReceivedDate: "2026-11-02"})
	db.Create(&model.StationCard{CardVersion: "2026 春", ImagePath: "/uploads/card.png", IsActive: true, SortOrder: 1})
	db.Create(&model.StationCard{CardVersion: "内部", ImagePath: "/etc/passwd", IsActive: true, SortOrder: 2})
	// false 是零值，GORM 创建时会丢掉，库默认又是 true。先插入再改掉。
	off := model.StationCard{CardVersion: "停用", ImagePath: "/uploads/old.png", IsActive: true}
	db.Create(&off)
	db.Model(&off).Update("is_active", false)

	s, ok := buildSnapshot(db)
	if !ok {
		t.Fatal("快照应该成功")
	}
	if s.CallSign != "BH1TST" || s.Name != "测试台" || s.QSO != 2 || s.Sent != 1 || s.Signed != 1 || s.Received != 1 {
		t.Fatalf("计数或呼号不对: %+v", s)
	}
	if len(s.Recent) != 2 || s.Recent[0].CallSign != "NEW1" || s.Recent[0].Band != "20m" {
		t.Fatalf("最近通联应按日期倒序，实际 %+v", s.Recent)
	}
	if len(s.Cards) != 2 || s.Cards[0].ImagePath != "/uploads/card.png" || s.Cards[1].ImagePath != "" {
		t.Fatalf("卡片图路径放行不对: %+v", s.Cards)
	}

	p := &Showcase{db: db}
	resetCache()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?limit=1", nil)
	p.feed(c)
	body := w.Body.String()
	for _, secret := range []string{"SECRET-TEL-9", "SECRET-ADDR-9", "secret@example.com", "SECRET-QTH-9", "SECRET-NOTE-9", "/etc/passwd"} {
		if strings.Contains(body, secret) {
			t.Errorf("响应里不该出现 %s", secret)
		}
	}
	if !strings.Contains(body, "NEW1") || strings.Contains(body, "OLD1") {
		t.Fatalf("limit=1 应只剩最新一条，实际 %s", body)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("展示 JSON 应允许任意来源读取")
	}
}

func TestBadgeEscapesCallsign(t *testing.T) {
	svg := badgeSVG(snapshot{CallSign: `A<B>&"C`, QSO: 3, Sent: 2, Signed: 1, Received: 4})
	if strings.Contains(svg, "<script") || strings.Contains(svg, `A<B>`) {
		t.Fatalf("呼号被当成标记写进了 SVG: %s", svg)
	}
	if !strings.Contains(svg, "A&lt;B&gt;&amp;&#34;C") {
		t.Fatalf("呼号没有按 XML 转义: %s", svg)
	}
	if !strings.Contains(svg, ">3<") || !strings.Contains(svg, ">4<") {
		t.Fatalf("计数没有写进徽章: %s", svg)
	}
}

func TestPublicBase(t *testing.T) {
	base, httpsOK := publicBase(" https://radio.example/qsl/ ")
	if base != "https://radio.example/qsl" || !httpsOK {
		t.Fatalf("https 地址应保留路径，实际 %q %v", base, httpsOK)
	}
	if _, ok := publicBase("http://radio.example"); ok {
		t.Fatal("http 不算 https")
	}
	for _, raw := range []string{"", "/showcase", "javascript:alert(1)", "https://user:pw@radio.example"} {
		if base, ok := publicBase(raw); ok || base != "" {
			t.Fatalf("%q 不该被当成站点地址，得到 %q", raw, base)
		}
	}
}

func TestEmbedRequiresHTTPSSite(t *testing.T) {
	db := openMem(t)
	db.Create(&model.SystemSetting{Key: "site_url", Value: "https://radio.example"})
	db.Create(&model.StationProfile{CallSign: "BH1TST", Name: "测试台"})
	resetCache()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	(&Showcase{db: db}).embed(c)
	body := w.Body.String()
	if !strings.Contains(body, "https://radio.example/api/public/ext/showcase/badge.svg") {
		t.Fatalf("徽章地址应拼在站点地址后面: %s", body)
	}
	if !strings.Contains(body, "BH1TST") || !strings.Contains(body, `"https":true`) {
		t.Fatalf("嵌入信息缺呼号或 https 标记: %s", body)
	}
}

func openMem(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&model.StationProfile{},
		&model.QsoRecord{},
		&model.CardRecord{},
		&model.ReceiveRecord{},
		&model.StationCard{},
		&model.SystemSetting{},
	); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	return db
}

func resetCache() {
	cache.mu.Lock()
	cache.ok = false
	cache.at = time.Time{}
	cache.mu.Unlock()
}
