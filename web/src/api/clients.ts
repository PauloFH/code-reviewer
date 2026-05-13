import { http } from './http'

export interface Client {
  id: number
  name: string
  platform: string
  base_url: string
  token: string
  webhook_secret: string
  username: string
  created_at: string
  updated_at: string
}

export type ClientInput = Omit<Client, 'id' | 'created_at' | 'updated_at'>

export async function listClients(): Promise<Client[]> {
  const res = await http.get('/clients')
  return res.data
}

export async function getClient(name: string): Promise<Client> {
  const res = await http.get(`/clients/${name}`)
  return res.data
}

export async function createClient(data: ClientInput): Promise<Client> {
  const res = await http.post('/clients', data)
  return res.data
}

export async function updateClient(name: string, data: ClientInput): Promise<Client> {
  const res = await http.put(`/clients/${name}`, data)
  return res.data
}

export async function deleteClient(name: string): Promise<void> {
  await http.delete(`/clients/${name}`)
}
