# Modos de Revisão — Fase 2 / Milestone 1: Clone local `@hashed` (same-machine) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Quando o serviço roda na mesma máquina de um GitLab self-hosted, clonar a working tree a partir do bare repo no disco (`@hashed`, via `file://`) em vez de pela rede, com fallback automático para o clone remoto via token.

**Architecture:** O GitLab client passa a preencher `RepoSource.LocalPath` quando `GITLAB_REPOS_PATH` está configurado, calculando o caminho `@hashed` a partir do ID numérico do projeto (`SHA256(id)`). O `GitWorkspaceFetcher` prefere `LocalPath` (`git clone file://...`) e cai no `RemoteURL` se o local falhar. `reposPath` é injetado no client via factory.

**Tech Stack:** Go (stdlib `crypto/sha256`, `encoding/hex`, `path/filepath`, `os/exec`). Testes com `httptest` e repo git temporário (git-guarded).

**Pré-requisitos:** Fase 1 completa (`RepoSource`, `RepoLocator`, `GitWorkspaceFetcher`, `AgenticConfig`).

**Referências (ler antes):**
- `internal/infrastructure/platform/gitlab/client.go` — `RepoSource` atual (Fase 1) + `newClient`/`New`/`NewSelfHosted`.
- `internal/infrastructure/platform/gitfetch/fetcher.go` — `Checkout` atual.
- `internal/infrastructure/platform/factory.go` — `Build`/`BuildAll`/`Factory`/`BuildFromDomain`.
- `internal/config/config.go` — `AgenticConfig`.
- `cmd/server/main.go:104` — `factory := platform.NewFactory()`.

**Contexto do hashed storage do GitLab:** o disk path de um projeto é `<repos>/@hashed/<h[0:2]>/<h[2:4]>/<h>.git`, onde `h = hex(SHA256(<project_id em decimal>))`. Ex.: id `1` → `SHA256("1") = 6b86b273...` → `@hashed/6b/86/6b86b273...b4b.git`.

---

## File Structure

- **Modify:** `internal/config/config.go` — `GitLabReposPath` em `AgenticConfig`.
- **Modify:** `internal/infrastructure/platform/gitlab/client.go` — campo `reposPath`, param no construtor, `hashedRepoPath`, `RepoSource` com `LocalPath`.
- **Modify:** `internal/infrastructure/platform/gitlab/client_test.go` — testes de `hashedRepoPath` e `RepoSource` com reposPath; atualizar call de `New`.
- **Modify:** `internal/infrastructure/platform/gitlab/webhook_test.go` — atualizar `newTestClient` (novo param).
- **Modify:** `internal/infrastructure/platform/factory.go` — thread `reposPath`.
- **Modify:** `internal/infrastructure/platform/gitfetch/fetcher.go` — preferir `LocalPath` + fallback (refactor `cloneFrom`).
- **Modify:** `internal/infrastructure/platform/gitfetch/fetcher_test.go` — testes local + fallback.
- **Modify:** `cmd/server/main.go` — `NewFactory(cfg.Agentic.GitLabReposPath)`.

---

## Task 1: Config — GitLabReposPath

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Adicionar o campo**

Em `AgenticConfig`, adicionar:

```go
	TimeoutSeconds int    // total git operation timeout
	GitLabReposPath string // root of GitLab @hashed storage for same-machine local clones (empty = disabled)
```

No `Load`, no literal `Agentic: AgenticConfig{...}`, adicionar:

```go
		TimeoutSeconds: getEnvInt("AGENTIC_TIMEOUT_SECONDS", 120),
		GitLabReposPath: getEnv("GITLAB_REPOS_PATH", ""),
```

- [ ] **Step 2: Build + commit**

Run: `go build ./internal/config/ && go test ./internal/config/...`
Expected: sucesso.

```bash
git add internal/config/config.go
git commit -m "feat(config): GITLAB_REPOS_PATH para clone local @hashed"
```

---

## Task 2: GitLab — LocalPath via @hashed

