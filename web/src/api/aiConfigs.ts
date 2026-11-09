import { http } from './http'
import { libraryApi } from './library'
import type { AIMode } from '@/domain'

// Catálogo de provedores vindo do backend: rótulos, defaults e quem aceita login de conta.
export interface AIProviderInfo {
  id: string
  label: string
  default_base_url: string
  default_model: string
  supports_account: boolean
  // 0 = o provedor decide o limite de saída.
  default_max_tokens: number
}

export interface AIConfig {
  id: number
  name: string
  provider: string
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

// api_key ausente na edição mantém a credencial atual; o servidor nunca devolve chave nem token.
export interface AIConfigInput {
  name: string
  provider: string
  mode: AIMode
  base_url: string
  model: string
  api_key?: string
  price_per_m_input_tokens: number
  price_per_m_output_tokens: number
  max_tokens: number
  timeout_seconds: number
}

export interface OAuthStartResponse {
  authorize_url: string
  state: string
}

export interface TestResult {
  review_markdown: string
  inline_comments: { path: string; line: number; body: string }[] | null
  usage: { prompt_tokens: number; completion_tokens: number; total_tokens: number }
}

// O teste sempre responde 200: ok=false traz a causa real em `error`, porque a CDN
// engoliria o corpo de um 5xx. Com ok=true, os campos de TestResult vêm preenchidos.
export interface TestResponse extends Partial<TestResult> {
  ok: boolean
  error?: string
}

export const aiConfigsApi = libraryApi<AIConfig, AIConfigInput>('/ai-configs')

export async function listProviders(): Promise<AIProviderInfo[]> {
  const res = await http.get('/ai-configs/providers')
  return res.data
}

export async function startClaudeOAuth(): Promise<OAuthStartResponse> {
  const res = await http.post('/ai-configs/oauth/claude/start')
  return res.data
}

export async function exchangeClaudeOAuth(configId: number, state: string, code: string): Promise<AIConfig> {
  const res = await http.post('/ai-configs/oauth/claude/exchange', { config_id: configId, state, code })
  return res.data
}

export async function testConfig(id: number): Promise<TestResponse> {
  const res = await http.post(`/ai-configs/${id}/test`)
  return res.data
}
