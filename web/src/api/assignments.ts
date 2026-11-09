import { http } from './http'
import type { Scope } from '@/domain'

// Mesmo formato para prompts, modelos de IA e perfis de revisão; o backend limpa client_name/project_ref conforme o escopo.
export interface Assignment {
  id: number
  target_id: number
  target_name: string
  scope: Scope
  client_name: string
  project_ref: string
}

export type AssignmentInput = Omit<Assignment, 'id' | 'target_name'>

export type AssignmentApi = ReturnType<typeof assignmentApi>

export function assignmentApi(resource: string) {
  const url = `${resource}/assignments`
  return {
    async list(): Promise<Assignment[]> {
      return (await http.get(url)).data
    },
    async set(data: AssignmentInput): Promise<void> {
      await http.put(url, data)
    },
    async remove(a: Pick<Assignment, 'scope' | 'client_name' | 'project_ref'>): Promise<void> {
      await http.delete(url, { params: { scope: a.scope, client_name: a.client_name, project_ref: a.project_ref } })
    },
  }
}
