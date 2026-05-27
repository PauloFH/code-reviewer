# CLAUDE.md — internal/transport/http

## Papel

Camada HTTP mais externa. Recebe webhooks e requisições do dashboard, valida e delega para os usecases. Não contém lógica de negócio.

## Arquivos

### `router.go`
Monta o `http.ServeMux` com todas as rotas e aplica middlewares.

| Método | Rota | Auth | Handler |
|--------|------|------|---------|
| POST | `/webhook/{clientName}` | webhookAuth | WebhookHandler |
| GET | `/healthz` | — | inline |
| POST | `/api/v1/auth/login` | — | AuthHandler.Login |
| GET | `/api/v1/events` | JWT | APIHandler.ListEvents |
| GET | `/api/v1/events/stats` | JWT | APIHandler.GetStats |
| GET | `/api/v1/events/daily` | JWT | APIHandler.GetDailyCosts |
| GET | `/api/v1/events/stream` | JWT (query param) | EventBroadcaster.ServeSSE |
| GET | `/api/v1/clients` | JWT | APIHandler.ListClients |
| POST | `/api/v1/clients` | JWT | APIHandler.CreateClient |
| GET | `/api/v1/clients/{name}` | JWT | APIHandler.GetClient |
| PUT | `/api/v1/clients/{name}` | JWT | APIHandler.UpdateClient |
| DELETE | `/api/v1/clients/{name}` | JWT | APIHandler.DeleteClient |
| GET | `/api/v1/prompts` | JWT | PromptHandler.ListPrompts |
| POST | `/api/v1/prompts` | JWT | PromptHandler.CreatePrompt |
| GET | `/api/v1/prompts/{id}` | JWT | PromptHandler.GetPrompt |
| PUT | `/api/v1/prompts/{id}` | JWT | PromptHandler.UpdatePrompt |
| DELETE | `/api/v1/prompts/{id}` | JWT | PromptHandler.DeletePrompt |
| GET | `/api/v1/prompts/assignments` | JWT | PromptHandler.ListAssignments |
| PUT | `/api/v1/prompts/assignments` | JWT | PromptHandler.SetAssignment |
| DELETE | `/api/v1/prompts/assignments` | JWT | PromptHandler.RemoveAssignment |
| GET/POST | `/api/v1/ai-configs` | JWT (super/gestor) | AIConfigHandler.List/CreateConfig |
| GET/PUT/DELETE | `/api/v1/ai-configs/{id}` | JWT (super/gestor) | AIConfigHandler.Get/Update/DeleteConfig |
| POST | `/api/v1/ai-configs/{id}/test` | JWT (super/gestor) | AIConfigHandler.TestConfig |
| GET/PUT/DELETE | `/api/v1/ai-configs/assignments` | JWT (super/gestor) | AIConfigHandler.List/Set/RemoveAssignment |
| POST | `/api/v1/ai-configs/oauth/claude/start` | JWT (super/gestor) | AIConfigHandler.StartClaudeOAuth |
| POST | `/api/v1/ai-configs/oauth/claude/exchange` | JWT (super/gestor) | AIConfigHandler.ExchangeClaudeOAuth |
| `/*` | SPA catch-all | — | frontendHandler |

`{name}` usa Go 1.22 `r.PathValue("name")`.

### `auth.go`
- `AuthHandler.Login` — valida credenciais e retorna JWT HS256 (24h)
- `jwtAuth(secret)` — middleware; aceita Bearer header ou query param `?token=`
- JWT implementado com stdlib apenas (`crypto/hmac`, `crypto/sha256`, `encoding/base64`)
- Se `DASHBOARD_JWT_SECRET` estiver vazio, gera secret aleatório por boot (tokens não sobrevivem restart)

### `api_handler.go`
- `APIHandler` — endpoints `/api/v1/events/*` e `/api/v1/clients/*`
- Token e webhook_secret retornados como `"***"` (redact)
- Paginação via `?limit=` e `?offset=`; filtros via `?client=`, `?since=`, `?until=`

### `prompt_handler.go`
- `PromptHandler` — endpoints `/api/v1/prompts/*` e `/api/v1/prompts/assignments`
- CRUD de prompts (nome, system_prompt HTML, output_template HTML)
- Atribuições: upsert por (scope, client_name, project_ref); remoção via query params
- Validação: scope deve ser `global`, `client` ou `project`; `prompt_id` obrigatório

### `ai_config_handler.go`
- `AIConfigHandler` — endpoints `/api/v1/ai-configs/*`
- CRUD de configs + atribuições; respostas usam `aiConfigDTO` que **mascara segredos** (só `has_api_key`/`account_connected`/`oauth_expires_at`)
- No update, chave vazia mantém a existente; tokens OAuth são sempre preservados (gerenciados pelo connect)
- OAuth: `StartClaudeOAuth` (gera authorize_url + state) e `ExchangeClaudeOAuth` (troca código → `UpdateOAuthTokens`)
- `TestConfig` delega para a interface local `ConfigTester` (o `ai.ResolvingReviewer`) e retorna o review + usage; erro de upstream vira `502`
- Depende das interfaces locais `ClaudeOAuthFlow` e `ConfigTester` (adaptadas no `main.go`) para não importar o pacote `ai`

### `sse.go`
- `EventBroadcaster` — fan-out não-bloqueante de `domain.ReviewEvent` para N clientes SSE
- `Publish(e)` — chamado pelo usecase após cada review; usa `select/default` para não bloquear
- `ServeSSE(w, r)` — HTTP handler; envia ping inicial, depois eventos até context cancelado
- `subscribe()` / `unsubscribe()` — gerenciamento thread-safe de canais

### `frontend.go`
- Usa `embed.FS` com `//go:embed frontend/dist` para embeber a SPA Vue no binário
- Serve arquivos estáticos; paths desconhecidos retornam `index.html` (SPA routing)

### `middleware.go`
- `webhookAuth` — lê body uma única vez, normaliza headers, valida secret via `PlatformClient.ValidateWebhook`
- `logging` — loga method, path, status e duração

### `handler.go`
- `WebhookHandler.Handle` — parseia webhook via `PlatformClient.ParseWebhook` e chama o usecase

### `context.go`
- `withRawWebhook` / `rawWebhookFromCtx` — armazenam `rawWebhookCarrier` no contexto HTTP

## Testes

```bash
go test ./internal/transport/http/... -v
```

`auth_test.go` cobre: issueToken+validate, token expirado, senha errada, body inválido, middleware com Bearer/query param/sem token.
