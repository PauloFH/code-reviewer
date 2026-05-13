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
8. repository.MarkAsReviewed(...)           → best-effort: loga erro, não retorna
9. saveEvent(...)                           → best-effort: salva ReviewEvent + publica SSE
```

Steps 8 e 9 são best-effort. `saveEvent` chama `publisher.Publish(e)` para o live feed SSE.

Dependências injetadas:
- `registry domain.ClientRegistry`
- `repository domain.ReviewRepository` — idempotência
- `eventRepo domain.ReviewEventRepository` — analytics
- `reviewer domain.AIReviewer`
- `publisher EventPublisher` — interface local para SSE broadcaster
- `pricePerMInput`, `pricePerMOut float64` — preços por milhão de tokens

### `cost.go`

`CalculateCost(usage domain.TokenUsage, pricePerMInput, pricePerMOutput float64) float64`

Calcula custo USD: `(promptTokens/1M * priceInput) + (completionTokens/1M * priceOutput)`.

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
