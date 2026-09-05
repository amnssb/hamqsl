package plugin

import (
	"log"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

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
	// Routes 向 /api/ext/<name> 组注册路由（主题类留空）
	Routes(rg *gin.RouterGroup)
}

// Manager 插件注册表与运行时管理。
// 启停状态持久化在 system_settings（key = plugin_enabled_<name>），
// 功能路由常驻注册、按请求校验开关 → 启停即时生效，无需重启进程。
type Manager struct {
	mu    sync.RWMutex
	db    *gorm.DB
	items map[string]Plugin
	order []string
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

func (m *Manager) getSetting(key string) string {
	var s model.SystemSetting
	if err := m.db.Where("key = ?", key).First(&s).Error; err != nil {
		return ""
	}
	return s.Value
}

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
	return m.getSetting(m.settingKey(name)) == "true"
}

func (m *Manager) SetEnabled(name string, on bool) {
	v := "false"
	if on {
		v = "true"
	}
	m.setSetting(m.settingKey(name), v)
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
	api.GET("/plugins", middleware.AuthRequired(), m.handleList)
	api.POST("/plugins/:name/enable", middleware.AuthRequired(), m.handleToggle(true))
	api.POST("/plugins/:name/disable", middleware.AuthRequired(), m.handleToggle(false))
	for _, name := range m.order {
		p := m.items[name]
		if p.Meta().Kind == "theme" {
			continue
		}
		g := api.Group("/ext/"+name, middleware.AuthRequired(), m.enabledGuard(name))
		p.Routes(g)
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
	items := make([]gin.H, 0, len(m.order))
	for _, name := range m.order {
		meta := m.items[name].Meta()
		items = append(items, gin.H{
			"name":        meta.Name,
			"title":       meta.Title,
			"version":     meta.Version,
			"description": meta.Description,
			"kind":        meta.Kind,
			"enabled":     m.getSetting(m.settingKey(name)) == "true",
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
	items := make([]gin.H, 0)
	for _, name := range m.order {
		meta := m.items[name].Meta()
		if meta.Kind == "theme" && m.getSetting(m.settingKey(name)) == "true" {
			items = append(items, gin.H{"name": meta.Name, "title": meta.Title, "version": meta.Version})
		}
	}
	response.OK(c, gin.H{"items": items})
}
