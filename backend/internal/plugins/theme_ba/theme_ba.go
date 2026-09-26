package themeba

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// ThemeBA 蔚蓝档案（Blue Archive）主题插件：
// 晨光蔚蓝基调 × 暖黄点缀 × 赛璐璐减法美学，带平行四边形斜切按键与菱形折射网格。
// 后端只提供元数据与启停状态，皮肤本体定义在前端 src/plugins/registry.js。
type ThemeBA struct{}

func New() *ThemeBA { return &ThemeBA{} }

func (t *ThemeBA) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_ba",
		Title:       "蔚蓝档案 📘",
		Version:     "1.0.0",
		Description: "基底蔚蓝 × 晨光白 × 暖黄点缀：平行四边形斜切按键、极淡折射菱形网格、清透赛璐璐柔光质感。",
		Kind:        "theme",
	}
}

func (t *ThemeBA) Init(db *gorm.DB) error { return nil }

func (t *ThemeBA) Routes(rg *gin.RouterGroup) {}