**Files:**
- Modify: `internal/infrastructure/platform/gitlab/client.go`
- Modify: `internal/infrastructure/platform/gitlab/client_test.go`
- Modify: `internal/infrastructure/platform/gitlab/webhook_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Em `internal/infrastructure/platform/gitlab/client_test.go`, adicionar (e atualizar imports: `crypto/sha256`, `encoding/hex`, `net/http`, `net/http/httptest` conforme necessário):

```go
func TestHashedRepoPath(t *testing.T) {
	// SHA256("1") = 6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b
	got := hashedRepoPath("/var/opt/gitlab/git-data/repositories", 1)
	want := "/var/opt/gitlab/git-data/repositories/@hashed/6b/86/6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b.git"
	if got != want {
		t.Fatalf("hashedRepoPath = %q, want %q", got, want)
	}
}

func TestRepoSourceWithLocalReposPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":1,"path_with_namespace":"grp/repo"}`))
	}))
	defer srv.Close()

	c := New("test", srv.URL, "tok", "wh", "bot", 30*time.Second, "/repos")
	src, err := c.RepoSource("grp/repo")
	if err != nil {
		t.Fatalf("RepoSource: %v", err)
	}
	if src.RemoteURL == "" || !strings.HasSuffix(src.RemoteURL, "/grp/repo.git") {
		t.Fatalf("RemoteURL inesperado: %q", src.RemoteURL)
	}
	wantLocal := "/repos/@hashed/6b/86/6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b.git"
	if src.LocalPath != wantLocal {
		t.Fatalf("LocalPath = %q, want %q", src.LocalPath, wantLocal)
	}
}

func TestRepoSourceWithoutReposPathHasNoLocal(t *testing.T) {
	c := New("test", "https://gitlab.example.com", "tok", "wh", "bot", 30*time.Second, "")
	src, err := c.RepoSource("grp/repo")
	if err != nil {
		t.Fatalf("RepoSource: %v", err)
	}
	if src.LocalPath != "" {
		t.Fatalf("LocalPath should be empty without reposPath, got %q", src.LocalPath)
	}
}
```

E atualizar o `TestRepoSourcePathRef` existente (se houver) para o novo construtor com `""` no fim:
```go
	c := New("test", "https://gitlab.example.com", "secret-token", "wh", "bot", 30*time.Second, "")
