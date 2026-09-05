package themedark

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// ThemeDark 暗色主题插件：后端只提供元数据与启停状态，
// 皮肤本体（CSS 变量 + 附加规则）定义在前端 src/plugins/registry.js。
type ThemeDark struct{}

func New() *ThemeDark { return &ThemeDark{} }

func (t *ThemeDark) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_dark",
		Title:       "暗色主题",
		Version:     "1.0.0",
		Description: "深色工作台皮肤：深蓝黑底、暖橙点缀，后台与公开页同步生效，切换即时预览。",
		Kind:        "theme",
	}
}

func (t *ThemeDark) Init(db *gorm.DB) error { return nil }

func (t *ThemeDark) Routes(rg *gin.RouterGroup) {}
