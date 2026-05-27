import { http } from './http'

export type AIProvider = 'openai' | 'claude' | 'gemini' | 'deepseek'
export type AIMode = 'api' | 'account'

export interface AIConfig {
  id: number
  name: string
  provider: AIProvider
  mode: AIMode
  base_url: string
  model: string
  has_api_key: boolean
  account_connected: boolean
  oauth_expires_at: string
  price_per_m_input_tokens: number
  price_per_m_output_tokens: number
  max_tokens: number
  timeout_seconds: number
  created_at: string
  updated_at: string
}

export interface AIConfigInput {
  name: string
  provider: AIProvider
  mode: AIMode
  base_url: string
  model: string
  api_key?: string
  price_per_m_input_tokens: number
  price_per_m_output_tokens: number
  max_tokens: number
  timeout_seconds: number
}

export interface AIConfigAssignment {
  id: number
  config_id: number
  config_name: string
  scope: 'global' | 'client' | 'project'
  client_name: string
  project_ref: string
}

export interface AIAssignmentInput {
  config_id: number
  scope: 'global' | 'client' | 'project'
  client_name: string
  project_ref: string
}

export interface OAuthStartResponse {
  authorize_url: string
  state: string
}

export interface TestResult {
  review_markdown: string
  inline_comments: { Path: string; Line: number; Body: string }[] | null
  usage: { PromptTokens: number; CompletionTokens: number; TotalTokens: number }
}

export async function listConfigs(): Promise<AIConfig[]> {
  const res = await http.get('/ai-configs')
  return res.data
}

export async function getConfig(id: number): Promise<AIConfig> {
  const res = await http.get(`/ai-configs/${id}`)
  return res.data
}

export async function createConfig(data: AIConfigInput): Promise<AIConfig> {
  const res = await http.post('/ai-configs', data)
  return res.data
}

export async function updateConfig(id: number, data: AIConfigInput): Promise<AIConfig> {
  const res = await http.put(`/ai-configs/${id}`, data)
  return res.data
}

export async function deleteConfig(id: number): Promise<void> {
  await http.delete(`/ai-configs/${id}`)
}

export async function listConfigAssignments(): Promise<AIConfigAssignment[]> {
  const res = await http.get('/ai-configs/assignments')
  return res.data
}

export async function setConfigAssignment(data: AIAssignmentInput): Promise<void> {
  await http.put('/ai-configs/assignments', data)
}

export async function removeConfigAssignment(scope: string, clientName: string, projectRef: string): Promise<void> {
  await http.delete('/ai-configs/assignments', { params: { scope, client_name: clientName, project_ref: projectRef } })
}

export async function startClaudeOAuth(): Promise<OAuthStartResponse> {
  const res = await http.post('/ai-configs/oauth/claude/start')
  return res.data
}

export async function exchangeClaudeOAuth(configId: number, state: string, code: string): Promise<AIConfig> {
  const res = await http.post('/ai-configs/oauth/claude/exchange', { config_id: configId, state, code })
  return res.data
}

export async function testConfig(id: number): Promise<TestResult> {
  const res = await http.post(`/ai-configs/${id}/test`)
  return res.data
}
