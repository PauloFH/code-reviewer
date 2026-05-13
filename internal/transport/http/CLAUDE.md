# CLAUDE.md — internal/transport/http

## Papel

Camada HTTP mais externa. Recebe webhooks das plataformas Git, valida e transforma a entrada, e chama o usecase. Não contém lógica de negócio.

## Arquivos

### `router.go`
- `NewRouter(uc, registry, log)` — monta o `http.ServeMux` com todas as rotas
- `chain(handler, middlewares...)` — aplica middlewares da esquerda para direita
- `registryAdapter` — adapta `domain.ClientRegistry` para a interface local `clientRegistry` (inversão de dependência)

**Rotas:**
- `POST /webhook/{clientName}` — recebe webhook; passa por `webhookAuth` antes do handler
- `GET /healthz` — liveness probe; retorna `200 OK`

### `middleware.go`
- `webhookAuth(registry, log)` — middleware de autenticação do webhook:
  1. Extrai `clientName` da URL
  2. Resolve o `PlatformClient` no registry
  3. Lê o body uma vez (limite: 10 MB) e armazena em memória
  4. Constrói `domain.RawWebhook` com headers em lowercase
  5. Chama `client.ValidateWebhook(raw)` — plataforma-específico
  6. Armazena no contexto via `withRawWebhook(r, raw)`
- `logging(log)` — loga method, path, status e duração de toda requisição

### `handler.go`
- `WebhookHandler.Handle(w, r)` — handler principal:
  1. Lê `rawWebhookCarrier` do contexto (preenchido pelo middleware)
  2. Resolve `PlatformClient` pelo clientName
  3. Chama `client.ParseWebhook(raw)` para normalizar o payload
  4. Chama `usecase.Execute(event)`
  5. Retorna `200` para eventos ignorados (evita reenvios desnecessários)

### `context.go`
- `withRawWebhook` / `rawWebhookFromCtx` — armazenam/recuperam o `rawWebhookCarrier` no contexto HTTP sem expor o tipo para fora do pacote

## Decisões de design

- **Body lido uma vez no middleware:** o body HTTP pode ser lido apenas uma vez. O middleware lê, armazena em `rawWebhookCarrier` e passa via contexto para o handler.
- **Headers em lowercase:** normalização feita no middleware garante que `client.ValidateWebhook` e `client.ParseWebhook` não precisam se preocupar com capitalização (`X-Gitlab-Token` vs `x-gitlab-token`).
- **200 para eventos ignorados:** plataformas interpretam `4xx/5xx` como falha e reenviam o webhook. Retornar `200` para eventos de tipo não suportado (ex: push event) evita reenvios desnecessários.
- **Interface local `clientRegistry`:** o handler e o middleware usam a interface local em vez de `domain.ClientRegistry` diretamente, facilitando testes sem dependência do pacote domain.

## Como as interfaces se encaixam

```
HTTP Request
    ↓
logging middleware
    ↓
webhookAuth middleware
    ↓  (valida secret via PlatformClient.ValidateWebhook)
WebhookHandler.Handle
    ↓  (parseia payload via PlatformClient.ParseWebhook)
usecase.Execute(event)
```
