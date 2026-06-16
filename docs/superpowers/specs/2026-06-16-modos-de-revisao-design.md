# Spec — Modos de revisão (camadas combináveis + revisão agêntica)

- **Data:** 2026-06-16
- **Status:** Aprovado (desenho) — pronto para plano de implementação da Fase 1
- **Autor:** Paulo Holanda (brainstorming assistido)

## 1. Contexto e objetivo

Hoje o serviço revisa um MR/PR usando **somente o diff**: o `usecase.ProcessMergeRequest`
busca o diff via API da plataforma, manda para o `ai.ResolvingReviewer` (que resolve a
`AIConfig` por escopo) e posta o review como comentário. O prompt é montado em
`internal/infrastructure/ai/prompt.go` com título, descrição e diff.

O objetivo é tornar a revisão **mais profunda e configurável**, com três níveis de contexto
que se combinam:

1. **Só diff** (comportamento atual).
2. **Checkout da branch + investigação agêntica**: clonar a working tree da branch do MR e
   deixar a IA navegar o repositório (ler arquivos, listar diretórios, fazer grep, ver
   histórico) para julgar se a mudança faz o que deveria, no contexto real do código.
3. **Contexto relacionado**: ler o que está no MR e nos itens relacionados — issues
   vinculadas (incluindo verificar se o MR cumpre a descrição da issue), MRs relacionados e
   a thread de discussão do próprio MR.

Foi avaliada também a viabilidade de, instalando o serviço **na mesma máquina** de um GitLab
self-hosted, percorrer o workspace do GitLab no disco. Conclusão: é viável, porém o GitLab
guarda repositórios **bare** (`@hashed/.../<hash>.git`) sem working tree — então mesmo no
disco local é preciso materializar uma working tree (clone/worktree/archive). O ganho do
acesso local é **velocidade** (clonar de um path local dispensa rede e token), não evitar o
checkout. Por isso o acesso local entra como otimização (Fase 2), com o clone remoto via
token como base (Fase 1).

## 2. Decisões de design (decision log)

Decisões tomadas no brainstorming, na ordem em que foram fechadas:

| # | Tema | Decisão |
|---|------|---------|
| 1 | Motor de revisão | **Agêntico** — a IA navega o repo num loop de tool-use (não apenas contexto estático). |
| 2 | Taxonomia | **Camadas combináveis** — base `diff` (sempre) + flags `related` e `agentic` que se somam. |
| 3 | Acesso ao repositório | **Clone local + fallback remoto** — bare `@hashed` via `file://` quando configurado; senão clone HTTPS via token. (Local fica para Fase 2.) |
| 4 | Sandbox de tools | **Leitura + execução sandboxed** — leitura agora; execução (testes/build) em sandbox na Fase 2. |
| 5 | Seleção de modo | **Por escopo + override por comando** — perfil resolvido global→client→project, sobrescrevível por `/review deep\|full\|...`. |
| 6 | Runtime de sandbox | **Interface `Sandbox` + Docker primeiro** — Podman/gVisor plugáveis depois. (Fase 2.) |
| 7 | Provider agêntico | **Todos tool-capable** — loop genérico com adaptadores por transporte (Claude/OpenAI/Gemini/DeepSeek). |
| 8 | Contexto relacionado | **Issues + MRs relacionados + discussão do MR** — bundle único (não flags separadas). |
| 9 | Faseamento | **Spec guarda-chuva + Fase 1** — desenho completo aqui; plano de implementação cobre a Fase 1. |

Confirmações de padrão (aprovadas):

- **(a)** A UI no `web/` está **incluída na Fase 1**, espelhando as telas de prompts/AIConfig.
- **(b)** **Sem perfil resolvido = `diff`-only**, preservando 100% o comportamento legado.

## 3. Arquitetura por camadas

A revisão deixa de ser monolítica e passa a ser composta por **camadas** resolvidas por um
*perfil de revisão* (`ReviewProfile`):

