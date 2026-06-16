# CLAUDE.md — Code Reviewer

## O que é este projeto

Serviço HTTP em Go que recebe webhooks de plataformas Git (GitLab, GitHub, Gitea), detecta quando um usuário configurado foi marcado como reviewer/assignee em um MR/PR, busca o diff via API, envia para uma IA e posta o review como comentário no MR — uma única vez por MR (idempotente).

A IA é configurável pelo frontend (entidade `AIConfig`, atribuível por escopo global → client → projeto, igual aos prompts), em dois modos:
- **api** — chave de API. Providers: `openai`, `deepseek` (via `/chat/completions`), `gemini` (`generateContent`) e `claude` (API nativa `/v1/messages`).
- **account** — login OAuth na assinatura Claude (fluxo estilo Claude Code: PKCE + code-paste manual + refresh automático). Só `claude`.

Credenciais são cifradas em repouso (AES-256-GCM). Se nenhuma `AIConfig` for resolvida, o serviço cai no **fallback configurado por env** (`gemini` ou `chat_completions`), preservando o comportamento legado.

## Modos de revisão (perfis)

A revisão é composta por **camadas combináveis**, definidas por um `ReviewProfile` resolvido por escopo (global → client → projeto, igual a prompts/AIConfig). Sem perfil resolvido, o default é `diff`-only (comportamento legado):

- **base `diff`** (sempre): título/descrição/diff.
- **`related`** (flag `include_related`): puxa issues vinculadas (incl. `closes_issues`), MRs relacionados e a thread de discussão, e pede checagem de compatibilidade com as issues. Capacidade opcional da plataforma via `domain.RelatedContextProvider` (só GitLab por ora).
- **`agentic`** (flag `agentic`): clona a working tree da branch no `DiffHeadSHA` e roda um loop de tool-use (read_file/list_dir/grep) onde a IA navega o repo. Funciona com Claude/OpenAI/DeepSeek/Gemini. Repositório obtido via `domain.RepoLocator` (GitLab) + `gitfetch.GitWorkspaceFetcher`: prefere o bare local `@hashed` (via `file://`) quando `GITLAB_REPOS_PATH` está setado (same-machine), senão clona pela rede via token. Falhas degradam graciosamente para o single-shot.
- **`execution`** (flag `execution`): adiciona a tool `run` ao loop, executando comandos (testes/build/lint) num sandbox Docker isolado (`--network=none`, `--read-only`, `-v <ws>:/src:ro`, sem env/segredos). Habilitada só quando `SANDBOX_IMAGE` está setado. Implementação em `internal/infrastructure/sandbox`.

O perfil pode ser sobrescrito por MR via comentário `/review <preset>`: `diff` | `related` | `deep` (agêntico) | `full` (related+agêntico). Spec e planos em `docs/superpowers/`.

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

A injeção em `main.go` é o `ai.ResolvingReviewer` (implementa `domain.AIReviewer`): ele resolve a `AIConfig` por escopo, garante token OAuth válido e despacha para o transporte certo. Para substituir toda a camada de IA, implemente `domain.AIReviewer` em um novo pacote e troque a injeção em `main.go`.

### Adicionar novo provider de IA

1. `internal/infrastructure/ai/<provider>.go` — implementa a interface interna `transport`
2. `internal/infrastructure/ai/client.go` — adiciona `case` em `transportKindForProvider`, `baseURLForConfig`, `modelForConfig` e no `transport()`
3. `internal/domain/ai_config.go` — adiciona a constante `AIProvider`
4. `internal/transport/http/ai_config_handler.go` — adiciona ao switch de `validateConfig`

### Trocar parâmetros do OAuth de conta (Claude)

Os parâmetros públicos do fluxo (client_id, endpoints, escopos, header beta) ficam em `internal/infrastructure/ai/oauth_claude.go` e `anthropic.go`. É área cinzenta (reaproveita o client do Claude Code); se quebrar, suspeite de mudança do lado da Anthropic.

## Estrutura de arquivos chave

