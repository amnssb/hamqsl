package plugins

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// themeBA 蔚蓝档案主题。配色经 /api/public/theme.css 下发；斜切按键和菱形网格在 frontend/src/styles/themes/theme_ba.css。
type themeBA struct{}

func init() {
	plugin.Add(40, func(*gorm.DB) plugin.Plugin { return &themeBA{} }) // 排在鱼板之后；两个都开着时蔚蓝档案盖住鱼板
}

func (t *themeBA) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_ba",
		Title:       "蔚蓝档案 📘",
		Version:     "2.0.0",
		Description: "基底蔚蓝 × 晨光白 × 暖黄点缀：平行四边形斜切按键、极淡折射菱形网格、清透赛璐璐柔光质感。",
		Kind:        "theme",
		Swatches:    []string{"#E8F4FD", "#45AFFF", "#006EB0", "#FFD700"},
	}
}

func (t *themeBA) Init(db *gorm.DB) error { return nil }

func (t *themeBA) Routes(rg *gin.RouterGroup) {}

// ThemeCSS 返回该主题的 CSS 自定义属性声明（不含选择器）。
func (t *themeBA) ThemeCSS() string {
	return `
color-scheme:light;
--p-bg:#e8f4fd;
--p-surface:#ffffff;
--p-surface-2:#f0f7fe;
--p-text:#2c3e50;
--p-text-2:#5a6b7c;
--p-heading:#006eb0;
--p-line:rgba(69,175,255,.24);
--p-brand:#45afff;
--p-brand-2:#2f95e8;
--p-on-brand:#ffffff;
--p-ok:#20b2aa;
--p-warn:#ffd700;
--p-danger:#ff6b6b;
--p-radius:10px;
--p-shadow-1:0 8px 24px rgba(69,175,255,.08);
--p-shadow-2:0 14px 34px rgba(69,175,255,.14);
--p-glass-bg:rgba(255,255,255,.94);
--p-glass-blur:12px;
--p-chrome-bg:#f0f7ff;
--p-chrome-text:#2c3e50;
--p-chrome-muted:#5a6b7c;`
}
