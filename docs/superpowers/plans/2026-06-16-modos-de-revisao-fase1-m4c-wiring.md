# Modos de Revisão — Fase 1 / Milestone 4c: Wiring do agêntico (keystone) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Conectar o agêntico ao pipeline: quando `profile.Agentic`, o usecase resolve a `RepoSource` (via `RepoLocator` do GitLab), faz checkout da working tree no `DiffHeadSHA` (via `GitWorkspaceFetcher`) e plumba em `ReviewInput.Workspace`, com cleanup garantido e degradação graciosa. Inclui a config de workspace e o wiring no `main.go`.

**Architecture:** O `usecase.ProcessMergeRequest` ganha uma dependência `RepoFetcher` (interface local satisfeita por `gitfetch.GitWorkspaceFetcher`). No `Execute`, quando o perfil é agêntico e o client implementa `domain.RepoLocator`, faz checkout best-effort e injeta a `Workspace`. O GitLab implementa `RepoLocator` montando a URL de clone autenticada. A config ganha `AgenticConfig` (workspace dir, profundidade, timeout). `main.go` constrói o fetcher e injeta.

**Tech Stack:** Go (stdlib): `net/url`, `time`. Testes com mocks por interface.

**Pré-requisitos:** M4a (`gitfetch.GitWorkspaceFetcher`, `domain.CodeWorkspace/RepoLocator/RepoSource`), M4b (caminho agêntico no `Client`), M1–M3.

**Decisão de escopo:** `AGENTIC_MAX_ITERATIONS` por env NÃO entra agora (o loop já default 12 e `ReviewProfile.MaxIterations` cobre por perfil). Evita config morta e um parâmetro extra no construtor. Pode ser adicionado depois.

**Referências de padrão (ler antes):**
- `internal/infrastructure/platform/gitlab/client.go` — `c.baseURL`, `c.token`, `encodeProjectRef`, `c.get`.
- `internal/usecase/process_merge_request.go` — `Execute`, `resolveProfile`, coleta de related (M3), construtor.
- `internal/config/config.go` — `Config`, `Load`, helpers de env.
- `cmd/server/main.go:149-153` — chamada de `NewProcessMergeRequest`; `aiTimeout` (time.Duration) já existe.

---

## File Structure

- **Modify:** `internal/config/config.go` — `AgenticConfig` + parse no `Load` + helper `getEnvInt`.
- **Modify:** `internal/infrastructure/platform/gitlab/client.go` — implementa `domain.RepoLocator`.
- **Create:** `internal/infrastructure/platform/gitlab/client_test.go` — teste de `RepoSource` (caso path, sem rede).
- **Modify:** `internal/usecase/process_merge_request.go` — `RepoFetcher` + checkout no `Execute`.
- **Modify:** `internal/usecase/process_merge_request_test.go` — mocks + testes + call sites.
- **Modify:** `cmd/server/main.go` — constrói `gitfetch.GitWorkspaceFetcher` e injeta.

---

## Task 1: Config — AgenticConfig

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Adicionar a struct e o parse**

Em `internal/config/config.go`:

(a) Adicionar o campo em `Config` (após `Dashboard DashboardConfig`):

```go
	Dashboard         DashboardConfig
	Agentic           AgenticConfig
```

(b) Adicionar a struct (após `AIConfig`):

```go
// AgenticConfig tunes the agentic review's repository checkout.
type AgenticConfig struct {
	WorkspaceDir   string // root for ephemeral checkouts
	CloneDepth     int    // shallow clone depth (0 = full clone)
	TimeoutSeconds int    // total git operation timeout
}
```

(c) No `Load`, adicionar ao literal `cfg` (após o bloco `Dashboard: ...`):

```go
		Agentic: AgenticConfig{
			WorkspaceDir:   getEnv("WORKSPACE_DIR", "./data/workspaces"),
			CloneDepth:     getEnvInt("AGENTIC_CLONE_DEPTH", 1),
			TimeoutSeconds: getEnvInt("AGENTIC_TIMEOUT_SECONDS", 120),
		},
```

(d) Adicionar o helper (junto aos outros, no fim do arquivo):

```go
// getEnvInt reads an integer env var, returning fallback when unset or invalid.
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
```

- [ ] **Step 2: Build + testes de config**

