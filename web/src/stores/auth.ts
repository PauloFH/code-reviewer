import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { loginAPI } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('jwt'))

  const isAuthenticated = computed(() => !!token.value)

  async function login(username: string, password: string) {
    const t = await loginAPI(username, password)
    token.value = t
    localStorage.setItem('jwt', t)
  }

  function logout() {
    token.value = null
    localStorage.removeItem('jwt')
  }

  return { token, isAuthenticated, login, logout }
})
