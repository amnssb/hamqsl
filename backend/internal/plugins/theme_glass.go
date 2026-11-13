package plugins

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// themeGlass 鸣门鱼板主题。配色和玻璃质感都由后端下发，前端用变量消费，不用逐个选择器写 backdrop-filter。
// 纹理在 frontend/src/styles/themes/theme_glass.css。
type themeGlass struct{}

func init() {
	plugin.Add(30, func(*gorm.DB) plugin.Plugin { return &themeGlass{} }) // 排在纸墨之后；两个都开着时鱼板盖住纸墨
}

func (t *themeGlass) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_glass",
		Title:       "鸣门鱼板 🍥",
		Version:     "2.0.0",
		Description: "鸣门鱼板 × 磨砂玻璃拟态：纯正粉白漩涡质感、梦幻蓝粉光晕、通透毛玻璃卡片。",
		Kind:        "theme",
		Swatches:    []string{"#f1f5fd", "#ec5f8f", "#ffb3cf", "#5b8ff9"},
	}
}

func (t *themeGlass) Init(db *gorm.DB) error { return nil }

func (t *themeGlass) Routes(rg *gin.RouterGroup) {}

// ThemeCSS 返回该主题的 CSS 自定义属性声明（不含选择器）。
func (t *themeGlass) ThemeCSS() string {
	return `
color-scheme:light;
--p-bg:#f1f5fd;
--p-surface:#ffffff;
--p-surface-2:#eaf0fc;
--p-text:#1e2942;
--p-text-2:#68769c;
--p-heading:#16203a;
--p-line:rgba(215,226,245,.85);
--p-brand:#ec5f8f;
--p-brand-2:#d94b80;
--p-on-brand:#ffffff;
--p-ok:#10b981;
--p-warn:#ff9ebb;
--p-danger:#ef4444;
--p-radius:10px;
--p-shadow-1:0 10px 32px rgba(76,110,245,.08);
--p-shadow-2:0 20px 40px rgba(76,110,245,.15);
--p-glass-bg:rgba(255,255,255,.70);
--p-glass-blur:20px;
--p-chrome-bg:rgba(36,52,92,.85);
--p-chrome-text:#f8fafc;
--p-chrome-muted:#c3cde3;`
}
