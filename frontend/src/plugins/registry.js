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
      '--qsl-panel': 'rgba(255,255,255,.62)',
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
      /* 玻璃卡片与容器 */
      .el-card, .stat-card, .workflow-card, .modern-card, .service-card, .stats-panel, .rcard, .login-card {
        background: rgba(255,255,255,.62) !important;
        backdrop-filter: blur(20px) saturate(170%) !important;
        -webkit-backdrop-filter: blur(20px) saturate(170%) !important;
        border: 1px solid rgba(255,255,255,.75) !important;
        box-shadow: 0 10px 34px rgba(76,110,245,.12), inset 0 1px 0 rgba(255,255,255,.85) !important;
      }
      /* 顶部栏玻璃效果 */
      .layout-header {
        background: rgba(255,255,255,.65) !important;
        backdrop-filter: blur(20px) saturate(160%) !important;
        -webkit-backdrop-filter: blur(20px) saturate(160%) !important;
        border-bottom: 1px solid rgba(255,255,255,.75) !important;
      }
      .user-chip, .admin-link, .hero-btn-secondary {
        background: rgba(255,255,255,.55) !important;
        border-color: rgba(255,255,255,.8) !important;
        backdrop-filter: blur(12px) !important;
      }
      /* 玻璃弹窗与浮层 */
      .el-dialog, .el-message-box {
        background: rgba(255,255,255,.85) !important;
        backdrop-filter: blur(24px) saturate(160%) !important;
        -webkit-backdrop-filter: blur(24px) saturate(160%) !important;
        border: 1px solid rgba(255,255,255,.8) !important;
      }
      .el-popper.is-light { background: rgba(255,255,255,.9) !important; backdrop-filter: blur(16px); }
      /* 表格：透明行落在玻璃卡片上 */
      .el-table {
        --el-table-header-bg-color: rgba(255,255,255,.72) !important;
        --el-table-row-hover-bg-color: rgba(255,214,229,.45) !important;
        --el-bg-color: transparent !important; --el-table-tr-bg-color: transparent !important;
        --el-table-border-color: rgba(190,205,238,.6) !important;
        background: transparent !important;
      }
      .el-table tr, .el-table td.el-table__cell, .el-table th.el-table__cell { background: transparent !important; }
      /* 输入控件：高透白玻璃 */
      .el-input__wrapper, .el-textarea__inner, .el-select__wrapper {
        background: rgba(255,255,255,.78) !important;
        box-shadow: 0 0 0 1px rgba(160,180,225,.55) inset !important;
        backdrop-filter: blur(8px);
      }
      /* 后台侧栏：深蓝玻璃 */
      .layout-aside {
        background: rgba(38,58,102,.80) !important;
        backdrop-filter: blur(18px) saturate(140%) !important;
        -webkit-backdrop-filter: blur(18px) saturate(140%) !important;
        border-right: 1px solid rgba(255,255,255,.18) !important;
      }
      .side-menu.el-menu, .side-menu .el-menu { background: rgba(44,68,120,.55) !important; }
      /* 登录页侧板：蓝粉渐变玻璃 */
      .login-aside {
        background: linear-gradient(160deg, rgba(38,58,102,.90), rgba(154,84,143,.84)) !important;
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
      }
      /* 语言切换器 */
      .lang-switch {
        background: rgba(255,255,255,.75) !important;
        backdrop-filter: blur(12px) !important;
        border-color: rgba(255,255,255,.8) !important;
      }
      .lang-switch button.active {
        background: var(--el-color-primary) !important;
        color: #fff !important;
      }
      /* 主按钮悬停跟随粉色系 */
      .el-button--primary:hover { border-color: #d94b80 !important; background: #d94b80 !important; }
      /* 🍥 品牌装饰 */
      .brand::after {
        content: '🍥';
        font-size: 17px;
        margin-left: auto;
        opacity: .92;
        filter: drop-shadow(0 2px 6px rgba(236,95,143,.45));
      }
      .brand.collapsed::after { display: none; }
      @media (max-width:640px){
        body::after { width:280px; height:280px; right:-95px; top:-85px; opacity:.7; }
        .el-card, .stat-card, .workflow-card, .modern-card { backdrop-filter: blur(14px) saturate(165%); }
      }
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
      '--qsl-navy': '#f1f0ec',
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
    extraCss: `
      /* 全局背景与滚动条 */
      body { background: #0f151c !important; color: #e6e4df !important; }
      ::-webkit-scrollbar-thumb { background: #2a3644; }
      ::-webkit-scrollbar-thumb:hover { background: #3d4e61; }

      /* 输入控件 */
      .el-input__wrapper, .el-textarea__inner, .el-select__wrapper {
        background: #101820 !important;
        box-shadow: 0 0 0 1px #2a3644 inset !important;
        color: #e6e4df !important;
      }
      .el-input__wrapper:hover, .el-select__wrapper:hover {
        box-shadow: 0 0 0 1px #3d4e61 inset !important;
      }
      .el-input__inner, .el-textarea__inner { color: #e6e4df !important; }

      /* 表格 */
      .el-table {
        --el-table-header-bg-color: #121b24 !important;
        --el-table-row-hover-bg-color: #1a2736 !important;
        --el-bg-color: #16202b !important;
        --el-table-tr-bg-color: #16202b !important;
        --el-table-border-color: #2a3644 !important;
        background: #16202b !important;
        color: #e6e4df !important;
      }
      .el-table th.el-table__cell {
        background-color: #121b24 !important;
        color: #f1f0ec !important;
        border-bottom-color: #2a3644 !important;
      }
      .el-table td.el-table__cell {
        background-color: #16202b !important;
        color: #e6e4df !important;
        border-bottom-color: #2a3644 !important;
      }
      .el-table--border { border-color: #2a3644 !important; }

      /* 卡片与容器 */
      .el-card, .stat-card, .workflow-card, .modern-card, .service-card, .stats-panel, .rcard, .login-card {
        background: #16202b !important;
        border-color: #2a3644 !important;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3) !important;
        color: #e6e4df !important;
      }
      .el-card__header { border-bottom-color: #2a3644 !important; color: #f1f0ec !important; }

      /* 弹窗与确认框 */
      .el-dialog, .el-message-box {
        background: #16202b !important;
        border-color: #2a3644 !important;
        color: #e6e4df !important;
      }
      .el-dialog__header, .el-dialog__footer { border-color: #2a3644 !important; }
      .el-dialog__title { color: #f1f0ec !important; }
      .el-popper.is-light { background: #16202b !important; border-color: #2a3644 !important; color: #e6e4df !important; }

      /* 后台顶栏与侧栏 */
      .layout-header {
        background: #16202b !important;
        border-bottom-color: #2a3644 !important;
      }
      .layout-aside {
        background: #0b1017 !important;
        border-right-color: #2a3644 !important;
      }
      .user-chip, .admin-link, .hero-btn-secondary {
        background: #101820 !important;
        border-color: #2a3644 !important;
        color: #e6e4df !important;
      }
      .record-item, .card-item {
        background: #16202b !important;
        border-color: #2a3644 !important;
      }
      .record-item:hover, .card-item:hover {
        background: #101820 !important;
        border-color: #ff7a45 !important;
      }

      /* 语言切换器 */
      .lang-switch {
        background: #16202b !important;
        border-color: #2a3644 !important;
      }
      .lang-switch button { color: #8f9aa6 !important; }
      .lang-switch button.active { color: #fff !important; background: #ff7a45 !important; }
      .lang-switch button:hover:not(.active) { background: #223243 !important; }

      /* 主按钮悬停 */
      .el-button--primary:hover { border-color: #e9632b !important; background: #e9632b !important; }
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
      /* 纸墨底衬与输入控件 */
      body { background: #efe6d0 !important; color: #2b2317 !important; }
      .el-input__wrapper, .el-textarea__inner, .el-select__wrapper {
        background: #fbf7ec !important;
        box-shadow: 0 0 0 1px #cfc2a4 inset !important;
        color: #2b2317 !important;
      }
      /* 表格 */
      .el-table {
        --el-table-header-bg-color: #e7dcc2 !important;
        --el-table-row-hover-bg-color: #f3ead4 !important;
        --el-bg-color: #faf5e8 !important;
        --el-table-tr-bg-color: #faf5e8 !important;
        --el-table-border-color: #cfc2a4 !important;
        background: #faf5e8 !important;
        color: #2b2317 !important;
      }
      .el-table th.el-table__cell { background-color: #e7dcc2 !important; color: #4a3b26 !important; border-bottom-color: #cfc2a4 !important; }
      .el-table td.el-table__cell { background-color: #faf5e8 !important; border-bottom-color: #cfc2a4 !important; color: #2b2317 !important; }
      /* 卡片与容器 */
      .el-card, .stat-card, .workflow-card, .modern-card, .service-card, .stats-panel, .rcard, .login-card {
        background: #faf5e8 !important;
        border-color: #cfc2a4 !important;
        color: #2b2317 !important;
      }
      .el-card__header { border-bottom-color: #cfc2a4 !important; color: #4a3b26 !important; }
      /* 顶栏与侧栏 */
      .layout-header {
        background: #faf5e8 !important;
        border-bottom-color: #cfc2a4 !important;
      }
      .layout-aside {
        background: #362a1c !important;
        border-right-color: #cfc2a4 !important;
      }
      .user-chip, .admin-link, .hero-btn-secondary {
        background: #efe6d0 !important;
        border-color: #cfc2a4 !important;
        color: #2b2317 !important;
      }
      .record-item, .card-item {
        background: #faf5e8 !important;
        border-color: #cfc2a4 !important;
      }
      .record-item:hover, .card-item:hover {
        background: #efe6d0 !important;
        border-color: #b4552d !important;
      }
      .lang-switch {
        background: #faf5e8 !important;
        border-color: #cfc2a4 !important;
      }
      .lang-switch button { color: #8a7d63 !important; }
      .lang-switch button.active { background: #b4552d !important; color: #fff !important; }
    `,
  },
}

const STYLE_ID = 'qsl-plugin-theme-extra'

// resetTheme 清除全部主题覆盖，回到默认现代化外观
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
