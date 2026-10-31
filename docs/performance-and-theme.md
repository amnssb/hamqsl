# 性能优化 & 主题系统重构方案

> 目标读者：接手本项目的开发 / AI。
> 本文只讨论两件事：**（A）性能瓶颈在哪、怎么改**，**（B）"主题只是蒙版"的根因与根治方案**。
> 文中所有 `文件:行号` 均基于当前 main 分支实测。

---

## 0. 结论先行

| # | 问题 | 现状代价 | 修复收益（估计） | 风险 |
|---|---|---|---|---|
| A1 | 全量注册 Element Plus 图标 + 全量 `app.use(ElementPlus)` | 入口 chunk 里塞进 **293** 个图标组件（实测 `@element-plus/icons-vue@2.3.2`）+ 整个 EP | 首屏 JS **-60%~70%** | 低 |
| A2 | 主题在 `onMounted` 后异步拉取再注入 | **每次硬刷新都白屏闪一下**（FOUC），且强绑定一次 API 往返 | 首帧即为最终外观 | 低 |
| A3 | 后端缺关键索引 + 8 条独立 `COUNT(*)` | 列表/看板随数据量线性变慢；公开查询走全表扫 | 看板 SQL 次数 8→1，公开查询 O(n)→O(log n) | 低 |
| A4 | 分页 `size` 无上限 | `?size=1000000` 一次拉全表 | 消除一类自伤 DoS | 低 |
| A5 | `plugin.IsEnabled` / `getSettingValue` 每次请求查库 | 公开页首屏 5+2 次 `system_settings` 往返 | 缓存后近似 0 | 低 |
| B1 | **主题是"补丁蒙版"**：靠逐条 `!important` 枚举选择器 | 148 处硬编码色 + 每个新页面都要补 CSS；暗色主题永远盖不全 | 主题只写 ~25 个原语，组件自动跟随 | 中 |

**A 类可以分批独立上线；B 类是结构性重构，建议一次做完。**

---

## 第一部分：主题为什么只是"蒙版"

### 1.1 现象

切到 `theme_dark` 之后，卡片/侧栏/弹窗确实变暗了，但这些东西仍然是浅色：`el-tag`、分页器、`el-dropdown`、日期选择器、`el-tooltip`、`el-empty`、`el-switch`、`el-checkbox`、`el-tabs`、`el-input-number`、移动端 `.rcard` 卡片、`/apply` 与 `/track` 整页。

### 1.2 三层根因

**根因一：组件里硬编码了颜色，绕开了变量。**
`frontend/src` 下有 **148 处** 十六进制色值（`ResponsiveTable.vue:52-72`、`Layout.vue:113-304`、`ApplyForm.vue:451-673`、`RequestStatus.vue:372-396` 等）。
这些是 `<style scoped>`，只在编译期加 `[data-v-xxx]`，颜色本身是字面量 —— 变量怎么改都影响不到它们。

**根因二：Element Plus 自身的 token 从未被赋值。**
`frontend/src/styles/theme.css:14-18` 只给了 `--el-color-primary*`。而 EP 组件的"底色/文字色/填充色/边框色/遮罩色"全都挂在另一组 token 上：

```
--el-bg-color  --el-bg-color-page  --el-bg-color-overlay
--el-text-color-primary  --el-text-color-regular  --el-text-color-secondary
--el-text-color-placeholder  --el-text-color-disabled
--el-border-color  --el-border-color-light  --el-border-color-lighter
--el-fill-color  --el-fill-color-light  --el-fill-color-blank
--el-mask-color  --el-overlay-color-lighter
--el-box-shadow  --el-box-shadow-light
```

这组全是 EP 默认的**浅色**字面量。所以 `el-tag`、`el-pagination`、`el-select-dropdown` 等**根本不会**因为主题而改变 —— 它们读的是变量。

**根因三（结果）：为了补前两个坑，每个主题都长出一大段 `!important` 补丁。**
`frontend/src/plugins/registry.js` 里：
- `theme_glass.extraCss` ≈ 190 行
- `theme_ba.extraCss` ≈ 140 行
- `theme_rhodes.extraCss` ≈ 130 行
- `theme_dark.extraCss` ≈ 90 行
- `theme_paper.extraCss` ≈ 55 行

合计约 **600 行手写 CSS，每个选择器都带 `!important`**。这就是"蒙版"的实体：不是在定义主题，而是在给已经画好的浅色界面刷漆。后果：

1. **永远盖不全**。新增任何页面 / 任何 EP 组件都要回来补一条。
2. **互相打架**。`theme.css:115` 的聚焦环写死 `rgba(37,99,235,.15)`（蓝色），粉色的 glass、棕色的 paper 主题下都是蓝环。
3. **无法预防**。没有任何机制阻止下一个人再写 `color: #1a1a1a`。
4. **`--qsl-navy` 语义被重载**。它同时被当"标题色"（`theme.css:60,133,200`）和"深色侧栏上的文字"用。`theme_glass` 必须让它保持深色（HANDOFF 第 85 条已记录这个坑），于是没法同时满足两张语义 —— 现在只是靠 `Layout.vue` 里硬编码 `#f8fafc` 蒙对了。

