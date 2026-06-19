# Modos de Revisão — Fase 1 / Milestone 4a: Fundação agêntica (workspace + fetcher) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Criar a fundação para a revisão agêntica: as interfaces de domínio (`CodeWorkspace`, `RepoLocator`, `RepoSource`), um workspace de leitura sobre um diretório local (com proteção contra path traversal) e um `GitWorkspaceFetcher` que clona+faz checkout da branch do MR.

**Architecture:** `domain` ganha interfaces puras (sem dependências). Um novo pacote `internal/infrastructure/platform/gitfetch` implementa o `CodeWorkspace` sobre um diretório e clona via `git` (os/exec) para um diretório efêmero, retornando a workspace + `cleanup`. Nada é cabeado no pipeline ainda (isso é o M4c). O loop de tool-use que consome a workspace é o M4b.

**Tech Stack:** Go (stdlib): `os`, `os/exec`, `context`, `path/filepath`, `regexp`, `bufio`. Testes com `testing` + `t.TempDir()`; o teste do clone usa `git` e faz `t.Skip` se o binário não existir.

**Pré-requisitos:** M1–M3 concluídos.

**Referências de padrão (ler antes):**
- `internal/domain/related_context.go` — estilo de entidade + interface opcional (`RelatedContextProvider`); o `RepoLocator` segue o mesmo padrão.
- `internal/infrastructure/platform/gitlab/client.go` — só para conhecer o estilo geral do pacote platform.

---

## File Structure

- **Create:** `internal/domain/workspace.go` — `CodeWorkspace`, `RepoLocator`, `RepoSource`, `DirEntry`, `GrepHit`.
- **Modify:** `internal/domain/review.go` — campo `Workspace CodeWorkspace` em `ReviewInput`.
- **Create:** `internal/infrastructure/platform/gitfetch/workspace.go` — `localWorkspace` (impl de `CodeWorkspace`) + `newLocalWorkspace`.
- **Create:** `internal/infrastructure/platform/gitfetch/workspace_test.go` — testes puros (temp dir, sem git).
- **Create:** `internal/infrastructure/platform/gitfetch/fetcher.go` — `GitWorkspaceFetcher`.
- **Create:** `internal/infrastructure/platform/gitfetch/fetcher_test.go` — teste de clone/checkout (git-guarded).

---

## Task 1: Domínio — interfaces de workspace

**Files:**
- Create: `internal/domain/workspace.go`
- Modify: `internal/domain/review.go`

- [ ] **Step 1: Criar as interfaces**

Criar `internal/domain/workspace.go`:

```go
package domain

// DirEntry is a single entry returned by CodeWorkspace.ListDir.
type DirEntry struct {
	Name  string
	IsDir bool
}

// GrepHit is a single line match returned by CodeWorkspace.Grep.
type GrepHit struct {
	Path string
	Line int
	Text string
}

// CodeWorkspace gives the agentic review read-only access to a checked-out
// working tree. All paths are relative to the workspace root; implementations
// MUST reject paths that escape the root (path traversal).
type CodeWorkspace interface {
	ReadFile(path string) ([]byte, error)
	ListDir(path string) ([]DirEntry, error)
	Grep(pattern, pathGlob string) ([]GrepHit, error)
	Root() string
}

// RepoSource describes where a project's repository can be cloned from.
type RepoSource struct {
	RemoteURL string // authenticated HTTPS clone URL (Fase 1)
	LocalPath string // local bare-repo path (Fase 2; empty if N/A)
}

// RepoLocator is an OPTIONAL capability a PlatformClient may implement to
// describe where a project's repository can be obtained. Mirrors the
// RelatedContextProvider optional-interface pattern.
type RepoLocator interface {
	RepoSource(projectRef string) (RepoSource, error)
}
```

Em `internal/domain/review.go`, adicionar a `ReviewInput` (após `Related *RelatedContext`):

```go
	Related *RelatedContext
	// Workspace is the checked-out working tree for agentic review.
	// Nil when the agentic layer is disabled.
	Workspace CodeWorkspace
```

- [ ] **Step 2: Build**

Run: `go build ./internal/domain/`
Expected: sucesso.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/workspace.go internal/domain/review.go
git commit -m "feat(domain): interfaces CodeWorkspace/RepoLocator + ReviewInput.Workspace"
```

---

## Task 2: localWorkspace — leitura segura sobre um diretório

**Files:**
- Create: `internal/infrastructure/platform/gitfetch/workspace.go`
- Create: `internal/infrastructure/platform/gitfetch/workspace_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Criar `internal/infrastructure/platform/gitfetch/workspace_test.go`:

