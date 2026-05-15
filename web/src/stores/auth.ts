import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { loginAPI, getMeAPI } from '@/api/auth'

export interface UserInfo {
  id: number
  username: string
  role: 'super' | 'gestor' | 'padrao'
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('jwt'))
  const user = ref<UserInfo | null>(null)

  const isAuthenticated = computed(() => !!token.value)
  const isSuper = computed(() => user.value?.role === 'super')
  const isGestor = computed(() => user.value?.role === 'gestor')
  const canManageUsers = computed(() => isSuper.value)
  const canManageResources = computed(() => isSuper.value || isGestor.value)

  async function login(username: string, password: string) {
    const t = await loginAPI(username, password)
    token.value = t
    localStorage.setItem('jwt', t)
    await fetchMe()
  }

  async function fetchMe() {
    try {
      const data = await getMeAPI()
      user.value = data
    } catch {
      user.value = null
    }
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('jwt')
  }

  if (token.value) {
    fetchMe()
  }

  return { token, user, isAuthenticated, isSuper, isGestor, canManageUsers, canManageResources, login, logout, fetchMe }
})