### 1.3 附带的时序问题（同时也是 A2）

`App.vue:35-40`：`onMounted` → `GET /api/public/plugins` → `applyEnabledThemes` → `document.documentElement.style.setProperty(...)` + 注入 `<style id="qsl-plugin-theme-extra">`。

也就是说**第一帧永远是默认浅色主题**，等网络回来才变。每次硬刷新（含手机端从后台切回）都能看到这个跳变。而且 5 个主题的 CSS 字符串全在入口 chunk 里，即使只启用一个。

---

## 第二部分：主题系统重构方案（根治"蒙版"）

### 2.1 设计原则

> **主题只提供"原料"，组件只读取"语义变量"，`--el-*` 由语义变量推导。**

三层 token，改一层不影响其它两层：

```
Layer 0  原语 (primitives)     每个主题 ~25 个值，纯色值，不引用别的变量
Layer 1  语义 (semantic)       --qsl-*：全局唯一一份，从 Layer 0 派生
Layer 2  EP 适配 (--el-*)     全局唯一一份，从 Layer 1 派生
────────
Layer 3  皮肤 (skin)           每个主题的可选 CSS：纹理/SVG/几何变形，不做调色
```

关键点：**Layer 1、Layer 2 全局只写一次**。新增主题 = 只写 Layer 0（+ 可选 Layer 3）。

### 2.2 Layer 0：每个主题的原语

`frontend/src/styles/themes/<name>.css`，用 `html[data-theme="..."]` 选择器：

```css
html[data-theme="theme_dark"] {
  color-scheme: dark;              /* 原生滚动条 / 表单控件 / autofill 一起变暗 */

  --p-bg:        #0f151c;          /* 页面底色 */
  --p-surface:   #16202b;          /* 卡片/输入框底色 */
  --p-surface-2: #121b24;          /* 次级面（表头、斑马纹） */
  --p-text:      #e6e4df;          /* 正文 */
  --p-text-2:    #8f9aa6;          /* 次要文字 */
  --p-heading:   #f1f0ec;          /* 标题（比正文更亮） */
  --p-line:      #2a3644;          /* 分隔线/描边 */
  --p-brand:     #ff7a45;          /* 主色 */
  --p-brand-ink: #ffffff;          /* 主色之上的文字 */
  --p-ok:        #3aa396;
  --p-warn:      #f3c969;
  --p-danger:    #ef5350;

  --p-radius:      8px;
  --p-shadow-sm:   0 1px 3px rgba(0,0,0,.35);
  --p-shadow-md:   0 8px 24px rgba(0,0,0,.45);
  --p-glass-bg:    var(--p-surface);   /* 需要磨砂时换成 rgba() */
  --p-glass-blur:  0px;                /* 不磨砂就是 0，CSS 里统一写 blur(var(--p-glass-blur)) */

  /* 深色侧栏（glass / paper / rhodes 用；默认与 surface 一致） */
  --p-chrome-bg:   #0b1017;
  --p-chrome-text: #e6e4df;
}
```

配套的默认主题就是 `:root` 里同一组变量（现在 `theme.css:1-23` 的内容搬过去，改名）。

> 参考：`theme_glass` 的 `--p-glass-bg: rgba(255,255,255,.70)`、`--p-glass-blur: 20px`；`theme_paper` 的 `--p-surface: #faf5e8`；`theme_rhodes` 的 `--p-radius: 3px`。**圆角、阴影、透明度都变成 token** —— 现在这些散落在 `extraCss` 里逐条 `!important`。

### 2.3 Layer 1：语义变量（全局一份）

`frontend/src/styles/tokens.css`：

