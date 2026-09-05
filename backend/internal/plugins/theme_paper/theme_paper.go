package themepaper

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// ThemePaper 纸墨主题插件：仿老式 QSL 卡的暖黄纸面 + 墨黑文字。
type ThemePaper struct{}

func New() *ThemePaper { return &ThemePaper{} }

func (t *ThemePaper) Meta() plugin.Info {
	return plugin.Info{
		Name:        "theme_paper",
		Title:       "纸墨主题",
		Version:     "1.0.0",
		Description: "仿老式 QSL 卡片质感：暖黄纸面、墨黑文字、砖红点缀，适合打印与怀旧风格。",
		Kind:        "theme",
	}
}

func (t *ThemePaper) Init(db *gorm.DB) error { return nil }

func (t *ThemePaper) Routes(rg *gin.RouterGroup) {}
