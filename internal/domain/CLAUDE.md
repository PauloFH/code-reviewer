# CLAUDE.md — internal/domain

## Papel

Camada mais interna da Clean Architecture. Define **o que** o sistema faz, sem depender de nenhuma tecnologia externa (sem HTTP, sem banco, sem plataformas Git).

## Regra fundamental

Este pacote **não pode importar nenhum pacote externo**. Apenas stdlib Go. Qualquer violação dessa regra quebra o isolamento da arquitetura.

## Arquivos

### `platform.go`
Contém o **Strategy Pattern** central do projeto.

- `PlatformType` — tipo string enumerado: `"gitlab"`, `"gitlab-selfhosted"`, `"github"`, `"gitea"`, `"gitea-selfhosted"`
- `RawWebhook` — abstração do request HTTP sem importar `net/http`; permite que a interface seja testável sem HTTP
- `WebhookEvent` — evento normalizado produzido por qualquer plataforma
- `PlatformClient` — interface Strategy; cada plataforma implementa este contrato
- `ClientRegistry` — interface do registro de clients; implementada em `infrastructure/platform/registry.go`

### `merge_request.go`
- `MergeRequest` — entidade central; usa `ProjectRef string` para ser agnóstica à plataforma (GitLab usa ID numérico como string, GitHub/Gitea usam `"owner/repo"`)
- `IsUserAssigned(username string) bool` — regra de negócio: verifica reviewers e assignees

### `review.go`
- `ReviewInput` — entrada para o revisor de IA
- `ReviewResult` — saída do revisor de IA
- `ReviewedMergeRequest` — modelo de persistência

### `repository.go`
- `ReviewRepository` — contrato de persistência; assinatura inclui `clientName` e `projectRef` para isolar registros entre plataformas
- `AIReviewer` — contrato do cliente de IA

## Onde mexer

- **Adicionar campo em MergeRequest:** edite `merge_request.go` e propague aos mappers em cada platform client.
- **Adicionar nova plataforma:** adicione a constante em `platform.go` e implemente `PlatformClient` na infrastructure.
- **Mudar regra de detecção de usuário:** edite `IsUserAssigned` em `merge_request.go` — é o único lugar.
