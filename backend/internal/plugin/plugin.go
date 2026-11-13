package plugin

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

// pluginStateTTL 启用状态快照的存活时间。
// 开关本身走 SetEnabled 主动失效，这个 TTL 只是兜底（比如直接改库的情况）。
const pluginStateTTL = 5 * time.Second

// Info 插件元数据
type Info struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Kind        string `json:"kind"` // feature = 功能插件（注入路由）| theme = 主题插件（前端皮肤，后端只管启停）
}

// Plugin 插件接口。
// 主题类插件只需 Meta + 空实现；功能类插件在 Routes 中向 /api/ext/<name> 组注入路由，
// 该组已自动携带认证与启停守卫（插件禁用时路由返回 404 QSL-PLUGIN-OFF）。
type Plugin interface {
	Meta() Info
	// Init 启动时对已启用插件调用一次（建表/迁移/预热），失败仅记日志不阻塞启动
	Init(db *gorm.DB) error
	// Routes 向 /api/ext/<name> 组注册路由（主题类留空）。这组要登录。
	Routes(rg *gin.RouterGroup)
}

// OpenReader 可选。功能插件再实现它，就会多一组无需登录的只读路由
// /api/public/ext/<name>，仍受启停守卫（禁用返回 404）。
// 给博客、QRZ 简介这种外部页面取展示数据。不要在这里放写操作。
type OpenReader interface {
	OpenRoutes(rg *gin.RouterGroup)
}

// SkinProvider 由主题类插件可选实现：返回该主题的 CSS 自定义属性声明（不含选择器）。
//
// 配色放在后端是有意为之——只有服务端知道"现在哪个主题生效"，也才能把它写进
// 首帧就会加载的 /api/public/theme.css，从而彻底消除主题闪烁。
// 纯装饰（纹理、SVG、几何变形）仍然留在前端，按需动态加载。
type SkinProvider interface {
	ThemeCSS() string
}

// Manager 插件注册表与运行时管理。
// 启停状态持久化在 system_settings（key = plugin_enabled_<name>），
// 功能路由常驻注册、按请求校验开关 → 启停即时生效，无需重启进程。
type Manager struct {
	mu    sync.RWMutex
	db    *gorm.DB
	items map[string]Plugin
	order []string

	// 启用状态的内存缓存：IsEnabled 在每个 /api/ext/* 请求上都会调用，
	// HandlePublicThemes 还会对每个主题各调一次（5 个主题 = 5 次单行 SELECT）。
	// 开关切换频率极低，用 5 秒 TTL 的快照，SetEnabled 时主动失效。
	stateMu   sync.RWMutex
	stateAt   time.Time
	stateSnap map[string]bool
}

func NewManager(db *gorm.DB) *Manager {
	return &Manager{db: db, items: make(map[string]Plugin)}
}

// Register 注册插件（按注册顺序展示；重名忽略并记日志）
func (m *Manager) Register(p Plugin) {
	meta := p.Meta()
	if meta.Name == "" {
		log.Printf("[plugin] 忽略未命名插件")
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, dup := m.items[meta.Name]; dup {
		log.Printf("[plugin] 重复注册 %s，已忽略", meta.Name)
		return
	}
	m.items[meta.Name] = p
	m.order = append(m.order, meta.Name)
}

func (m *Manager) settingKey(name string) string { return "plugin_enabled_" + name }

// setSetting 先加载再更新的 upsert 模式：
// db.Save 对 ID=0 的 struct 走 INSERT，会撞 key 唯一索引并静默失败（见 HANDOFF 坑记录）
func (m *Manager) setSetting(key, val string) {
	var s model.SystemSetting
	if err := m.db.Where("key = ?", key).First(&s).Error; err != nil {
		m.db.Create(&model.SystemSetting{Key: key, Value: val})
		return
	}
	m.db.Model(&s).Update("value", val)
}

// IsEnabled 插件是否启用（未注册的插件恒为 false；未设置过开关的插件默认禁用 → 升级零行为变化）
func (m *Manager) IsEnabled(name string) bool {
	m.mu.RLock()
	_, ok := m.items[name]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	return m.enabledSnapshot()[name]
}

// enabledSnapshot 一次性读出全部 plugin_enabled_* 开关。
// 单次全表读（几十行）远比"每个插件查一次"便宜，尤其在公开页首屏。
func (m *Manager) enabledSnapshot() map[string]bool {
	m.stateMu.RLock()
	if m.stateSnap != nil && time.Since(m.stateAt) < pluginStateTTL {
		snap := m.stateSnap
		m.stateMu.RUnlock()
		return snap
	}
	m.stateMu.RUnlock()

	// 公开页首屏会同时要主题列表和 theme.css，两处都读这份快照。
	// 过期时只让一个请求查库，另一个等结果，避免同一瞬间打两次。
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	if m.stateSnap != nil && time.Since(m.stateAt) < pluginStateTTL {
		return m.stateSnap
	}

	var rows []model.SystemSetting
	snap := make(map[string]bool)
	if err := m.db.Model(&model.SystemSetting{}).
		Select("key, value").
		Where("key LIKE ?", m.settingKey("")+"%").
		Find(&rows).Error; err != nil {
		if m.stateSnap != nil {
			return m.stateSnap // 这次没读到就继续用上一份
		}
		return snap
	}
	for _, r := range rows {
		snap[strings.TrimPrefix(r.Key, m.settingKey(""))] = r.Value == "true"
	}
	m.stateSnap = snap
	m.stateAt = time.Now()
	return snap
}

// invalidatePluginState 让下一次 IsEnabled 重新查库
func (m *Manager) invalidatePluginState() {
	m.stateMu.Lock()
	m.stateSnap = nil
	m.stateAt = time.Time{}
	m.stateMu.Unlock()
}

func (m *Manager) SetEnabled(name string, on bool) {
	v := "false"
	if on {
		v = "true"
	}
	m.setSetting(m.settingKey(name), v)
	// 开关立即生效，不等 TTL
	m.invalidatePluginState()
}

// InitEnabled 启动时对已启用插件执行 Init
func (m *Manager) InitEnabled() {
	for _, name := range m.order {
		if !m.IsEnabled(name) {
			continue
		}
		if err := m.items[name].Init(m.db); err != nil {
			log.Printf("[plugin] %s Init 失败: %v", name, err)
		} else {
			log.Printf("[plugin] 已启用: %s", name)
		}
	}
}

// Mount 注册管理端点并挂载功能插件路由。
// 管理端点自带 AuthRequired；功能插件统一挂 /api/ext/<name>/...，
// 与 /api/plugins/:name/... 参数段分离，规避 gin 路由树静态/参数段混挂歧义。
func (m *Manager) Mount(api *gin.RouterGroup) {
	api.GET("/plugins", middleware.AuthRequired(m.db), m.handleList)
	api.POST("/plugins/:name/enable", middleware.AuthRequired(m.db), m.handleToggle(true))
	api.POST("/plugins/:name/disable", middleware.AuthRequired(m.db), m.handleToggle(false))
	for _, name := range m.order {
		p := m.items[name]
		if p.Meta().Kind == "theme" {
			continue
		}
		g := api.Group("/ext/"+name, middleware.AuthRequired(m.db), m.enabledGuard(name))
		p.Routes(g)
	}
}

// MountOpen 把实现了 OpenReader 的插件挂到 /api/public/ext/<name>。
// pub 是已经建好的 /api/public 组。不挂认证：调用方是别人的网页，没有本站令牌。
func (m *Manager) MountOpen(pub *gin.RouterGroup) {
	for _, name := range m.order {
		p := m.items[name]
		op, ok := p.(OpenReader)
		if !ok {
			continue
		}
		g := pub.Group("/ext/"+name, m.enabledGuard(name))
		op.OpenRoutes(g)
	}
}

func (m *Manager) enabledGuard(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.IsEnabled(name) {
			c.AbortWithStatusJSON(404, gin.H{"code": "QSL-PLUGIN-OFF", "message": "插件未启用"})
			return
		}
		c.Next()
	}
}