```css
:root, html[data-theme] {
  /* —— 语义映射 —— */
  --qsl-paper:      var(--p-bg);          /* 兼容旧名，逐步淘汰 */
  --qsl-bg:         var(--p-bg);
  --qsl-panel:      var(--p-surface);
  --qsl-panel-2:    var(--p-surface-2);
  --qsl-ink:        var(--p-text);
  --qsl-ink-2:      var(--p-text-2);
  --qsl-muted:      var(--p-text-2);
  --qsl-line:       var(--p-line);
  --qsl-heading:    var(--p-heading);     /* 新增：替代被重载的 --qsl-navy */
  --qsl-navy:       var(--p-heading);     /* 旧名保留 = 标题色 */
  --qsl-primary:    var(--p-brand);
  --qsl-orange:     var(--p-brand);
  --qsl-green:      var(--p-ok);
  --qsl-yellow:     var(--p-warn);
  --qsl-danger:     var(--p-danger);

  /* —— 深色侧栏 —— */
  --qsl-chrome-bg:   var(--p-chrome-bg);
  --qsl-chrome-text: var(--p-chrome-text);
  --qsl-chrome-muted: color-mix(in srgb, var(--p-chrome-text) 62%, var(--p-chrome-bg));

  /* —— 形状与质感 —— */
  --qsl-radius:    var(--p-radius);
  --qsl-radius-sm: calc(var(--p-radius) - 2px);
  --qsl-card-bg:   var(--p-glass-bg);
  --qsl-blur:      var(--p-glass-blur);
  --qsl-shadow-1:  var(--p-shadow-sm);
  --qsl-shadow-2:  var(--p-shadow-md);

  /* —— 派生：焦点/悬停/选中，全部从主色算出来，不再逐主题手调 —— */
  --qsl-focus-ring:  color-mix(in srgb, var(--qsl-primary) 22%, transparent);
  --qsl-hover-tint:  color-mix(in srgb, var(--qsl-primary) 8%, transparent);
  --qsl-active-tint: color-mix(in srgb, var(--qsl-primary) 14%, transparent);
}
```

`color-mix()` 支持：Chrome/Edge 111+、Safari 16.2+、Firefox 113+。本项目是自建站 + 现代浏览器，可用；若要兜底，在 token 文件里给同名字面量做为 `@supports not (color: color-mix(in srgb, red, blue))` 的降级。

### 2.4 Layer 2：Element Plus token 适配（全局一份）

```css
:root, html[data-theme] {
  /* 主色 + 5 级浅色，从 --p-brand 推导（现在每个主题手写 5 个十六进制，5 个主题 = 25 处） */
  --el-color-primary:          var(--qsl-primary);
  --el-color-primary-light-3:  color-mix(in srgb, var(--qsl-primary) 70%, var(--qsl-bg));
  --el-color-primary-light-5:  color-mix(in srgb, var(--qsl-primary) 50%, var(--qsl-bg));
  --el-color-primary-light-7:  color-mix(in srgb, var(--qsl-primary) 30%, var(--qsl-bg));
  --el-color-primary-light-8:  color-mix(in srgb, var(--qsl-primary) 20%, var(--qsl-bg));
  --el-color-primary-light-9:  color-mix(in srgb, var(--qsl-primary) 10%, var(--qsl-bg));
  --el-color-primary-dark-2:   color-mix(in srgb, var(--qsl-primary) 80%, #000);

  --el-color-success: var(--qsl-green);
  --el-color-warning: var(--qsl-yellow);
  --el-color-danger:  var(--qsl-danger);
  --el-color-error:   var(--qsl-danger);
  --el-color-info:    var(--qsl-muted);

  /* 背景 / 文字 / 边框 / 填充 —— 这一组是"暗色盖不全"的真正解药 */
  --el-bg-color:                 var(--qsl-panel);
  --el-bg-color-page:            var(--qsl-bg);
  --el-bg-color-overlay:         var(--qsl-panel);
  --el-text-color-primary:       var(--qsl-ink);
  --el-text-color-regular:       var(--qsl-ink);
  --el-text-color-secondary:     var(--qsl-muted);
  --el-text-color-placeholder:   var(--qsl-muted);
  --el-text-color-disabled:      color-mix(in srgb, var(--qsl-muted) 70%, transparent);
  --el-border-color:             var(--qsl-line);
  --el-border-color-light:       var(--qsl-line);
  --el-border-color-lighter:     var(--qsl-line);
  --el-border-color-extra-light: color-mix(in srgb, var(--qsl-line) 60%, transparent);
  --el-border-color-dark:        color-mix(in srgb, var(--qsl-line) 80%, var(--qsl-ink));
  --el-fill-color:               var(--qsl-panel-2);
  --el-fill-color-light:         var(--qsl-panel-2);
  --el-fill-color-lighter:       color-mix(in srgb, var(--qsl-panel-2) 60%, var(--qsl-panel));
  --el-fill-color-blank:         var(--qsl-panel);
  --el-fill-color-dark:          color-mix(in srgb, var(--qsl-panel-2) 80%, var(--qsl-ink));
  --el-mask-color:               color-mix(in srgb, var(--qsl-bg) 82%, transparent);
  --el-mask-color-extra-light:   color-mix(in srgb, var(--qsl-bg) 30%, transparent);
  --el-overlay-color-lighter:    color-mix(in srgb, var(--qsl-ink) 16%, transparent);
  --el-box-shadow:               var(--qsl-shadow-2);
  --el-box-shadow-light:         var(--qsl-shadow-1);
  --el-box-shadow-lighter:       var(--qsl-shadow-1);
  --el-disabled-bg-color:        var(--qsl-panel-2);
  --el-disabled-text-color:      var(--qsl-muted);
  --el-disabled-border-color:    var(--qsl-line);

  /* 形状 */
  --el-border-radius-base:  var(--qsl-radius);
  --el-border-radius-small: var(--qsl-radius-sm);
}
```

