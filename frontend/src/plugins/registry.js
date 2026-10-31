// 前端主题注册表。
//
// 职责边界（重要，改之前先看懂这张表）：
//
//   配色 (--p-* 原语)   → 后端 internal/plugins/theme_*/  由 /api/public/theme.css 下发
//                         只有这样，index.html 的 <link> 才能在首帧就命中正确配色，
//                         也就没有"先白一下再变暗"的闪烁。
//   装饰 (Layer 3)      → src/styles/themes/<name>.css    按需动态 import()
//   元数据               → 本文件的 themeRegistry          仅用于 /admin/plugins 展示
//
// 上一版的 5 个主题各自带一大段 extraCss（合计约 600 行 !important），
// 靠逐条选择器去覆盖已经画好的浅色界面——也就是"蒙版"。
// 现在配色走 token，装饰走独立文件，这里不应该再出现任何颜色规则。
//
// 新增主题三步：
//   1. 后端 internal/plugins/<name>/ 实现 plugin.Plugin + ThemeCSS()，main.go 里 pm.Register；
//   2. 若需要纹理/几何装饰，加 src/styles/themes/<name>.css（所有规则挂在
//      html[data-theme="<name>"] 下），并在这里补 skin；
//   3. 在下面补一条元数据（title/version/description/swatches）。

export const themeRegistry = {
  theme_glass: {
    name: 'theme_glass',
    title: '鸣门鱼板 🍥',
    version: '2.0.0',
    description: '鸣门鱼板 × 磨砂玻璃拟态：粉白漩涡质感、蓝粉光晕、通透毛玻璃卡片。',
    swatches: ['#f1f5fd', '#ec5f8f', '#ffb3cf', '#5b8ff9'],
    skin: () => import('../styles/themes/glass.css'),
  },
  theme_ba: {
    name: 'theme_ba',
    title: '蔚蓝档案 📘',
    version: '2.0.0',
    description: '基底蔚蓝 × 晨光白 × 暖黄点缀：斜切按键、菱形网格、赛璐璐柔光。',
    swatches: ['#E8F4FD', '#45AFFF', '#006EB0', '#FFD700'],
    skin: () => import('../styles/themes/ba.css'),
  },
  theme_rhodes: {
    name: 'theme_rhodes',
    title: '罗德岛终端 📐',
    version: '2.0.0',
    description: '工业美学：纸感浅灰基座、精密标线、品牌青操作面板、黑底 chip 铭牌。',
    swatches: ['#F4F5F6', '#171B26', '#2FA6DE', '#E14A4A'],
    skin: () => import('../styles/themes/rhodes.css'),
  },
  theme_dark: {
    name: 'theme_dark',
    title: '暗色主题',
    version: '2.0.0',
    description: '深色工作台皮肤：深蓝黑底、暖橙点缀，后台与公开页同步生效。',
    swatches: ['#0f151c', '#16202b', '#ff7a45', '#3aa396'],
    // 纯配色主题，不需要装饰层
  },
  theme_paper: {
    name: 'theme_paper',
    title: '纸墨主题',
    version: '2.0.0',
    description: '仿老式 QSL 卡片：暖黄纸面、墨黑文字、砖红点缀，适合打印与怀旧风格。',
    swatches: ['#efe6d0', '#faf5e8', '#b4552d', '#5e7d4e'],
    // 纯配色主题，不需要装饰层
  },
}

// 已加载过的皮肤（按主题名去重，避免重复 import）
const loadedSkins = new Set()

/**
 * 应用主题。
 *
 * - 配色：只改 <html data-theme>，变量早已由 /api/public/theme.css 下发；
 *   传空值即回到默认主题（移除属性 → :root 里的默认原语生效）。
 * - 装饰：动态 import 对应的 CSS，浏览器注入后立刻生效；
 *   这些规则都限定在 html[data-theme="..."] 下，切走自然失效，无需卸载。
 *
 * @param {string} name 主题插件名；空串 / 'default' 表示默认主题
 * @returns {Promise<void>} 装饰样式加载完成（失败静默，配色仍然正确）
 */
export async function applyTheme(name) {
  const root = document.documentElement
  const theme = themeRegistry[name]

  if (!theme) {
    root.removeAttribute('data-theme')
    return
  }

  // 配色：一个属性搞定
  root.setAttribute('data-theme', theme.name)

  // 装饰：按需加载
  if (!theme.skin || loadedSkins.has(theme.name)) return
  try {
    await theme.skin()
    loadedSkins.add(theme.name)
  } catch {
    /* 装饰加载失败不影响配色，静默降级 */
  }
}

/** 回到默认主题 */
export function resetTheme() {
  document.documentElement.removeAttribute('data-theme')
}

/**
 * 重新拉取 /api/public/theme.css。
 *
 * 为什么需要：主题开关是在后端改的，而配色变量来自 index.html 里那个 <link>。
 * 首次加载时该文件已经包含"所有已启用主题"的 [data-theme] 块，所以正常切换
 * 只改属性即可；但"新启用一个此前没启用的主题"时，它的块还不在已加载的样式表里，
 * 必须重新拉一次。
 *
 * 做法：新建一个同源 <link> 追加到 <head> 末尾（带时间戳绕开缓存），
 * 加载完成后移除旧的。新的在旧之后 → 同特异度下后者生效。
 */
export function reloadThemeCSS() {
  return new Promise(resolve => {
    const old = document.querySelector('link[data-qsl-theme]')
    const fresh = document.createElement('link')
    fresh.rel = 'stylesheet'
    fresh.setAttribute('data-qsl-theme', 'reload')
    fresh.href = '/api/public/theme.css?t=' + Date.now()

    let settled = false
    const done = () => {
      if (settled) return
      settled = true
      if (old) old.remove()
      resolve()
    }
    fresh.onload = done
    fresh.onerror = () => {
      fresh.remove()
      done()
    }
    document.head.appendChild(fresh)
    // 兜底：网络挂住时不要卡住调用方
    setTimeout(done, 3000)
  })
}

/**
 * 应用后端给出的已启用主题列表。
 * 多主题同时启用时按注册顺序取最后一个生效（与后端 HandleThemeCSS 的遍历顺序一致：
 * 越晚注册越优先）。
 */
export function applyEnabledThemes(enabledThemeNames) {
  const names = (enabledThemeNames || []).filter((n) => themeRegistry[n])
  if (!names.length) {
    resetTheme()
    return Promise.resolve()
  }
  return applyTheme(names[names.length - 1])
}

// 兼容旧调用点（Plugins.vue 曾经用 applyThemeByName）
export const applyThemeByName = applyTheme
