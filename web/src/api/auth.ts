import { http } from './http'
import type { Role } from '@/domain'

export interface UserInfo {
  id: number
  username: string
  role: Role
}

export async function loginAPI(username: string, password: string): Promise<string> {
  const res = await http.post('/auth/login', { username, password }, { skipAuthRedirect: true })
  return res.data.token as string
}

export async function getMeAPI(): Promise<UserInfo> {
  const res = await http.get('/auth/me', { skipAuthRedirect: true })
  return res.data
}