**这一层落地后，`extraCss` 里的"颜色补丁"整段可以删掉**（大约砍掉 500 行 `!important`），只保留"身份"部分。

### 2.5 Layer 3：皮肤（每个主题可选，只做形状/纹理）

只有无法 token 化的东西留在这里：

| 主题 | 保留的 Layer 3 |
|---|---|
| `theme_glass` | 🍥 鱼板 SVG (`body::after`)、四色径向光斑 (`body::before`)、玻璃卡片 = `background: var(--qsl-card-bg); backdrop-filter: blur(var(--qsl-blur))` |
| `theme_ba` | 菱形网格纹理、`skewX(-12deg)` 斜切按钮、黄色标题下划线 |
| `theme_rhodes` | 斜纹警戒条、`letter-spacing` 工业字距、mono 字体 eyebrow |
| `theme_dark` | **无**（纯配色，Layer 0 就够） |
| `theme_paper` | 可选纸纹 |

文件拆开：`frontend/src/styles/themes/<name>.css`，由主题注册表用 **动态 `import()`** 按需加载 —— 目前 5 段 CSS 字符串（约 30KB）全打进入口 chunk。

### 2.6 应用机制：消灭 FOUC

**方案：后端出一个 `theme.css`，`index.html` 里用 `<link>` 预置。**

理由：
- CSP 是 `style-src 'self'`（`backend/internal/middleware/security.go:14-17`），`script-src 'self'`，**内联 `<script>` 会被拦**。`<link rel="stylesheet" href="/api/...">` 是同源样式表，不违反 CSP，不用加 nonce。
- `<link>` 在 `<head>` 里 → 浏览器渲染第一帧前就命中主题，**零闪烁**。
- 后端本来就已知"哪个主题启用"（`plugin.Manager.HandlePublicThemes`）。

```go
// backend/internal/plugin/plugin.go
// GET /api/public/theme.css —— 返回当前生效主题的 Layer 0 + Layer 3
func (m *Manager) HandleThemeCSS(c *gin.Context) {
    name, ok := m.activeTheme()               // 复用现有 order + IsEnabled 逻辑
    if !ok { c.String(200, "/* 默认主题 */"); return }
    c.Header("Content-Type", "text/css; charset=utf-8")
    c.Header("Cache-Control", "public, max-age=300")
    c.String(200, m.items[name].(interface{ CSS() string }).CSS())
}
```

```html
<!-- frontend/index.html -->
<link rel="stylesheet" href="/api/public/theme.css" />
```

配套：
- `theme.css` 静态文件里 `@import` 或由后端拼接 Layer 1 / Layer 2（它们是全局的，可以直接打进主 CSS，不必进接口）。
- 管理端切换主题时：`document.documentElement.dataset.theme = name` + `await import('./styles/themes/'+name+'.css')` → **不刷新页面即时生效**，且下次进来由 `<link>` 承接。
- 加 `prefers-color-scheme: dark` 自动跟随（`theme_dark` 或任一主题作为 auto 目标），用一个 `data-theme="auto"` 分支。

### 2.7 防回退机制（关键，否则半年后又变蒙版）

1. **清掉 SFC 里的 148 处硬编码色**。除以下场景外一律换 `var(--qsl-*)`：
   - 打印/二维码画布（`CardList.vue:638,675,686-696` 是 `canvas` 与 `qrcode` 的 `color.dark/light`，必须保持字面量）；
   - 邮件 HTML 模板（在 Go 侧，本来就是白底邮件）；
   - `body::before/::after` 里的装饰渐变（Layer 3 允许）。
2. **加 stylelint**：
   ```json
   { "rules": { "color-no-hex": [true, { "ignore": ["**/styles/**"] }] } }
   ```
   把 `src/styles/**` 与打印/二维码模块加白名单，其余 `*.vue` 出现 `#rrggbb` 直接 CI 失败。
3. **拆 `--qsl-navy` 的重载语义**：标题用 `--qsl-heading`，深色侧栏文字用 `--qsl-chrome-text`，两者都能独立跟随主题。
4. **加一个主题巡检页**（dev-only，或挂在 `/admin/plugins` 的"预览"按钮里）：一页渲染所有 EP 组件各一次（button 四态 / input / select / tag 五色 / table / pagination / dropdown / date-picker / tooltip / dialog / switch / checkbox / radio / tabs / empty / upload / steps），一键切 5 个主题肉眼核对。

---

## 第三部分：性能优化清单

按"收益/风险"排序，每一项都可以独立上线。

### P0-1 前端按需引入（最大单项收益）

**现状** `frontend/src/main.js:8,17-19`：

```js
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)     // 实测 293 个图标组件全量注册
}
```

`app.use(ElementPlus)` 又把整库装进去；`import 'element-plus/dist/index.css'` 拉全量 CSS。

