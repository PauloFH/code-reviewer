# CLAUDE.md — internal/domain

## Papel

Camada mais interna da Clean Architecture. Define **o que** o sistema faz, sem depender de nenhuma tecnologia externa (sem HTTP, sem banco, sem plataformas Git).

## Regra fundamental

Este pacote **não pode importar nenhum pacote externo**. Apenas stdlib Go. Qualquer violação dessa regra quebra o isolamento da arquitetura.

## Arquivos

### `platform.go`
- `PlatformType` — enumerado: `"gitlab"`, `"gitlab-selfhosted"`, `"github"`, `"gitea"`, `"gitea-selfhosted"`
- `RawWebhook` — abstração do request HTTP sem importar `net/http`
- `WebhookEvent` — evento normalizado produzido por qualquer plataforma
- `PlatformClient` — interface Strategy; cada plataforma implementa este contrato
- `ErrInlineCommentsUnsupported` — erro sentinela para plataforma sem suporte a comentários inline no diff
- `ClientRegistry` — interface do registro de clients; implementada em `infrastructure/platform/registry.go`

### `merge_request.go`
- `MergeRequest` — entidade central; `ProjectRef string` agnóstica à plataforma
- `IsUserAssigned(username string) bool` — regra de negócio: verifica reviewers e assignees

### `review.go`
- `ReviewInput` — entrada para o revisor de IA
- `ReviewResult` — saída do revisor de IA com `Usage TokenUsage`
- `TokenUsage` — prompt/completion/total tokens retornados pela API de IA
- `InlineComment` — comentário estruturado para postagem inline no diff (`Path`, `Line`, `Body`)

### `client.go`
- `Client` — entidade de um client de plataforma gerenciado (espelho de `config.ClientConfig` no domínio para evitar ciclo de imports). Todos os campos têm json tags snake_case.

### `prompt.go`
- `Prompt` — entidade de um prompt reutilizável com `SystemPrompt` e `OutputTemplate` (armazenados como HTML do editor Tiptap). Todos os campos têm json tags snake_case.
- `PromptAssignment` — vincula um `Prompt` a um escopo: `"global"`, `"client"` ou `"project"`. Campos `ClientName` e `ProjectRef` definem a especificidade.

### `review_event.go`
- `ReviewEvent` — registro completo de um processamento de webhook (tokens, custo, duração, status)
- `ReviewEventFilter` — filtros para consulta de eventos
- `ReviewStats` — agregados globais e por cliente para o dashboard
- `ClientStat` — stat por cliente dentro de `ReviewStats`
- `DailyCostPoint` — ponto do gráfico de custo diário (`date`, `cost_usd`, `reviews`)
- Todos os structs têm json tags snake_case.

### `repository.go`
Interfaces de persistência:
- `ReviewRepository` — idempotência de reviews
- `AIReviewer` — contrato do cliente de IA
- `ClientRepository` — CRUD de clients + `SeedFromConfigs` + `CountClients`
- `ReviewEventRepository` — `SaveEvent`, `ListEvents`, `CountEvents`, `GetStats`, `GetDailyCosts`
- `PromptRepository` — CRUD de prompts + atribuições + `ResolvePrompt(clientName, projectRef)` (hierarquia projeto > client > global)

## Onde mexer

- **Adicionar campo em MergeRequest:** edite `merge_request.go` e propague aos mappers em cada platform client.
- **Adicionar nova plataforma:** adicione a constante em `platform.go` e implemente `PlatformClient` na infrastructure.
- **Mudar regra de detecção de usuário:** edite `IsUserAssigned` em `merge_request.go`.
- **Novos agregados de stats:** adicione em `ReviewStats`/`ClientStat` e implemente em `sqlite_repository.go`.
