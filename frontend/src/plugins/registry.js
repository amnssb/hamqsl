/*
主题装饰层。

配色不在这里。配色由 /api/public/theme.css 在首帧下发。
这个文件只做两件事：把 <html data-theme> 设成当前主题，并按需加载纹理。
纹理文件是 frontend/src/styles/themes/<插件名>.css，文件名就是插件名。
放进那个文件夹就有装饰，删掉文件就只剩配色。不用改这里。

调用示例：
  await applyTheme('theme_dark')
  await applyEnabledThemes(['theme_paper', 'theme_dark'])
  resetTheme()
  await reloadThemeCSS()
*/

const skins = import.meta.glob('../styles/themes/*.css') // 键是相对本文件的路径

const loadedSkins = new Set() // 同一个主题的纹理只加载一次

/**
 * 应用一个主题。
 *
 * @param {string} name 插件名，例如 theme_dark。空串或 default 回到默认配色
 * @returns {Promise<void>} 纹理加载完成。没有同名 css，或加载失败，配色仍然生效
 */
export async function applyTheme(name) {
  const root = document.documentElement
  if (!name || name === 'default') {
    root.removeAttribute('data-theme') // 去掉属性，:root 里的默认配色生效
    return
  }

  root.setAttribute('data-theme', name) // 配色块已经在 theme.css 里，改一个属性就换色
  const load = skins[`../styles/themes/${name}.css`]
  if (!load || loadedSkins.has(name)) return // 没有纹理文件，或这一页已经加载过
  try {
    await load()
    loadedSkins.add(name)
  } catch {
    // 纹理没加载到不影响配色，配色来自服务端下发的 theme.css
  }
}

/** 回到默认主题 */
export function resetTheme() {
  document.documentElement.removeAttribute('data-theme')
}

/**
 * 重新拉取 /api/public/theme.css。
 *
 * 主题开关改在后端。index.html 里的 <link> 只包含当时已启用的主题。
 * 新启用一个以前没开过的主题时，它的配色块还不在已加载的样式表里，必须再拉一次。
 *
 * @returns {Promise<void>} 新样式表加载完成，或 3 秒后不再等
 */
export function reloadThemeCSS() {
  return new Promise(resolve => {
    const old = document.querySelector('link[data-qsl-theme]')
    const fresh = document.createElement('link')
    fresh.rel = 'stylesheet'
    fresh.setAttribute('data-qsl-theme', 'reload')
    fresh.href = '/api/public/theme.css?t=' + Date.now() // 时间戳绕开缓存，否则拿到的还是旧配色

    let settled = false
    const done = () => {
      if (settled) return
      settled = true
      if (old) old.remove() // 新的在旧的后面，同特异度下后者生效，然后拆掉旧的
      resolve()
    }
    fresh.onload = done
    fresh.onerror = () => {
      fresh.remove()
      done()
    }
    document.head.appendChild(fresh)
    setTimeout(done, 3000) // 网络挂住时不要卡住调用方
  })
}

/**
 * 应用后端给出的已启用主题列表。
 * 多个同时启用时，列表里最后一个生效，和服务端 HandleThemeCSS 的遍历顺序一致。
 *
 * @param {string[]} enabledThemeNames 按登记顺序排列的已启用主题名
 * @returns {Promise<void>}
 */
export function applyEnabledThemes(enabledThemeNames) {
  const names = (enabledThemeNames || []).filter(Boolean)
  if (!names.length) {
    resetTheme()
    return Promise.resolve()
  }
  return applyTheme(names[names.length - 1])
}
