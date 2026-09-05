// 前端插件注册表：主题插件的视觉资源（CSS 变量 + 附加规则）在前端定义，
// 后端 plugin.Manager 只负责插件元数据与启停状态；两者以插件名（theme_*）关联。
// 新增主题：后端注册 kind=theme 的插件 + 在 themeRegistry 中补充同名条目即可。
export const themeRegistry = {
  theme_dark: {
    name: 'theme_dark',
    title: '暗色主题',
    version: '1.0.0',
    swatches: ['#0f151c', '#16202b', '#ff7a45', '#3aa396'],
    vars: {
      '--qsl-ink': '#e6e4df',
      '--qsl-muted': '#8f9aa6',
      '--qsl-line': '#2a3644',
      '--qsl-paper': '#0f151c',
      '--qsl-panel': '#16202b',
      '--qsl-navy': '#e6e4df',
      '--qsl-navy-soft': '#223243',
      '--qsl-orange': '#ff7a45',
      '--qsl-yellow': '#f3c969',
      '--qsl-green': '#3aa396',
      '--el-color-primary': '#ff7a45',
      '--el-color-primary-light-3': '#ff9a6e',
      '--el-color-primary-light-5': '#ffba96',
      '--el-color-primary-light-7': '#ffd9c0',
      '--el-color-primary-light-9': '#3a2a22',
    },
    // theme.css 中硬编码的浅色规则与 .el-table 局部变量无法用 :root 变量覆盖，用附加样式表补齐
    extraCss: `
      .el-input__wrapper,.el-textarea__inner,.el-select__wrapper{background:#101820!important;box-shadow:0 0 0 1px #2a3644 inset!important;}
      .el-input__wrapper:hover,.el-select__wrapper:hover{box-shadow:0 0 0 1px #3d4e61 inset!important;}
      .el-table{--el-table-header-bg-color:#1b2632!important;--el-table-row-hover-bg-color:#1c2836!important;--el-bg-color:#16202b;--el-table-tr-bg-color:#16202b;color:#e6e4df;}
      .el-card{box-shadow:3px 3px 0 rgba(0,0,0,.4)!important;}
      .el-dialog{background:#16202b;}
      .el-button--primary:hover{border-color:#e9632b;background:#e9632b;}
    `,
  },
  theme_paper: {
    name: 'theme_paper',
    title: '纸墨主题',
    version: '1.0.0',
    swatches: ['#efe6d0', '#faf5e8', '#b4552d', '#5e7d4e'],
    vars: {
      '--qsl-ink': '#2b2317',
      '--qsl-muted': '#8a7d63',
      '--qsl-line': '#cfc2a4',
      '--qsl-paper': '#efe6d0',
      '--qsl-panel': '#faf5e8',
      '--qsl-navy': '#4a3b26',
      '--qsl-navy-soft': '#5d4c31',
      '--qsl-orange': '#b4552d',
      '--qsl-yellow': '#d9a441',
      '--qsl-green': '#5e7d4e',
      '--el-color-primary': '#b4552d',
      '--el-color-primary-light-3': '#c47453',
      '--el-color-primary-light-5': '#d39679',
      '--el-color-primary-light-7': '#e2b9a2',
      '--el-color-primary-light-9': '#f1ddd2',
    },
    extraCss: `
      .el-input__wrapper,.el-textarea__inner,.el-select__wrapper{background:#fbf7ec!important;}
      .el-table{--el-table-header-bg-color:#e7dcc2!important;--el-table-row-hover-bg-color:#f3ead4!important;}
    `,
  },
}

const STYLE_ID = 'qsl-plugin-theme-extra'

// resetTheme 清除全部主题覆盖，回到默认"米白纸面"外观
export function resetTheme() {
  Object.values(themeRegistry).forEach(t => {
    Object.keys(t.vars).forEach(k => document.documentElement.style.removeProperty(k))
  })
  const el = document.getElementById(STYLE_ID)
  if (el) el.textContent = ''
}

export function applyThemeByName(name) {
  const t = themeRegistry[name]
  resetTheme()
  if (!t) return
  Object.entries(t.vars).forEach(([k, v]) => document.documentElement.style.setProperty(k, v))
  let el = document.getElementById(STYLE_ID)
  if (!el) {
    el = document.createElement('style')
    el.id = STYLE_ID
    document.head.appendChild(el)
  }
  el.textContent = t.extraCss || ''
}

// applyEnabledThemes 应用已启用主题列表：多主题同时启用时，注册顺序靠后者优先（列表末位生效）
export function applyEnabledThemes(enabledThemeNames) {
  const names = (enabledThemeNames || []).filter(n => themeRegistry[n])
  if (!names.length) {
    resetTheme()
    return
  }
  applyThemeByName(names[names.length - 1])
}
