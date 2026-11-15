package plugins

// 对外展示。启用后多出三条出口。前两条不要登录，给别人的网页用；第三条要登录，给站长复制代码：
//
//	GET /api/public/ext/showcase            JSON：呼号、计数、最近通联、卡片图
//	GET /api/public/ext/showcase/badge.svg  同一份计数画成 SVG，用 <img> 引用
//	GET /api/ext/showcase/embed             返回上面两条的绝对地址和一段 HTML
//
// 外部站点对接的字段契约（实测响应、错误码、示例代码）见 docs/showcase-api.md。
// 公开页 /showcase 读第一条。QRZ 简介和博客会丢掉 script 和 iframe，所以对外只给图片和链接。
// 不公开：地址、电话、邮箱、备注、QTH、设备、申请正文。
// 数字和列表共用一份快照，徽章和 JSON 不会各算各的。
// 快照最多留 60 秒；通联、卡片、收卡、本台资料一写入就作废，避免和后台对不上。
// 响应头仍是 Cache-Control: public, max-age=60，已经贴出去的 <img> 最多再旧一分钟。

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/internal/plugin"
	"qsl-management/pkg/response"
)

const (
	snapshotTTL = 60 * time.Second
	recentCap   = 20
	cardCap     = 6
)

// Showcase 只读展示。不建表，直接读通联、卡片和本台呼号。
type Showcase struct{ db *gorm.DB }

func init() {
	// 70：排在每日统计后面。这个数字只决定插件页上的先后。
	plugin.Add(70, func(db *gorm.DB) plugin.Plugin { return &Showcase{db: db} })
}

func (p *Showcase) Meta() plugin.Info {
	return plugin.Info{
		Name:        "showcase",
		Title:       "对外展示",
		Version:     "1.0.1",
		Description: "把呼号、通联和卡片数量公开成只读数据。博客和 QRZ 简介贴同一段图片链接即可展示，不含地址、邮箱和备注。",
		Kind:        "feature",
	}
}

func (p *Showcase) Init(db *gorm.DB) error {
	registerShowcaseInvalidation(db)
	return nil
}

func (p *Showcase) Routes(rg *gin.RouterGroup) {
	rg.GET("/embed", p.embed)
}

func (p *Showcase) OpenRoutes(rg *gin.RouterGroup) {
	rg.GET("", p.feed)
	rg.GET("/badge.svg", p.badge)
	// 预检要自己回 204。全局 CORS 对站外来源不再直接 403，好让这里把 * 写上。
	rg.OPTIONS("", p.preflight)
	rg.OPTIONS("/badge.svg", p.preflight)
}

type qsoLine struct {
	CallSign string `json:"call_sign"`
	Date     string `json:"date"`
	Time     string `json:"time"`
	Band     string `json:"band"`
	Mode     string `json:"mode"`
	Freq     string `json:"freq"`
	RstSent  string `json:"rst_sent"`
	RstRcvd  string `json:"rst_rcvd"`
}

type cardFace struct {
	CardVersion string `json:"card_version"`
	ImagePath   string `json:"image_path,omitempty"`
}

type snapshot struct {
	CallSign string
	Name     string
	NameEn   string
	QSO      int64
	Sent     int64
	Signed   int64
	Received int64
	Recent   []qsoLine
	Cards    []cardFace
}

type snapshotCache struct {
	mu  sync.Mutex
	at  time.Time
	val snapshot
	ok  bool
	gen uint64
}

var cache snapshotCache

// showcaseHooks 记住已经挂过失效回调的库。Init 在启动和每次启用时都会跑。
var showcaseHooks sync.Map

