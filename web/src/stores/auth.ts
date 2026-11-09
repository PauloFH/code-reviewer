import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'
import { loginAPI, getMeAPI, type UserInfo } from '@/api/auth'
import { bindAuth } from '@/api/http'
import { readStorage, writeStorage, removeStorage } from '@/brand/storage'
import { roleCan, type Capability } from '@/domain'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(readStorage('jwt'))
  const user = ref<UserInfo | null>(null)

  const isAuthenticated = computed(() => !!token.value)
  const can = (need: Capability | undefined) => roleCan(user.value?.role, need)

  async function login(username: string, password: string) {
    const t = await loginAPI(username, password)
    token.value = t
    writeStorage('jwt', t)
    await loadUser()
  }

  async function loadUser() {
    if (!token.value) return
    try {
      user.value = await getMeAPI()
    } catch (e) {
      user.value = null
      // Token vencido: sai sem recarregar, porque /auth/me também roda em páginas públicas.
      if (axios.isAxiosError(e) && e.response?.status === 401) logout()
    }
  }

  function logout() {
    token.value = null
    user.value = null
    removeStorage('jwt')
  }

  bindAuth(
    () => token.value,
    () => {
      logout()
      window.location.href = '/login'
    },
  )

  const initPromise = loadUser()

  return { token, user, isAuthenticated, can, login, logout, initPromise }
})
