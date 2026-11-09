import axios from 'axios'

export const API_BASE = '/api/v1'

declare module 'axios' {
  interface AxiosRequestConfig {
    // 401 nesta chamada é resposta esperada (senha errada, token vencido) e fica com quem chamou.
    skipAuthRedirect?: boolean
  }
}

// O store de auth registra aqui como ler o token e o que fazer no 401; http não importa o store (evita ciclo).
let getToken: () => string | null = () => null
let onUnauthorized: () => void = () => {}

export function bindAuth(token: () => string | null, unauthorized: () => void): void {
  getToken = token
  onUnauthorized = unauthorized
}

export function authHeader(): Record<string, string> {
  const t = getToken()
  return t ? { Authorization: `Bearer ${t}` } : {}
}

// Erro 4xx do backend vem em texto puro (http.Error) e já é a mensagem para o usuário.
// 5xx e corpos de outro tipo (HTML da CDN num 502) caem no fallback em pt-BR.
export function apiErrorMessage(e: unknown, fallback: string): string {
  const res = axios.isAxiosError(e) ? e.response : undefined
  if (!res || res.status >= 500) return fallback
  const type = String(res.headers['content-type'] ?? '')
  const data = res.data
  return type.startsWith('text/plain') && typeof data === 'string' && data.trim() ? data.trim() : fallback
}

export const http = axios.create({ baseURL: API_BASE })

http.interceptors.request.use((config) => {
  config.headers.set(authHeader())
  return config
})

http.interceptors.response.use(
  (r) => r,
  (err) => {
    if (err.response?.status === 401 && !err.config?.skipAuthRedirect) onUnauthorized()
    return Promise.reject(err)
  },
)