**改法**

推荐两步走：

1. **图标**：项目实际只用到 **27 个**（`ArrowDown/ArrowRight/BellFilled/Box/Check/CircleCheck/CircleClose/Connection/DataBoard/Document/InfoFilled/Loading/Location/MagicStick/Menu/Message/Notebook/Plus/Postcard/Promotion/Search/Switch/SwitchButton/Tools/UploadFilled/Van` + Dashboard 动态用的 `Bell`）。建 `src/plugins/icons.js` 只注册这些。**这一项单独就能砍掉一大块入口体积，且零视觉变化。**

   ⚠️ **改之前必须先处理这个坑**：`Dashboard.vue:103-113` 的 `icon` 是**字符串**（`icon: 'Document'`），模板里是 `<component :is="card.icon" />`（`Dashboard.vue:19,44`）—— 靠全局注册才能解析。去掉全量注册后这里会渲染成空。
   两种改法，任选：
   - 把 `icon: 'Document'` 改成 `icon: Document`（`import { Document } from '@element-plus/icons-vue'`），让 `<component :is>` 收到组件对象；
   - 或保留全局注册，但只注册这 5 个（`Document / Postcard / Promotion / Box / Bell`）。
   建议第一种，去掉"字符串 → 全局组件名"这层隐式耦合。

2. **EP 组件**：引入 `unplugin-vue-components` + `ElementPlusResolver`，删掉 `app.use(ElementPlus)`（`ElConfigProvider` 已在 `App.vue:11` 单独 import，不受影响），样式按需。

> 注意：`main.js` 里 `import zhCn from 'element-plus/dist/locale/zh-cn.mjs'` 只用于 `app.use(ElementPlus, { locale })`；`App.vue:12-13` 已另有 `element-plus/es/locale/lang/*` 的 `ElConfigProvider` 路径（真正生效的是后者）。删掉 `app.use` 时可以把 `main.js:7` 那行一起删。

### P0-2 构建产物分离 + 压缩 + 强缓存

**现状** `frontend/vite.config.js` 只有 `outDir/assetsDir`，没有 `manualChunks`。

```js
build: {
  target: 'es2020',
  cssCodeSplit: true,
  rollupOptions: {
    output: {
      manualChunks: {
        vue: ['vue', 'vue-router', 'pinia'],
        element: ['element-plus', '@element-plus/icons-vue'],
        vendor: ['axios', 'qrcode'],
      },
    },
  },
},
```

再加：
- 构建时预压缩（`vite-plugin-compression` 出 `.gz` / `.br`）；
- 后端静态托管补缓存头：`r.Static` 与 `main.go:101-113` 的 `NoRoute` 兜底目前**没有任何 `Cache-Control`/ETag**。`assets/` 下文件名带 hash，应给 `immutable, max-age=31536000`；`index.html` 给 `no-cache`。

### P0-3 主题时序（同 B 类，但可独立先做最小版）

最小改动版（不做 B 类重构也能立刻消掉闪烁）：
- 在 `index.html` 的 `<head>` 里加 `<link rel="stylesheet" href="/api/public/theme.css">`；
- 后端加一个只回"当前启用主题的 `extraCss` + `:root` 变量"的端点；
- `App.vue` 的 `onMounted` 保留，仅用于**运行期切换**。

### P0-4 后端索引

`backend/internal/model/models.go`，这些字段被 `WHERE` / `ORDER BY` 高频使用但没有索引：

| 模型 | 字段 | 被谁用 |
|---|---|---|
| `CardRecord` | `tracking_number` ⚠️ | `public.go:484` 公开查询（未登录！） |
| `CardRecord` | `flow_status` | `card.go:60` 列表筛选 |
| `CardRecord` | `scene_type` / `mail_type` | `card.go:59,61` |
| `CardRecord` | `return_mail_enabled` + `return_mailed_at` | `card.go:63-70` 三态筛选 |
| `CardRecord` | `card_sent` / `card_received` / `card_issued` | `dashboard.go:18-22`、`public.go:199-200` |
| `ExchangeRequest` | `review_status` | `dashboard.go:23`、`public.go:201`、`exchange.go:49` |
| `ExchangeRequest` | `scene_type` | `exchange.go:50` |
| `QsoRecord` | `date`（或 `date,time` 复合） | `qso.go:34` `ORDER BY date DESC, time DESC` |

```go
TrackingNumber string `gorm:"size:50;index" json:"tracking_number"`
FlowStatus     string `gorm:"size:30;default:PENDING_ISSUE;index"`
ReviewStatus   string `gorm:"size:20;default:PENDING;index"`
```

> `TrackingNumber` 这条尤其重要：它是**公开未鉴权**端点，一次 `?tracking_number=` 就是一次全表扫，放大成本极低。

### P0-5 看板 8 条 COUNT → 1 条

