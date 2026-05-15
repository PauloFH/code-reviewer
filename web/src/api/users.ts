import { http } from './http'

export interface User {
  id: number
  username: string
  role: 'super' | 'gestor' | 'padrao'
  created_at: string
  updated_at: string
}

export interface CreateUserInput {
  username: string
  password: string
  role: 'super' | 'gestor' | 'padrao'
}

export interface UpdateUserInput {
  username: string
  password?: string
  role: 'super' | 'gestor' | 'padrao'
}

export async function listUsers(): Promise<User[]> {
  const res = await http.get('/users')
  return res.data
}

export async function createUser(data: CreateUserInput): Promise<User> {
  const res = await http.post('/users', data)
  return res.data
}

export async function getUser(id: number): Promise<User> {
  const res = await http.get(`/users/${id}`)
  return res.data
}

export async function updateUser(id: number, data: UpdateUserInput): Promise<User> {
  const res = await http.put(`/users/${id}`, data)
  return res.data
}

export async function deleteUser(id: number): Promise<void> {
  await http.delete(`/users/${id}`)
}
