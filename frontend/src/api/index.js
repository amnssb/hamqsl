import axios from 'axios'
import { ElMessage } from 'element-plus'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 令牌自动刷新：并发 401 共享同一个刷新请求，刷新成功后重放原请求
let refreshingPromise = null

function tryRefresh() {
  const rt = localStorage.getItem('refresh_token')
  if (!rt) return Promise.resolve(false)
  if (!refreshingPromise) {
    refreshingPromise = axios.post('/api/auth/refresh', { refresh_token: rt })
      .then(resp => {
        const d = resp?.data?.data ?? resp?.data
        if (d && d.access_token) {
          localStorage.setItem('token', d.access_token)
          if (d.refresh_token) localStorage.setItem('refresh_token', d.refresh_token)
          return true
        }
        return false
      })
      .catch(() => false)
      .finally(() => { refreshingPromise = null })
  }
  return refreshingPromise
}

function clearSession() {
  localStorage.removeItem('token')
  localStorage.removeItem('refresh_token')
  window.location.href = '/login'
}

api.interceptors.response.use(
  response => {
    // API 统一返回 { code, message, data }，组件直接使用 data。
    const payload = response.data
    return payload && Object.prototype.hasOwnProperty.call(payload, 'data') ? payload.data : payload
  },
  async error => {
    const status = error.response?.status
    const config = error.config || {}
    const url = config.url || ''
    // 登录/刷新本身失败不进入刷新流程
    const isAuthCall = url.includes('/auth/login') || url.includes('/auth/refresh')
    if (status === 401 && !isAuthCall && !config._retriedAfterRefresh) {
      config._retriedAfterRefresh = true
      const ok = await tryRefresh()
      if (ok) {
        config.headers = config.headers || {}
        config.headers.Authorization = `Bearer ${localStorage.getItem('token')}`
        return api(config)
      }
      clearSession()
      return Promise.reject(error)
    }
    if (status === 401) {
      clearSession()
      return Promise.reject(error)
    }
    const msg = error.response?.data?.message || error.response?.data?.detail || error.message || '请求失败'
    ElMessage({ message: msg, type: 'error', grouping: true })
    return Promise.reject(error)
  }
)

export default api
