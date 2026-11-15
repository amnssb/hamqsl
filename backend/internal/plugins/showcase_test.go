package plugins

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/internal/plugin"
)

func TestBuildSnapshotOmitsPrivateFields(t *testing.T) {
	db := openMem(t)
	db.Create(&model.StationProfile{
		CallSign:   "BH1TST",
		Name:       "测试台",
		NameEn:     "Test Station",
		Telephone:  "SECRET-TEL-9",
		PostalCode: "SECRET-POST-9",
		Address:    "SECRET-ADDR-9",
		AddressEn:  "SECRET-ADDR-EN-9",
		Email:      "secret@example.com",
		Remarks:    "SECRET-STATION-NOTE-9",
	})
	db.Create(&model.QsoRecord{
		RecordCode: "Q1", Date: "2026-01-01", Time: "08:00", CallSign: "OLD1",
		Band: "40m", Mode: "CW", Freq: "7.010", RstSent: "599", RstRcvd: "589",
		QTH: "SECRET-QTH-9", MyQTH: "SECRET-MYQTH-9", Operator: "SECRET-OP-9",
		Remarks: "SECRET-NOTE-9",
	})
	db.Create(&model.QsoRecord{
		RecordCode: "Q2", Date: "2026-11-01", Time: "12:30", CallSign: "NEW1",
		Band: "20m", Mode: "SSB", Freq: "14.200",
	})
	db.Create(&model.CardRecord{
		CardCode: "C1", CallSign: "NEW1", CardSent: true, ReceiptConfirmed: true,
		MailTargetEmail: "secret-card@example.com", BusinessRemarks: "SECRET-CARD-NOTE-9",
	})
	db.Create(&model.CardRecord{CardCode: "C2", CallSign: "OLD1", CardSent: false})
	// card_received 是「对方已收」，不计入 RX。RX 只数收卡记录。
	db.Create(&model.CardRecord{CardCode: "C3", CallSign: "NEW1", CardSent: true, CardReceived: true})
	db.Create(&model.ReceiveRecord{ReceiveCode: "R1", CallSign: "NEW1", ReceivedDate: "2026-11-02"})
	db.Create(&model.StationCard{CardVersion: "2026 春", ImagePath: "/uploads/card.png", IsActive: true, SortOrder: 1, Remarks: "SECRET-VERSION-NOTE-9"})
	db.Create(&model.StationCard{CardVersion: "内部", ImagePath: "/etc/passwd", IsActive: true, SortOrder: 2})
	db.Create(&model.StationCard{CardVersion: "绝对地址", ImagePath: "https://radio.example/uploads/from-url.png?x=1", IsActive: true, SortOrder: 3})
	// false 是零值，GORM 创建时会丢掉，库默认又是 true。先插入再改掉。
	off := model.StationCard{CardVersion: "停用", ImagePath: "/uploads/old.png", IsActive: true}
	db.Create(&off)
	db.Model(&off).Update("is_active", false)

	s, ok := buildSnapshot(db)
	if !ok {
		t.Fatal("快照应该成功")
	}
	if s.CallSign != "BH1TST" || s.Name != "测试台" || s.QSO != 2 || s.Sent != 2 || s.Signed != 1 || s.Received != 1 {
		t.Fatalf("计数或呼号不对: %+v", s)
	}
	if len(s.Recent) != 2 || s.Recent[0].CallSign != "NEW1" || s.Recent[0].Band != "20m" {
		t.Fatalf("最近通联应按日期倒序，实际 %+v", s.Recent)
	}
	if len(s.Cards) != 3 || s.Cards[0].ImagePath != "/uploads/card.png" || s.Cards[1].ImagePath != "" || s.Cards[2].ImagePath != "/uploads/from-url.png" {
		t.Fatalf("卡片图路径放行不对: %+v", s.Cards)
	}

	p := &Showcase{db: db}
	resetCache()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?limit=1", nil)
	p.feed(c)
	body := w.Body.String()
	for _, secret := range []string{
		"SECRET-TEL-9", "SECRET-POST-9", "SECRET-ADDR-9", "SECRET-ADDR-EN-9", "secret@example.com",
		"SECRET-STATION-NOTE-9", "SECRET-QTH-9", "SECRET-MYQTH-9", "SECRET-OP-9", "SECRET-NOTE-9",
		"secret-card@example.com", "SECRET-CARD-NOTE-9", "SECRET-VERSION-NOTE-9", "/etc/passwd",
	} {
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

func TestBadgeStripsMarkupAndControls(t *testing.T) {
	svg := badgeSVG(snapshot{CallSign: "A</title><script>alert(1)</script>\nB\x00\u202eC"})
	if strings.Contains(svg, "<script") || strings.Contains(svg, "</title><script") || strings.Contains(svg, "\n") || strings.Contains(svg, "\x00") {
		t.Fatalf("控制字符或标记进了 SVG: %s", svg)
	}
	if !strings.Contains(svg, "A&lt;/title&gt;&lt;script&gt;alert(1)&lt;/script&gt;BC") {
		t.Fatalf("呼号没有按文本转义: %s", svg)
	}
}

func TestPublicImagePath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/uploads/card.png", "/uploads/card.png"},
		{" /uploads/my..card.png ", "/uploads/my..card.png"},
		{"/uploads/card.png?v=1", "/uploads/card.png"},
		{"https://radio.example/uploads/from-url.png?x=1", "/uploads/from-url.png"},
		{"https://user:pw@radio.example/uploads/card.png", ""},
		{"https://evil.example/uploads/%2e%2e/%2e%2e/etc/passwd", ""},
		{"/uploads/../../etc/passwd", ""},
		{"/uploads/foo/bar.png", ""},
		{"/etc/passwd", ""},
		{"javascript:alert(1)", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := publicImagePath(tc.in); got != tc.want {
			t.Errorf("publicImagePath(%q) = %q，期望 %q", tc.in, got, tc.want)
		}
	}
}

func TestRecentOrderUsesWallClock(t *testing.T) {
	db := openMem(t)
	db.Create(&model.StationProfile{CallSign: "BH1TST", Name: "测试台"})
	// 8:05 的原文比 12:30 大；20261101 的原文也比 2026-11-02 大。按字符串倒序会把更早的通联排到前面。
	db.Create(&model.QsoRecord{RecordCode: "Q1", Date: "20261101", Time: "8:05", CallSign: "early1", Band: "40m"})
	db.Create(&model.QsoRecord{RecordCode: "Q2", Date: "2026-11-02", Time: "12:30", CallSign: "late1", Band: "20m"})
	db.Create(&model.QsoRecord{RecordCode: "Q3", Date: "2026-11-02", Time: "8:05", CallSign: "mid1", Band: "40m"})

	s, ok := buildSnapshot(db)
	if !ok || len(s.Recent) != 3 {
		t.Fatalf("应有 3 条通联: %+v ok=%v", s.Recent, ok)
	}
	got := []string{s.Recent[0].CallSign, s.Recent[1].CallSign, s.Recent[2].CallSign}
	want := []string{"LATE1", "MID1", "EARLY1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("最近通联顺序应为 %v，实际 %v", want, got)
	}
	if s.Recent[1].Time != "08:05" || s.Recent[2].Date != "2026-11-01" {
		t.Fatalf("对外格式不对: %+v", s.Recent)
	}
}

func TestWallClockAndCall(t *testing.T) {
	if got := publicCall(" bi1abc \n"); got != "BI1ABC" {
		t.Fatalf("呼号应去掉空白并转大写，得到 %q", got)
	}
	if got := wallDate("20261101"); got != "2026-11-01" {
		t.Fatalf("紧凑日期应变成 YYYY-MM-DD，得到 %q", got)
	}
	if got := wallDate("2026-11-01T00:30:00Z"); got != "2026-11-01" {
		t.Fatalf("日期字段只取日历日，得到 %q", got)
	}
	if got := wallDate("not-a-date"); got != "not-a-date" {
		t.Fatalf("认不出的日期应原样留下，得到 %q", got)
	}
	for _, tc := range []struct{ in, want string }{
		{"12:30", "12:30"},
		{"12:30:45", "12:30"},
		{"9:05", "09:05"},
		{"093045", "09:30"},
		{"99:99", "99:99"},
	} {
		if got := wallClock(tc.in); got != tc.want {
			t.Errorf("wallClock(%q) = %q，期望 %q", tc.in, got, tc.want)
		}
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
	if base, ok := publicBase("https://radio.example:8443/qsl"); base != "https://radio.example:8443/qsl" || !ok {
		t.Fatalf("端口应保留，实际 %q %v", base, ok)
	}
	for _, raw := range []string{"", "/showcase", "javascript:alert(1)", "https://user:pw@radio.example", "https://radio.example/qsl/../admin", "https://radio.example/qsl/%2e%2e/admin", "https://radio.example\r\nX"} {
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

func TestEmbedEscapesCallSign(t *testing.T) {
	db := openMem(t)
	db.Create(&model.SystemSetting{Key: "site_url", Value: "https://radio.example/qsl/"})
	db.Create(&model.StationProfile{CallSign: `bh1tst"><script>alert(1)</script>`})
	resetCache()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	(&Showcase{db: db}).embed(c)

	var body struct {
		Data struct {
			HTML  string `json:"html"`
			HTTPS bool   `json:"https"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON: %v %s", err, w.Body.String())
	}
	html := body.Data.HTML
	if !body.Data.HTTPS || !strings.Contains(html, "https://radio.example/qsl/showcase") {
		t.Fatalf("嵌入地址不对: %s", html)
	}
	if strings.Contains(html, "<script") || strings.Contains(html, `bh1tst">`) {
		t.Fatalf("呼号被当成 HTML: %s", html)
	}
	if !strings.Contains(html, "BH1TST&#34;&gt;&lt;SCRIPT&gt;ALERT(1)&lt;/SCRIPT&gt;") {
		t.Fatalf("呼号没有按属性转义: %s", html)
	}
}

func TestSnapshotRefreshesAfterWrite(t *testing.T) {
	db := openMem(t)
	db.Create(&model.StationProfile{CallSign: "bh1tst", Name: "测试台"})
	p := &Showcase{db: db}
	if err := p.Init(db); err != nil {
		t.Fatal(err)
	}
	if err := p.Init(db); err != nil {
		t.Fatal(err)
	}
	resetCache()

	s, ok := p.current()
	if !ok || s.QSO != 0 || s.CallSign != "BH1TST" {
		t.Fatalf("初始快照不对: %+v ok=%v", s, ok)
	}
	db.Create(&model.QsoRecord{
		RecordCode: "Q1", Date: "20261101", Time: "093045", CallSign: "bi1abc",
		Band: "20m", Mode: "SSB",
	})
	s, ok = p.current()
	if !ok || s.QSO != 1 || len(s.Recent) != 1 {
		t.Fatalf("写入后应立刻看到新通联，实际 %+v ok=%v", s, ok)
	}
	line := s.Recent[0]
	if line.CallSign != "BI1ABC" || line.Date != "2026-11-01" || line.Time != "09:30" {
		t.Fatalf("呼号或时间格式不对: %+v", line)
	}
}

func TestFeedUsesStaleSnapshotThen503(t *testing.T) {
	db := openMem(t)
	db.Create(&model.StationProfile{CallSign: "BH1TST", Name: "测试台"})
	db.Create(&model.QsoRecord{RecordCode: "Q1", Date: "2026-11-01", Time: "12:30", CallSign: "BI1ABC"})
	p := &Showcase{db: db}
	resetCache()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	p.feed(c)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "BI1ABC") {
		t.Fatalf("第一份应成功: %d %s", w.Code, w.Body.String())
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	p.feed(c)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "BI1ABC") {
		t.Fatalf("数据库抖动时应继续用上一份，实际 %d %s", w.Code, w.Body.String())
	}

	resetCache()
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/badge.svg", nil)
	p.badge(c)
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "<svg") || strings.Contains(w.Body.String(), ">0<") {
		t.Fatalf("没有快照时不应回一张 0 徽章: %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("失败响应也要能被跨域读到且不能缓存: %v", w.Header())
	}
	if !strings.Contains(w.Body.String(), "QSL-503") {
		t.Fatalf("应返回 QSL-503: %s", w.Body.String())
	}
}

func TestPublicRoutesWhenPluginOffAndOn(t *testing.T) {
	db := openMem(t)
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.StationProfile{
		CallSign: "bh1tst", Name: "测试台", Email: "secret@example.com", Address: "SECRET-ADDR-9",
	})
	db.Create(&model.User{Username: "admin", HashPassword: "x", IsActive: true})

	p := &Showcase{db: db}
	m := plugin.NewManager(db)
	m.Register(p)
	r := gin.New()
	r.Use(middleware.CORSMiddleware())
	api := r.Group("/api")
	m.Mount(api)
	m.MountOpen(api.Group("/public"))

	get := func(method, path, origin, auth string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		r.ServeHTTP(w, req)
		return w
	}

	off := get(http.MethodGet, "/api/public/ext/showcase", "https://blog.example", "")
	if off.Code != http.StatusNotFound || !strings.Contains(off.Body.String(), "QSL-PLUGIN-OFF") {
		t.Fatalf("禁用时应 404，实际 %d %s", off.Code, off.Body.String())
	}
	if strings.Contains(off.Body.String(), "BH1TST") || strings.Contains(off.Body.String(), "secret@example.com") || strings.Contains(off.Body.String(), "SECRET-ADDR-9") {
		t.Fatalf("禁用时不应带出数据: %s", off.Body.String())
	}
	if off.Header().Get("Access-Control-Allow-Origin") != "*" || off.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("禁用的 404 要能被跨域读到且不缓存: %v", off.Header())
	}
	if off.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("公开响应不能同时带凭据头")
	}

	pre := get(http.MethodOptions, "/api/public/ext/showcase", "https://blog.example", "")
	if pre.Code != http.StatusNoContent || pre.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("站外预检应 204，实际 %d %v", pre.Code, pre.Header())
	}

	loginPre := get(http.MethodOptions, "/api/auth/login", "https://blog.example", "")
	if loginPre.Code != http.StatusForbidden {
		t.Fatalf("登录接口的站外预检仍应拒绝，实际 %d", loginPre.Code)
	}

	if got := get(http.MethodGet, "/api/ext/showcase/embed", "", ""); got.Code != http.StatusUnauthorized {
		t.Fatalf("嵌入代码未登录应 401，实际 %d %s", got.Code, got.Body.String())
	}

	resetCache()
	if err := m.SetEnabled("showcase", true); err != nil {
		t.Fatal(err)
	}
	on := get(http.MethodGet, "/api/public/ext/showcase?limit=8", "http://localhost:3000", "")
	if on.Code != http.StatusOK || !strings.Contains(on.Body.String(), "BH1TST") {
		t.Fatalf("启用后应返回大写呼号，实际 %d %s", on.Code, on.Body.String())
	}
	if strings.Contains(on.Body.String(), "secret@example.com") || strings.Contains(on.Body.String(), "SECRET-ADDR-9") {
		t.Fatalf("启用后仍不应带出地址和邮箱: %s", on.Body.String())
	}
	if on.Header().Get("Access-Control-Allow-Origin") != "*" || on.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("开发来源也不能把 * 和凭据头放在一起: %v", on.Header())
	}
	if on.Header().Get("Cross-Origin-Resource-Policy") != "cross-origin" {
		t.Fatalf("徽章和 JSON 要允许被别的网页引用: %v", on.Header())
	}

	middleware.SetSecret("showcase-test-secret")
	access, _, err := middleware.GenerateToken("admin", 0, []string{"pwd"})
	if err != nil {
		t.Fatal(err)
	}
	embed := get(http.MethodGet, "/api/ext/showcase/embed", "", "Bearer "+access)
	if embed.Code != http.StatusOK {
		t.Fatalf("登录后应能取嵌入代码，实际 %d %s", embed.Code, embed.Body.String())
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
	invalidateShowcaseCache()
}
