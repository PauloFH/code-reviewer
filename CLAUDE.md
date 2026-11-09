# CLAUDE.md — Code Reviewer

## O que é este projeto

**Mergeye** é o nome visível do produto (painel, README e telas). O repositório, o módulo Go, o binário e a imagem Docker continuam `code-reviewer`.

Serviço HTTP em Go que recebe webhooks de plataformas Git (GitLab, GitHub, Gitea), detecta quando um usuário configurado foi marcado como reviewer/assignee em um MR/PR, busca o diff via API, envia para uma IA e posta o review como comentário no MR — uma única vez por MR (idempotente).

A IA é configurável pelo frontend (entidade `AIConfig`, atribuível por escopo global → client → projeto, igual aos prompts), em dois modos:
- **api** — chave de API. Providers: `openai`, `deepseek` (via `/chat/completions`), `gemini` (`generateContent`) e `claude` (API nativa `/v1/messages`).
- **account** — login OAuth na assinatura Claude (fluxo estilo Claude Code: PKCE + code-paste manual + refresh automático). Só `claude`.

Credenciais são cifradas em repouso (AES-256-GCM). Se nenhuma `AIConfig` for resolvida, o serviço cai no **fallback configurado por env** (`gemini` ou `chat_completions`), preservando o comportamento legado.

## Retentativas

O `WebhookHandler` responde `200` ao webhook assim que o evento é parseado e processa a MR numa
goroutine em background (`internal/transport/http/webhook.go`). Dentro do pipeline
(`internal/usecase/process_merge_request.go`), as chamadas de rede (buscar MR/diff na plataforma,
enviar à IA, postar o comentário) são retentadas com backoff exponencial (`RETRY_MAX_ATTEMPTS`,
default 3; `RETRY_BASE_DELAY_SECONDS`, default 2). Se todas as tentativas de um passo falharem, o
serviço posta um comentário no MR explicando o erro e pedindo para comentar `/review` mais tarde —
a claim de idempotência já é liberada nesse caso, então o `/review` não é bloqueado como duplicata.
Essa retentativa é em memória (não sobrevive a um restart do processo no meio da janela de retry).

## Retenção de dados

O histórico de revisões (`review_events`: autor, título e URL do MR) é apagado após 12 meses por uma goroutine em `cmd/server/main.go` (`PurgeEventsBefore`, no boot e a cada 24 h), como declara a política de privacidade em `/privacidade` (`web/src/views/PrivacyView.vue`). Das claims de idempotência (`reviewed_merge_requests`) só o `review_body` é esvaziado: a linha fica para o MR não ser revisado de novo, e um commit novo num MR tão antigo recebe revisão completa em vez de incremental.

## Modos de revisão (perfis)

A revisão é composta por **camadas combináveis**, definidas por um `ReviewProfile` resolvido por escopo (global → client → projeto, igual a prompts/AIConfig). Sem perfil resolvido, o default é `diff`-only (comportamento legado):

- **base `diff`** (sempre): título/descrição/diff.
- **`related`** (flag `include_related`): puxa issues vinculadas (incl. `closes_issues`), MRs relacionados e a thread de discussão, e pede checagem de compatibilidade com as issues. Capacidade opcional da plataforma via `domain.RelatedContextProvider` (só GitLab por ora).
- **`agentic`** (flag `agentic`): clona a working tree da branch no `HeadSHA` e roda um loop de tool-use (read_file/list_dir/grep) onde a IA navega o repo. Funciona com Claude/OpenAI/DeepSeek/Gemini. Repositório obtido via `domain.RepoLocator` (GitLab) + `gitfetch.GitWorkspaceFetcher`: prefere o bare local `@hashed` (via `file://`) quando `GITLAB_REPOS_PATH` está setado (same-machine), senão clona pela rede, com o token passado num header que não fica gravado no clone. O workspace é confinado por `os.Root`: symlinks do MR não escapam dele e `.git` fica inacessível. Falhas degradam graciosamente para o single-shot.
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

