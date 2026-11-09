import { http } from './http'
import type { Platform } from '@/domain'

export interface Client {
  id: number
  name: string
  platform: Platform
  base_url: string
  token: string
  webhook_secret: string
  username: string
  created_at: string
  updated_at: string
}

// Na edição, token e webhook_secret vazios mantêm os valores gravados (o servidor só devolve "***").
export type ClientInput = Omit<Client, 'id' | 'created_at' | 'updated_at'>

export async function listClients(): Promise<Client[]> {
  const res = await http.get('/clients')
  return res.data
}

export async function createClient(data: ClientInput): Promise<Client> {
  const res = await http.post('/clients', data)
  return res.data
}

// `name` é o nome atual: é a chave da rota mesmo que o corpo traga outro.
export async function updateClient(name: string, data: ClientInput): Promise<Client> {
  const res = await http.put(`/clients/${encodeURIComponent(name)}`, data)
  return res.data
}