// registerShowcaseInvalidation 在写入提交之后丢掉快照。
//
// 挂在 gorm:commit_or_rollback_transaction 后面：测试库默认包事务，这样读到的是已提交的行。
// 生产配置 SkipDefaultTransaction 时 GORM 会拿掉那个回调，这条钩子就落在写入之后，效果一样。
func registerShowcaseInvalidation(db *gorm.DB) {
	if db == nil {
		return
	}
	if _, loaded := showcaseHooks.LoadOrStore(db, struct{}{}); loaded {
		return
	}
	hook := func(tx *gorm.DB) {
		if tx.Error != nil {
			return
		}
		invalidateShowcaseCache()
	}
	_ = db.Callback().Create().After("gorm:commit_or_rollback_transaction").Register("showcase:invalidate_create", hook)
	_ = db.Callback().Update().After("gorm:commit_or_rollback_transaction").Register("showcase:invalidate_update", hook)
	_ = db.Callback().Delete().After("gorm:commit_or_rollback_transaction").Register("showcase:invalidate_delete", hook)
}

func invalidateShowcaseCache() {
	cache.mu.Lock()
	cache.ok = false
	cache.at = time.Time{}
	cache.gen++
	cache.mu.Unlock()
}

// current 返回快照。第二个返回值表示这份数据能不能对外展示。
// 数据库暂时失败时继续用上一份，避免徽章在抖动时变成全 0；一份都没有才算失败。
// 读库时不占着锁。若这期间有写入，gen 对不上，这次结果不放进缓存，下一请求再读。
func (p *Showcase) current() (snapshot, bool) {
	cache.mu.Lock()
	if cache.ok && time.Since(cache.at) < snapshotTTL {
		s := cache.val
		cache.mu.Unlock()
		return s, true
	}
	gen := cache.gen
	cache.mu.Unlock()

	s, ok := buildSnapshot(p.db)

	cache.mu.Lock()
	defer cache.mu.Unlock()
	if !ok {
		if cache.ok && cache.gen == gen {
			return cache.val, true
		}
		if cache.ok && time.Since(cache.at) < snapshotTTL {
			return cache.val, true
		}
		return snapshot{Recent: []qsoLine{}, Cards: []cardFace{}}, false
	}
	if cache.gen != gen {
		return s, true
	}
	cache.val = s
	cache.at = time.Now()
	cache.ok = true
	return s, true
}

// buildSnapshot 只选出对外展示用得到的列。
// 备注、地址、QTH 不在 SELECT 里，响应里就不会出现。
func buildSnapshot(db *gorm.DB) (snapshot, bool) {
	var s snapshot
	s.Recent = []qsoLine{}
	s.Cards = []cardFace{}

	var station model.StationProfile
	err := db.Model(&model.StationProfile{}).
		Select("call_sign, name, name_en").
		Order("id").
		First(&station).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return s, false
	}
	// 呼号全站都是大写。这里再转一次，库里残留的小写不会出现在徽章上。
	s.CallSign = publicCall(station.CallSign)
	s.Name = cleanText(station.Name)
	s.NameEn = cleanText(station.NameEn)

	if err := db.Model(&model.QsoRecord{}).Count(&s.QSO).Error; err != nil {
		return s, false
	}
	var agg struct {
		Sent   int64
		Signed int64
	}
	if err := db.Model(&model.CardRecord{}).Select(`
		COALESCE(SUM(CASE WHEN card_sent THEN 1 ELSE 0 END), 0) AS sent,
		COALESCE(SUM(CASE WHEN receipt_confirmed THEN 1 ELSE 0 END), 0) AS signed
	`).Scan(&agg).Error; err != nil {
		return s, false
	}
	s.Sent = agg.Sent
	s.Signed = agg.Signed
	// cards_received 是收卡记录的行数（已经入库的回卡），不是 card_received。
	// card_received 表示「对方已收我们寄出的卡」，首页签收数字不看它，徽章上的 RX 也不看它。
	if err := db.Model(&model.ReceiveRecord{}).Count(&s.Received).Error; err != nil {
		return s, false
	}

	// 通联列表按原文字符串排序。日期选择器给出的是 YYYY-MM-DD，时间框却允许写成 8:05。
	// 按原文倒序时，8:05 会排到 12:30 前面，8 位紧凑日期也会排到更晚的日历日前面。
	// 这里先收成可比较的日历日和补零时刻，SQLite 和 PostgreSQL 都认 substr / ||。
	if err := db.Model(&model.QsoRecord{}).
		Select("call_sign, date, time, band, mode, freq, rst_sent, rst_rcvd").
		Order(`CASE WHEN length(date) = 8 THEN substr(date, 1, 4) || '-' || substr(date, 5, 2) || '-' || substr(date, 7, 2) ELSE date END DESC, ` +
			`CASE WHEN length(time) >= 4 AND substr(time, 2, 1) = ':' THEN '0' || time ELSE time END DESC, id DESC`).
		Limit(recentCap).
		Find(&s.Recent).Error; err != nil {
		return s, false
	}
	if s.Recent == nil {
		s.Recent = []qsoLine{}
	}
	for i := range s.Recent {
		s.Recent[i].CallSign = publicCall(s.Recent[i].CallSign)
		s.Recent[i].Date = wallDate(s.Recent[i].Date)
		s.Recent[i].Time = wallClock(s.Recent[i].Time)
		s.Recent[i].Band = cleanText(s.Recent[i].Band)
		s.Recent[i].Mode = cleanText(s.Recent[i].Mode)
		s.Recent[i].Freq = cleanText(s.Recent[i].Freq)
		s.Recent[i].RstSent = cleanText(s.Recent[i].RstSent)
		s.Recent[i].RstRcvd = cleanText(s.Recent[i].RstRcvd)
	}

	var cards []model.StationCard
	if err := db.Model(&model.StationCard{}).
		Select("card_version, image_path").
		Where("is_active = ?", true).
		Order("sort_order, id").
		Limit(cardCap).
		Find(&cards).Error; err != nil {
		return s, false
	}
	for _, card := range cards {
		s.Cards = append(s.Cards, cardFace{
			CardVersion: cleanText(card.CardVersion),
			ImagePath:   publicImagePath(card.ImagePath),
		})
	}
	return s, true
}