1. `internal/domain/platform.go` — constante `PlatformType` e entrada em `Platforms` (a validação de clients deriva daí)
2. `internal/infrastructure/platform/<nome>/` — implementa `domain.PlatformClient` sobre `platform/forge` (identidade, REST autenticado, HMAC); uma API no dialeto GitHub reaproveita `platform/github` como o Gitea
3. `internal/infrastructure/platform/factory.go` — `case` em `Factory.Build`
4. `web/src/domain.ts` — entrada em `PLATFORMS` (rótulo e iniciais)

### Trocar SQLite por outro banco

Implemente os repositórios de `internal/domain/repository.go` no novo pacote de storage e substitua a injeção em `cmd/server/main.go`. Nenhuma outra parte muda.

### Trocar a API de IA

A injeção em `main.go` é o `ai.Reviewer` (implementa `domain.AIReviewer`): ele resolve a `AIConfig` por escopo (sem nenhuma, usa a `AIConfig` montada das env vars), garante token OAuth válido, conversa com o provider e calcula o custo. Para substituir toda a camada de IA, implemente `domain.AIReviewer` em um novo pacote e troque a injeção em `main.go`.

### Adicionar novo provider de IA

1. `internal/domain/ai_config.go` — constante `AIProvider` e linha em `AIProviders` (rótulo, base URL e modelo default, se aceita modo conta). A validação, os defaults e o formulário do frontend (via `GET /api/v1/ai-configs/providers`) derivam dessa tabela.
2. Se o provider fala um protocolo novo (não OpenAI-compatible), um adaptador em `internal/infrastructure/ai/` implementando `conversation` e o `case` em `newConversation`.

### Trocar parâmetros do OAuth de conta (Claude)

Os parâmetros públicos do fluxo (client_id, endpoints, escopos, header beta) ficam em `internal/infrastructure/ai/oauth_claude.go` e `anthropic.go`. É área cinzenta (reaproveita o client do Claude Code); se quebrar, suspeite de mudança do lado da Anthropic.

## Estrutura de arquivos chave

| Arquivo | Papel |
|---|---|
| `internal/domain/platform.go` | Interface Strategy `PlatformClient`, `Platforms`, `RawWebhook` |
| `internal/domain/scope.go` | `Scope`/`ScopeKey`/`Assignment`: atribuição por escopo (global → client → projeto) de prompts, perfis e configs de IA |
| `internal/domain/repository.go` | Contratos de persistência; `ScopedLibrary[T]` serve prompts, perfis e configs de IA |
| `internal/domain/errors.go` | `ErrNotFound`, `ErrConflict`, `Invalid(...)` — o HTTP traduz em 404/409/400 |
| `internal/domain/ai_config.go` | `AIConfig` (modos api/account) e a tabela `AIProviders` |
| `internal/infrastructure/platform/factory.go` | Constrói o client de cada plataforma |
| `internal/infrastructure/platform/forge/forge.go` | O que todo client de plataforma compartilha |
| `internal/infrastructure/ai/reviewer.go` | `Reviewer`: resolve config e prompt por escopo, refresh OAuth, custo |
| `internal/infrastructure/ai/adapter.go` | Um adaptador por protocolo (Anthropic, chat completions, Gemini) para single-shot e agêntico |
| `internal/infrastructure/ai/payload.go` | Tabela das seções da revisão: gera o schema do prompt, decodifica e monta o Markdown |
| `internal/infrastructure/ai/diff.go` | Modelo de linhas do diff: filtra, anota `[new:N]` e valida comentários inline |
| `internal/infrastructure/ai/oauth_claude.go` | Fluxo OAuth Claude (PKCE, exchange, refresh) |
| `internal/infrastructure/storage/storage.go` | `Open`: migra, verifica a chave e cifra segredos legados antes de devolver o repositório |
| `internal/infrastructure/storage/library.go` | CRUD genérico (`record[T]`) e resolução por escopo (`scopedLibrary[T]`) |
| `internal/usecase/process_merge_request.go` | Toda a regra de negócio da revisão |
| `internal/transport/http/server.go` | Todas as rotas e o papel exigido por cada uma |
| `cmd/server/main.go` | Wiring de dependências e boot |
| `web/src/domain.ts` | Tabelas do frontend (escopos, papéis, plataformas, status, navegação) |
| `clients.json` | Seed dos clientes no primeiro boot (não commitado) |

