import { http } from './http'

export interface Prompt {
  id: number
  name: string
  system_prompt: string
  output_template: string
  created_at: string
  updated_at: string
}

export type PromptInput = Pick<Prompt, 'name' | 'system_prompt' | 'output_template'>

export interface PromptAssignment {
  id: number
  prompt_id: number
  prompt_name: string
  scope: 'global' | 'client' | 'project'
  client_name: string
  project_ref: string
}

export interface AssignmentInput {
  prompt_id: number
  scope: 'global' | 'client' | 'project'
  client_name: string
  project_ref: string
}

export async function listPrompts(): Promise<Prompt[]> {
  const res = await http.get('/prompts')
  return res.data
}

export async function getPrompt(id: number): Promise<Prompt> {
  const res = await http.get(`/prompts/${id}`)
  return res.data
}

export async function createPrompt(data: PromptInput): Promise<Prompt> {
  const res = await http.post('/prompts', data)
  return res.data
}

export async function updatePrompt(id: number, data: PromptInput): Promise<Prompt> {
  const res = await http.put(`/prompts/${id}`, data)
  return res.data
}

export async function deletePrompt(id: number): Promise<void> {
  await http.delete(`/prompts/${id}`)
}

export async function listAssignments(): Promise<PromptAssignment[]> {
  const res = await http.get('/prompts/assignments')
  return res.data
}

export async function setAssignment(data: AssignmentInput): Promise<void> {
  await http.put('/prompts/assignments', data)
}

export async function removeAssignment(scope: string, clientName: string, projectRef: string): Promise<void> {
  await http.delete('/prompts/assignments', { params: { scope, client_name: clientName, project_ref: projectRef } })
}
