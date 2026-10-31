package themeglass

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// ThemeGlass 鸣门鱼板主题插件：鸣门卷鱼板 × 磨砂玻璃拟态 + 梦幻粉蓝白渐变 + 🍥 鱼板漩涡装饰。
//
// 配色与"玻璃质感"（--p-glass-bg / --p-glass-blur）都由后端下发，
// 前端 theme.css 统一用 var(--qsl-card-bg) + blur(var(--qsl-blur)) 消费，
// 所以这里只需要给出原语值，不用再逐个选择器写 backdrop-filter。
type ThemeGlass struct{}

func New() *ThemeGlass { return &ThemeGlass{} }

func (t *ThemeGlass) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_glass",
		Title:       "鸣门鱼板 🍥",
		Version:     "2.0.0",
		Description: "鸣门鱼板 × 磨砂玻璃拟态：纯正粉白漩涡质感、梦幻蓝粉光晕、通透毛玻璃卡片。",
		Kind:        "theme",
	}
}

func (t *ThemeGlass) Init(db *gorm.DB) error { return nil }

func (t *ThemeGlass) Routes(rg *gin.RouterGroup) {}

// ThemeCSS 返回该主题的 CSS 自定义属性声明（不含选择器）。
func (t *ThemeGlass) ThemeCSS() string {
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
