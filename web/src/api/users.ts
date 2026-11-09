import { http } from './http'
import type { Role } from '@/domain'

export interface User {
  id: number
  username: string
  role: Role
  created_at: string
  updated_at: string
}

// Senha vazia na edição mantém a atual.
export interface UserInput {
  username: string
  password?: string
  role: Role
}

export async function listUsers(): Promise<User[]> {
  const res = await http.get('/users')
  return res.data
}

export async function createUser(data: UserInput): Promise<User> {
  const res = await http.post('/users', data)
  return res.data
}

export async function updateUser(id: number, data: UserInput): Promise<User> {
  const res = await http.put(`/users/${id}`, data)
  return res.data
}

export async function deleteUser(id: number): Promise<void> {
  await http.delete(`/users/${id}`)
}
