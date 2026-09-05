// 前端插件注册表：主题插件的视觉资源（CSS 变量 + 附加规则）在前端定义，
// 后端 plugin.Manager 只负责插件元数据与启停状态；两者以插件名（theme_*）关联。
// 新增主题：后端注册 kind=theme 的插件 + 在 themeRegistry 中补充同名条目即可。
export const themeRegistry = {
  theme_glass: {
    name: 'theme_glass',
    title: '磨砂漩涡 🍥',
    version: '1.0.0',
    swatches: ['#f1f5ff', '#ec5f8f', '#5b8ff9', '#ffd6e5'],
    vars: {
      '--qsl-ink': '#35426b',
      '--qsl-muted': '#7d88ab',
      '--qsl-line': '#d4def3',
      '--qsl-paper': '#f1f5ff',
      '--qsl-panel': 'rgba(255,255,255,.58)',
      '--qsl-navy': '#273a6b',
      '--qsl-navy-soft': '#3d5590',
      '--qsl-orange': '#ec5f8f',
      '--qsl-yellow': '#ffa7c4',
      '--qsl-green': '#38b2a3',
      '--el-color-primary': '#ec5f8f',
      '--el-color-primary-light-3': '#f183ab',
      '--el-color-primary-light-5': '#f5a7c4',
      '--el-color-primary-light-7': '#f9cbdc',
      '--el-color-primary-light-9': '#fde9f1',
    },
    // 磨砂玻璃需要 backdrop-filter 与 theme.css 硬编码浅色规则的覆盖，
    // 全部走注入样式表（<style id="qsl-plugin-theme-extra">）
    extraCss: `
      /* 蓝粉白渐变底 + 柔焦光斑（磨砂玻璃的折射底衬） */
      body::before {
        content: '';
        position: fixed; inset: -20%; z-index: -2; pointer-events: none;
        background:
          radial-gradient(42% 38% at 18% 22%, rgba(96,165,250,.50), transparent 70%),
          radial-gradient(38% 34% at 82% 18%, rgba(244,114,182,.44), transparent 70%),
          radial-gradient(46% 40% at 78% 82%, rgba(167,139,250,.30), transparent 72%),
          radial-gradient(40% 36% at 22% 84%, rgba(125,211,252,.40), transparent 70%),
          linear-gradient(135deg, #eef4ff 0%, #fdf0f6 46%, #ffffff 100%);
      }
      /* 🍥 漩涡切片：粉白辐条环纹 + 挖空圆心 + 柔焦，悬浮于右上角 */
      body::after {
        content: '';
        position: fixed; right: -140px; top: -120px;
        width: 460px; height: 460px; border-radius: 50%;
        z-index: -1; pointer-events: none;
        background: repeating-conic-gradient(from 8deg, rgba(255,143,184,.55) 0deg 22deg, rgba(255,255,255,.70) 22deg 45deg);
        -webkit-mask: radial-gradient(closest-side, transparent 26%, #000 27%);
        mask: radial-gradient(closest-side, transparent 26%, #000 27%);
        filter: blur(26px) saturate(135%);
        opacity: .85;
      }
      /* 玻璃卡片 */
      .el-card {
        background: rgba(255,255,255,.58) !important;
        backdrop-filter: blur(20px) saturate(170%);
        -webkit-backdrop-filter: blur(20px) saturate(170%);
        border: 1px solid rgba(255,255,255,.75) !important;
        box-shadow: 0 10px 34px rgba(76,110,245,.14), inset 0 1px 0 rgba(255,255,255,.85) !important;
      }
      /* 玻璃弹窗与浮层（保证可读性用较高不透明度） */
      .el-dialog, .el-message-box {
        background: rgba(255,255,255,.85) !important;
        backdrop-filter: blur(24px) saturate(160%);
        -webkit-backdrop-filter: blur(24px) saturate(160%);
        border: 1px solid rgba(255,255,255,.8);
      }
      .el-popper.is-light { background: rgba(255,255,255,.9) !important; backdrop-filter: blur(16px); }
      /* 表格：透明行落在玻璃卡片上，表头微白、悬停淡粉 */
      .el-table {
        --el-table-header-bg-color: rgba(255,255,255,.72) !important;
        --el-table-row-hover-bg-color: rgba(255,214,229,.45) !important;
        --el-bg-color: transparent; --el-table-tr-bg-color: transparent;
        --el-table-border-color: rgba(190,205,238,.6) !important;
      }
      .el-table tr, .el-table td.el-table__cell, .el-table th.el-table__cell.is-leaf { background: transparent !important; }
      /* 输入控件：高透白玻璃 */
      .el-input__wrapper, .el-textarea__inner, .el-select__wrapper {
        background: rgba(255,255,255,.78) !important;
        box-shadow: 0 0 0 1px rgba(160,180,225,.55) inset !important;
        backdrop-filter: blur(8px);
      }
      /* 后台侧栏：深蓝玻璃 */
      .layout-aside {
        background: rgba(38,58,102,.80) !important;
        backdrop-filter: blur(18px) saturate(140%);
        -webkit-backdrop-filter: blur(18px) saturate(140%);
        border-right: 1px solid rgba(255,255,255,.18) !important;
      }
      .side-menu.el-menu, .side-menu .el-menu { background: rgba(44,68,120,.55) !important; }
      /* 登录页侧板：蓝粉渐变玻璃 */
      .login-aside {
        background: linear-gradient(160deg, rgba(38,58,102,.90), rgba(154,84,143,.84)) !important;
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
      }
      /* 主按钮悬停跟随粉色系 */
      .el-button--primary:hover { border-color: #d94b80 !important; background: #d94b80 !important; }
      /* 🍥 品牌装饰：侧栏品牌行右侧（收起时隐藏） */
      .brand::after {
        content: '🍥';
        font-size: 17px;
        margin-left: auto;
        opacity: .92;
        filter: drop-shadow(0 2px 6px rgba(236,95,143,.45));
      }
      .brand.collapsed::after { display: none; }
    `,
  },
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