// publicImagePath 只放行本站上传目录里的一个文件名。
//
// 后台有时存的是浏览器地址栏里的绝对地址（http://主机/uploads/xxx.png）。
// 原样丢掉的话，公开页和外部引用都会少图。这里只取出路径，仍然由本站 /uploads 提供。
// 带查询串、目录穿越、盘符路径一律不发出去。
func publicImagePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if strings.Contains(p, "://") {
		u, err := url.Parse(p)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			return ""
		}
		p = u.Path
	} else if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if dec, err := url.PathUnescape(p); err == nil {
		p = dec
	}
	p = path.Clean(p)
	if !strings.HasPrefix(p, "/uploads/") {
		return ""
	}
	name := path.Base(p)
	if name != strings.TrimPrefix(p, "/uploads/") || !safeUploadName(name) {
		return ""
	}
	return "/uploads/" + name
}

func safeUploadName(name string) bool {
	if name == "" || name == "." || name == ".." || len(name) > 200 {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

// publicCall 对外呼号：去掉控制字符和双向文字覆盖，再转成大写。
func publicCall(s string) string {
	return strings.ToUpper(cleanText(s))
}

// cleanText 去掉不能出现在 XML / HTML 文本里的控制字符，以及用来伪造呼号方向的双向控制符。
func cleanText(s string) string {
	s = strings.TrimSpace(s)
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		switch r {
		case 0x202A, 0x202B, 0x202C, 0x202D, 0x202E, 0x2066, 0x2067, 0x2068, 0x2069:
			return -1
		default:
			return r
		}
	}, s)
}

// wallDate 把认得出的日期收成 YYYY-MM-DD。认不出就原样返回，不把乱字符串改成一个假日期。
func wallDate(s string) string {
	s = cleanText(s)
	raw := s
	if len(s) == 8 && digits(s) {
		s = s[:4] + "-" + s[4:6] + "-" + s[6:]
	}
	if len(s) >= 10 {
		if _, err := time.Parse("2006-01-02", s[:10]); err == nil {
			return s[:10]
		}
	}
	return raw
}

