# CLAUDE.md — internal/usecase

## Papel

Contém toda a regra de negócio do sistema. Não conhece HTTP, SQLite, GitLab ou IA diretamente — só fala com interfaces do domain.

## Arquivos

### `process_merge_request.go`

Pipeline de `Execute(event *domain.WebhookEvent)`:

```
1. Resolve PlatformClient pelo event.ClientName no registry
2. GetMergeRequest(projectRef, mrIID)       → com retry; retorna erro em falha de API
3. mr.IsUserAssigned(username)              → skip se usuário não está no MR
4. repository.AlreadyReviewed(...)          → skip se já revisado (idempotência)
5. GetMergeRequestDiff(projectRef, mrIID)   → com retry; retry esgotado → notifyFailure + erro
6. reviewer.ReviewDiff(input)               → com retry; retry esgotado → notifyFailure + erro
7. CreateMergeRequestNote(projectRef, ...)  → com retry; retry esgotado → notifyFailure + erro
8. postInlineComments(...)                   → best-effort: filtra e posta comentários inline quando suportado
9. repository.MarkAsReviewed(...)           → best-effort: loga erro, não retorna
10. saveEvent(...)                           → best-effort: salva ReviewEvent + publica SSE
```

Steps 8, 9 e 10 são best-effort. `saveEvent` chama `publisher.Publish(e)` para o live feed SSE.

**Retentativas (retry.go):** os passos 2, 5, 6 e 7 (chamadas de rede — plataforma Git e IA) são
envolvidos em `uc.retry.run(log, op, fn)`, que tenta `fn` até `RetryConfig.MaxAttempts` vezes com
backoff exponencial a partir de `BaseDelay` (2s, 4s, 8s, ...). Passos locais (`ClaimReview`, lookup
no registry) não são retentados — falha ali é considerada permanente, não transitória.

Quando os passos 5, 6 ou 7 esgotam as tentativas, `notifyFailure` posta um comentário best-effort no
MR explicando o erro e convidando a tentar de novo comentando `/review` (o `defer` que libera o
`ClaimReview` em caso de falha já garante que essa nova tentativa não seja bloqueada como duplicata).
O passo 2 (buscar a MR) é retentado mas **não** notifica — nesse ponto ainda não se sabe se o usuário
monitorado está de fato envolvido na MR, então postar comentário arriscaria ruído em MRs irrelevantes.

`RetryConfig` é injetado via `NewProcessMergeRequest` (lido de `RETRY_MAX_ATTEMPTS` /
`RETRY_BASE_DELAY_SECONDS` em `config.go`, default 3 tentativas / 2s). O `WebhookHandler` roda
`Execute` numa goroutine em background (ver `transport/http/CLAUDE.md`), então esses `time.Sleep`
entre tentativas não bloqueiam a resposta HTTP do webhook.

Dependências injetadas:
- `registry domain.ClientRegistry`
- `repository domain.ReviewRepository` — idempotência
- `eventRepo domain.ReviewEventRepository` — analytics
- `reviewer domain.AIReviewer`
- `publisher EventPublisher` — interface local para SSE broadcaster
- `reviewer domain.AIReviewer` — na prática é o `ai.ResolvingReviewer`, que escolhe a config por escopo
- `pricePerMInput`, `pricePerMOut float64` — preços por milhão de tokens (defaults de env)
- `retry RetryConfig` — tentativas e delay base para as chamadas de rede (ver acima)

`saveEvent` recebe `*domain.ReviewResult`: usa os preços vindos da config no resultado (`PricePerM*`) quando > 0, senão cai nos defaults injetados. Nos caminhos de erro/skip o resultado é `nil` (custo zero).

### `retry.go`

`RetryConfig{MaxAttempts, BaseDelay}` + `run(log, op, fn) error` — helper genérico de retry com
backoff exponencial, usado apenas pelo `process_merge_request.go`.

### `cost.go`

`CalculateCost(usage domain.TokenUsage, pricePerMInput, pricePerMOutput float64) float64`

Calcula custo USD: `(promptTokens/1M * priceInput) + (completionTokens/1M * priceOutput)`.

### `inline_comments.go`

Filtra os comentários inline sugeridos pela IA:
- mantém apenas comentários com `path`, `line` e `body`
- aceita somente linhas adicionadas do lado novo do diff
- remove duplicados
- limita a `maxInlineCommentsPerReview`

### `client_management.go`

`ClientManagementService` — CRUD de clients com hot-reload do registry.

Interface `ClientFactory`:
```go
BuildFromDomain(c domain.Client, timeout time.Duration) (domain.PlatformClient, error)
```

Métodos:
- `ListClients()` → lista do DB
- `GetClient(name)` → lookup por nome
- `CreateClient(c)` → persiste no DB + registra no registry; rollback se build falhar
- `UpdateClient(oldName, c)` → persiste no DB + replace no registry (suporta rename)
- `DeleteClient(name)` → remove do DB + unregister do registry

## Testes

```bash
go test ./internal/usecase/... -v
```

- `process_merge_request_test.go` — cenários de skip, idempotência, erros em cada step, e retry (recuperação após falha transitória, esgotamento de tentativas + comentário de falha)
- `cost_test.go` — 5 casos (zero, input only, output only, mixed, custom prices)