```go
package gitfetch

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\nfunc Run() {}\n")
	mustWrite(t, filepath.Join(root, "internal", "svc.go"), "package internal\n// TODO: fix\nfunc Do() {}\n")
	return root
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceReadFile(t *testing.T) {
	ws := newLocalWorkspace(writeTree(t))
	b, err := ws.ReadFile("main.go")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(b) == "" || !contains(string(b), "func Run()") {
		t.Fatalf("unexpected content: %q", b)
	}
}

func TestWorkspaceListDir(t *testing.T) {
	ws := newLocalWorkspace(writeTree(t))
	entries, err := ws.ListDir(".")
	if err != nil {
		t.Fatalf("ListDir: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	if !hasAll(names, "main.go", "internal") {
		t.Fatalf("ListDir = %v, want main.go and internal", names)
	}
}

func TestWorkspaceGrep(t *testing.T) {
	ws := newLocalWorkspace(writeTree(t))
	hits, err := ws.Grep("TODO", "")
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(hits) != 1 || hits[0].Path != filepath.FromSlash("internal/svc.go") || hits[0].Line != 2 {
		t.Fatalf("Grep hits = %+v, want internal/svc.go:2", hits)
	}
}

func TestWorkspaceGrepPathGlob(t *testing.T) {
	ws := newLocalWorkspace(writeTree(t))
	hits, err := ws.Grep("func", "*.go") // only top-level *.go
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	for _, h := range hits {
		if h.Path != "main.go" {
			t.Fatalf("glob *.go matched %q", h.Path)
		}
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one hit in main.go")
	}
}

func TestWorkspaceRejectsTraversal(t *testing.T) {
	ws := newLocalWorkspace(writeTree(t))
	if _, err := ws.ReadFile("../../etc/passwd"); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}

// tiny helpers (avoid extra imports in the test)
func contains(s, sub string) bool { return len(s) >= len(sub) && indexOf(s, sub) >= 0 }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
func hasAll(set []string, want ...string) bool {
	for _, w := range want {
		found := false
		for _, s := range set {
			if s == w {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/platform/gitfetch/ -run TestWorkspace -v`
Expected: FAIL — `newLocalWorkspace` indefinido.

- [ ] **Step 3: Implementar**

Criar `internal/infrastructure/platform/gitfetch/workspace.go`:

