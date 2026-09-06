// 手机视口布局审计：检测每页横向溢出与超界元素
// 用法: node frontend/_mobile_audit.mjs
import { chromium } from 'playwright-core'

const BASE = 'http://localhost:3000'
const API = 'http://localhost:8000'

const browser = await chromium.launch({ channel: 'chrome', headless: true })
const ctx = await browser.newContext({
  viewport: { width: 375, height: 812 },
  isMobile: true, hasTouch: true,
})
const page = await ctx.newPage()

const login = await fetch(API + '/api/auth/login', {
  method: 'POST', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'admin', password: 'admin123' }),
}).then(r => r.json())

async function audit(path, name, { authed = false } = {}) {
  if (authed) {
    await ctx.addInitScript(([tok, rt]) => {
      localStorage.setItem('token', tok)
      localStorage.setItem('refresh_token', rt)
    }, [login.data.access_token, login.data.refresh_token])
  }
  await page.goto(BASE + path, { waitUntil: 'networkidle', timeout: 30000 }).catch(() => {})
  await page.waitForTimeout(900)
  const r = await page.evaluate(() => {
    const vw = document.documentElement.clientWidth
    const out = { vw, scrollW: document.documentElement.scrollWidth, overflows: [] }
    if (out.scrollW > vw + 1) {
      const els = document.querySelectorAll('*')
      for (const el of els) {
        const b = el.getBoundingClientRect()
        if (b.right > vw + 2 && b.width > 0) {
          const cls = (el.className && typeof el.className === 'string') ? el.className.split(' ').slice(0, 2).join('.') : ''
          out.overflows.push(`${el.tagName.toLowerCase()}${cls ? '.' + cls : ''} right=${Math.round(b.right)} w=${Math.round(b.width)}`)
          if (out.overflows.length >= 8) break
        }
      }
    }
    // 关键布局尺寸
    const aside = document.querySelector('.layout-aside')
    const header = document.querySelector('.layout-header')
    const main = document.querySelector('.layout-main')
    out.layout = {
      aside: aside ? { x: Math.round(aside.getBoundingClientRect().x), w: Math.round(aside.getBoundingClientRect().width), pos: getComputedStyle(aside).position } : null,
      header: header ? Math.round(header.getBoundingClientRect().height) : null,
      mainW: main ? Math.round(main.getBoundingClientRect().width) : null,
    }
    return out
  })
  console.log(`\n=== ${name} (${path}) vw=${r.vw} scrollW=${r.scrollW} ${r.scrollW > r.vw + 1 ? '*** H-OVERFLOW ***' : 'ok'}`)
  if (r.overflows.length) console.log('  OVERFLOW: ' + r.overflows.join(' | '))
  console.log('  LAYOUT: ' + JSON.stringify(r.layout))
}

await audit('/', 'home')
await audit('/apply', 'apply')
await audit('/status', 'status')
await audit('/login', 'login')
await audit('/admin/dashboard', 'dashboard', { authed: true })
await audit('/admin/cards', 'cards', { authed: true })
await audit('/admin/settings', 'settings', { authed: true })
await audit('/admin/plugins', 'plugins', { authed: true })
await audit('/admin/qso', 'qso', { authed: true })
await audit('/admin/exchange/online', 'exchange', { authed: true })
await audit('/admin/receive', 'receive', { authed: true })
await audit('/admin/address', 'address', { authed: true })

await browser.close()
