package plugins

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// themeDark 暗色主题。配色经 /api/public/theme.css 下发；纹理在 frontend/src/styles/themes/theme_dark.css。
type themeDark struct{}

func init() {
	plugin.Add(10, func(*gorm.DB) plugin.Plugin { return &themeDark{} }) // 五个主题里最先登记；数字更大的同时开着时盖住它
}

func (t *themeDark) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_dark",
		Title:       "暗色主题",
		Version:     "2.0.0",
		Description: "深色工作台皮肤：深蓝黑底、暖橙点缀，后台与公开页同步生效。",
		Kind:        "theme",
		Swatches:    []string{"#0f151c", "#16202b", "#ff7a45", "#3aa396"},
	}
}

func (t *themeDark) Init(db *gorm.DB) error { return nil }

func (t *themeDark) Routes(rg *gin.RouterGroup) {}

// ThemeCSS 返回该主题的 CSS 自定义属性声明（不含选择器）。
func (t *themeDark) ThemeCSS() string {
	return `
color-scheme:dark;
--p-bg:#0f151c;
--p-surface:#16202b;
--p-surface-2:#121b24;
--p-text:#e6e4df;
--p-text-2:#8f9aa6;
--p-heading:#f1f0ec;
--p-line:#2a3644;
--p-brand:#ff7a45;
--p-brand-2:#e9632b;
--p-on-brand:#ffffff;
--p-ok:#3aa396;
--p-warn:#f3c969;
--p-danger:#ef5350;
--p-radius:8px;
--p-shadow-1:0 4px 12px rgba(0,0,0,.30);
--p-shadow-2:0 16px 32px rgba(0,0,0,.50);
--p-glass-bg:var(--p-surface);
--p-glass-blur:0px;
--p-chrome-bg:#0b1017;
--p-chrome-text:#e6e4df;
--p-chrome-muted:#8f9aa6;`
}
