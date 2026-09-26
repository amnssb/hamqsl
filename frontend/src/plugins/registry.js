// 前端插件注册表：主题插件的视觉资源（CSS 变量 + 附加规则）在前端定义，
// 后端 plugin.Manager 只负责插件元数据与启停状态；两者以插件名（theme_*）关联。
// 新增主题：后端注册 kind=theme 的插件 + 在 themeRegistry 中补充同名条目即可。
export const themeRegistry = {
  theme_glass: {
    name: 'theme_glass',
    title: '鸣门鱼板 🍥',
    version: '1.1.0',
    swatches: ['#f1f5fd', '#ec5f8f', '#ffb3cf', '#5b8ff9'],
    vars: {
      '--qsl-ink': '#1e2942',
      '--qsl-muted': '#68769c',
      '--qsl-line': 'rgba(215, 226, 245, 0.85)',
      '--qsl-paper': '#f1f5fd',
      '--qsl-panel': 'rgba(255, 255, 255, 0.70)',
      '--qsl-navy': '#16203a',
      '--qsl-navy-soft': '#2c3b63',
      '--qsl-orange': '#ec5f8f',
      '--qsl-yellow': '#ff9ebb',
      '--qsl-green': '#10b981',
      '--el-color-primary': '#ec5f8f',
      '--el-color-primary-light-3': '#f188af',
      '--el-color-primary-light-5': '#f5abc7',
      '--el-color-primary-light-7': '#f9cee0',
      '--el-color-primary-light-9': '#fdeef5',
    },
    extraCss: `
      /* 全局容器完全透明，让底层的鱼板漩涡与磨砂光晕透出 */
      .layout-container, .main-container, .layout-main, .portal-page, .public-page, .login-container {
        background: transparent !important;
        background-image: none !important;
      }
      body {
        background: #f1f5fd !important;
        color: #1e2942 !important;
        min-height: 100vh;
      }

      /* 蓝粉白渐变底 + 柔焦光斑（磨砂玻璃的折射底衬） */
      body::before {
        content: '';
        position: fixed; inset: -20%; z-index: -2; pointer-events: none;
        background:
          radial-gradient(42% 38% at 18% 22%, rgba(96,165,250,.45), transparent 70%),
          radial-gradient(38% 34% at 82% 18%, rgba(244,114,182,.40), transparent 70%),
          radial-gradient(46% 40% at 78% 82%, rgba(167,139,250,.28), transparent 70%),
          radial-gradient(40% 36% at 22% 84%, rgba(125,211,252,.35), transparent 70%),
          linear-gradient(135deg, #eef4ff 0%, #fdf0f6 46%, #ffffff 100%);
      }

      /* 🍥 鸣门鱼板真实漩涡纹理：右上角悬浮切片，带锯齿轮廓与经典粉红螺旋 */
      body::after {
        content: '';
        position: fixed; right: -40px; top: -40px;
        width: 380px; height: 380px;
        z-index: -1; pointer-events: none;
        background-repeat: no-repeat;
        background-size: contain;
        background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 200 200'%3E%3Cdefs%3E%3Cfilter id='f' x='-20%25' y='-20%25' width='140%25' height='140%25'%3E%3CfeDropShadow dx='0' dy='8' stdDeviation='10' flood-color='%23ec5f8f' flood-opacity='0.25'/%3E%3C/filter%3E%3C/defs%3E%3Cpath d='M100 22 C115 23 125 32 138 35 C152 38 165 48 172 61 C180 75 178 91 180 106 C178 122 173 138 162 150 C150 162 135 170 119 175 C104 178 88 176 74 170 C60 163 48 152 40 139 C32 126 31 110 32 95 C33 79 40 64 51 53 C62 42 77 34 93 32 Z' fill='%23ffffff' opacity='0.88' filter='url(%23f)' stroke='%23f9cbdc' stroke-width='4'/%3E%3Cpath d='M100 58 A42 42 0 0 1 142 100 A42 42 0 0 1 100 142 A42 42 0 0 1 58 100 A22 22 0 0 1 100 78 A22 22 0 0 1 122 100 A12 12 0 0 1 100 112 A4 4 0 0 1 96 100' fill='none' stroke='%23ec5f8f' stroke-width='14' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
        opacity: .82;
        filter: drop-shadow(0 15px 35px rgba(236,95,143,.22));
      }

      /* 玻璃卡片与容器全层级深度适配 */
      .el-card, .stat-card, .workflow-card, .modern-card, .service-card, .stats-panel, .rcard, .login-card, .site-notice {
        background: rgba(255, 255, 255, 0.70) !important;
        backdrop-filter: blur(20px) saturate(175%) !important;
        -webkit-backdrop-filter: blur(20px) saturate(175%) !important;
        border: 1px solid rgba(255, 255, 255, 0.85) !important;
        box-shadow: 0 10px 32px rgba(76, 110, 245, 0.08), inset 0 1px 0 rgba(255, 255, 255, 0.95) !important;
        border-radius: 14px !important;
      }
      .el-card__header, .card-header {
        border-bottom: 1px solid rgba(215, 226, 245, 0.75) !important;
      }

      /* 顶部栏玻璃效果 */
      .layout-header {
        background: rgba(255, 255, 255, 0.72) !important;
        backdrop-filter: blur(20px) saturate(160%) !important;
        -webkit-backdrop-filter: blur(20px) saturate(160%) !important;
        border-bottom: 1px solid rgba(215, 226, 245, 0.85) !important;
      }
      .user-chip, .admin-link, .hero-btn-secondary {
        background: rgba(255, 255, 255, 0.65) !important;
        border-color: rgba(255, 255, 255, 0.9) !important;
        backdrop-filter: blur(12px) !important;
        color: #1e2942 !important;
      }

      /* 玻璃弹窗与浮层 */
      .el-dialog, .el-message-box {
        background: rgba(255, 255, 255, 0.88) !important;
        backdrop-filter: blur(24px) saturate(160%) !important;
        -webkit-backdrop-filter: blur(24px) saturate(160%) !important;
        border: 1px solid rgba(255, 255, 255, 0.9) !important;
        box-shadow: 0 20px 40px rgba(76, 110, 245, 0.15) !important;
        border-radius: 16px !important;
      }
      .el-popper.is-light, .el-picker-panel {
        background: rgba(255, 255, 255, 0.92) !important;
        backdrop-filter: blur(20px) !important;
        border: 1px solid rgba(255, 255, 255, 0.9) !important;
        box-shadow: 0 12px 36px rgba(100, 120, 180, 0.18) !important;
        border-radius: 12px !important;
      }
      .el-select-dropdown__item.is-hovering, .el-select-dropdown__item.is-selected {
        background: rgba(236, 95, 143, 0.12) !important;
        color: #ec5f8f !important;
      }

      /* 表格：透明行落在玻璃卡片上 */
      .el-table {
        --el-table-header-bg-color: rgba(248, 250, 255, 0.8) !important;
        --el-table-row-hover-bg-color: rgba(236, 95, 143, 0.12) !important;
        --el-bg-color: transparent !important;
        --el-table-tr-bg-color: transparent !important;
        --el-table-border-color: rgba(212, 222, 243, 0.75) !important;
        background: rgba(255, 255, 255, 0.5) !important;
        backdrop-filter: blur(16px) !important;
        border-radius: 12px !important;
      }
      .el-table tr, .el-table td.el-table__cell, .el-table th.el-table__cell {
        background: transparent !important;
      }
      .el-table th.el-table__cell {
        background-color: rgba(245, 248, 255, 0.8) !important;
      }

      /* 移动端字段卡片 */
      .rcard {
        background: rgba(255, 255, 255, 0.75) !important;
        backdrop-filter: blur(14px) !important;
        border: 1px solid rgba(236, 95, 143, 0.25) !important;
        border-radius: 14px !important;
        box-shadow: 0 4px 16px rgba(236, 95, 143, 0.08) !important;
      }
      .rcard-head {
        border-bottom-color: rgba(236, 95, 143, 0.18) !important;
      }

      /* 输入控件：高透白玻璃 */
      .el-input__wrapper, .el-textarea__inner, .el-select__wrapper {
        background: rgba(255, 255, 255, 0.80) !important;
        box-shadow: 0 0 0 1px rgba(180, 198, 235, 0.65) inset !important;
        backdrop-filter: blur(10px) !important;
        border-radius: 8px !important;
      }
      .el-input__wrapper.is-focus, .el-select__wrapper.is-focused, .el-textarea__inner:focus {
        box-shadow: 0 0 0 1px #ec5f8f inset, 0 0 0 3px rgba(236, 95, 143, 0.22) !important;
      }

      /* 后台侧栏：深蓝磨砂玻璃 */
      .layout-aside {
        background: rgba(36, 52, 92, 0.85) !important;
        backdrop-filter: blur(20px) saturate(140%) !important;
        -webkit-backdrop-filter: blur(20px) saturate(140%) !important;
        border-right: 1px solid rgba(255, 255, 255, 0.16) !important;
      }
      .side-menu.el-menu, .side-menu .el-menu { background: transparent !important; }
      .side-menu .el-menu-item.is-active {
        background: linear-gradient(135deg, #ec5f8f, #f472b6) !important;
        color: #ffffff !important;
        box-shadow: 0 4px 12px rgba(236, 95, 143, 0.35) !important;
      }

      /* 登录页侧板：蓝粉渐变玻璃 */
      .login-aside {
        background: linear-gradient(160deg, rgba(38, 58, 102, 0.90), rgba(180, 84, 140, 0.86)) !important;
        backdrop-filter: blur(20px);
        -webkit-backdrop-filter: blur(20px);
      }

      /* 语言切换器 */
      .lang-switch {
        background: rgba(255, 255, 255, 0.75) !important;
        backdrop-filter: blur(12px) !important;
        border-color: rgba(255, 255, 255, 0.8) !important;
      }
      .lang-switch button.active {
        background: #ec5f8f !important;
        color: #fff !important;
      }

      /* 主按钮悬停跟随粉色系 */
      .el-button--primary {
        background: #ec5f8f !important;
        border-color: #ec5f8f !important;
        box-shadow: 0 2px 8px rgba(236, 95, 143, 0.3) !important;
      }
      .el-button--primary:hover {
        background: #d94b80 !important;
        border-color: #d94b80 !important;
        box-shadow: 0 4px 12px rgba(236, 95, 143, 0.4) !important;
      }

      /* 🍥 品牌与标识装饰 */
      .brand-mark, .logo-icon {
        background: linear-gradient(135deg, #ec5f8f, #f472b6) !important;
        border-radius: 10px !important;
      }
      .brand::after, .portal-brand::after {
        content: '🍥';
        font-size: 18px;
        margin-left: 8px;
        opacity: .92;
        filter: drop-shadow(0 2px 6px rgba(236, 95, 143, 0.45));
      }
      .brand.collapsed::after { display: none; }

      @media (max-width: 640px) {
        body::after { width: 220px; height: 220px; right: -25px; top: -25px; opacity: .75; }
        .el-card, .stat-card, .workflow-card, .modern-card { backdrop-filter: blur(14px) saturate(165%); }
      }
    `,
  },

  theme_ba: {
    name: 'theme_ba',
    title: '蔚蓝档案 📘',
    version: '1.0.0',
    swatches: ['#E8F4FD', '#45AFFF', '#006EB0', '#FFD700'],
    vars: {
      '--qsl-ink': '#2C3E50',
      '--qsl-muted': '#5A6B7C',
      '--qsl-line': 'rgba(69, 175, 255, 0.24)',
      '--qsl-paper': '#E8F4FD',
      '--qsl-panel': 'rgba(255, 255, 255, 0.94)',
      '--qsl-navy': '#006EB0',
      '--qsl-navy-soft': '#45AFFF',
      '--qsl-orange': '#FFD700',
      '--qsl-yellow': '#FFD700',
      '--qsl-green': '#20B2AA',
      '--el-color-primary': '#45AFFF',
      '--el-color-primary-light-3': '#7FCFFF',
      '--el-color-primary-light-5': '#A2DEFF',
      '--el-color-primary-light-7': '#C5ECFF',
      '--el-color-primary-light-9': '#EFF8FF',
    },
    extraCss: `
      /* 容器背景透出天空底色 */
      .layout-container, .main-container, .layout-main, .portal-page, .public-page, .login-container {
        background: transparent !important;
        background-image: none !important;
      }
      body {
        background: linear-gradient(160deg, #F0F7FF 0%, #E8F4FD 40%, #FDFEFF 100%) !important;
        color: #2C3E50 !important;
        min-height: 100vh;
      }

      /* 蔚蓝档案招牌折射菱形网格纹理（周期 12px，透明度 ≤0.04） */
      body::before {
        content: '';
        position: fixed; inset: 0; z-index: -2; pointer-events: none;
        background-image:
          repeating-linear-gradient(45deg, transparent, transparent 12px, rgba(69, 175, 255, 0.035) 12px, rgba(69, 175, 255, 0.035) 13px),
          repeating-linear-gradient(-45deg, transparent, transparent 12px, rgba(69, 175, 255, 0.035) 12px, rgba(69, 175, 255, 0.035) 13px);
      }

      /* 右上角光环（Halo）柔光符号 */
      body::after {
        content: '';
        position: fixed; right: -40px; top: -40px; width: 340px; height: 340px;
        border-radius: 50%;
        z-index: -1; pointer-events: none;
        background: radial-gradient(circle, rgba(69, 175, 255, 0.12) 0%, rgba(69, 175, 255, 0.03) 60%, transparent 70%);
        box-shadow: inset 0 0 30px rgba(69, 175, 255, 0.15), 0 0 40px rgba(69, 175, 255, 0.10);
        border: 2px dashed rgba(69, 175, 255, 0.22);
        opacity: 0.85;
      }

      /* 平行四边形斜切按键（主按钮 skewX(-12deg)，内层文字反向回正） */
      .el-button--primary, .hero-btn-primary, .login-button {
        transform: skewX(-12deg) !important;
        border-radius: 4px !important;
        background: linear-gradient(180deg, #6BC8FF 0%, #3E9EF8 100%) !important;
        border: 1px solid #45AFFF !important;
        box-shadow: 0 4px 14px rgba(62, 158, 248, 0.3) !important;
        font-weight: 700 !important;
        letter-spacing: 0.02em !important;
      }
      .el-button--primary:hover, .hero-btn-primary:hover, .login-button:hover {
        filter: brightness(1.08) !important;
        box-shadow: 0 6px 18px rgba(62, 158, 248, 0.45) !important;
      }
      .el-button--primary > *, .hero-btn-primary > *, .login-button > * {
        transform: skewX(12deg) !important;
        display: inline-flex; align-items: center; justify-content: center;
      }

      /* 标题黄色下划线装饰（BA 标志性黄色点睛） */
      .page-heading h1, .hero-title, .card-header h2 {
        position: relative;
      }
      .page-heading h1::after, .card-header h2::after, .section-heading h2::after {
        content: '';
        display: block;
        width: 44px;
        height: 3.5px;
        background: #FFD700;
        border-radius: 2px;
        margin-top: 6px;
        box-shadow: 0 1px 4px rgba(255, 215, 0, 0.4);
      }

      /* 赛璐璐轻透卡片与面板（柔和浅蓝投影，去重边框） */
      .el-card, .stat-card, .workflow-card, .modern-card, .service-card, .stats-panel, .rcard, .login-card {
        background: rgba(255, 255, 255, 0.94) !important;
        backdrop-filter: blur(12px) !important;
        border: 1px solid rgba(69, 175, 255, 0.22) !important;
        border-radius: 14px !important;
        box-shadow: 0 8px 24px rgba(69, 175, 255, 0.08), 0 2px 8px rgba(69, 175, 255, 0.04) !important;
      }
      .el-card__header, .card-header {
        border-bottom: 1px solid rgba(69, 175, 255, 0.16) !important;
      }

      /* 顶栏与侧栏：清爽晨光蓝白 */
      .layout-header {
        background: rgba(255, 255, 255, 0.92) !important;
        border-bottom: 1px solid rgba(69, 175, 255, 0.20) !important;
        backdrop-filter: blur(12px) !important;
      }
      .layout-aside {
        background: rgba(240, 247, 255, 0.95) !important;
        border-right: 1px solid rgba(69, 175, 255, 0.20) !important;
        box-shadow: 2px 0 12px rgba(69, 175, 255, 0.06) !important;
      }
      .side-menu .el-menu-item, .side-menu .el-sub-menu__title { color: #5A6B7C !important; }
      .side-menu .el-menu-item:hover, .side-menu .el-sub-menu__title:hover {
        background: rgba(69, 175, 255, 0.1) !important;
        color: #006EB0 !important;
      }
      .side-menu .el-menu-item.is-active {
        background: #45AFFF !important;
        color: #FFFFFF !important;
        border-radius: 6px !important;
        box-shadow: 0 4px 12px rgba(69, 175, 255, 0.3) !important;
      }
      .brand-mark, .logo-icon {
        background: linear-gradient(135deg, #45AFFF, #006EB0) !important;
        border-radius: 10px !important;
        box-shadow: 0 4px 12px rgba(69, 175, 255, 0.35) !important;
      }
      .brand::after, .portal-brand::after {
        content: '✦ SCHALE';
        color: #45AFFF;
        font-size: 11px;
        font-weight: 800;
        letter-spacing: 0.1em;
        margin-left: 8px;
        font-family: monospace, sans-serif;
      }
      .brand.collapsed::after { display: none; }

      .user-chip, .admin-link, .hero-btn-secondary {
        background: #ffffff !important;
        border-color: rgba(69, 175, 255, 0.28) !important;
        color: #2C3E50 !important;
      }
      .el-input__wrapper, .el-textarea__inner, .el-select__wrapper {
        background: #ffffff !important;
        box-shadow: 0 0 0 1px rgba(69, 175, 255, 0.28) inset !important;
        border-radius: 8px !important;
      }
      .el-input__wrapper.is-focus, .el-select__wrapper.is-focused, .el-textarea__inner:focus {
        box-shadow: 0 0 0 1px #45AFFF inset, 0 0 0 3px rgba(69, 175, 255, 0.2) !important;
      }
      .el-table {
        --el-table-header-bg-color: #f0f7fe !important;
        --el-table-row-hover-bg-color: rgba(69, 175, 255, 0.08) !important;
        --el-table-border-color: rgba(69, 175, 255, 0.18) !important;
        background: #ffffff !important;
      }
      .el-table th.el-table__cell { color: #006EB0 !important; }
      .el-dialog, .el-message-box {
        border-radius: 16px !important;
        border: 1px solid rgba(69, 175, 255, 0.25) !important;
        box-shadow: 0 14px 34px rgba(69, 175, 255, 0.14) !important;
      }
    `,
  },

  theme_rhodes: {
    name: 'theme_rhodes',
    title: '罗德岛终端 📐',
    version: '1.0.0',
    swatches: ['#F4F5F6', '#171B26', '#2FA6DE', '#E14A4A'],
    vars: {
      '--qsl-ink': '#141517',
      '--qsl-muted': '#6A6F78',
      '--qsl-line': '#D0D4DA',
      '--qsl-paper': '#F4F5F6',
      '--qsl-panel': '#FFFFFF',
      '--qsl-navy': '#171B26',
      '--qsl-navy-soft': '#2C313C',
      '--qsl-orange': '#E14A4A',
      '--qsl-yellow': '#F2A93B',
      '--qsl-green': '#2FA6DE',
      '--el-color-primary': '#2FA6DE',
      '--el-color-primary-light-3': '#58BCE8',
      '--el-color-primary-light-5': '#83CEF0',
      '--el-color-primary-light-7': '#B4E2F7',
      '--el-color-primary-light-9': '#E7F6FC',
    },
    extraCss: `
      /* 浅色纸感基座与工业字距 */
      body {
        background: #F4F5F6 !important;
        color: #141517 !important;
      }
      h1, h2, .hero-title, .page-heading h1, .brand-title {
        letter-spacing: -0.04em !important;
        font-weight: 850 !important;
      }
      .eyebrow, .brand-sub {
        letter-spacing: 0.16em !important;
        font-family: monospace, sans-serif !important;
      }

      /* 工业斜纹警戒条（Card Header 点睛） */
      .card-header::after, .section-heading::after {
        content: '';
        display: block;
        height: 3px;
        width: 42px;
        background: repeating-linear-gradient(-45deg, #2FA6DE, #2FA6DE 3px, transparent 3px, transparent 6px);
        margin-top: 6px;
      }

      /* 工业黑底 chip 铭牌标签 */
      .live-pill, .welcome-badge, .aside-pill, .status-tag {
        background: #171B26 !important;
        color: #FFFFFF !important;
        border-radius: 2px !important;
        font-weight: 700 !important;
        font-size: 11px !important;
        letter-spacing: 0.06em !important;
        padding: 2px 7px !important;
        border: none !important;
      }

      /* 精密工业卡片：硬朗小圆角 + 发丝级标线 */
      .el-card, .stat-card, .workflow-card, .modern-card, .service-card, .stats-panel, .rcard, .login-card {
        background: #FFFFFF !important;
        border: 1px solid #D0D4DA !important;
        border-radius: 3px !important;
        box-shadow: 0 2px 6px rgba(0, 0, 0, 0.04) !important;
      }
      .el-card__header {
        border-bottom: 1px solid #D0D4DA !important;
        font-weight: 750 !important;
        color: #171B26 !important;
      }

      /* 罗德岛指令终端顶栏与侧栏 */
      .layout-aside {
        background: #171B26 !important;
        border-right: 1px solid #2C313C !important;
      }
      .side-menu .el-menu-item { border-radius: 2px !important; }
      .side-menu .el-menu-item.is-active {
        background: #2FA6DE !important;
        color: #FFFFFF !important;
        border-radius: 2px !important;
      }
      .layout-header {
        background: #FFFFFF !important;
        border-bottom: 1px solid #D0D4DA !important;
      }
      .brand-mark, .logo-icon {
        background: #171B26 !important;
        border: 1.5px solid #2FA6DE !important;
        border-radius: 2px !important;
        color: #2FA6DE !important;
      }
      .brand::after, .portal-brand::after {
        content: '◆ RHODES';
        color: #2FA6DE;
        font-size: 11px;
        font-weight: 800;
        letter-spacing: 0.12em;
        margin-left: 8px;
        font-family: monospace, sans-serif;
      }
      .brand.collapsed::after { display: none; }

      /* 操作按钮：品牌青色，硬直边缘 */
      .el-button { border-radius: 2px !important; }
      .el-button--primary, .hero-btn-primary, .login-button {
        background: #2FA6DE !important;
        border-color: #2FA6DE !important;
        color: #FFFFFF !important;
        font-weight: 700 !important;
        letter-spacing: 0.03em !important;
        border-radius: 2px !important;
        box-shadow: 0 2px 6px rgba(47, 166, 222, 0.25) !important;
      }
      .el-button--primary:hover, .hero-btn-primary:hover, .login-button:hover {
        background: #238fc2 !important;
        border-color: #238fc2 !important;
      }

      /* 输入控件 */
      .el-input__wrapper, .el-textarea__inner, .el-select__wrapper {
        border-radius: 2px !important;
        box-shadow: 0 0 0 1px #D0D4DA inset !important;
      }
      .el-input__wrapper.is-focus, .el-select__wrapper.is-focused, .el-textarea__inner:focus {
        box-shadow: 0 0 0 1.5px #2FA6DE inset, 0 0 0 3px rgba(47, 166, 222, 0.18) !important;
      }

      /* 表格 */
      .el-table {
        --el-table-header-bg-color: #EBECEF !important;
        --el-table-row-hover-bg-color: rgba(47, 166, 222, 0.08) !important;
        --el-table-border-color: #D0D4DA !important;
        border-radius: 3px !important;
      }
      .el-table th.el-table__cell {
        color: #171B26 !important;
        font-weight: 750 !important;
        letter-spacing: 0.02em;
      }

      /* 弹窗 */
      .el-dialog, .el-message-box {
        border-radius: 4px !important;
        border: 1px solid #D0D4DA !important;
        box-shadow: 0 16px 32px rgba(23, 27, 38, 0.2) !important;
      }
      .user-chip, .admin-link, .hero-btn-secondary {
        background: #FFFFFF !important;
        border-color: #D0D4DA !important;
        border-radius: 2px !important;
        color: #141517 !important;
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