- **base — `diff`** (sempre): título/descrição/diff, como hoje.
- **`related`** (flag): injeta issues vinculadas + MRs relacionados + thread de discussão e
  pede checagem de compatibilidade com as issues.
- **`agentic`** (flag): clona a working tree e roda o loop de tool-use de leitura.
- **`execution`** (flag, **Fase 2**): habilita tools de execução (testes/build) no sandbox.

As camadas são ortogonais: `diff`, `diff+related`, `diff+agentic`, `diff+related+agentic`
são todas combinações válidas.

A Clean Architecture é preservada: `domain` continua sem dependências externas (só novas
entidades e **interfaces**); o git, o filesystem do checkout, os adaptadores de tool-use e o
sandbox vivem na `infrastructure`; o `usecase` orquestra via interfaces injetadas.

## 4. Modelo de dados e resolução de escopo

Espelha o que já existe para `Prompt`/`AIConfig`.

### 4.1 Entidades de domínio (`internal/domain/review_profile.go`)

```go
// ReviewProfile descreve quais camadas de contexto a revisão usa.
type ReviewProfile struct {
    ID             int64     `json:"id"`
    Name           string    `json:"name"`
    IncludeRelated bool      `json:"include_related"` // issues + MRs relacionados + discussão
    Agentic        bool      `json:"agentic"`         // checkout + navegação por tool-use
    Execution      bool      `json:"execution"`       // sandbox run (Fase 2)
    MaxIterations  int       `json:"max_iterations"`  // teto do loop agêntico (0 = default do env)
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}

// ReviewProfileAssignment vincula um ReviewProfile a um escopo.
// Espelha PromptAssignment/AIConfigAssignment.
type ReviewProfileAssignment struct {
    ID          int64  `json:"id"`
    ProfileID   int64  `json:"profile_id"`
    ProfileName string `json:"profile_name,omitempty"`
    Scope       string `json:"scope"` // "global" | "client" | "project"
    ClientName  string `json:"client_name"`
    ProjectRef  string `json:"project_ref"`
}
```

### 4.2 Resolução

`ResolveProfile(clientName, projectRef)` segue a mesma cascata dos prompts:
projeto → client → global. **Quando nada resolve, o perfil efetivo é `diff`-only**
(`IncludeRelated=false, Agentic=false`), garantindo que MRs sem configuração se comportem
exatamente como hoje.

### 4.3 Persistência

Nova migration (`migration007`) cria `review_profiles` e `review_profile_assignments`,
espelhando o schema de prompts. **Sem criptografia** — perfis não guardam segredo.

## 5. Override por comando

O comando `/review` já existe (`WebhookEvent.IsReanalyze`, `CommentAuthor`, validação de
autor/usuário registrado no usecase). Estendemos o parser para aceitar um preset nomeado:

| Comando | Camadas efetivas |
|---------|------------------|
| `/review` ou `/review diff` | base |
| `/review related` | diff + related |
| `/review deep` | diff + agentic |
| `/review full` | diff + related + agentic |

- Novo campo `WebhookEvent.RequestedProfile string` (vazio = sem override).
- O parser de webhook de cada plataforma extrai o token após `/review`.
- O usecase aplica o override **apenas quando o autor é autorizado** (reaproveita a checagem
  já existente: autor do MR ou usuário registrado). Override inválido/desconhecido é
  ignorado e logado, caindo no perfil resolvido por escopo.

## 6. Contratos de domínio novos

Mantendo `domain` puro (apenas stdlib + interfaces):

