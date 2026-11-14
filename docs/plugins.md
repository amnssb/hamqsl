# 插件开发指南

> 目标读者：第一次给本项目写插件的人（包括零背景的接手者 / AI）。
> 读完就能照着写出一个能启停、能出现在插件页上的完整插件。
> 机制源头在 `backend/internal/plugin/plugin.go`（接口与管理器）；本文只讲怎么用。

## 一句话原理

**一个插件 = 后端一个文件 + （可选）前端一个文件 + （可选）主题纹理一个文件。文件名对上就生效，删掉就消失，任何中央清单都不用改。**

```
backend/internal/plugins/<插件名>.go          后端：登记、路由、Init（必须）
frontend/src/plugins/pages/<插件名>.vue       插件页上这个插件自己的按钮/弹窗（可选）
frontend/src/styles/themes/<插件名>.css       主题的纹理装饰（只有主题插件才可能有）
```

三个文件都用插件名对齐：后端 `Meta().Name` 叫什么，前端、纹理文件就叫什么。

## 最小可用的功能插件

新建 `backend/internal/plugins/my_plugin.go`，整份内容如下：

```go
package plugins

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/plugin"
	"qsl-management/pkg/response"
)

type myPlugin struct{ db *gorm.DB }

func init() {
	// order 决定插件页上的先后。现有占位：主题 10-50，stats_daily 60。
	// 挑一个没用过的空档（如 70），不要改别人已经在用的数字。
	plugin.Add(70, func(db *gorm.DB) plugin.Plugin { return &myPlugin{db: db} })
}

func (p *myPlugin) Meta() plugin.Info {
	return plugin.Info{
		Name:        "my_plugin", // 就是文件名，也是前端界面文件名
		Title:       "我的插件",
		Version:     "1.0.0",
		Description: "一句话说清它给站长带来什么。",
		Kind:        "feature", // "theme" 是主题插件，见下文
	}
}

// Init 在「启动时发现已启用」和「后台点启用」两个时机都会被调。
// 建表、迁移、预热写在这里；必须幂等（重复调用不能出错）。
func (p *myPlugin) Init(db *gorm.DB) error { return nil }

// Routes 挂到 /api/ext/my_plugin/...，这组自带登录校验和启停守卫。
// 插件被禁用时这里的所有路由返回 404 QSL-PLUGIN-OFF，不用自己判断开关。
func (p *myPlugin) Routes(rg *gin.RouterGroup) {
	rg.GET("/hello", func(c *gin.Context) {
		response.OK(c, gin.H{"hello": "world"})
	})
}
```

做完这些重启后端，`GET /api/plugins` 里就会出现它。到这里不需要改任何别的文件。

## 给它一个界面（可选）

上一步的插件在插件页上只有一个开关。想要一个「查看数据」之类的按钮：

新建 `frontend/src/plugins/pages/my_plugin.vue`，文件名 = `Meta().Name`：

```vue
<template>
  <el-button size="small" text type="primary" @click="open">打个招呼</el-button>
  <el-dialog v-model="visible" title="我的插件" width="480px">
    <p>{{ msg }}</p>
  </el-dialog>
</template>

<script setup>
import { ref } from 'vue'
import api from '../../api'

const visible = ref(false)
const msg = ref('')

async function open() {
  const res = await api.get('/ext/my_plugin/hello') // 已带登录态，响应已解包成 data
  msg.value = res?.hello || ''
  visible.value = true
}
</script>
```

`Plugins.vue` 启动时按文件名自动挂载这些组件（`import.meta.glob`）：
文件存在且插件已启用 → 按钮出现在插件卡片上；删掉文件按钮就消失。
组件收一个 `plugin` prop（就是 `/api/plugins` 里那一行，含 `name/enabled/kind` 等）。

注意：

- **别在这里请求别的插件的接口**，也别 import 其它插件文件——一个插件一个文件，互不知道。
- 弹窗里跳别的页面用 `useRouter()`（参考 `pages/showcase.vue` 的 `goSettings`）。
- 颜色只用 `var(--qsl-*)` / `var(--el-*)`，写裸 `#rrggbb` 过不了 CI 的 stylelint。

## 主题插件

主题只管配色，不管路由。把 `Kind` 设成 `"theme"`，实现 `ThemeCSS()`：

