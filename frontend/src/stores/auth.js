import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../api'
import { publicKeyToRequestOptions, credentialToJSON } from '../utils/webauthn'

/*
登录状态。

密码登录现在可能只走一半：服务器返回 mfa_required 时只给了票据，必须再过一次
动态码 / 通行密钥 / 恢复码才拿到令牌。所以 login() 的返回值是
{ mfaRequired: false } 或 { mfaRequired: true, ticket, methods }，
调用方（Login.vue）据此决定要不要显示第二阶段。

调用示例：
	const r = await auth.login(u, p)
	if (r.mfaRequired) showSecondFactor(r.ticket, r.methods)
	await auth.completeTotp(ticket, '123456')
	await auth.loginWithPasskey()          // 免密，不输账号
*/
export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(null)
  const isLoggedIn = computed(() => !!token.value)

  // 只有真正拿到令牌才落盘。半登录状态绝不能当登录成功。
  function applyTokens(res) {
    if (!res?.access_token) return false
    token.value = res.access_token
    localStorage.setItem('token', res.access_token)
    if (res.refresh_token) localStorage.setItem('refresh_token', res.refresh_token)
    return true
  }

  async function login(username, password) {
    const res = await api.post('/auth/login', { username, password })
    if (res?.mfa_required) {
      return { mfaRequired: true, ticket: res.ticket, methods: res.methods || [] }
    }
    applyTokens(res)
    await fetchUser()
    return { mfaRequired: false }
  }

  // 动态码
  async function completeTotp(ticket, code) {
    applyTokens(await api.post('/auth/mfa/totp', { ticket, code }))
    await fetchUser()
    return true
  }

  // 恢复码（手机丢了）
  async function completeRecovery(ticket, code) {
    applyTokens(await api.post('/auth/mfa/recovery', { ticket, code }))
    await fetchUser()
    return true
  }

  // 通行密钥作为第二因子
  async function completePasskey(ticket) {
    const begin = await api.post('/auth/mfa/passkey/begin', { ticket })
    const assertion = await navigator.credentials.get({
      publicKey: publicKeyToRequestOptions(begin.publicKey),
    })
    applyTokens(
      await api.post(
        '/auth/mfa/passkey/finish?ticket=' + encodeURIComponent(begin.ticket),
        credentialToJSON(assertion)
      )
    )
    await fetchUser()
    return true
  }

  // 通行密钥免密登录：浏览器直接弹账号选择器，不用输用户名
  async function loginWithPasskey() {
    const begin = await api.post('/auth/passkey/login/begin', {})
    const assertion = await navigator.credentials.get({
      publicKey: publicKeyToRequestOptions(begin.publicKey),
    })
    applyTokens(
      await api.post(
        '/auth/passkey/login/finish?ticket=' + encodeURIComponent(begin.ticket),
        credentialToJSON(assertion)
      )
    )
    await fetchUser()
    return true
  }

  async function fetchUser() {
    try {
      user.value = await api.get('/auth/me')
    } catch {
      logout()
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('refresh_token')
  }

  if (token.value) fetchUser()

  return {
    token,
    user,
    isLoggedIn,
    login,
    completeTotp,
    completeRecovery,
    completePasskey,
    loginWithPasskey,
    logout,
    fetchUser,
    // 改密/改名/解绑等操作会让后端换发新令牌，页面拿到后用它落盘，避免被打回登录页
    adoptTokens: applyTokens,
  }
})
