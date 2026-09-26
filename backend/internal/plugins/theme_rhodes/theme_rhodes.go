package themerhodes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// ThemeRhodes 罗德岛终端（明日方舟 / 舟味）主题插件：
// 浅色纸感基座 × 品牌青操作条 × 工业信息设计骨架，带黑底 chip 铭牌与工业警示质感。
// 后端只提供元数据与启停状态，皮肤本体定义在前端 src/plugins/registry.js。
type ThemeRhodes struct{}

func New() *ThemeRhodes { return &ThemeRhodes{} }

func (t *ThemeRhodes) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_rhodes",
		Title:       "罗德岛终端 📐",
		Version:     "1.0.0",
		Description: "明日方舟官方 UI 工业美学：纸感浅灰基座、精密标线、品牌青操作面板、黑底 chip 铭牌与工业警示质感。",
		Kind:        "theme",
	}
}

func (t *ThemeRhodes) Init(db *gorm.DB) error { return nil }

func (t *ThemeRhodes) Routes(rg *gin.RouterGroup) {}
