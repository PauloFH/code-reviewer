import axios from 'axios'
import { useAuthStore } from '@/stores/auth'

export const http = axios.create({
  baseURL: '/api/v1',
})

http.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

http.interceptors.response.use(
  (r) => r,
  (err) => {
    if (err.response?.status === 401) {
      // Don't auto-logout on /auth/me — it's expected to fail with stale tokens
      const url = err.config?.url || ''
      if (!url.includes('/auth/me')) {
        const auth = useAuthStore()
        auth.logout()
        window.location.href = '/login'
      }
    }
    return Promise.reject(err)
  },
)