| Arquivo | Papel |
|---|---|
| `internal/domain/platform.go` | Interface Strategy `PlatformClient` |
| `internal/domain/repository.go` | Interfaces `ReviewRepository`, `AIReviewer`, `PromptRepository`, `AIConfigRepository` |
| `internal/domain/prompt.go` | Entidades `Prompt` e `PromptAssignment` |
| `internal/domain/ai_config.go` | Entidades `AIConfig` e `AIConfigAssignment` (modos api/account, providers) |
| `internal/infrastructure/platform/factory.go` | Ponto de extensão para novas plataformas |
| `internal/infrastructure/ai/resolver.go` | `ResolvingReviewer`: resolve config por escopo, refresh OAuth, despacho |
| `internal/infrastructure/ai/client.go` | Constrói o transporte por provider/modo; defaults por provider |
| `internal/infrastructure/ai/{gemini,chat_completions,anthropic}.go` | Transportes por provider |
| `internal/infrastructure/ai/oauth_claude.go` | Fluxo OAuth Claude (PKCE, exchange, refresh) |
| `internal/infrastructure/ai/prompt.go` | Construção do prompt + filtro de diff |
| `internal/infrastructure/ai/html_to_text.go` | Conversão HTML (Tiptap) → texto plano |
| `internal/infrastructure/storage/crypto.go` | AES-256-GCM para credenciais em repouso |
| `internal/infrastructure/storage/ai_config_repository.go` | CRUD + `ResolveConfig` + tokens OAuth |
| `internal/usecase/process_merge_request.go` | Toda a regra de negócio |
| `internal/transport/http/router.go` | Rotas HTTP e wiring de middlewares |
| `internal/transport/http/prompt_handler.go` | CRUD de prompts e atribuições |
| `internal/transport/http/ai_config_handler.go` | CRUD de configs de IA, atribuições, OAuth e teste |
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
| `GET/POST` | `/api/v1/ai-configs` | Lista / cria config de IA |
| `GET/PUT/DELETE` | `/api/v1/ai-configs/{id}` | Lê / atualiza / remove config |
| `POST` | `/api/v1/ai-configs/{id}/test` | Roda um review de diff de exemplo na config |
| `GET/PUT/DELETE` | `/api/v1/ai-configs/assignments` | Atribuições de config por escopo |
| `POST` | `/api/v1/ai-configs/oauth/claude/start` | Inicia o OAuth (gera authorize_url + state) |
| `POST` | `/api/v1/ai-configs/oauth/claude/exchange` | Troca o código por tokens e persiste |
| `GET/POST` | `/api/v1/review-profiles` | Lista / cria perfil de revisão |
| `GET/PUT/DELETE` | `/api/v1/review-profiles/{id}` | Lê / atualiza / remove perfil |
| `GET/PUT/DELETE` | `/api/v1/review-profiles/assignments` | Atribuições de perfil por escopo |

Todas as rotas de `prompts`, `clients`, `ai-configs` e `review-profiles` exigem JWT + `requireRole(super, gestor)`; `users` exige `super`.

## Variáveis de ambiente obrigatórias

`AI_PROVIDER` define o **fallback** de IA (`gemini` ou `chat_completions`), usado quando nenhuma `AIConfig` for resolvida no banco. Para Gemini, `GEMINI_API_KEY` é obrigatória; `GEMINI_BASE_URL` e `GEMINI_MODEL` têm defaults. Para `chat_completions`, `AI_BASE_URL` e `AI_TOKEN` são obrigatórias.

`AI_CONFIG_ENC_KEY` (base64 de 32 bytes) cifra as credenciais de IA gravadas pelo frontend. É obrigatória apenas quando existem configs persistidas com segredo — sem ela, o boot falha rápido ao tentar lê-las. Gere com `openssl rand -base64 32` e **mantenha estável** (chave diferente torna os segredos ilegíveis).

A revisão agêntica (perfil com `agentic`) usa variáveis opcionais com defaults: `WORKSPACE_DIR` (raiz dos checkouts efêmeros, default `./data/workspaces`), `AGENTIC_CLONE_DEPTH` (profundidade do clone, default `1`; 0 = clone completo), `AGENTIC_TIMEOUT_SECONDS` (timeout das operações git, default `120`) e `GITLAB_REPOS_PATH` (raiz do `@hashed` para clone local same-machine; vazio = só remoto). O `WORKSPACE_DIR` fica sob `data/` (já no `.gitignore`).

A camada `execution` usa `SANDBOX_IMAGE` (imagem Docker; vazio = desabilitado), `SANDBOX_CPUS` (default `2`), `SANDBOX_MEMORY` (default `2g`) e `SANDBOX_TIMEOUT_SECONDS` (default `120`).

## Testes

Todos os testes são unitários, sem dependências externas (mocks via interfaces e `httptest`). O pacote `storage` tem testes só de `crypto.go`; o resto do SQLite é testado via integração manual.

```bash
go test ./... -v   # 94 testes, 0 falhas
```

## Dependências externas

Apenas uma: `github.com/mattn/go-sqlite3` (CGo). Tudo mais é stdlib Go.
