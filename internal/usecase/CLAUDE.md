# CLAUDE.md — internal/usecase

## Papel

Contém toda a regra de negócio do sistema. Não conhece HTTP, SQLite, GitLab ou IA diretamente — só fala com interfaces do domain.

## Arquivos

### `process_merge_request.go`

Pipeline de `Execute(event *domain.WebhookEvent)`:

```
1. Resolve PlatformClient pelo event.ClientName no registry
2. GetMergeRequest(projectRef, mrIID)       → retorna erro em falha de API
3. mr.IsUserAssigned(username)              → skip se usuário não está no MR
4. repository.AlreadyReviewed(...)          → skip se já revisado (idempotência)
5. GetMergeRequestDiff(projectRef, mrIID)   → retorna erro em falha de API
6. reviewer.ReviewDiff(input)               → retorna erro se IA falhar
7. CreateMergeRequestNote(projectRef, ...)  → retorna erro se post falhar
8. postInlineComments(...)                   → best-effort: filtra e posta comentários inline quando suportado
9. repository.MarkAsReviewed(...)           → best-effort: loga erro, não retorna
10. saveEvent(...)                           → best-effort: salva ReviewEvent + publica SSE
```

Steps 8, 9 e 10 são best-effort. `saveEvent` chama `publisher.Publish(e)` para o live feed SSE.

Dependências injetadas:
- `registry domain.ClientRegistry`
- `repository domain.ReviewRepository` — idempotência
- `eventRepo domain.ReviewEventRepository` — analytics
- `reviewer domain.AIReviewer`
- `publisher EventPublisher` — interface local para SSE broadcaster
- `reviewer domain.AIReviewer` — na prática é o `ai.ResolvingReviewer`, que escolhe a config por escopo
- `pricePerMInput`, `pricePerMOut float64` — preços por milhão de tokens (defaults de env)

`saveEvent` recebe `*domain.ReviewResult`: usa os preços vindos da config no resultado (`PricePerM*`) quando > 0, senão cai nos defaults injetados. Nos caminhos de erro/skip o resultado é `nil` (custo zero).

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

- `process_merge_request_test.go` — 9 cenários (skip, idempotência, erros em cada step)
- `cost_test.go` — 5 casos (zero, input only, output only, mixed, custom prices)
