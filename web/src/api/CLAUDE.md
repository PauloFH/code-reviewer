# CLAUDE.md — web/src/api/

## Papel

Camada de acesso à API REST. Cada módulo exporta funções async e os tipos TypeScript correspondentes. Todos os tipos espelham as structs Go com json tags (snake_case).

---

## `http.ts` — instância axios

```ts
export const http = axios.create({ baseURL: '/api/v1' })
```

**Interceptor de request:** injeta `Authorization: Bearer <token>` quando `auth.token` existe.

**Interceptor de response:** em resposta `401` chama `auth.logout()` e redireciona para `/login` via `window.location.href`. Isso cobre tokens expirados sem precisar verificar em cada chamada.

---

## `auth.ts`

```ts
loginAPI(username: string, password: string): Promise<string>
```

`POST /api/v1/auth/login` → retorna o JWT como string. O store `auth` armazena o token e chama essa função.

---

## `events.ts` — tipos e funções

### Tipos

| Tipo | Descrição |
|------|-----------|
| `ReviewEvent` | Registro completo de um review: tokens, custo, status, MR metadata |
| `ReviewStats` | Agregados globais + `by_client: ClientStat[]` |
| `DailyCostPoint` | Ponto do gráfico: `{ date, cost_usd, reviews }` |
| `EventFilter` | Filtros para `listEvents`: `client?, since?, until?, limit?, offset?` |
| `EventsResponse` | `{ items: ReviewEvent[], total: number }` |

`ReviewEvent.status`: `'success' | 'skipped' | 'error'`

### Funções

```ts
listEvents(filter?: EventFilter): Promise<EventsResponse>
// GET /events?client=&since=&until=&limit=&offset=

getStats(client?: string): Promise<ReviewStats>
// GET /events/stats?client=

getDailyCosts(client?: string, days?: number): Promise<DailyCostPoint[]>
// GET /events/daily?days=30&client=

createEventSource(token: string): EventSource
// new EventSource('/api/v1/events/stream?token=<jwt>')
// SSE — EventSource não suporta headers, por isso o token vai na query string
```

---

## `clients.ts` — tipos e funções

### Tipos

```ts
interface Client {
  id: number; name: string; platform: string
  base_url: string; token: string; webhook_secret: string
  username: string; created_at: string; updated_at: string
}

type ClientInput = Omit<Client, 'id' | 'created_at' | 'updated_at'>
```

O servidor retorna `token` e `webhook_secret` como `"***"` (redacted). Ao editar um client, envie strings vazias para manter os valores existentes.

### Funções

```ts
listClients(): Promise<Client[]>          // GET /clients
getClient(name: string): Promise<Client>  // GET /clients/{name}
createClient(data: ClientInput): Promise<Client>           // POST /clients
updateClient(name: string, data: ClientInput): Promise<Client>  // PUT /clients/{name}
deleteClient(name: string): Promise<void>                  // DELETE /clients/{name}
```

`updateClient` usa o `name` original (antes de qualquer rename) como path param.

---

## `aiConfigs.ts` — tipos e funções

### Tipos

- `AIConfig` — config de IA (resposta mascarada: `has_api_key`/`account_connected`, sem segredos), com `provider`, `mode`, `base_url`, `model`, preços, `max_tokens`, `timeout_seconds`.
- `AIConfigInput` — payload de create/update; `api_key` opcional (vazio mantém a atual).
- `AIConfigAssignment` / `AIAssignmentInput` — atribuição por escopo (espelha prompts).
- `TestResult` — `{ review_markdown, inline_comments, usage }`.

### Funções

```ts
listConfigs() / getConfig(id) / createConfig(data) / updateConfig(id, data) / deleteConfig(id)
listConfigAssignments() / setConfigAssignment(data) / removeConfigAssignment(scope, client, project)
startClaudeOAuth(): Promise<{ authorize_url, state }>     // POST /ai-configs/oauth/claude/start
exchangeClaudeOAuth(configId, state, code): Promise<AIConfig>  // .../exchange
testConfig(id): Promise<TestResult>                       // POST /ai-configs/{id}/test
```

O servidor nunca devolve chave/token — só as flags de presença. Ao editar, envie `api_key` vazio para manter a credencial atual.
