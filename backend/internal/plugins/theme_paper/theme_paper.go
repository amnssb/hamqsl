package themepaper

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// ThemePaper 纸墨主题插件：仿老式 QSL 卡的暖黄纸面 + 墨黑文字。
// 配色由后端下发（plugin.SkinProvider），装饰层在前端 src/styles/themes/。
type ThemePaper struct{}

func New() *ThemePaper { return &ThemePaper{} }

func (t *ThemePaper) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_paper",
		Title:       "纸墨主题",
		Version:     "2.0.0",
		Description: "仿老式 QSL 卡片质感：暖黄纸面、墨黑文字、砖红点缀，适合打印与怀旧风格。",
		Kind:        "theme",
	}
}

func (t *ThemePaper) Init(db *gorm.DB) error { return nil }

func (t *ThemePaper) Routes(rg *gin.RouterGroup) {}

// ThemeCSS 返回该主题的 CSS 自定义属性声明（不含选择器）。
func (t *ThemePaper) ThemeCSS() string {
	return `
color-scheme:light;
--p-bg:#efe6d0;
--p-surface:#faf5e8;
--p-surface-2:#e7dcc2;
--p-text:#2b2317;
--p-text-2:#8a7d63;
--p-heading:#4a3b26;
--p-line:#cfc2a4;
--p-brand:#b4552d;
--p-brand-2:#9a4726;
--p-on-brand:#ffffff;
--p-ok:#5e7d4e;
--p-warn:#d9a441;
--p-danger:#b4472d;
--p-radius:8px;
--p-shadow-1:0 1px 3px rgba(74,59,38,.10);
--p-shadow-2:0 18px 30px -8px rgba(74,59,38,.25);
--p-glass-bg:var(--p-surface);
--p-glass-blur:0px;
--p-chrome-bg:#362a1c;
--p-chrome-text:#faf5e8;
--p-chrome-muted:#a89676;`
}