Run: `go build ./internal/config/ && go test ./internal/config/...`
Expected: sucesso.

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): AgenticConfig (workspace dir, clone depth, timeout)"
```

---

## Task 2: GitLab — implementar RepoLocator

**Files:**
- Modify: `internal/infrastructure/platform/gitlab/client.go`
- Create: `internal/infrastructure/platform/gitlab/client_test.go`

- [ ] **Step 1: Escrever o teste que falha**

Criar `internal/infrastructure/platform/gitlab/client_test.go`:

```go
package gitlab

import (
	"strings"
	"testing"
	"time"
)

func TestRepoSourcePathRef(t *testing.T) {
	c := New("test", "https://gitlab.example.com", "secret-token", "wh", "bot", 30*time.Second)

	src, err := c.RepoSource("grp/sub/repo")
	if err != nil {
		t.Fatalf("RepoSource: %v", err)
	}
	// must be an authenticated HTTPS URL ending in the repo path + .git
	if !strings.HasPrefix(src.RemoteURL, "https://oauth2:") {
		t.Fatalf("RemoteURL missing oauth2 userinfo: %q", src.RemoteURL)
	}
	if !strings.Contains(src.RemoteURL, "@gitlab.example.com/") {
		t.Fatalf("RemoteURL missing host: %q", src.RemoteURL)
	}
	if !strings.HasSuffix(src.RemoteURL, "/grp/sub/repo.git") {
		t.Fatalf("RemoteURL missing repo path: %q", src.RemoteURL)
	}
	if !strings.Contains(src.RemoteURL, "secret-token") {
		t.Fatalf("RemoteURL missing token: %q", src.RemoteURL)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/platform/gitlab/ -run TestRepoSource -v`
Expected: FAIL — `RepoSource` indefinido.

- [ ] **Step 3: Implementar**

Em `internal/infrastructure/platform/gitlab/client.go`, adicionar o import `"net/url"` (já presente — confirme) e o método:

```go
// ---------- RepoLocator ----------

// RepoSource builds an authenticated HTTPS clone URL for the project. When the
// projectRef is a numeric ID, it resolves the path via the API.
func (c *Client) RepoSource(projectRef string) (domain.RepoSource, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return domain.RepoSource{}, fmt.Errorf("gitlab RepoSource: parsing base URL: %w", err)
	}

	path := projectRef
	if !strings.Contains(projectRef, "/") {
		var raw struct {
			PathWithNamespace string `json:"path_with_namespace"`
		}
		apiURL := fmt.Sprintf("%s/api/v4/projects/%s", c.baseURL, encodeProjectRef(projectRef))
		if err := c.get(apiURL, &raw); err != nil {
			return domain.RepoSource{}, fmt.Errorf("gitlab RepoSource: resolving project path: %w", err)
		}
		path = raw.PathWithNamespace
	}

	u.User = url.UserPassword("oauth2", c.token)
	u.Path = "/" + strings.TrimPrefix(path, "/") + ".git"
	return domain.RepoSource{RemoteURL: u.String()}, nil
}
```

Adicionar o assert de compilação no fim do arquivo:

```go
// Compile-time check: GitLab can locate a repository for agentic checkout.
var _ domain.RepoLocator = (*Client)(nil)
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/platform/gitlab/ -v`
Expected: PASS (o caso path não acessa rede; o caso numérico é testado manualmente, como o resto do client).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/platform/gitlab/client.go internal/infrastructure/platform/gitlab/client_test.go
git commit -m "feat(gitlab): implementa RepoLocator (URL de clone autenticada)"
```

---

## Task 3: Usecase — checkout agêntico no pipeline

**Files:**
- Modify: `internal/usecase/process_merge_request.go`
- Modify: `internal/usecase/process_merge_request_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Em `internal/usecase/process_merge_request_test.go`, adicionar:

```go
// stubWorkspace is a no-op CodeWorkspace for wiring tests.
type stubWorkspace struct{}

func (stubWorkspace) ReadFile(string) ([]byte, error)            { return nil, nil }
func (stubWorkspace) ListDir(string) ([]domain.DirEntry, error)  { return nil, nil }
func (stubWorkspace) Grep(string, string) ([]domain.GrepHit, error) { return nil, nil }
func (stubWorkspace) Root() string                              { return "/stub" }

type mockRepoFetcher struct {
	ws      domain.CodeWorkspace
	err     error
	cleaned bool
}

func (m *mockRepoFetcher) Checkout(_ domain.RepoSource, _ string) (domain.CodeWorkspace, func(), error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.ws, func() { m.cleaned = true }, nil
}

// mockAgenticClient implements PlatformClient + domain.RepoLocator.
type mockAgenticClient struct {
	mockClient
}

func (m *mockAgenticClient) RepoSource(_ string) (domain.RepoSource, error) {
	return domain.RepoSource{RemoteURL: "https://oauth2:t@h/x.git"}, nil
}

func TestExecute_AgenticCheckoutSetsWorkspace(t *testing.T) {
	mr := happyMR("alice")
	client := &mockAgenticClient{mockClient: mockClient{name: "test-client", username: "alice", mr: mr, diff: "--- a/x.go\n+x"}}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	profiles := &mockProfileRepo{resolved: &domain.ReviewProfile{Name: "deep", Agentic: true}}
	fetcher := &mockRepoFetcher{ws: stubWorkspace{}}

	uc := newUsecaseWithProfiles(&client.mockClient, &mockRepository{}, ai, profiles)
	uc.registry = &mockRegistry{client: client}
	uc.repoFetcher = fetcher

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Workspace == nil {
		t.Fatal("expected workspace set for agentic profile")
	}
	if !fetcher.cleaned {
		t.Error("expected cleanup to be called after Execute")
	}
}

func TestExecute_AgenticNoLocatorFallsBack(t *testing.T) {
	// plain mockClient does NOT implement RepoLocator → no workspace, no error
	client := &mockClient{name: "test-client", username: "alice", mr: happyMR("alice"), diff: "--- a/x.go\n+x"}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	profiles := &mockProfileRepo{resolved: &domain.ReviewProfile{Name: "deep", Agentic: true}}

	uc := newUsecaseWithProfiles(client, &mockRepository{}, ai, profiles)
	uc.repoFetcher = &mockRepoFetcher{ws: stubWorkspace{}}

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Workspace != nil {
		t.Fatal("expected no workspace when client lacks RepoLocator")
	}
}
```

Atualizar o helper `newUsecaseWithProfiles` para passar `nil` como `repoFetcher` (novo parâmetro do construtor), e o call site direto em `TestExecute_UnknownClient_ReturnsError` idem. Ver Task Step 3(b).

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/usecase/ -run TestExecute_Agentic -v`
Expected: FAIL de compilação — construtor sem `repoFetcher`, campo/checkout inexistentes.

- [ ] **Step 3: Implementar**

Em `internal/usecase/process_merge_request.go`:

(a) Adicionar a interface e o campo. Após a interface `EventPublisher` (topo do arquivo), adicionar:

```go
// RepoFetcher checks out the MR working tree for agentic review.
// Satisfied by gitfetch.GitWorkspaceFetcher.
type RepoFetcher interface {
	Checkout(src domain.RepoSource, sha string) (domain.CodeWorkspace, func(), error)
}
```

Na struct `ProcessMergeRequest`, após `related *RelatedCollector`:

```go
	related        *RelatedCollector
	repoFetcher    RepoFetcher
```

(b) No construtor `NewProcessMergeRequest`, adicionar o parâmetro `repoFetcher RepoFetcher` logo após `profileRepo domain.ReviewProfileRepository,` e atribuí-lo (`repoFetcher: repoFetcher,`). Atualize a assinatura e o corpo.

Atualize TODOS os call sites:
- `cmd/server/main.go` (Task 4).
- Em `process_merge_request_test.go`, o helper:

```go
func newUsecaseWithProfiles(client *mockClient, repo *mockRepository, ai *mockAI, profiles domain.ReviewProfileRepository) *ProcessMergeRequest {
	reg := &mockRegistry{}
	if client != nil {
		reg.client = client
	}
	return NewProcessMergeRequest(reg, repo, &mockEventRepo{}, &mockUserRepoUC{}, profiles, nil, ai, nil, 3.0, 15.0, discardLogger())
}
```

- E o call site direto em `TestExecute_UnknownClient_ReturnsError`:

```go
	uc := NewProcessMergeRequest(reg, &mockRepository{}, &mockEventRepo{}, &mockUserRepoUC{}, &mockProfileRepo{}, nil, &mockAI{}, nil, 3.0, 15.0, discardLogger())
```

(c) Adicionar o helper de checkout (no fim do arquivo):

```go
// checkoutWorkspace clones the MR working tree for agentic review. Best-effort:
// returns ok=false (and the review proceeds without a workspace) on any problem.
func (uc *ProcessMergeRequest) checkoutWorkspace(log *slog.Logger, client domain.PlatformClient, event *domain.WebhookEvent, mr *domain.MergeRequest) (domain.CodeWorkspace, func(), bool) {
	if uc.repoFetcher == nil {
		return nil, nil, false
	}
	locator, ok := client.(domain.RepoLocator)
	if !ok {
		log.Warn("agentic profile but platform has no RepoLocator; proceeding without workspace",
			slog.String("platform", string(client.Platform())))
		return nil, nil, false
	}
	src, err := locator.RepoSource(event.ProjectRef)
	if err != nil {
		log.Error("resolving repo source for agentic review", slog.String("error", err.Error()))
		return nil, nil, false
	}
	ws, cleanup, err := uc.repoFetcher.Checkout(src, mr.DiffHeadSHA)
	if err != nil {
		log.Error("checking out working tree, proceeding without workspace", slog.String("error", err.Error()))
		return nil, nil, false
	}
	return ws, cleanup, true
}
```

(d) No `Execute`, depois da coleta de related (M3) e antes de montar o `ReviewInput`, adicionar:

```go
	var workspace domain.CodeWorkspace
	if profile.Agentic {
		if ws, cleanup, ok := uc.checkoutWorkspace(log, client, event, mr); ok {
			workspace = ws
			defer cleanup()
			log.Info("agentic workspace checked out", slog.String("root", ws.Root()))
		}
	}
```

E acrescentar `Workspace: workspace,` ao literal `domain.ReviewInput{...}`:

```go
		Profile:      &profile,
		Related:      related,
		Workspace:    workspace,
	})
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/usecase/ -v`
Expected: PASS (todos, incluindo os 2 novos de agentic checkout).

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/process_merge_request.go internal/usecase/process_merge_request_test.go
git commit -m "feat(usecase): checkout agêntico best-effort quando profile.Agentic"
```

---

## Task 4: Wiring no main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Construir o fetcher e injetar**

Em `cmd/server/main.go`:

(a) Adicionar o import do pacote `gitfetch` no bloco de imports:

```go
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/gitfetch"
```

(b) Antes da criação do usecase (`uc := usecase.NewProcessMergeRequest(...)`), construir o fetcher:

```go
	agenticFetcher := gitfetch.NewGitWorkspaceFetcher(
		cfg.Agentic.WorkspaceDir,
		cfg.Agentic.CloneDepth,
		time.Duration(cfg.Agentic.TimeoutSeconds)*time.Second,
		log,
	)
```

(c) Passar `agenticFetcher` como o novo argumento `repoFetcher` (após o `profileRepo`, que é o 5º `repo`):

```go
	uc := usecase.NewProcessMergeRequest(
		registry, repo, repo, repo, repo, agenticFetcher, reviewer, broadcaster,
		cfg.AI.PricePerMInputTokens, cfg.AI.PricePerMOutputTokens,
		log,
	)
```

(Confirme que `time` já está importado em `main.go` — `aiTimeout` o usa; se não, adicione.)

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: sucesso.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(main): injeta GitWorkspaceFetcher no pipeline agêntico"
```

---

## Verificação final do milestone

- [ ] **Build:** `go build ./...` → sucesso.
- [ ] **Testes:** `go test ./...` → tudo verde, sem regressão.
- [ ] **Sanidade end-to-end (mental):** perfil `deep`/`full` + client GitLab → `Execute` resolve `RepoSource`, clona no `DiffHeadSHA`, injeta `Workspace`; o `Client` (Claude) roda o loop de tool-use; cleanup remove o dir ao fim. Falha em qualquer etapa → segue sem workspace (single-shot/related). GitHub/Gitea (sem `RepoLocator`) → sem workspace, sem erro.

## Notas para o executor

- **Best-effort total:** nenhuma falha de checkout pode derrubar a review. `checkoutWorkspace` retorna `ok=false` e o pipeline segue.
- **Cleanup:** o `defer cleanup()` no `Execute` garante remoção do working tree mesmo em erro posterior.
- **Não logar o RemoteURL:** ele contém o token. Logue só `ws.Root()`.
- **Ordem dos parâmetros do construtor:** `repoFetcher` vem logo após `profileRepo`. Atualize main.go + os 2 call sites de teste. O `*SQLiteRepository` (`repo`) é passado 5x (review/event/user/profile) + `agenticFetcher` como `repoFetcher`.
- **`mr.DiffHeadSHA`:** é o SHA para checkout (já populado pelo GitLab/GitHub em `GetMergeRequest`).
