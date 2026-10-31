package themeba

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// ThemeBA 蔚蓝档案（Blue Archive）主题插件：
// 晨光蔚蓝基调 × 暖黄点缀 × 赛璐璐减法美学，带平行四边形斜切按键与菱形折射网格。
// 配色由后端下发，装饰层在前端 src/styles/themes/ba.css。
type ThemeBA struct{}

func New() *ThemeBA { return &ThemeBA{} }

func (t *ThemeBA) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_ba",
		Title:       "蔚蓝档案 📘",
		Version:     "2.0.0",
		Description: "基底蔚蓝 × 晨光白 × 暖黄点缀：平行四边形斜切按键、极淡折射菱形网格、清透赛璐璐柔光质感。",
		Kind:        "theme",
	}
}

func (t *ThemeBA) Init(db *gorm.DB) error { return nil }

func (t *ThemeBA) Routes(rg *gin.RouterGroup) {}

// ThemeCSS 返回该主题的 CSS 自定义属性声明（不含选择器）。
func (t *ThemeBA) ThemeCSS() string {
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