**现状** `backend/internal/handler/dashboard.go:16-23`：8 次独立 `COUNT(*)`，每次全表扫。

```go
func (h *DashboardHandler) Summary(c *gin.Context) {
    var card struct {
        Total, Sent, Received, PendingIssue, PendingSent, PendingReceive int64
    }
    h.db.Model(&model.CardRecord{}).Select(`
        COUNT(*) AS total,
        COALESCE(SUM(CASE WHEN card_sent       THEN 1 ELSE 0 END), 0) AS sent,
        COALESCE(SUM(CASE WHEN card_received   THEN 1 ELSE 0 END), 0) AS received,
        COALESCE(SUM(CASE WHEN NOT card_issued AND call_sign <> '' THEN 1 ELSE 0 END), 0) AS pending_issue,
        COALESCE(SUM(CASE WHEN card_issued AND NOT card_sent THEN 1 ELSE 0 END), 0) AS pending_sent,
        COALESCE(SUM(CASE WHEN card_sent AND NOT card_received THEN 1 ELSE 0 END), 0) AS pending_receive
    `).Scan(&card)

    var totalQso, pendingExchange int64
    h.db.Model(&model.QsoRecord{}).Count(&totalQso)
    h.db.Model(&model.ExchangeRequest{}).Where("review_status = 'PENDING'").Count(&pendingExchange)
    // ...
}
```

`SUM(CASE WHEN …)` 语法 SQLite 与 PostgreSQL 都支持（避免用 `FILTER`，SQLite 老版本不支持）。
顺带修一个 bug：`Dashboard.vue:98-99` 读 `total_exchange_requests` / `total_offline_activities`，但后端从未返回 → 页面上这两项恒为 0。

### P0-6 分页 `size` 上限

`qso.go:18`、`card.go:54`、`exchange.go:46`、`receive.go` 都直接 `strconv.Atoi` 后无上界。

```go
func pageParams(c *gin.Context) (page, size int) {
    page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
    size, _ = strconv.Atoi(c.DefaultQuery("size", "20"))
    if page < 1 { page = 1 }
    if size < 1 { size = 20 }
    if size > 200 { size = 200 }   // ← 新增
    return
}
```
放进 `pkg/response` 或新建 `handler/params.go`，四处统一调用。

### P0-7 设置 / 插件状态缓存

**现状**
- `plugin.go:87-95` `IsEnabled` 每次调用查一次 `system_settings`；
- `plugin.go:184-195` `HandlePublicThemes` 对**每个**已注册主题查一次 → 现在 5 次；
- `tracking_query.go:36-42` `getSettingValue` 每次查一次；`public.go:211-220` `SiteInfo` 连查 3 次、`kuaidi100Configured` 连查 2 次。

**改法**：`system_settings` 是小表（几十行），全量读进 `atomic.Value` 的 `map[string]string`，`setSetting` / 任何写路径后置为 dirty 异步重载（或直接同步重载，写入频率极低）。

预期：公开页首屏少 7 次 DB 往返。

### P1-1 批量更新 N+1

`card.go:432-439`：

```go
h.db.Where("card_sent = ? AND mail_type = ? AND tracking_number <> ''", true, "REGISTERED").Find(&cards)
for i := range cards {
    if cards[i].TrackingStatus == "" { ...; h.db.Save(&cards[i]) }   // N 次 UPDATE + N 次全字段回写
}
```

改为单条：
```go
h.db.Model(&model.CardRecord{}).
    Where("card_sent = ? AND mail_type = ? AND tracking_number <> '' AND tracking_status = ''", true, "REGISTERED").
    Updates(map[string]any{
        "tracking_status":     "查询中",
        "tracking_updated_at": cstNow().Format("2006-01-02 15:04:05"),
    })
```

### P1-2 `LIKE '%x%'` 无法走索引

`qso.go:24`、`card.go:58`、`exchange.go:48` 都是前置通配符。呼号是**前缀语义**，改成 `LIKE 'CS%'` 就能吃索引；姓名/地址类模糊搜索如果确需中缀匹配，再考虑 SQLite FTS5（PG 侧用 `pg_trgm`）。

### P1-3 玻璃主题的合成开销

`registry.js:65-128` 给 `.el-table`（整表）、每张卡、侧栏、弹窗、下拉、`.rcard` 都上了 `backdrop-filter`。移动端中低端机上是每一帧的合成成本，长列表滚动会明显掉帧。

建议：
- 只给**容器层**（卡片、侧栏、顶栏、弹窗）保留 blur；表格本体改半透明纯色 `background: rgba(255,255,255,.5)`，不加 `backdrop-filter`；
- 小屏 + `@media (max-width: 640px)` 统一 `--qsl-blur: 8px` 或直接 `0px`（现在 `registry.js:211-214` 只是把 20px 降到 14px，仍然是全屏 blur）；
- 加低端降级开关：`html[data-perf="low"] { --qsl-blur: 0px }`，用 `navigator.hardwareConcurrency <= 4 || navigator.deviceMemory <= 2` 判定。

