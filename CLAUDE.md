# CLAUDE.md — Code Reviewer

## O que é este projeto

Serviço HTTP em Go que recebe webhooks de plataformas Git (GitLab, GitHub, Gitea), detecta quando um usuário configurado foi marcado como reviewer/assignee em um MR/PR, busca o diff via API, envia para uma API de IA compatível com `chat/completions` e posta o review como comentário no MR — uma única vez por MR (idempotente).

## Comandos essenciais

```bash
# Rodar o servidor
go run ./cmd/server

# Rodar todos os testes
go test ./...

# Build do binário
go build -o code-reviewer ./cmd/server

# Build da imagem Docker
docker compose up --build

# Verificar compilação sem rodar
go build ./...

# Verificar lint (se golangci-lint instalado)
golangci-lint run ./...
```

## Arquitetura

Clean Architecture com Strategy Pattern para suporte multi-plataforma:

```
transport/http  →  usecase  →  domain interfaces
                                    ↑
                         infrastructure (implementações)
```

- **domain** — entidades e contratos. Sem dependências externas.
- **usecase** — orquestra a regra de negócio. Depende só do domain.
- **infrastructure** — implementa os contratos: platform clients, AI, SQLite, logger.
- **transport** — recebe webhooks HTTP, valida, chama usecase.
- **config** — lê env vars e `clients.json`.

## Convenções do projeto

### Adicionar nova plataforma Git

1. `internal/infrastructure/platform/<nome>/client.go` — implementa `domain.PlatformClient`
2. `internal/infrastructure/platform/<nome>/webhook.go` — implementa `ValidateWebhook` e `ParseWebhook`
3. `internal/domain/platform.go` — adiciona constante `PlatformType`
4. `internal/infrastructure/platform/factory.go` — adiciona `case`
5. `internal/config/clients.go` — adiciona ao switch de validação

Nenhuma outra parte muda.

### Trocar SQLite por outro banco

Implemente `domain.ReviewRepository` no novo pacote de storage e substitua a injeção em `cmd/server/main.go`. Nenhuma outra parte muda.

### Trocar a API de IA

Implemente `domain.AIReviewer` em um novo pacote e substitua a injeção em `main.go`.

## Estrutura de arquivos chave

| Arquivo | Papel |
|---|---|
| `internal/domain/platform.go` | Interface Strategy `PlatformClient` |
| `internal/domain/repository.go` | Interfaces `ReviewRepository`, `AIReviewer`, `PromptRepository` |
| `internal/domain/prompt.go` | Entidades `Prompt` e `PromptAssignment` |
| `internal/infrastructure/platform/factory.go` | Ponto de extensão para novas plataformas |
| `internal/infrastructure/ai/prompt.go` | Construção do prompt + filtro de diff |
| `internal/infrastructure/ai/html_to_text.go` | Conversão HTML (Tiptap) → texto plano |
| `internal/usecase/process_merge_request.go` | Toda a regra de negócio |
| `internal/transport/http/router.go` | Rotas HTTP e wiring de middlewares |
| `internal/transport/http/prompt_handler.go` | CRUD de prompts e atribuições |
| `cmd/server/main.go` | Wiring de dependências e boot |
| `clients.json` | Configuração dos clientes monitorados (não commitado) |

## Configuração mínima para rodar

```bash
cp .env.example .env
cp clients.example.json clients.json
mkdir -p data
# editar .env e clients.json com dados reais
go run ./cmd/server
```

## Rotas disponíveis

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/webhook/{clientName}` | Recebe webhook de qualquer plataforma |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/api/v1/prompts` | Lista prompts |
| `POST` | `/api/v1/prompts` | Cria prompt |
| `GET` | `/api/v1/prompts/{id}` | Busca prompt por ID |
| `PUT` | `/api/v1/prompts/{id}` | Atualiza prompt |
| `DELETE` | `/api/v1/prompts/{id}` | Remove prompt |
| `GET` | `/api/v1/prompts/assignments` | Lista atribuições de prompts |
| `PUT` | `/api/v1/prompts/assignments` | Cria/atualiza atribuição |
| `DELETE` | `/api/v1/prompts/assignments` | Remove atribuição |

## Variáveis de ambiente obrigatórias

`AI_BASE_URL`, `AI_TOKEN` — sem estas o serviço não inicia (panic com mensagem clara).

## Testes

Todos os testes são unitários, sem dependências externas (mocks via interfaces). O pacote `storage` não tem testes unitários — o SQLite é testado via integração manual.

```bash
go test ./... -v   # 49 testes, 0 falhas
```

## Dependências externas

Apenas uma: `github.com/mattn/go-sqlite3` (CGo). Tudo mais é stdlib Go.
