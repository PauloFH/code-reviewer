// API fictícia só para desenvolvimento da interface: `node web/tests/mock-api.mjs [--empty]`.
// Escuta em :8080 (alvo do proxy do Vite). Não faz parte do build nem dos testes.
import { createServer } from 'node:http'

const EMPTY = process.argv.includes('--empty')
const PORT = 8080
const now = Date.now()
const ago = (min) => new Date(now - min * 60_000).toISOString()

const longError = ('Falha ao postar o comentário no MR: a plataforma respondeu 502 após três tentativas. ' +
  'Detalhe do backend sem quebras: ' + 'x'.repeat(300)).slice(0, 300)

const ev = (id, client, project, iid, title, status, tokens, cost, ms, min, err = '') => ({
  id, client_name: client, project_ref: project, mr_iid: iid, mr_id: 9000 + id, mr_title: title,
  mr_url: `https://git.example.com/${project}/-/merge_requests/${iid}`, author_username: 'ana.souza',
  prompt_tokens: Math.round(tokens * 0.8), completion_tokens: Math.round(tokens * 0.2), total_tokens: tokens,
  cost_usd: cost, duration_ms: ms, status, error_message: err, reviewed_at: ago(min),
})

const events = EMPTY ? [] : [
  ev(1, 'acme', 'acme/api', 412, 'Corrige paginação da listagem de pedidos', 'success', 18400, 0.0842, 21300, 3),
  ev(2, 'acme', 'acme/web', 98, 'Migra formulário de checkout para o novo componente', 'success', 25100, 0.1131, 33800, 14),
  ev(3, 'globex', 'globex/billing', 57, 'Ajusta arredondamento de impostos', 'skipped', 0, 0, 120, 38),
  ev(4, 'initech', 'initech/core', 233, 'Refatora fila de notificações', 'success', 9300, 0.0417, 12900, 75),
  ev(5, 'globex', 'globex/billing', 58, 'Adiciona exportação de faturas em CSV', 'error', 1200, 0.0051, 4100, 130, 'timeout ao chamar o provedor de IA'),
  ev(6, 'acme', 'acme/api',
    413, 'Reescreve integralmente o módulo de autenticação para suportar múltiplos provedores de identidade, tokens rotativos e revogação em cascata de sessões',
    'error', 2300, 0.0102, 8800, 190, longError),
  ev(7, 'initech', 'initech/infra/terraform/modules/rede-privada-compartilhada-producao', 12, 'Atualiza módulo de rede', 'success', 14200, 0.0633, 17000, 260),
]

const byClient = EMPTY ? [] : [
  { client_name: 'acme', review_count: 214, total_tokens: 3_900_000, total_cost_usd: 17.42 },
  { client_name: 'globex', review_count: 131, total_tokens: 2_100_000, total_cost_usd: 9.88 },
  { client_name: 'initech', review_count: 97, total_tokens: 1_480_000, total_cost_usd: 6.31 },
]
const stats = EMPTY
  ? { total_reviews: 0, total_tokens: 0, total_cost_usd: 0, reviews_last_30d: 0, cost_last_30d: 0, by_client: [] }
  : { total_reviews: 1284, total_tokens: 7_480_000, total_cost_usd: 38.17, reviews_last_30d: 442, cost_last_30d: 33.61, by_client: byClient }

const daily = (days) => EMPTY ? [] : Array.from({ length: days }, (_, i) => {
  const d = new Date(now - (days - 1 - i) * 86_400_000).toISOString().slice(0, 10)
  const r = Math.round(8 + 10 * Math.abs(Math.sin(i * 0.7)))
  return { date: d, cost_usd: +(r * 0.11).toFixed(4), reviews: r }
})

