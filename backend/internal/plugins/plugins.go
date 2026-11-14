// Package plugins 是会不断增加的插件，一个文件一个。
//
// 每个插件在 init 里调用 plugin.Add 登记自己。加一个插件只要在这个目录新建文件，
// 不用改 main.go，也不用改别的清单。配套文件按插件名对齐：
//
//	backend/internal/plugins/<插件名>.go      本目录：登记、路由、Init
//	frontend/src/plugins/pages/<插件名>.vue   插件页上它自己的按钮/弹窗（可选）
//	frontend/src/styles/themes/<插件名>.css   主题的纹理装饰（可选）
//
// 三个文件都对上名字就生效，删掉就消失。完整教程见 docs/plugins.md。
//
// 调用示例：
//
//	plugins.Install(pm, db)
package plugins

import (
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
)

// Install 把本目录里已经登记的插件装进管理器。
func Install(m *plugin.Manager, db *gorm.DB) {
	m.Install(db) // 先后由各文件里的 order 决定，不由这里决定
}