```go
func (p *myTheme) ThemeCSS() string {
	// 一组 CSS 自定义属性声明（不含选择器），由 /api/public/theme.css 下发。
	// 原语名和默认值抄 frontend/src/styles/tokens.css 的 Layer 0 注释，
	// 每个都要给全：缺一个就回落到默认主题的值。
	return `color-scheme:dark;
--p-bg:#0f151c;
--p-surface:#16202b;
...`
}
```

- 色板：`Info.Swatches` 填 3-4 个代表色，插件页和主题巡检页用它画色点。
- 纹理（可选）：`frontend/src/styles/themes/<插件名>.css`，选择器统一写
  `html[data-theme="<插件名>"] ...`，只做装饰（纹理/几何），**不写颜色**——颜色在后端。
- 生效规则：多个主题同时启用时，`order` 大的盖住小的（后装上的赢）。
- 主题巡检页 `/admin/theme-lab` 能逐个预览，不用真启停。

## 给外部网页用的只读接口（可选）

实现 `OpenReader`，路由会挂到 `/api/public/ext/<name>/...`，**无需登录**：

```go
func (p *myPlugin) OpenRoutes(rg *gin.RouterGroup) {
	rg.GET("/feed", p.feed)
}
```

规则（`showcase.go` 是完整范例，外部对接的字段契约见 [showcase-api.md](showcase-api.md)）：

- 只读。不做任何写操作；不输出地址、邮箱、备注这类隐私列——SELECT 里就不查它们。
- 禁用时同样 404，守卫已挂好。
- 给别人网页 `<img>` / fetch 用的话，响应上加 `Access-Control-Allow-Origin: *`
  （只加在这条响应上，别动全站 CORS，见 `showcase.go` 的 `allowPublicRead`）。

## 启停与 Init 的时序

| 时机 | 发生什么 |
|---|---|
| 进程启动 | `Install` 按 order 装上所有插件 → `InitEnabled` 对**已启用**的逐个跑 `Init` |
| 后台点「启用」 | 落盘开关 → 立刻跑 `Init`；**Init 失败会把开关回退并返回 500**，接口不会带着半初始化状态上线 |
| 后台点「禁用」 | 落盘开关，下一次请求起 `/api/ext/<name>` 全部 404。路由常驻，不需要重启 |
| 开关读取 | 5 秒 TTL 快照 + 主动失效。直接改数据库的话最多 5 秒后生效 |

所以 `Init` 里放建表/迁移没问题（AutoMigrate 幂等），但别放「只能跑一次」的逻辑。

## 元数据与 order 约定

`plugin.Info` 字段：

| 字段 | 说明 |
|---|---|
| `Name` | 唯一名。文件名、前端界面文件名、URL 段都用它 |
| `Title` | 插件页显示名 |
| `Version` | 语义化版本 |
| `Description` | 插件页上的一句话说明 |
| `Kind` | `"feature"` 或 `"theme"` |
| `Swatches` | 主题色点，只有主题填 |

order 现状：主题 10-50（暗色 10、纸墨 20、鱼板 30、蔚蓝档案 40、罗德岛 50），功能插件 60 起。新插件挑空档；两个插件同号时按文件名先 `init` 的排前面。

## 验证清单（交插件前跑一遍）

```bash
# 后端
cd backend && gofmt -l . && go vet ./... && CGO_ENABLED=1 go test ./...
# 前端
cd frontend && npm run lint:css && npm run build
```

手工冒烟（后端起来后）：

1. `GET /api/plugins` 能看到新插件，默认禁用。
2. 点启用：有界面文件的按钮出现；`/api/ext/<name>/...` 通；重启后保持启用。
3. 点禁用：接口立刻 404（`QSL-PLUGIN-OFF`）。
4. 主题：`/api/public/theme.css` 含自己的变量块；插件页/巡检页色点正常；切页无闪烁。
5. OpenReader：无令牌 `curl` 能拿到 200；禁用后 404。

## 已知约束（写代码前看一眼）

- 插件目前随二进制编译（Go 内置插件，不是运行时加载 `.so`）。加/删插件要重新编译部署——但启停不用。
- 插件间不互相 import；需要共享的查询放 `internal/model` / `internal/handler` 里。
- `system_settings` 里的开关 key 是 `plugin_enabled_<Name>`，手工删掉这行 = 重置为禁用。
- E2E（`scripts/e2e_verify.ps1`）覆盖插件列表/启停/守卫；改插件机制时记得跑。
