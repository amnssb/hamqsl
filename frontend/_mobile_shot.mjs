// 手机视口截图脚本：375x812 逐页截图到 scripts/_shots/
// 用法: node scripts/_mobile_shot.mjs
import { chromium } from 'playwright-core'
import fs from 'node:fs'

const BASE = 'http://localhost:3000'
const API = 'http://localhost:8000'
const OUT = 'scripts/_shots'

fs.mkdirSync(OUT, { recursive: true })

const browser = await chromium.launch({ channel: 'chrome', headless: true })
const ctx = await browser.newContext({
  viewport: { width: 375, height: 812 },
  deviceScaleFactor: 2,
  isMobile: true,
  hasTouch: true,
})
const page = await ctx.newPage()

async function shot(path, name, { authed = false } = {}) {
  if (authed) {
    // 注入登录态（拦截器读 localStorage.token）
    const r = await fetch(API + '/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: 'admin', password: 'admin123' }),
    })
    const j = await r.json()
    await ctx.addInitScript(([tok, rt]) => {
      localStorage.setItem('token', tok)
      localStorage.setItem('refresh_token', rt)
    }, [j.data.access_token, j.data.refresh_token])
  }
  await page.goto(BASE + path, { waitUntil: 'networkidle', timeout: 30000 }).catch(() => {})
  await page.waitForTimeout(900)
  await page.screenshot({ path: `${OUT}/${name}.png`, fullPage: true })
  console.log('SHOT', name)
}

// 公开页
await shot('/', '01-home')
await shot('/apply', '02-apply')
await shot('/status', '03-status')
await shot('/login', '04-login')
// 后台页（登录态）
await shot('/admin/dashboard', '05-dashboard', { authed: true })
await shot('/admin/cards', '06-cards', { authed: true })
await shot('/admin/settings', '07-settings', { authed: true })
await shot('/admin/plugins', '08-plugins', { authed: true })

await browser.close()
console.log('DONE')