const stamp = '2026-10-01T12:00:00Z'
const clients = EMPTY ? [] : [
  { id: 1, name: 'acme', platform: 'gitlab', base_url: 'https://git.acme.example', token: '********', webhook_secret: '********', username: 'revisor', created_at: stamp, updated_at: stamp },
  { id: 2, name: 'globex', platform: 'github', base_url: 'https://api.github.com', token: '********', webhook_secret: '********', username: 'revisor-bot', created_at: stamp, updated_at: stamp },
  { id: 3, name: 'initech', platform: 'gitea', base_url: 'https://gitea.initech.example', token: '********', webhook_secret: '********', username: 'revisor', created_at: stamp, updated_at: stamp },
]
const prompts = EMPTY ? [] : [
  { id: 1, name: 'Padrão', system_prompt: '<p>Você é um revisor de código criterioso.</p>', output_template: '## Resumo\n\n## Problemas', created_at: stamp, updated_at: stamp },
  { id: 2, name: 'Segurança', system_prompt: '<p>Foque em vulnerabilidades.</p>', output_template: '## Riscos', created_at: stamp, updated_at: stamp },
]
const promptAssignments = EMPTY ? [] : [
  { id: 1, prompt_id: 1, prompt_name: 'Padrão', scope: 'global', client_name: '', project_ref: '' },
  { id: 2, prompt_id: 2, prompt_name: 'Segurança', scope: 'client', client_name: 'globex', project_ref: '' },
]
const aiConfigs = EMPTY ? [] : [
  { id: 1, name: 'Claude Sonnet', provider: 'claude', mode: 'api', base_url: '', model: 'claude-sonnet-5-5', has_api_key: true, account_connected: false, oauth_expires_at: '', price_per_m_input_tokens: 3, price_per_m_output_tokens: 15, max_tokens: 8192, timeout_seconds: 120, created_at: stamp, updated_at: stamp },
  { id: 2, name: 'Gemini', provider: 'gemini', mode: 'api', base_url: '', model: 'gemini-2.5-pro', has_api_key: true, account_connected: false, oauth_expires_at: '', price_per_m_input_tokens: 1.25, price_per_m_output_tokens: 10, max_tokens: 8192, timeout_seconds: 120, created_at: stamp, updated_at: stamp },
]
const aiAssignments = EMPTY ? [] : [{ id: 1, config_id: 1, config_name: 'Claude Sonnet', scope: 'global', client_name: '', project_ref: '' }]
const profiles = EMPTY ? [] : [
  { id: 1, name: 'Somente diff', include_related: false, agentic: false, execution: false, max_iterations: 0, created_at: stamp, updated_at: stamp },
  { id: 2, name: 'Completo', include_related: true, agentic: true, execution: false, max_iterations: 12, created_at: stamp, updated_at: stamp },
]
const profileAssignments = EMPTY ? [] : [{ id: 1, profile_id: 1, profile_name: 'Somente diff', scope: 'global', client_name: '', project_ref: '' }]
const users = [
  { id: 1, username: 'admin', role: 'super', created_at: stamp, updated_at: stamp },
  ...(EMPTY ? [] : [
    { id: 2, username: 'marina', role: 'gestor', created_at: stamp, updated_at: stamp },
    { id: 3, username: 'caio', role: 'padrao', created_at: stamp, updated_at: stamp },
  ]),
]

const sseClients = new Set()
let nextId = 100

// Escolhe um evento aleatório a cada 20s para o feed "Ao vivo" (não roda no modo --empty).
if (!EMPTY) {
  setInterval(() => {
    if (!sseClients.size) return
    const base = events[Math.floor(Math.random() * events.length)]
    const e = { ...base, id: 0, reviewed_at: new Date().toISOString() }
    for (const res of sseClients) res.write(`data: ${JSON.stringify(e)}\n\n`)
  }, 20_000)
}

const send = (res, code, body) => {
  res.writeHead(code, { 'Content-Type': 'application/json' })
  res.end(body === undefined ? '' : JSON.stringify(body))
}

const readBody = (req) => new Promise((resolve) => {
  let raw = ''
  req.on('data', (c) => { raw += c })
  req.on('end', () => { try { resolve(raw ? JSON.parse(raw) : {}) } catch { resolve({}) } })
})