### P1-4 双份 resize 监听、无节流

`Layout.vue:90-95` 与 `ResponsiveTable.vue:36-38` 各注册一个 `window.resize`，都没有节流；每个列表页都会实例化一次 `ResponsiveTable`。
改为 `matchMedia('(max-width: 720px)')` 的 `change` 事件（只在跨越断点时触发一次），或抽一个 `useBreakpoint()` composable 全局共享一个监听。

### P1-5 启动时的全表 UPDATE

`backend/cmd/server/main.go:63-66`：每次启动无条件跑 4 条全表 `UPDATE`。

```go
db.Model(&model.ExchangeRequest{}).Where("scene_type = ?", "ONLINE_EYEBALL").Update("scene_type", "EYEBALL")
// ×4
```

数据量大了之后每次重启都写一遍 WAL。改为：`Where(...).Limit(1).Update(...)` 判断 `RowsAffected`，为 0 就跳过；或用一次性的迁移标记写进 `system_settings`。

### P2 其它可做项
- **SQLite DSN**：加 `?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-20000)`。并发读不再互相阻塞，写入也快。
- **限流桶重复计数**：`main.go:126` 给整个 `/api` 挂了 `APIRateLimitMiddleware`，`main.go:129` 又给 `/api/public` 挂了一次 → 公开接口每个请求消耗 **2** 次配额（现在 600/分钟，公开页也能被自己撑爆）。删掉 `:129` 那次。
- **公开统计缓存**：`public.go:197-207` 三条 count 是全表扫，首页每次刷新都跑。加 60s 内存缓存即可（这数据本来就不需要秒级）。
- **`gin.Default()`**：生产环境换 `gin.New()` + `gin.Recovery()` + 精简日志，减少每请求的格式化和锁。
- **连接池**：PG 部署时设 `sqlDB.SetMaxOpenConns/SetMaxIdleConns/SetConnMaxLifetime`（SQLite 单文件不建议超过 1 writer，配 WAL 后可放宽读）。
- **`randomCode` / `uniqueCode`**（`card.go:22-39`、`exchange.go:31-42`）：每次插入前 `COUNT` 一次。可改 `INSERT ... ON CONFLICT DO NOTHING` + 重试，省一次查询；同时修掉 `nextCode` 读-改-写无锁的并发重号隐患。
- **`AutoMigrate`**（`main.go:56-60`）：启动时对每张表做一遍 schema 反射，库大后启动变慢。可加 `MIGRATE=0` 开关，容器化部署时由显式的迁移步骤负责。
- **`/uploads` 静态服务**：补 `Cache-Control: public, max-age=31536000, immutable`（文件名随机，天然不可变）。

---

## 第五部分：顺手可以修的缺陷（读代码时发现，与性能/主题无关）

1. **`public.css` 里的 `:deep()` 是死代码。**
   `frontend/src/styles/public.css:89`、`:110`、`:114` 用了 `:deep(.el-form-item__label)` / `:deep(.el-radio-button)`，但这是 **Vue SFC `<style scoped>` 的编译期语法**，在纯 `.css` 文件里浏览器不认，整条规则被丢弃。也就是说 `/apply` 表单标签样式、单选按钮组样式**从来没生效过**。
   改法：去掉 `:deep()` 直接写 `.modern-form .el-form-item__label`（普通 CSS 全局作用域无需 `:deep`）。

2. **Dashboard 两个指标恒为 0。**
   `Dashboard.vue:98-99` 读 `total_exchange_requests` / `total_offline_activities`，`backend/internal/handler/dashboard.go` 从未返回（已在 P0-5 的改法里一并补上）。

3. **公开接口被限流桶计两次。**
   `main.go:126` 给整个 `/api` 挂了 `APIRateLimitMiddleware`，`main.go:129` 又给 `/api/public` 挂了一次 → 公开页每个请求消耗 2 次配额。

4. **`nextCode` 并发重号。**
   `exchange.go:18-28` 是"读 `system_settings` → +1 → 写回"，无事务无锁。并发提交申请时可能拿到同一个编号，靠 `uniqueCode`（`:31-42`）的 count 兜底也会在竞争下漏。建议 `db.Transaction` + `SELECT ... FOR UPDATE`（PG）/ 单写者串行化（SQLite）。

5. **`gin.Default()` 的 Logger 在 `NoRoute` 托管场景下会为每个静态资源打一行日志**，长列表/多图页面刷屏。生产换精简日志。

---

## 第六部分：建议的执行顺序

