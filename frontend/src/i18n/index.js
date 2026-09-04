// 公开门户轻量 i18n：语言包按页面拆分于 ./pages/*.zh.js / *.en.js
// 用法：import { t, locale, setLocale } from '@/i18n'（相对路径引入）
// 模板中 {{ t('home.title') }}，动态属性 :placeholder="t('apply.name')"
import { reactive } from 'vue'
import commonZh from './pages/common.zh'
import commonEn from './pages/common.en'
import homeZh from './pages/home.zh'
import homeEn from './pages/home.en'
import applyZh from './pages/apply.zh'
import applyEn from './pages/apply.en'
import statusZh from './pages/status.zh'
import statusEn from './pages/status.en'
import confirmZh from './pages/confirm.zh'
import confirmEn from './pages/confirm.en'
import trackZh from './pages/track.zh'
import trackEn from './pages/track.en'

export const messages = {
  zh: { ...commonZh, ...homeZh, ...applyZh, ...statusZh, ...confirmZh, ...trackZh },
  en: { ...commonEn, ...homeEn, ...applyEn, ...statusEn, ...confirmEn, ...trackEn }
}

const saved = (() => { try { return localStorage.getItem('qsl_lang') } catch (e) { return null } })()
const browserLang = ((navigator.language || 'zh-CN') || '').toLowerCase().startsWith('zh') ? 'zh' : 'en'

// 首次访问跟随浏览器语言；用户手动切换后以 localStorage 为准
export const locale = reactive({ current: saved === 'zh' || saved === 'en' ? saved : browserLang })

export function setLocale(l) {
  if (l !== 'zh' && l !== 'en') return
  locale.current = l
  try { localStorage.setItem('qsl_lang', l) } catch (e) { /* 隐私模式忽略 */ }
  document.documentElement.lang = l === 'zh' ? 'zh-CN' : 'en'
}

document.documentElement.lang = locale.current === 'zh' ? 'zh-CN' : 'en'

// 取词：当前语言缺失时回落中文，再缺回落 key 本身；支持 {name} 插值
export function t(key, params) {
  let v = locale.current === 'en' ? messages.en[key] : undefined
  if (v === undefined || v === '') v = messages.zh[key]
  if (v === undefined) return key
  if (params) {
    v = v.replace(/\{(\w+)\}/g, (m, k) => (params[k] !== undefined ? String(params[k]) : m))
  }
  return v
}