// Listas de recursos: GET lista, POST cria (eco com id), PUT/DELETE respondem sem alterar o estado.
const resources = {
  clients: { list: clients, key: 'name' },
  prompts: { list: prompts, key: 'id', assignments: promptAssignments },
  'ai-configs': { list: aiConfigs, key: 'id', assignments: aiAssignments },
  'review-profiles': { list: profiles, key: 'id', assignments: profileAssignments },
  users: { list: users, key: 'id' },
}

createServer(async (req, res) => {
  const url = new URL(req.url, `http://localhost:${PORT}`)
  const path = url.pathname.replace(/^\/api\/v1/, '')
  const m = req.method
  const q = url.searchParams

  if (m === 'POST' && path === '/auth/login') {
    const b = await readBody(req)
    if (!b.username || !b.password) return send(res, 401, { error: 'credenciais inválidas' })
    return send(res, 200, { token: 'mock-token' })
  }
  if (m === 'GET' && path === '/auth/me') return send(res, 200, { id: 1, username: 'admin', role: 'super' })

  if (m === 'GET' && path === '/events/stats') {
    const c = q.get('client')
    return send(res, 200, c ? { ...stats, by_client: stats.by_client.filter((x) => x.client_name === c) } : stats)
  }
  if (m === 'GET' && path === '/events/daily') {
    return send(res, 200, daily(Number(q.get('days')) || 30))
  }
  if (m === 'GET' && path === '/events/stream') {
    res.writeHead(200, { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache', Connection: 'keep-alive' })
    res.write(': conectado\n\n')
    sseClients.add(res)
    req.on('close', () => sseClients.delete(res))
    return
  }
  if (m === 'GET' && path === '/events') {
    const client = q.get('client')
    const search = (q.get('search') || '').toLowerCase()
    let items = events.filter((e) => (!client || e.client_name === client) &&
      (!search || `${e.mr_title} ${e.project_ref}`.toLowerCase().includes(search)))
    const total = items.length
    const offset = Number(q.get('offset')) || 0
    const limit = Number(q.get('limit')) || 50
    items = items.slice(offset, offset + limit)
    return send(res, 200, { items, total })
  }
  if (m === 'POST' && path === '/reanalyze') return send(res, 202)

  // /<recurso>/assignments tem de vir antes de /<recurso>/{id}
  const parts = path.split('/').filter(Boolean)
  const r = resources[parts[0]]
  if (r) {
    if (parts[1] === 'assignments') {
      if (m === 'GET') return send(res, 200, r.assignments || [])
      return send(res, 204)
    }
    if (parts[0] === 'ai-configs' && parts[1] === 'oauth') {
      return send(res, 200, parts[3] === 'start' ? { authorize_url: 'https://claude.ai/oauth/authorize?mock=1', state: 'mock-state' } : aiConfigs[0] ?? {})
    }
    if (parts[0] === 'ai-configs' && parts[2] === 'test' && m === 'POST') {
      return send(res, 200, { ok: true, review_markdown: '**Revisão de teste**\n\nNenhum problema encontrado.', inline_comments: null, usage: { PromptTokens: 120, CompletionTokens: 40, TotalTokens: 160 } })
    }
    if (parts.length === 1) {
      if (m === 'GET') return send(res, 200, r.list)
      if (m === 'POST') return send(res, 201, { id: nextId++, created_at: stamp, updated_at: stamp, ...(await readBody(req)) })
    } else {
      const item = r.list.find((x) => String(x[r.key]) === decodeURIComponent(parts[1]))
      if (m === 'GET') return item ? send(res, 200, item) : send(res, 404, { error: 'não encontrado' })
      if (m === 'PUT') return send(res, 200, { ...(item || {}), ...(await readBody(req)) })
      if (m === 'DELETE') return send(res, 204)
    }
  }
  send(res, 404, { error: 'rota não encontrada' })
}).listen(PORT, () => console.log(`API fictícia em :${PORT}${EMPTY ? ' (--empty)' : ''}`))