```go
package gitfetch

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const (
	maxFileBytes = 512 * 1024 // skip/limit files larger than this for reads/grep
	maxGrepHits  = 200
)

// localWorkspace is a read-only CodeWorkspace backed by a directory on disk.
type localWorkspace struct {
	root string
}

func newLocalWorkspace(root string) *localWorkspace {
	return &localWorkspace{root: root}
}

func (w *localWorkspace) Root() string { return w.root }

// resolve maps a workspace-relative path to an absolute path inside the root,
// rejecting anything that escapes the root.
func (w *localWorkspace) resolve(p string) (string, error) {
	clean := filepath.Clean("/" + filepath.FromSlash(p)) // anchor at root, kill ".."
	full := filepath.Join(w.root, clean)
	rel, err := filepath.Rel(w.root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q escapes workspace", p)
	}
	return full, nil
}

func (w *localWorkspace) ReadFile(path string) ([]byte, error) {
	full, err := w.resolve(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory", path)
	}
	if info.Size() > maxFileBytes {
		return nil, fmt.Errorf("%q too large (%d bytes)", path, info.Size())
	}
	return os.ReadFile(full)
}

func (w *localWorkspace) ListDir(path string) ([]domain.DirEntry, error) {
	full, err := w.resolve(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	out := make([]domain.DirEntry, 0, len(entries))
	for _, e := range entries {
		if e.Name() == ".git" {
			continue
		}
		out = append(out, domain.DirEntry{Name: e.Name(), IsDir: e.IsDir()})
	}
	return out, nil
}

func (w *localWorkspace) Grep(pattern, pathGlob string) ([]domain.GrepHit, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid grep pattern: %w", err)
	}

	var hits []domain.GrepHit
	walkErr := filepath.WalkDir(w.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(w.root, path)
		if pathGlob != "" {
			ok, _ := filepath.Match(pathGlob, rel)
			if !ok {
				return nil
			}
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxFileBytes {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		line := 0
		for sc.Scan() {
			line++
			text := sc.Text()
			if re.MatchString(text) {
				hits = append(hits, domain.GrepHit{Path: rel, Line: line, Text: strings.TrimSpace(text)})
				if len(hits) >= maxGrepHits {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return hits, nil
}

// compile-time check
var _ domain.CodeWorkspace = (*localWorkspace)(nil)
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/platform/gitfetch/ -run TestWorkspace -v`
Expected: PASS (5 testes).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/platform/gitfetch/workspace.go internal/infrastructure/platform/gitfetch/workspace_test.go
git commit -m "feat(gitfetch): localWorkspace de leitura com proteção contra traversal"
```

---

## Task 3: GitWorkspaceFetcher — clone + checkout

**Files:**
- Create: `internal/infrastructure/platform/gitfetch/fetcher.go`
- Create: `internal/infrastructure/platform/gitfetch/fetcher_test.go`

- [ ] **Step 1: Escrever o teste que falha (git-guarded)**

Criar `internal/infrastructure/platform/gitfetch/fetcher_test.go`:

```go
package gitfetch

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// makeOriginRepo creates a real git repo with one commit and returns its path
// and the HEAD sha. Skips the test if git is unavailable.
func makeOriginRepo(t *testing.T) (path, sha string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
		return string(out)
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("oi mundo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "first")
	sha = run("rev-parse", "HEAD")
	return dir, trimNL(sha)
}

func trimNL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func TestFetcherCheckout(t *testing.T) {
	origin, sha := makeOriginRepo(t)

	f := NewGitWorkspaceFetcher(t.TempDir(), 0, 60*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ws, cleanup, err := f.Checkout(domain.RepoSource{RemoteURL: "file://" + origin}, sha)
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	defer cleanup()

	b, err := ws.ReadFile("hello.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(b) != "oi mundo\n" {
		t.Fatalf("content = %q", b)
	}

	// cleanup removes the working tree
	root := ws.Root()
	cleanup()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected workspace removed after cleanup, stat err = %v", err)
	}
}

func TestFetcherEmptyRemoteURL(t *testing.T) {
	f := NewGitWorkspaceFetcher(t.TempDir(), 0, 10*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, _, err := f.Checkout(domain.RepoSource{}, ""); err == nil {
		t.Fatal("expected error for empty remote URL")
	}
}

var _ = context.Background // keep context import if unused after edits
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/platform/gitfetch/ -run TestFetcher -v`
Expected: FAIL — `NewGitWorkspaceFetcher` indefinido.

- [ ] **Step 3: Implementar**

Criar `internal/infrastructure/platform/gitfetch/fetcher.go`:

```go
package gitfetch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// GitWorkspaceFetcher clones the MR branch into an ephemeral directory and
// exposes it as a read-only CodeWorkspace.
type GitWorkspaceFetcher struct {
	workspaceDir string
	cloneDepth   int
	timeout      time.Duration
	log          *slog.Logger
}

func NewGitWorkspaceFetcher(workspaceDir string, cloneDepth int, timeout time.Duration, log *slog.Logger) *GitWorkspaceFetcher {
	return &GitWorkspaceFetcher{
		workspaceDir: workspaceDir,
		cloneDepth:   cloneDepth,
		timeout:      timeout,
		log:          log,
	}
}

// Checkout clones src and checks out sha, returning the workspace and a cleanup
// function that removes the working tree. The caller MUST call cleanup.
func (f *GitWorkspaceFetcher) Checkout(src domain.RepoSource, sha string) (domain.CodeWorkspace, func(), error) {
	if src.RemoteURL == "" {
		return nil, nil, fmt.Errorf("gitfetch: empty remote URL")
	}
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
	if f.cloneDepth > 0 {
		args = append(args, "--depth", strconv.Itoa(f.cloneDepth))
	}
	args = append(args, src.RemoteURL, dir)
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

func (f *GitWorkspaceFetcher) runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	// Never prompt for credentials — fail fast instead of hanging.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	return string(out), err
}
```

NOTA: o teste tem `var _ = context.Background` só para garantir o import enquanto o arquivo de teste evolui; pode remover se o linter reclamar de redundância após a implementação (o import `context` no teste não é usado de outra forma — se causar erro de "imported and not used", remova a linha do import e a referência).

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/platform/gitfetch/ -v`
Expected: PASS (`TestFetcherCheckout`, `TestFetcherEmptyRemoteURL` + os de workspace). Se `git` não existir no ambiente, `TestFetcherCheckout` aparece como SKIP — aceitável.

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/platform/gitfetch/fetcher.go internal/infrastructure/platform/gitfetch/fetcher_test.go
git commit -m "feat(gitfetch): GitWorkspaceFetcher (clone + checkout efêmero)"
```

---

## Verificação final do milestone

- [ ] **Build:** `go build ./...` → sucesso.
- [ ] **Testes:** `go test ./...` → tudo verde (o teste de clone faz SKIP sem git; sem falhas).

## Notas para o executor

- **Sem wiring ainda:** este milestone NÃO toca o pipeline, o `main.go`, nem a camada de IA. Só cria as interfaces e o pacote `gitfetch`. O consumo da workspace pelo loop é o M4b; o wiring é o M4c.
- **Segurança:** `resolve` deve rejeitar traversal; `GIT_TERMINAL_PROMPT=0` evita travar pedindo credencial; `cleanup` sempre remove o diretório.
- **Token na URL:** o `RemoteURL` virá com credencial embutida (montada no M4c pelo `RepoLocator` do GitLab). NUNCA logue o `RemoteURL` completo.
- **Teste sem dependência de rede:** o teste de clone usa um repositório git local via `file://` — não acessa rede. É a única parte que depende do binário `git`, e faz `t.Skip` se ele faltar.