// wallClock 把认得出的时刻收成 HH:mm（UTC+8 墙钟，和通联列表里存的是同一串，这里只补零、去掉秒）。
func wallClock(s string) string {
	s = cleanText(s)
	h, m, ok := splitClock(s)
	if !ok || h > 23 || m > 59 {
		return s
	}
	return fmt.Sprintf("%02d:%02d", h, m)
}

func splitClock(s string) (h, m int, ok bool) {
	switch {
	case len(s) >= 5 && s[2] == ':' && digits(s[:2]) && digits(s[3:5]):
		h, _ = strconv.Atoi(s[:2])
		m, _ = strconv.Atoi(s[3:5])
		return h, m, true
	case len(s) >= 4 && s[1] == ':' && digits(s[:1]) && digits(s[2:4]):
		h, _ = strconv.Atoi(s[:1])
		m, _ = strconv.Atoi(s[2:4])
		return h, m, true
	case len(s) >= 4 && digits(s[:4]):
		h, _ = strconv.Atoi(s[:2])
		m, _ = strconv.Atoi(s[2:4])
		return h, m, true
	default:
		return 0, 0, false
	}
}

func digits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (p *Showcase) feed(c *gin.Context) {
	limit := 8
	if d, err := strconv.Atoi(c.DefaultQuery("limit", "8")); err == nil && d >= 1 && d <= recentCap {
		limit = d
	}
	s, ok := p.current()
	if !ok {
		allowPublicError(c)
		response.FailCode(c, http.StatusServiceUnavailable, "QSL-503", "展示数据暂时不可用")
		return
	}
	recent := s.Recent
	if len(recent) > limit {
		recent = recent[:limit]
	}
	allowPublicRead(c)
	response.OK(c, gin.H{
		"station": gin.H{
			"call_sign": s.CallSign,
			"name":      s.Name,
			"name_en":   s.NameEn,
		},
		"counts": gin.H{
			"qso":            s.QSO,
			"cards_sent":     s.Sent,
			"cards_signed":   s.Signed,
			"cards_received": s.Received,
		},
		"recent": recent,
		"cards":  s.Cards,
	})
}

func (p *Showcase) badge(c *gin.Context) {
	s, ok := p.current()
	if !ok {
		// 不要回一张写着 0 的徽章。QRZ 上会把故障看成「这台没有通联」。
		allowPublicError(c)
		response.FailCode(c, http.StatusServiceUnavailable, "QSL-503", "展示数据暂时不可用")
		return
	}
	allowPublicRead(c)
	c.Data(http.StatusOK, "image/svg+xml; charset=utf-8", []byte(badgeSVG(s)))
}

