package themerhodes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// ThemeRhodes 罗德岛终端（明日方舟 / 舟味）主题插件：
// 浅色纸感基座 × 品牌青操作条 × 工业信息设计骨架，带黑底 chip 铭牌与工业警示质感。
// 配色由后端下发，装饰层在前端 src/styles/themes/rhodes.css。
type ThemeRhodes struct{}

func New() *ThemeRhodes { return &ThemeRhodes{} }

func (t *ThemeRhodes) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_rhodes",
		Title:       "罗德岛终端 📐",
		Version:     "2.0.0",
		Description: "明日方舟官方 UI 工业美学：纸感浅灰基座、精密标线、品牌青操作面板、黑底 chip 铭牌与工业警示质感。",
		Kind:        "theme",
	}
}

func (t *ThemeRhodes) Init(db *gorm.DB) error { return nil }

func (t *ThemeRhodes) Routes(rg *gin.RouterGroup) {}

// ThemeCSS 返回该主题的 CSS 自定义属性声明（不含选择器）。
func (t *ThemeRhodes) ThemeCSS() string {
	return `
color-scheme:light;
--p-bg:#f4f5f6;
--p-surface:#ffffff;
--p-surface-2:#ebecef;
--p-text:#141517;
--p-text-2:#6a6f78;
--p-heading:#171b26;
--p-line:#d0d4da;
--p-brand:#2fa6de;
--p-brand-2:#238fc2;
--p-on-brand:#ffffff;
--p-ok:#2fa6de;
--p-warn:#f2a93b;
--p-danger:#e14a4a;
--p-radius:2px;
--p-shadow-1:0 2px 6px rgba(0,0,0,.04);
--p-shadow-2:0 16px 32px rgba(23,27,38,.20);
--p-glass-bg:#ffffff;
--p-glass-blur:0px;
--p-chrome-bg:#171b26;
--p-chrome-text:#ffffff;
--p-chrome-muted:#8b929e;`
}