```go
// CodeWorkspace dá às tools agênticas acesso de leitura à working tree clonada.
// Implementada na infra sobre o diretório do checkout.
type CodeWorkspace interface {
    ReadFile(path string) ([]byte, error)
    ListDir(path string) ([]DirEntry, error)
    Grep(pattern, pathGlob string) ([]GrepHit, error)
    Root() string
}

// RepoLocator descreve como obter a working tree de um projeto.
// Implementada (opcionalmente) pelos platform clients.
type RepoLocator interface {
    RepoSource(projectRef string) (RepoSource, error)
}

type RepoSource struct {
    RemoteURL string // HTTPS com credencial embutida (Fase 1)
    LocalPath string // path do bare repo @hashed (Fase 2; vazio se N/A)
}
```

Entidades de contexto relacionado:

```go
type Issue struct {
    IID         int64
    Title       string
    Description string
    State       string
    WebURL      string
}

type Discussion struct {
    Author string
    Body   string
}

type RelatedContext struct {
    Issues      []Issue
    LinkedMRs   []MergeRequestRef // resumo: iid, título, estado, url
    Discussions []Discussion
}
```

`PlatformClient` cresce (best-effort por plataforma):

```go
GetIssue(projectRef string, issueIID int64) (*Issue, error)
GetLinkedMergeRequests(projectRef string, mrIID int64) ([]MergeRequestRef, error)
GetMergeRequestDiscussions(projectRef string, mrIID int64) ([]Discussion, error)
```

Plataformas que não suportarem um método retornam `ErrRelatedContextUnsupported` (sentinela
novo), tratado como best-effort no usecase — espelha o padrão de `ErrInlineCommentsUnsupported`.

`ReviewInput` ganha campos opcionais:

```go
type ReviewInput struct {
    MergeRequest *MergeRequest
    Diff         string
    ClientName   string
    ProjectRef   string
    Profile      *ReviewProfile  // novo
    Related      *RelatedContext // novo (nil quando related desligado)
    Workspace    CodeWorkspace   // novo (nil quando agentic desligado)
}
```

Novo repositório em `internal/domain/repository.go`:

```go
type ReviewProfileRepository interface {
    ListProfiles() ([]ReviewProfile, error)
    GetProfile(id int64) (*ReviewProfile, error)
    CreateProfile(p ReviewProfile) (*ReviewProfile, error)
    UpdateProfile(id int64, p ReviewProfile) (*ReviewProfile, error)
    DeleteProfile(id int64) error
    SetAssignment(a ReviewProfileAssignment) error
    RemoveAssignment(scope, clientName, projectRef string) error
    ListAssignments() ([]ReviewProfileAssignment, error)
    ResolveProfile(clientName, projectRef string) (*ReviewProfile, error)
}
```

## 7. Fluxo no usecase (`ProcessMergeRequest.Execute`)

```
... resolve client, GetMergeRequest, IsUserAssigned, idempotência (igual hoje) ...

profile := profileRepo.ResolveProfile(client, project)   // nil-resolve => diff-only
profile = applyCommandOverride(profile, event.RequestedProfile, autorizado)

diff := client.GetMergeRequestDiff(...)                   // igual hoje

var related *RelatedContext
if profile.IncludeRelated {
    related = relatedCollector.Collect(client, mr)        // best-effort; loga e segue
}

var ws CodeWorkspace
if profile.Agentic {
    ws, cleanup, err = repoFetcher.Checkout(client, mr)   // best-effort
    if err == nil { defer cleanup() } else { loga e segue sem ws }
}

result := reviewer.ReviewDiff({mr, diff, profile, related, ws})

... CreateMergeRequestNote, postInlineComments, MarkAsReviewed, saveEvent (igual hoje) ...
```

**Princípio de degradação graciosa:** qualquer falha nas camadas extras (clone, fetch de
issue, discussão) é logada e a revisão continua com o que tiver — nunca derruba o review.
Se o clone agêntico falhar, cai para `diff` (+`related` se disponível) em single-shot.

Dependências novas injetadas no `ProcessMergeRequest`:
- `profileRepo domain.ReviewProfileRepository`
- `relatedCollector` (serviço de usecase que usa os novos métodos do `PlatformClient`)
- `repoFetcher` (interface de usecase; impl na infra — ver §9)

