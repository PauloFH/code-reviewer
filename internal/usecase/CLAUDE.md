# CLAUDE.md — internal/usecase

## Papel

Contém toda a regra de negócio do sistema. Não conhece HTTP, SQLite, GitLab ou IA diretamente — só fala com interfaces do domain.

## `process_merge_request.go`

### Pipeline de `Execute(event *domain.WebhookEvent)`

```
1. Resolve PlatformClient pelo event.ClientName no registry
2. GetMergeRequest(projectRef, mrIID)       → retorna erro em falha de API
3. mr.IsUserAssigned(username)              → ignora se usuário não está no MR
4. repository.AlreadyReviewed(...)          → ignora se já foi revisado (idempotência)
5. GetMergeRequestDiff(projectRef, mrIID)   → retorna erro em falha de API
6. reviewer.ReviewDiff(input)               → retorna erro se IA falhar
7. CreateMergeRequestNote(projectRef, ...)  → retorna erro se post falhar
8. repository.MarkAsReviewed(...)           → best-effort: loga erro, não retorna
```

### Decisões de design

- **Step 8 é best-effort:** se `MarkAsReviewed` falhar após o comentário já ter sido postado, o usecase retorna `nil` (sucesso). A plataforma não vai receber `5xx` e não vai reenviar o webhook. O UNIQUE constraint no banco previne duplicatas em cenários de retry real.

- **Sem lógica de plataforma aqui:** o usecase não sabe se está falando com GitLab ou GitHub. Tudo passa pela interface `PlatformClient`.

- **Busca o MR completo sempre:** em vez de tentar inferir o assignee do payload do webhook (que varia muito entre plataformas), o usecase sempre busca o MR completo via API para ter os dados confiáveis.

## Dependências injetadas

```go
type ProcessMergeRequest struct {
    registry   domain.ClientRegistry   // resolve PlatformClient por nome
    repository domain.ReviewRepository // idempotência
    reviewer   domain.AIReviewer       // gera o review
    log        *slog.Logger
}
```

## Testes

`process_merge_request_test.go` testa com mocks via interfaces — sem nenhuma dependência externa. Casos cobertos:

- Usuário não atribuído → skip
- MR já revisado → skip
- Erro ao buscar MR → retorna erro
- Erro ao buscar diff → retorna erro
- Erro na IA → retorna erro, sem comentário postado
- Erro ao postar comentário → retorna erro, sem mark
- Erro no mark após post → retorna nil (best-effort)
- Erro no check de idempotência → retorna erro
- Client desconhecido → retorna erro

```bash
go test ./internal/usecase/...
```
