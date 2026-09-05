import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(null)
  const isLoggedIn = computed(() => !!token.value)

  async function login(username, password) {
    const res = await api.post('/auth/login', { username, password })
    token.value = res.access_token
    localStorage.setItem('token', token.value)
    if (res.refresh_token) {
      localStorage.setItem('refresh_token', res.refresh_token)
    }
    await fetchUser()
    return true
  }

  async function fetchUser() {
    try {
      const res = await api.get('/auth/me')
      user.value = res
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

  return { token, user, isLoggedIn, login, logout, fetchUser }
})