```

Em `internal/infrastructure/platform/gitlab/webhook_test.go`, atualizar `newTestClient`:
```go
func newTestClient() *Client {
	return New("test-gitlab", "https://gitlab.com", "token", "mysecret", "paulorfholanda", 30*time.Second, "")
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/platform/gitlab/ -run 'TestHashedRepoPath|TestRepoSource' -v`
Expected: FAIL de compilação — `New` ainda tem 6 params; `hashedRepoPath` indefinido.

- [ ] **Step 3: Implementar**

Em `internal/infrastructure/platform/gitlab/client.go`:

(a) Adicionar imports: `"crypto/sha256"`, `"encoding/hex"`, `"path/filepath"`, `"strconv"` (confira os já presentes).

(b) Adicionar `reposPath` ao struct `Client` (após `username string`):

```go
	username     string
	reposPath    string
```

(c) Atualizar `New`, `NewSelfHosted` e `newClient` para receber `reposPath string` como ÚLTIMO parâmetro e propagá-lo:

```go
func New(name, baseURL, token, secret, username string, timeout time.Duration, reposPath string) *Client {
	return newClient(domain.PlatformGitLab, name, baseURL, token, secret, username, timeout, reposPath)
}

func NewSelfHosted(name, baseURL, token, secret, username string, timeout time.Duration, reposPath string) *Client {
	return newClient(domain.PlatformGitLabSelfHosted, name, baseURL, token, secret, username, timeout, reposPath)
}

func newClient(pt domain.PlatformType, name, baseURL, token, secret, username string, timeout time.Duration, reposPath string) *Client {
	return &Client{
		platformType: pt,
		name:         name,
		baseURL:      strings.TrimRight(baseURL, "/"),
		token:        token,
		secret:       secret,
		username:     username,
		reposPath:    reposPath,
		http:         &http.Client{Timeout: timeout},
	}
}
```

(d) Adicionar o helper `hashedRepoPath`:

```go
// hashedRepoPath computes the GitLab @hashed storage path for a project id:
// <root>/@hashed/<h[0:2]>/<h[2:4]>/<h>.git where h = hex(sha256(decimal id)).
func hashedRepoPath(root string, projectID int64) string {
	sum := sha256.Sum256([]byte(strconv.FormatInt(projectID, 10)))
	h := hex.EncodeToString(sum[:])
	return filepath.Join(root, "@hashed", h[0:2], h[2:4], h+".git")
}
```

(e) Substituir o método `RepoSource` inteiro por:

```go
// RepoSource builds an authenticated HTTPS clone URL for the project and, when
// GITLAB_REPOS_PATH is configured, the local @hashed bare-repo path for fast
// same-machine clones. Resolves the project id/path via the API as needed.
func (c *Client) RepoSource(projectRef string) (domain.RepoSource, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return domain.RepoSource{}, fmt.Errorf("gitlab RepoSource: parsing base URL: %w", err)
	}

	path := projectRef
	var projectID int64

	// We need the API when the ref is numeric (to get the path) or when a local
	// repos path is configured (to get the id for @hashed).
	if !strings.Contains(projectRef, "/") || c.reposPath != "" {
		var raw struct {
			ID                int64  `json:"id"`
			PathWithNamespace string `json:"path_with_namespace"`
		}
		apiURL := fmt.Sprintf("%s/api/v4/projects/%s", c.baseURL, encodeProjectRef(projectRef))
		if err := c.get(apiURL, &raw); err != nil {
			return domain.RepoSource{}, fmt.Errorf("gitlab RepoSource: resolving project: %w", err)
		}
		if raw.PathWithNamespace != "" {
			path = raw.PathWithNamespace
		}
		projectID = raw.ID
	}

	u.User = url.UserPassword("oauth2", c.token)
	u.Path = "/" + strings.TrimPrefix(path, "/") + ".git"

	src := domain.RepoSource{RemoteURL: u.String()}
	if c.reposPath != "" && projectID != 0 {
		src.LocalPath = hashedRepoPath(c.reposPath, projectID)
	}
	return src, nil
}
```

(O assert `var _ domain.RepoLocator = (*Client)(nil)` já existe da Fase 1 — mantenha.)

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/platform/gitlab/ -v`
Expected: PASS (incluindo os novos testes e os antigos atualizados).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/platform/gitlab/client.go internal/infrastructure/platform/gitlab/client_test.go internal/infrastructure/platform/gitlab/webhook_test.go
git commit -m "feat(gitlab): RepoSource.LocalPath via @hashed quando GITLAB_REPOS_PATH setado"
```

---

## Task 3: Factory — propagar reposPath

**Files:**
- Modify: `internal/infrastructure/platform/factory.go`

- [ ] **Step 1: Threadar o parâmetro**

Em `internal/infrastructure/platform/factory.go`:

(a) `Build` recebe `reposPath string` e o passa só nos cases gitlab:

```go
func Build(cfg config.ClientConfig, timeout time.Duration, reposPath string) (domain.PlatformClient, error) {
	switch cfg.Platform {
	case "gitlab":
		return gitlab.New(cfg.Name, cfg.BaseURL, cfg.Token, cfg.WebhookSecret, cfg.Username, timeout, reposPath), nil
	case "gitlab-selfhosted":
		return gitlab.NewSelfHosted(cfg.Name, cfg.BaseURL, cfg.Token, cfg.WebhookSecret, cfg.Username, timeout, reposPath), nil
	case "github":
		return github.New(cfg.Name, cfg.BaseURL, cfg.Token, cfg.WebhookSecret, cfg.Username, timeout), nil
	case "gitea":
		return gitea.New(cfg.Name, cfg.BaseURL, cfg.Token, cfg.WebhookSecret, cfg.Username, timeout), nil
	case "gitea-selfhosted":
		return gitea.NewSelfHosted(cfg.Name, cfg.BaseURL, cfg.Token, cfg.WebhookSecret, cfg.Username, timeout), nil
	default:
		return nil, fmt.Errorf("unknown platform %q for client %q", cfg.Platform, cfg.Name)
	}
}
```

(b) `BuildAll` recebe e repassa `reposPath`:

```go
func BuildAll(configs []config.ClientConfig, timeout time.Duration, reposPath string) (*Registry, error) {
	reg := NewRegistry()
	for _, cfg := range configs {
		client, err := Build(cfg, timeout, reposPath)
		if err != nil {
			return nil, err
		}
		reg.Register(client)
	}
	return reg, nil
}
```

(c) `Factory` guarda `reposPath`:

```go
type Factory struct {
	reposPath string
}

