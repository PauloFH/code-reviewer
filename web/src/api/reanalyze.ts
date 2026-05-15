import { http } from './http'

export interface ReanalyzeInput {
  client_name: string
  project_ref: string
  mr_iid: number
}

export async function reanalyze(data: ReanalyzeInput): Promise<void> {
  await http.post('/reanalyze', data)
}
