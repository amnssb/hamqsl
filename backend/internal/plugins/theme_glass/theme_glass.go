package themeglass

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// ThemeGlass 鸣门鱼板主题插件：鸣门卷鱼板 × 磨砂玻璃拟态 + 梦幻粉蓝白渐变 + 🍥 鱼板漩涡装饰。
// 后端只提供元数据与启停状态，皮肤本体定义在前端 src/plugins/registry.js。
type ThemeGlass struct{}

func New() *ThemeGlass { return &ThemeGlass{} }

func (t *ThemeGlass) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_glass",
		Title:       "鸣门鱼板 🍥",
		Version:     "1.1.0",
		Description: "鸣门鱼板 × 磨砂玻璃拟态：纯正粉白漩涡质感、梦幻蓝粉光晕、通透毛玻璃卡片与立体浮光深度适配。",
		Kind:        "theme",
	}
}

func (t *ThemeGlass) Init(db *gorm.DB) error { return nil }

func (t *ThemeGlass) Routes(rg *gin.RouterGroup) {}
