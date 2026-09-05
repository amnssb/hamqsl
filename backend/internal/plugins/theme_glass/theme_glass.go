package themeglass

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// ThemeGlass 磨砂漩涡主题插件：磨砂玻璃拟态 + 蓝粉白渐变 + 🍥 漩涡装饰。
// 后端只提供元数据与启停状态，皮肤本体定义在前端 src/plugins/registry.js。
type ThemeGlass struct{}

func New() *ThemeGlass { return &ThemeGlass{} }

func (t *ThemeGlass) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_glass",
		Title:       "磨砂漩涡 🍥",
		Version:     "1.0.0",
		Description: "磨砂玻璃拟态 × 蓝粉白渐变：玻璃卡片与侧栏、柔焦蓝粉光斑、🍥 粉白漩涡切片虚化装饰，甜而不腻的工作台皮肤。",
		Kind:        "theme",
	}
}

func (t *ThemeGlass) Init(db *gorm.DB) error { return nil }

func (t *ThemeGlass) Routes(rg *gin.RouterGroup) {}