## 8. Camada `related`

### 8.1 Detecção de links (agnóstica de plataforma)

`relatedCollector` faz parse de `mr.Description` (que já temos) procurando:
- palavras-chave de fechamento: `Closes #N`, `Fixes #N`, `Resolves #N` (case-insensitive);
- referências simples `#N`;
- referências cross-project `grupo/proj#N`;
- URLs completas de issues.

Complementarmente, no GitLab usa o endpoint nativo `GET /merge_requests/:iid/closes_issues`
para issues que serão fechadas pelo MR.

### 8.2 Fetch e renderização

Para cada issue detectada chama `GetIssue`; coleta MRs relacionados via
`GetLinkedMergeRequests` e a thread via `GetMergeRequestDiscussions`. O resultado vira um
`RelatedContext`. O `prompt.go` ganha um bloco **"Contexto relacionado"** com:
- descrição de cada issue + pergunta explícita de compatibilidade ("o MR cumpre o que a
  issue #N pede? aponte divergências");
- resumo dos MRs relacionados (iid, título, estado, URL);
- a thread de discussão do MR (autor + corpo, truncada por tamanho).

GitLab é a plataforma de referência (implementação completa); GitHub/Gitea entram
best-effort (o que a API suportar; o resto retorna `ErrRelatedContextUnsupported`).

## 9. Acesso ao repositório (working tree)

### 9.1 Fase 1 — clone remoto via token

`GitWorkspaceFetcher` (infra) recebe um `RepoSource` (via `RepoLocator` do client) e o
`mr.DiffHeadSHA`, e produz a working tree:

```
git clone --depth <AGENTIC_CLONE_DEPTH> <RemoteURL> <WORKSPACE_DIR>/<run-id>
git -C <dir> checkout <DiffHeadSHA>
```

Retorna uma `CodeWorkspace` sobre `<dir>` e um `cleanup()` que remove o diretório. O
`RemoteURL` embute a credencial (ex.: `https://oauth2:<token>@host/grp/repo.git` no GitLab),
obtida do próprio client — token nunca é logado.

### 9.2 Fase 2 — clone local `@hashed` (otimização same-machine)

O client GitLab passa `RepoSource.LocalPath` apontando para o bare repo no disco
(`<GITLAB_REPOS_PATH>/@hashed/ab/cd/<hash>.git`). O fetcher prefere o path local
(`git clone file://<LocalPath> ...`) — rápido, sem rede nem token — e cai automaticamente
no `RemoteURL` se o local falhar/não existir. O mapeamento `project → @hashed` usa o
algoritmo de hashed storage do GitLab (SHA-256 do ID do projeto) e exige permissão de
leitura no `git-data`.

## 10. Motor agêntico (loop de tool-use, 4 providers)

### 10.1 Estrutura

Os transportes atuais são single-shot. Adicionamos um **loop agêntico genérico** em `ai/`:

- **Tools (Fase 1, só leitura):** `read_file(path)`, `list_dir(path)`, `grep(pattern, glob)`,
  `git_log(path?)`, `git_blame(path)`. Executadas por um *tool executor* que opera sobre a
  `CodeWorkspace` (git_log/blame via `git -C <root>`).
- **Tools (Fase 2):** `run(cmd)` via `Sandbox` (ver §11), habilitada só com `profile.Execution`.
- **Loop:** monta a mensagem inicial (instruções + diff + `related` + descrição das tools) →
  o modelo pede tool calls → o executor roda e devolve os resultados → repete até o modelo
  emitir o review final em JSON **ou** atingir `MaxIterations` (perfil ou `AGENTIC_MAX_ITERATIONS`).
- **Saída:** reaproveita `parseReviewPayload` (mesmo contrato JSON: `review_markdown` +
  `inline_comments`).

### 10.2 Adaptadores por transporte

Cada API formata tool-use de um jeito; um adaptador por transporte traduz o schema genérico
de tools e os turnos da conversa:
- **Claude** (`anthropic.go`): blocos `tools` nativos + `tool_use`/`tool_result`. Vale para
  modo `api` (`x-api-key`) e `account` (OAuth Claude Code — feito sob medida para agentic).
- **OpenAI/DeepSeek** (`chat_completions.go`): `tools` + `tool_calls` + mensagens `role:tool`.
- **Gemini** (`gemini.go`): `functionDeclarations` + `functionCall`/`functionResponse`.

### 10.3 Degradação

Se não houver `Workspace` (clone falhou) ou o transporte não suportar tool-use, o cliente
cai no caminho single-shot atual (com `related` incluído no prompt quando disponível) e loga.

## 11. Sandbox de execução (Fase 2)

Interface na infra:

```go
type Sandbox interface {
    Run(image, cmd string, limits SandboxLimits) (stdout string, exitCode int, err error)
}
```

Implementação Docker:

```
docker run --rm --network=none --read-only --tmpfs /tmp \
  --cpus <n> --memory <m> -v <workspace>:/src:ro <image> <cmd>
```

Sem rede, sem env/segredos, FS efêmero descartado ao fim, cap de CPU/mem/timeout. Habilita a
tool `run` apenas quando `profile.Execution`. Podman/gVisor podem ser plugados depois pela
mesma interface.

## 12. HTTP e Frontend

### 12.1 API (`internal/transport/http/review_profile_handler.go`)

Espelha `prompt_handler.go`:

| Método | Rota | Descrição |
|---|---|---|
| `GET/POST` | `/api/v1/review-profiles` | Lista / cria perfil |
| `GET/PUT/DELETE` | `/api/v1/review-profiles/{id}` | Lê / atualiza / remove |
| `GET/PUT/DELETE` | `/api/v1/review-profiles/assignments` | Atribuições por escopo |

Todas exigem JWT + `requireRole(super, gestor)`, como prompts e ai-configs.

### 12.2 Frontend (`web/`)

Telas de CRUD de perfil + atribuição por escopo, espelhando as de prompts/AIConfig.

## 13. Configuração / env

**Fase 1:**
- `WORKSPACE_DIR` — raiz dos checkouts efêmeros (default ex.: `./data/workspaces`).
- `AGENTIC_MAX_ITERATIONS` — teto default do loop (sobrescrito por `ReviewProfile.MaxIterations`).
- `AGENTIC_CLONE_DEPTH` — profundidade do clone (default raso).
- `AGENTIC_TIMEOUT` — timeout total da fase agêntica.

**Fase 2:**
- `GITLAB_REPOS_PATH` — raiz do `@hashed` para clone local.
- `SANDBOX_IMAGE`, `SANDBOX_CPUS`, `SANDBOX_MEMORY`, `SANDBOX_TIMEOUT` — parâmetros do sandbox.

## 14. Faseamento

### Fase 1 (plano de implementação a seguir)
Seções 4–8, 10 (modo **leitura**) e 12–13:
- domínio: `ReviewProfile`/assignment, `CodeWorkspace`, `RepoLocator`/`RepoSource`,
  `Issue`/`Discussion`/`RelatedContext`/`MergeRequestRef`, novos métodos do `PlatformClient`,
  `ReviewProfileRepository`, campos novos em `ReviewInput` e `WebhookEvent`;
- storage: `migration007` + repositório de perfis;
- usecase: resolução de perfil + override `/review`, `relatedCollector`, wiring do
  `repoFetcher`, degradação graciosa;
- plataforma: `GitWorkspaceFetcher` (clone remoto HTTPS+token), métodos related no GitLab
  (GitHub/Gitea best-effort);
- IA: loop agêntico genérico **só leitura** + adaptadores nos 4 transportes; render do bloco
  related no `prompt.go`;
- transport: handler REST de perfis + rotas;
- frontend: telas de perfil + atribuição;
- config/env + wiring em `main.go`.

### Fase 2 (especificada; plano depois)
- Sandbox `Run` + impl Docker e a tool `run` (camada `execution`);
- `RepoSource.LocalPath` (`@hashed`) com fallback automático para o remoto.

## 15. Plano de testes (unitário, sem dependências externas)

Seguindo o padrão do projeto (mocks via interfaces, `httptest`, sem rede):
- **Resolução de perfil:** cascata projeto→client→global e default diff-only.
- **Override de comando:** parsing de `/review deep|full|related|diff`, presets e autorização.
- **Detecção de links:** keywords, `#N`, cross-project, URLs.
- **Render do related:** bloco montado a partir de um `RelatedContext` fake.
- **Loop agêntico:** transporte fake que emite tool calls determinísticas + `CodeWorkspace`
  em memória; valida iterações, teto e parsing final.
- **GitWorkspaceFetcher:** clona de um repo git temporário local (`file://`), confere
  checkout no SHA e cleanup.
- **Adaptadores por transporte:** tradução do schema de tools e dos turnos (Claude/OpenAI/
  Gemini/DeepSeek) com `httptest`.
- **Handler REST:** CRUD + atribuições com `httptest` + role.

## 16. Riscos e mitigações

| Risco | Mitigação |
|---|---|
| Custo de tokens variável no agentic | `MaxIterations`/`AGENTIC_TIMEOUT`; perfil agentic é opt-in por escopo/comando. |
| Vazamento de token no clone | Credencial só na URL de clone em memória; nunca logada; cleanup do diretório. |
| Código não-confiável na leitura | Fase 1 é só leitura (sem execução); execução só na Fase 2, em sandbox isolado. |
| Disco dos checkouts | Diretórios efêmeros por `run-id` + cleanup garantido (defer) + `WORKSPACE_DIR` dedicado. |
| Divergência entre 4 formatos de tool-use | Schema genérico de tools + adaptadores isolados e testados por transporte. |
| Plataforma sem suporte a related | `ErrRelatedContextUnsupported` best-effort, espelhando inline comments. |
| Regressão no fluxo legado | Default sem perfil = diff-only; camadas extras são aditivas e best-effort. |

## 17. Arquivos afetados (visão de alto nível, Fase 1)

**Novos:**
- `internal/domain/review_profile.go`
- `internal/domain/related_context.go` (Issue, Discussion, RelatedContext, MergeRequestRef)
- `internal/domain/workspace.go` (CodeWorkspace, RepoLocator, RepoSource, DirEntry, GrepHit)
- `internal/infrastructure/storage/review_profile_repository.go`
- `internal/infrastructure/platform/gitfetch/fetcher.go` (GitWorkspaceFetcher + CodeWorkspace impl)
- `internal/infrastructure/ai/agentic.go` (loop genérico + tool executor)
- `internal/usecase/related_context.go` (relatedCollector + detecção de links)
- `internal/transport/http/review_profile_handler.go`
- `web/` — telas de perfil/atribuição

**Modificados:**
- `internal/domain/repository.go` (ReviewProfileRepository), `platform.go` (novos métodos +
  sentinela), `review.go` (ReviewInput)
- `internal/infrastructure/storage/migrations.go` (migration007)
- `internal/infrastructure/platform/gitlab/{client,webhook}.go` (related + RepoLocator + parse `/review <preset>`)
- `internal/infrastructure/platform/{github,gitea}/*` (related best-effort + parse `/review <preset>`)
- `internal/infrastructure/ai/{client,anthropic,chat_completions,gemini,prompt}.go` (tool-use + bloco related)
- `internal/usecase/process_merge_request.go` (orquestração das camadas)
- `internal/transport/http/router.go` (rotas)
- `cmd/server/main.go` (wiring), `internal/config/config.go` (novas env)
```
