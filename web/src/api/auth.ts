import axios from 'axios'
import { http } from './http'
import type { UserInfo } from '@/stores/auth'

export async function loginAPI(username: string, password: string): Promise<string> {
  const res = await axios.post('/api/v1/auth/login', { username, password })
  return res.data.token as string
}

export async function getMeAPI(): Promise<UserInfo> {
  const res = await http.get('/auth/me')
  return res.data
}
