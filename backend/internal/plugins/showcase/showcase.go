// Package showcase 对外展示插件。
//
// 启用后多出三条出口。前两条不要登录，给别人的网页用；第三条要登录，给站长复制代码：
//
//	GET /api/public/ext/showcase            JSON：呼号、计数、最近通联、卡片图
//	GET /api/public/ext/showcase/badge.svg  同一份计数画成 SVG，用 <img> 引用
//	GET /api/ext/showcase/embed             返回上面两条的绝对地址和一段 HTML
//
// 公开页 /showcase 读第一条，给人看。
//
// QRZ 简介是网页，会丢掉 script 和 iframe，图片和链接留得住。
// 本站响应头也不允许被嵌进别人的 iframe，所以对外只给 <img> + <a>。
//
// 不公开：地址、电话、邮箱、备注、QTH、设备、申请正文。
// 数字和列表共用一份 60 秒快照，徽章和 JSON 不会各算各的。
package showcase

import (
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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

func New(db *gorm.DB) *Showcase { return &Showcase{db: db} }

func (p *Showcase) Meta() plugin.Info {
	return plugin.Info{
		Name:        "showcase",
		Title:       "对外展示",
		Version:     "1.0.0",
		Description: "把呼号、通联和卡片数量公开成只读数据。博客和 QRZ 简介贴同一段图片链接即可展示，不含地址、邮箱和备注。",
		Kind:        "feature",
	}
}

func (p *Showcase) Init(db *gorm.DB) error { return nil }

func (p *Showcase) Routes(rg *gin.RouterGroup) {
	rg.GET("/embed", p.embed)
}

func (p *Showcase) OpenRoutes(rg *gin.RouterGroup) {
	rg.GET("", p.feed)
	rg.GET("/badge.svg", p.badge)
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
}

var cache snapshotCache

func (p *Showcase) current() snapshot {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.ok && time.Since(cache.at) < snapshotTTL {
		return cache.val
	}
	s, ok := buildSnapshot(p.db)
	if !ok {
		if cache.ok {
			return cache.val
		}
		return snapshot{Recent: []qsoLine{}, Cards: []cardFace{}}
	}
	cache.val = s
	cache.at = time.Now()
	cache.ok = true
	return s
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
	s.CallSign = strings.TrimSpace(station.CallSign)
	s.Name = strings.TrimSpace(station.Name)
	s.NameEn = strings.TrimSpace(station.NameEn)

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
	if err := db.Model(&model.ReceiveRecord{}).Count(&s.Received).Error; err != nil {
		return s, false
	}

	if err := db.Model(&model.QsoRecord{}).
		Select("call_sign, date, time, band, mode, freq, rst_sent, rst_rcvd").
		Order("date DESC, time DESC, id DESC").
		Limit(recentCap).
		Find(&s.Recent).Error; err != nil {
		return s, false
	}
	if s.Recent == nil {
		s.Recent = []qsoLine{}
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
			CardVersion: card.CardVersion,
			ImagePath:   publicImagePath(card.ImagePath),
		})
	}
	return s, true
}

// publicImagePath 只放行已经公开的上传路径。其它值（本机路径、..）不带出去。
func publicImagePath(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "/uploads/") && !strings.Contains(p, "..") {
		return p
	}
	return ""
}

func (p *Showcase) feed(c *gin.Context) {
	limit := 8
	if d, err := strconv.Atoi(c.DefaultQuery("limit", "8")); err == nil && d >= 1 && d <= recentCap {
		limit = d
	}
	s := p.current()
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
	s := p.current()
	allowPublicRead(c)
	c.Data(200, "image/svg+xml; charset=utf-8", []byte(badgeSVG(s)))
}

// allowPublicRead 只加在这两条展示响应上。
// 简单 GET 不发预检，浏览器看这个头决定博客能不能读到 JSON。
// 不要把全站 CORS 改成 *：登录接口还带着凭据。
func allowPublicRead(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Cache-Control", "public, max-age=60")
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
	s := p.current()
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
// javascript:、纯路径、带账号的 URL 都不拿去拼到别人的网页里。
func publicBase(raw string) (string, bool) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	path := strings.TrimRight(u.EscapedPath(), "/")
	return u.Scheme + "://" + u.Host + path, u.Scheme == "https"
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