func (p *Showcase) preflight(c *gin.Context) {
	allowPublicRead(c)
	c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Accept, Content-Type")
	c.Header("Access-Control-Max-Age", "600")
	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// allowPublicRead 只加在展示响应上。
// 简单 GET 不发预检，浏览器看这个头决定博客能不能读到 JSON。
// 不要把全站 CORS 改成 *：登录接口还带着凭据。
func allowPublicRead(c *gin.Context) {
	middleware.AllowPublicEmbed(c)
	c.Header("Cache-Control", "public, max-age=60")
}

func allowPublicError(c *gin.Context) {
	middleware.AllowPublicEmbed(c)
	c.Header("Cache-Control", "no-store")
}

type embedBody struct {
	SiteURL  string `json:"site_url"`
	PageURL  string `json:"page_url"`
	FeedURL  string `json:"feed_url"`
	BadgeURL string `json:"badge_url"`
	HTML     string `json:"html"`
	HTTPS    bool   `json:"https"`
}

func (p *Showcase) embed(c *gin.Context) {
	base, httpsOK := publicBase(settingValue(p.db, "site_url"))
	s, _ := p.current()
	body := embedBody{SiteURL: base, HTTPS: httpsOK}
	if base == "" {
		response.OK(c, body)
		return
	}
	body.PageURL = base + "/showcase"
	body.FeedURL = base + "/api/public/ext/showcase"
	body.BadgeURL = base + "/api/public/ext/showcase/badge.svg"
	alt := s.CallSign
	if alt == "" {
		alt = "QSL"
	}
	body.HTML = `<a href="` + htmlEscape(body.PageURL) + `" target="_blank" rel="noopener"><img src="` +
		htmlEscape(body.BadgeURL) + `" width="480" height="132" alt="` + htmlEscape(alt) + ` QSL"></a>`
	response.OK(c, body)
}

// publicBase 只接受 http(s) 且带主机名的站点地址。
// javascript:、纯路径、带账号、换行、目录穿越都不拿去拼到别人的网页里。
// 和邮件里的站点地址用同一套底线：请求头或设置值不能把链接改到别的主机。
func publicBase(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, "\r\n\\") {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Opaque != "" {
		return "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	if strings.ContainsAny(u.Host, "\r\n/\\ @") {
		return "", false
	}
	p := u.EscapedPath()
	if dec, err := url.PathUnescape(p); err == nil {
		p = dec
	}
	if strings.Contains(p, "..") {
		return "", false
	}
	p = path.Clean(p)
	if p == "." || p == "/" {
		p = ""
	} else {
		p = strings.TrimRight(p, "/")
	}
	return u.Scheme + "://" + u.Host + p, u.Scheme == "https"
}

func settingValue(db *gorm.DB, key string) string {
	var row model.SystemSetting
	if err := db.Select("value").Where("key = ?", key).First(&row).Error; err != nil {
		return ""
	}
	return row.Value
}

func htmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&#34;",
		"'", "&#39;",
	).Replace(s)
}

// badgeSVG 画一张 480×132 的徽章。颜色写死：它显示在别人的网页上，读不到本站主题变量。
// 文字只用 XML 转义后的呼号和数字，不插入任何标记。
func badgeSVG(s snapshot) string {
	call := s.CallSign
	if call == "" {
		call = "QSL"
	}
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="480" height="132" viewBox="0 0 480 132" role="img">`)
	b.WriteString(`<title>` + xmlEscape(call) + ` QSL</title>`)
	b.WriteString(`<rect width="480" height="132" rx="16" fill="#1a2d3d"/>`)
	b.WriteString(`<rect width="8" height="132" fill="#f5a623"/>`)
	b.WriteString(`<text x="28" y="52" fill="#ffffff" font-family="Segoe UI, PingFang SC, Microsoft YaHei, sans-serif" font-size="28" font-weight="700">`)
	b.WriteString(xmlEscape(call))
	b.WriteString(`</text>`)
	b.WriteString(`<text x="452" y="46" fill="#f5a623" font-family="Segoe UI, PingFang SC, Microsoft YaHei, sans-serif" font-size="13" font-weight="700" text-anchor="end">QSL</text>`)
	writeStat(&b, 28, "QSO", s.QSO)
	writeStat(&b, 140, "SENT", s.Sent)
	writeStat(&b, 252, "SIGNED", s.Signed)
	writeStat(&b, 364, "RX", s.Received)
	b.WriteString(`</svg>`)
	return b.String()
}

func writeStat(b *strings.Builder, x int, label string, n int64) {
	b.WriteString(`<text x="` + strconv.Itoa(x) + `" y="84" fill="#9c978d" font-family="Segoe UI, PingFang SC, Microsoft YaHei, sans-serif" font-size="11">`)
	b.WriteString(label)
	b.WriteString(`</text>`)
	b.WriteString(`<text x="` + strconv.Itoa(x) + `" y="110" fill="#ffffff" font-family="Segoe UI, PingFang SC, Microsoft YaHei, sans-serif" font-size="20" font-weight="700">`)
	b.WriteString(strconv.FormatInt(n, 10))
	b.WriteString(`</text>`)
}

func xmlEscape(s string) string {
	s = cleanText(s)
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&#34;",
		"'", "&#39;",
	).Replace(s)
}