func NewFactory(reposPath string) *Factory { return &Factory{reposPath: reposPath} }

func (f *Factory) BuildFromDomain(c domain.Client, timeout time.Duration) (domain.PlatformClient, error) {
	cfg := config.ClientConfig{
		Name:          c.Name,
		Platform:      string(c.Platform),
		BaseURL:       c.BaseURL,
		Token:         c.Token,
		WebhookSecret: c.WebhookSecret,
		Username:      c.Username,
	}
	return Build(cfg, timeout, f.reposPath)
}
```

- [ ] **Step 2: Build (vai falhar no main.go até a Task 5)**

Run: `go build ./internal/infrastructure/platform/`
Expected: sucesso neste pacote (o `main.go` será ajustado na Task 5).

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/platform/factory.go
git commit -m "feat(platform): factory propaga reposPath ao client GitLab"
```

---

## Task 4: Fetcher — preferir local com fallback

**Files:**
- Modify: `internal/infrastructure/platform/gitfetch/fetcher.go`
- Modify: `internal/infrastructure/platform/gitfetch/fetcher_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Em `internal/infrastructure/platform/gitfetch/fetcher_test.go`, adicionar (reutiliza `makeOriginRepo` da Fase 1):

```go
func TestFetcherPrefersLocalPath(t *testing.T) {
	origin, sha := makeOriginRepo(t)

	f := NewGitWorkspaceFetcher(t.TempDir(), 0, 60*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	// LocalPath set, RemoteURL empty → must clone from local
	ws, cleanup, err := f.Checkout(domain.RepoSource{LocalPath: origin}, sha)
	if err != nil {
		t.Fatalf("Checkout (local): %v", err)
	}
	defer cleanup()
	if b, err := ws.ReadFile("hello.txt"); err != nil || string(b) != "oi mundo\n" {
		t.Fatalf("local clone content wrong: %q err=%v", b, err)
	}
}

func TestFetcherFallsBackToRemoteWhenLocalFails(t *testing.T) {
	origin, sha := makeOriginRepo(t)

	f := NewGitWorkspaceFetcher(t.TempDir(), 0, 60*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	src := domain.RepoSource{
		LocalPath: "/nonexistent/path/repo.git", // fails → fallback
		RemoteURL: "file://" + origin,
	}
	ws, cleanup, err := f.Checkout(src, sha)
	if err != nil {
		t.Fatalf("Checkout (fallback): %v", err)
	}
	defer cleanup()
	if _, err := ws.ReadFile("hello.txt"); err != nil {
		t.Fatalf("fallback clone failed to read: %v", err)
	}
}
```

NOTA: `makeOriginRepo` faz `t.Skip` se `git` não estiver instalado; estes testes herdam esse skip.

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/platform/gitfetch/ -run 'TestFetcherPrefersLocal|TestFetcherFallsBack' -v`
Expected: FAIL — `Checkout` atual exige `RemoteURL` e ignora `LocalPath`.

- [ ] **Step 3: Implementar**

Substituir o método `Checkout` em `internal/infrastructure/platform/gitfetch/fetcher.go` por (e adicionar `log/slog` ao import se necessário — já está):

```go
// Checkout obtains the MR working tree. It prefers the local bare repo
// (LocalPath, fast, no network) and falls back to the authenticated RemoteURL.
// Returns the workspace and a cleanup that removes the working tree.
func (f *GitWorkspaceFetcher) Checkout(src domain.RepoSource, sha string) (domain.CodeWorkspace, func(), error) {
	if src.LocalPath != "" {
		ws, cleanup, err := f.cloneFrom("file://"+src.LocalPath, 0, sha)
		if err == nil {
			return ws, cleanup, nil
		}
		f.log.Warn("local clone failed, falling back to remote", slog.String("error", err.Error()))
	}
	if src.RemoteURL == "" {
		return nil, nil, fmt.Errorf("gitfetch: no usable repo source (no local path, empty remote URL)")
	}
	return f.cloneFrom(src.RemoteURL, f.cloneDepth, sha)
}

// cloneFrom clones repoURL (depth 0 = full) into an ephemeral dir and checks out sha.
func (f *GitWorkspaceFetcher) cloneFrom(repoURL string, depth int, sha string) (domain.CodeWorkspace, func(), error) {
	if err := os.MkdirAll(f.workspaceDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("gitfetch: creating workspace dir: %w", err)
	}
	dir, err := os.MkdirTemp(f.workspaceDir, "review-*")
	if err != nil {
		return nil, nil, fmt.Errorf("gitfetch: temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	args := []string{"clone", "--quiet"}
	if depth > 0 {
		args = append(args, "--depth", strconv.Itoa(depth))
	}
	args = append(args, repoURL, dir)
	if out, err := f.runGit(ctx, "", args...); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("gitfetch: clone failed: %w (%s)", err, out)
	}

	if sha != "" {
		if _, err := f.runGit(ctx, dir, "checkout", "--quiet", sha); err != nil {
			// A shallow clone may not contain an arbitrary sha; fetch it explicitly.
			if _, ferr := f.runGit(ctx, dir, "fetch", "--quiet", "--depth", "1", "origin", sha); ferr == nil {
				_, err = f.runGit(ctx, dir, "checkout", "--quiet", sha)
			}
			if err != nil {
				cleanup()
				return nil, nil, fmt.Errorf("gitfetch: checkout %s: %w", sha, err)
			}
		}
	}

	return newLocalWorkspace(dir), cleanup, nil
}
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/platform/gitfetch/ -v`
Expected: PASS (testes da Fase 1 + os 2 novos; `TestFetcherEmptyRemoteURL` continua válido pois sem LocalPath e sem RemoteURL retorna erro).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/platform/gitfetch/fetcher.go internal/infrastructure/platform/gitfetch/fetcher_test.go
git commit -m "feat(gitfetch): preferir clone local (file://) com fallback remoto"
```

---

## Task 5: Wiring no main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Passar reposPath ao factory**

Em `cmd/server/main.go`, trocar:

```go
	factory := platform.NewFactory()
```

por:

```go
	factory := platform.NewFactory(cfg.Agentic.GitLabReposPath)
```

(Se houver chamada a `platform.BuildAll(...)` em algum lugar, adicione `cfg.Agentic.GitLabReposPath` como 3º argumento. No `main.go` atual o registro usa `factory.BuildFromDomain`, então provavelmente só o `NewFactory` muda.)

- [ ] **Step 2: Build + testes**

Run: `go build ./... && go test ./...`
Expected: sucesso, tudo verde.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(main): injeta GITLAB_REPOS_PATH no factory"
```

---

## Verificação final do milestone

- [ ] **Build:** `go build ./...` → sucesso.
- [ ] **Testes:** `go test ./...` → tudo verde, sem regressão.
- [ ] **Sanidade:** com `GITLAB_REPOS_PATH` setado e permissão de leitura no `git-data`, perfis agênticos clonam do bare local (`file://.../@hashed/...`) — rápido, sem rede/token; se o local falhar, cai no remoto. Sem a env, comportamento idêntico à Fase 1 (só remoto).

## Notas para o executor

- **`SHA256("1")`** = `6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b` — valor fixo usado nos testes; o hash é do ID em **decimal** como string.
- **Fallback é decisão de design** (não workaround): local falho → remoto, logando o motivo. Não mascarar; apenas degradar.
- **Não logar `RemoteURL`** (token). O `LocalPath` pode ser logado.
- **Permissões:** o clone `file://` exige leitura no diretório do GitLab; em produção o serviço precisa rodar com um usuário no grupo `git` (ou similar). Documentar no M6 da Fase 2 / README.
- **`makeOriginRepo`** (Fase 1) já existe em `fetcher_test.go`; reutilize — não duplique.