| 批次 | 内容 | 影响面 | 可验证指标 |
|---|---|---|---|
| **第 1 批**（半天，纯收益无风险） | P0-1 图标按需、P0-2 分包+压缩+缓存头、P0-4 索引、P0-5 看板合并、P0-6 size 上限、P0-7 设置缓存、P2 限流去重 | 后端 6 个文件 + `vite.config.js` + `main.js` | 入口 JS 体积、看板 SQL 条数、`EXPLAIN QUERY PLAN` |
| **第 2 批**（1~2 天，主题根治） | tokens.css 三层落地 → 逐主题重写为 Layer 0 → 删 `extraCss` 的调色段 → SFC 硬编码色清理 → stylelint | 全部 5 个主题 + 17 个 SFC | 5 主题 × 全组件巡检页无遗漏；`!important` 数量下降 80%+ |
| **第 3 批**（1 天） | `theme.css` 后端端点 + `<link>` 预置 + 动态 import 皮肤 + `color-scheme` | `index.html`、`plugin.go`、`App.vue` | 硬刷新首帧即最终外观（录屏逐帧核对） |
| **第 4 批**（按需） | 玻璃降级、breakpoint composable、批量 SQL、WAL、FTS5 | 局部 | 移动端长列表滚动帧率 |

---

## 附：怎么量

**前端体积**
```powershell
cd frontend; npm run build
Get-ChildItem dist/assets | Sort-Object Length -Descending | Select-Object Name, @{n='KB';e={[math]::Round($_.Length/1KB,1)}} -First 15
```

**后端 SQL 次数**（SQLite）
```go
// 临时在 gorm.Config{Logger: ...} 里打开 Info 级别，或 query 前后打点
```
或直接 `EXPLAIN QUERY PLAN SELECT ...` 确认索引命中（要看到 `USING INDEX idx_...`，不能是 `SCAN card_records`）。

**首帧闪烁**
DevTools → Performance → 勾选 Screenshots → 硬刷新，逐帧看第 1 帧是否已是目标主题。

**主题完整性**
第 2 批做完后，在 `/admin/plugins` 加一个"主题巡检"入口，一页铺满所有 EP 组件 + 自有组件，切 5 个主题截图比对。

---

## 实施记录（本轮已落地）

| 指标 | 改前 | 改后 |
|---|---|---|
| 主题相关 `!important` | 约 600 行（5 个 `extraCss`） | **45 处**，全部是响应式布局与"身份"样式（斜切/字距/硬边），**没有一条是颜色补丁** |
| `frontend/src` 硬编码 `#rrggbb` | 148 处 | **5 处**，全在 `<script>` 里（二维码画布与打印 HTML 的颜色回退值） |
| 主题配色来源 | 前端 JS 注入 `<style>` | 后端 `GET /api/public/theme.css`，首帧 `<link>` 命中 |
| 主题 CSS 进入口 chunk | 5 段（约 30KB） | 0（装饰层按需 `import()`） |
| 图标注册 | 293 个 | 30 个 |
| 看板 SQL | 8 条 `COUNT(*)` | 1 条聚合 + 3 条小表 count |

落地文件：

- 新增 `frontend/src/styles/tokens.css`（Layer 1 + Layer 2 + 默认 Layer 0）
- 新增 `frontend/src/styles/themes/{glass,ba,rhodes}.css`（Layer 3）
- 新增 `frontend/src/plugins/icons.js`、`backend/internal/handler/{params.go,settings_cache.go}`
- 重写 `frontend/src/styles/theme.css`、`frontend/src/plugins/registry.js`
- 5 个 `backend/internal/plugins/theme_*/theme_*.go` 增加 `ThemeCSS()`
- `backend/internal/plugin/plugin.go` 增加 `SkinProvider` / `HandleThemeCSS` / 启用状态快照
- 详见 HANDOFF.md 第九节

### 第二轮（巡检 / 玻璃性能 / 发号器 / 按需引入）

- 新增 `frontend/src/views/ThemeLab.vue` + 路由 `/admin/theme-lab`：一页铺满所有组件，
  6 个主题芯片本地预览（不写站点设置），离开自动还原。
- 新增 `frontend/.stylelintrc.json`（只开 `color-no-hex`）+ `npm run lint:css`；
  为此把散落的裸色值收敛为 `--qsl-tone-*` / `--qsl-poster-*` / `--qsl-qr-bg|fg`。
- `theme.css` 与各组件**不再统一挂 `backdrop-filter`**：只有 `themes/glass.css`
  给容器层挂磨砂，且 `.rcard` / `.lang-switch` / `.el-table` 显式 `none`。
  其余 4 个主题完全不产生合成层。
- 新增 `backend/internal/handler/sequence.go` 统一发号器（进程内互斥 + 显式 upsert），
  替换 4 处重复的"读-改-写"；顺带修 `nextReceiveCode` 的切片越界 panic。
- `vite.config.js` 加 `unplugin-vue-components` **运行时探测**：没装 → 行为不变；
  装了 → 组件按需引入。`npm i -D unplugin-vue-components` 即可开启，卸载即回退。
  详见 HANDOFF.md 第十节。
