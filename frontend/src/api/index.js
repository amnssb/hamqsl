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

api.interceptors.response.use(
  response => {
    // API 统一返回 { code, message, data }，组件直接使用 data。
    const payload = response.data
    return payload && Object.prototype.hasOwnProperty.call(payload, 'data') ? payload.data : payload
  },
  error => {
    const msg = error.response?.data?.message || error.response?.data?.detail || error.message || '请求失败'
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    } else {
      ElMessage({ message: msg, type: 'error', grouping: true })
    }
    return Promise.reject(error)
  }
)

export default api