## Configuração mínima para rodar

```bash
cp .env.example .env
cp clients.example.json clients.json
mkdir -p data
# editar .env e clients.json com dados reais
go run ./cmd/server
```

Go >= 1.26.10. A lib `git.pauloholanda.com.br/paulorh/sdk-claude-oauth-go` é privada: use `GOPRIVATE=git.pauloholanda.com.br/*` e credencial de leitura no Git. O build Docker exige o segredo `git_token` (`docker compose up --build` lê `CI_GIT_TOKEN` do ambiente do host).

CI e deploy rodam no GitHub Actions (`.github/workflows/ci.yaml`): push no `main` testa e compila em runners hospedados; depois da aprovação no ambiente `production`, envia o binário por SSH ao `deploy/code-reviewer-deploy.sh`, instalado no servidor como comando forçado da chave. O servidor não tem token nem código. Preparação em `docs/deployment.md`.

## Rotas

Ficam em `internal/transport/http/server.go` (`NewServer`), cada uma com o papel exigido ao lado: `clients`, `prompts`, `review-profiles` e `ai-configs` exigem `super` ou `gestor`; `users`, só `super`; o resto, qualquer usuário logado. Webhooks chegam em `POST /webhook/{client}`.

## Variáveis de ambiente obrigatórias

`AI_PROVIDER` define o **fallback** de IA (`gemini` ou `chat_completions`), usado quando nenhuma `AIConfig` for resolvida no banco. Para Gemini, `GEMINI_API_KEY` é obrigatória; `GEMINI_BASE_URL` e `GEMINI_MODEL` têm defaults. Para `chat_completions`, `AI_BASE_URL` e `AI_TOKEN` são obrigatórias.

`AI_CONFIG_ENC_KEY` (base64 de 32 bytes) cifra as credenciais de IA gravadas pelo frontend e o `token`/`webhook_secret` dos clients (coluna `secrets_encrypted`; linhas antigas em texto puro são cifradas no boot por `storage.Open`, depois de conferir que a chave decifra o que já está gravado). Na prática é obrigatória assim que existe um client — sem ela, o boot falha rápido. Gere com `openssl rand -base64 32` e **mantenha estável** (chave diferente torna os segredos ilegíveis).

A revisão agêntica (perfil com `agentic`) usa variáveis opcionais com defaults: `WORKSPACE_DIR` (raiz dos checkouts efêmeros, default `./data/workspaces`), `AGENTIC_CLONE_DEPTH` (profundidade do clone, default `1`; 0 = clone completo), `AGENTIC_TIMEOUT_SECONDS` (timeout das operações git, default `120`) e `GITLAB_REPOS_PATH` (raiz do `@hashed` para clone local same-machine; vazio = só remoto). O `WORKSPACE_DIR` fica sob `data/` (já no `.gitignore`).

A camada `execution` usa `SANDBOX_IMAGE` (imagem Docker; vazio = desabilitado), `SANDBOX_CPUS` (default `2`), `SANDBOX_MEMORY` (default `2g`) e `SANDBOX_TIMEOUT_SECONDS` (default `120`).

## Testes

Testes unitários, sem serviços externos: fakes via interfaces, `httptest` e SQLite em arquivo temporário (`storage`).

```bash
go test -race ./...
cd web && npm test   # node --test; lógica testável fica em módulos .ts, não em .vue
```

## Dependências externas

`github.com/mattn/go-sqlite3` (CGo), `golang.org/x/crypto` (bcrypt) e a lib privada `sdk-claude-oauth-go`. O resto é stdlib.