func (m *Manager) handleList(c *gin.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	enabled := m.enabledSnapshot()
	items := make([]gin.H, 0, len(m.order))
	for _, name := range m.order {
		meta := m.items[name].Meta()
		items = append(items, gin.H{
			"name":        meta.Name,
			"title":       meta.Title,
			"version":     meta.Version,
			"description": meta.Description,
			"kind":        meta.Kind,
			"enabled":     enabled[name],
		})
	}
	response.OK(c, gin.H{"items": items})
}

func (m *Manager) handleToggle(on bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		m.mu.RLock()
		_, ok := m.items[name]
		m.mu.RUnlock()
		if !ok {
			response.Fail(c, 404, "插件不存在")
			return
		}
		m.SetEnabled(name, on)
		verb := "已禁用"
		if on {
			verb = "已启用"
		}
		response.OKMsg(c, "插件 "+name+" "+verb+"，即时生效")
	}
}

// HandlePublicThemes 公开端点：列出已启用的主题插件（公开页据此应用皮肤，无需登录）
func (m *Manager) HandlePublicThemes(c *gin.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	enabled := m.enabledSnapshot()
	items := make([]gin.H, 0)
	for _, name := range m.order {
		meta := m.items[name].Meta()
		if meta.Kind == "theme" && enabled[name] {
			items = append(items, gin.H{"name": meta.Name, "title": meta.Title, "version": meta.Version})
		}
	}
	response.OK(c, gin.H{"items": items})
}

// ThemeCSS 输出当前站点的主题样式表。
//
// 设计要点：
//  1. 当前生效主题的变量以 "无属性" 的 :root:root 块下发 → index.html 里的
//     <link rel="stylesheet" href="/api/public/theme.css"> 在第一帧就命中正确配色，
//     不必等 JS，故没有 FOUC。双伪类 :root:root 是为了稳定压过打包 CSS 里的 :root{}，
//     不依赖样式表先后顺序（Vite 会把产物 CSS 链接插到 <head> 末尾）。
//  2. 同时下发每个主题的 [data-theme="..."] 块 → 后台切换主题只需改一个属性，
//     不用重新请求样式表。
//  3. 内容只有 CSS 自定义属性（几十行），每次请求现拼即可，无需缓存。
func (m *Manager) HandleThemeCSS(c *gin.Context) {
	m.mu.RLock()
	enabled := m.enabledSnapshot()
	declsByTheme := make(map[string]string, len(m.order))
	active := ""
	for _, name := range m.order {
		p := m.items[name]
		if p.Meta().Kind != "theme" || !enabled[name] {
			continue
		}
		sp, ok := p.(SkinProvider)
		if !ok {
			continue
		}
		declsByTheme[name] = sp.ThemeCSS()
		active = name
	}
	names := append([]string(nil), m.order...)
	m.mu.RUnlock()

	var b strings.Builder
	b.WriteString("/* 由 /api/public/theme.css 生成：主题配色变量 */\n")
	if active != "" {
		b.WriteString(":root:root{" + declsByTheme[active] + "}\n")
	}
	for _, name := range names {
		if decls, ok := declsByTheme[name]; ok {
			b.WriteString(`:root:root[data-theme="` + name + `"]{` + decls + "}\n")
		}
	}

	c.Header("Cache-Control", "no-cache")
	c.Header("X-QSL-Theme", active)
	c.Data(http.StatusOK, "text/css; charset=utf-8", []byte(b.String()))
}
