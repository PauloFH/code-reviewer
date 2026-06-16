# Modos de Revisão — Fase 1 / Milestone 3: Camada `related` (contexto relacionado) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Quando o perfil tem `IncludeRelated`, coletar issues vinculadas (incl. as que o MR fecha), MRs relacionados e a thread de discussão, e injetar esse contexto no prompt da IA, pedindo checagem de compatibilidade com as issues.

**Architecture:** O contexto relacionado é capacidade **opcional** da plataforma: uma interface `domain.RelatedContextProvider` que só o GitLab implementa nesta fase. O `usecase.RelatedCollector` detecta referências na descrição do MR (lógica pura, testável), usa o provider via type-assertion (best-effort) e monta um `domain.RelatedContext`. O `usecase.ProcessMergeRequest` coleta quando `profile.IncludeRelated` e plumba em `ReviewInput.Related`. O `ai/prompt.go` renderiza um bloco "Contexto relacionado" no prompt.

**Tech Stack:** Go (stdlib), `regexp` para detecção de refs, testes com `testing` e mocks por interface.

**Pré-requisitos:** M1 (perfis) e M2 (`ReviewInput.Profile`, resolução no pipeline) concluídos.

**Decisão de design (desvio consciente da spec §4/§6):** a spec falava em crescer a interface `PlatformClient` e usar um sentinela `ErrRelatedContextUnsupported`. Em vez disso usamos uma **interface opcional separada** (`RelatedContextProvider`) + type-assertion — idêntico ao padrão `RepoLocator` que a própria spec já adota, sem churn em github/gitea/mocks e sem necessidade do sentinela. Mais idiomático e menor superfície.

**Referências de padrão (ler antes):**
- `internal/infrastructure/platform/gitlab/client.go` — helper `c.get`, `encodeProjectRef`, estilo dos métodos HTTP.
- `internal/infrastructure/ai/prompt.go` — `buildPrompt(input, outputTemplate)` (recebe o `input` inteiro) e o `fmt.Sprintf` do prompt.
- `internal/usecase/process_merge_request.go` — `Execute`, `resolveProfile`, plumbing de `ReviewInput`.

---

## File Structure

- **Create:** `internal/domain/related_context.go` — entidades `Issue`, `Discussion`, `MergeRequestRef`, `RelatedContext` e a interface opcional `RelatedContextProvider`.
- **Modify:** `internal/domain/review.go` — campo `Related *RelatedContext` em `ReviewInput`.
- **Create:** `internal/usecase/related_context.go` — `RelatedCollector` + detecção pura de refs.
- **Create:** `internal/usecase/related_context_test.go` — testes de detecção e montagem.
- **Modify:** `internal/usecase/process_merge_request.go` — coleta quando `profile.IncludeRelated`.
- **Modify:** `internal/usecase/process_merge_request_test.go` — injeção do collector + teste de plumbing.
- **Modify:** `internal/infrastructure/ai/prompt.go` — render do bloco related em `buildPrompt`.
- **Modify:** `internal/infrastructure/ai/prompt_test.go` — teste do render.
- **Modify:** `internal/infrastructure/platform/gitlab/client.go` — implementa `RelatedContextProvider`.

---

## Task 1: Domínio — entidades + interface opcional

**Files:**
- Create: `internal/domain/related_context.go`
- Modify: `internal/domain/review.go`

- [ ] **Step 1: Criar as entidades e a interface**

Criar `internal/domain/related_context.go`:

```go
package domain

// Issue is a minimal view of a tracker issue linked to a merge request.
type Issue struct {
	IID         int64
	Title       string
	Description string
	State       string
	WebURL      string
}

// Discussion is a single comment from a merge request's discussion thread.
type Discussion struct {
	Author string
	Body   string
}

// MergeRequestRef is a lightweight reference to a related merge request.
type MergeRequestRef struct {
	IID    int64
	Title  string
	State  string
	WebURL string
}

// RelatedContext bundles everything the "related" layer gathers around an MR.
type RelatedContext struct {
	Issues      []Issue
	LinkedMRs   []MergeRequestRef
	Discussions []Discussion
}

// IsEmpty reports whether no related context was gathered.
func (rc *RelatedContext) IsEmpty() bool {
	return rc == nil || (len(rc.Issues) == 0 && len(rc.LinkedMRs) == 0 && len(rc.Discussions) == 0)
}

// RelatedContextProvider is an OPTIONAL capability a PlatformClient may implement
// to supply linked issues and discussion threads. Clients that don't implement
// it simply yield no related context (best-effort). Mirrors the RepoLocator
// optional-interface pattern.
type RelatedContextProvider interface {
	GetIssue(projectRef string, issueIID int64) (*Issue, error)
	GetClosesIssues(projectRef string, mrIID int64) ([]Issue, error)
	GetMergeRequestDiscussions(projectRef string, mrIID int64) ([]Discussion, error)
}
```

Em `internal/domain/review.go`, adicionar o campo a `ReviewInput` (após `Profile *ReviewProfile`):

```go
	Profile *ReviewProfile
	// Related is the gathered related context (issues, linked MRs, discussion).
	// Nil when the related layer is disabled or yielded nothing.
	Related *RelatedContext
```

- [ ] **Step 2: Verificar compilação**

Run: `go build ./internal/domain/`
Expected: sucesso.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/related_context.go internal/domain/review.go
git commit -m "feat(domain): entidades de contexto relacionado + RelatedContextProvider"
```

---

## Task 2: Usecase — detecção de refs + collector

**Files:**
- Create: `internal/usecase/related_context.go`
- Create: `internal/usecase/related_context_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Criar `internal/usecase/related_context_test.go`:

```go
package usecase

import (
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestDetectIssueIIDs(t *testing.T) {
	desc := "Implementa X. Closes #12, also relates to #7 and #12 again. Not a ref: foo#9bar"
	got := detectIssueIIDs(desc)
	// unique, in first-seen order: 12, 7
	if len(got) != 2 || got[0] != 12 || got[1] != 7 {
		t.Fatalf("detectIssueIIDs = %v, want [12 7]", got)
	}
}

func TestDetectMRIIDs(t *testing.T) {
	desc := "Depends on !34 and supersedes !2."
	got := detectMRIIDs(desc)
	if len(got) != 2 || got[0] != 34 || got[1] != 2 {
		t.Fatalf("detectMRIIDs = %v, want [34 2]", got)
	}
}

// mock implementing PlatformClient + RelatedContextProvider for collector tests.
type mockRelatedClient struct {
	mockClient
	issuesByIID map[int64]*domain.Issue
	closes      []domain.Issue
	discussions []domain.Discussion
	mrsByIID    map[int64]*domain.MergeRequest
}

func (m *mockRelatedClient) GetIssue(_ string, iid int64) (*domain.Issue, error) {
	return m.issuesByIID[iid], nil
}
func (m *mockRelatedClient) GetClosesIssues(_ string, _ int64) ([]domain.Issue, error) {
	return m.closes, nil
}
func (m *mockRelatedClient) GetMergeRequestDiscussions(_ string, _ int64) ([]domain.Discussion, error) {
	return m.discussions, nil
}
func (m *mockRelatedClient) GetMergeRequest(_ string, iid int64) (*domain.MergeRequest, error) {
	return m.mrsByIID[iid], nil
}

func TestCollectAssemblesAndDedups(t *testing.T) {
	mr := &domain.MergeRequest{
		IID:         5,
		Description: "Closes #7. Depends on !34. Self ref !5 ignored.",
	}
	client := &mockRelatedClient{
		closes:      []domain.Issue{{IID: 7, Title: "Bug do login"}},
		issuesByIID: map[int64]*domain.Issue{7: {IID: 7, Title: "Bug do login"}},
		discussions: []domain.Discussion{{Author: "ana", Body: "revisem o caso X"}},
		mrsByIID:    map[int64]*domain.MergeRequest{34: {IID: 34, Title: "Refactor", State: "merged", WebURL: "u"}},
	}

	collector := NewRelatedCollector(discardLogger())
	rc := collector.Collect(client, mr)

	if rc == nil {
		t.Fatal("expected non-nil related context")
	}
	if len(rc.Issues) != 1 || rc.Issues[0].IID != 7 { // #7 from closes and desc → deduped to 1
		t.Fatalf("issues = %+v, want single #7", rc.Issues)
	}
	if len(rc.LinkedMRs) != 1 || rc.LinkedMRs[0].IID != 34 { // !5 (self) excluded
		t.Fatalf("linkedMRs = %+v, want single !34", rc.LinkedMRs)
	}
	if len(rc.Discussions) != 1 {
		t.Fatalf("discussions = %+v, want 1", rc.Discussions)
	}
}

func TestCollectUnsupportedClientReturnsNil(t *testing.T) {
	// plain mockClient does NOT implement RelatedContextProvider
	collector := NewRelatedCollector(discardLogger())
	rc := collector.Collect(&mockClient{}, &domain.MergeRequest{IID: 1, Description: "Closes #1"})
	if rc != nil {
		t.Fatalf("expected nil for client without RelatedContextProvider, got %+v", rc)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/usecase/ -run 'TestDetect|TestCollect' -v`
Expected: FAIL — `detectIssueIIDs`/`detectMRIIDs`/`NewRelatedCollector` indefinidos.

- [ ] **Step 3: Implementar**

Criar `internal/usecase/related_context.go`:

```go
package usecase

import (
	"log/slog"
	"regexp"
	"strconv"

	"github.com/paulofh/code-reviewer/internal/domain"
)

var (
	issueRefPattern = regexp.MustCompile(`(?:^|[^\w/])#(\d+)`)
	mrRefPattern    = regexp.MustCompile(`(?:^|[^\w/])!(\d+)`)
)

// detectIssueIIDs extracts GitLab-style issue references (#N) from text,
// returning unique IIDs in first-seen order.
func detectIssueIIDs(text string) []int64 {
	return uniqueIIDs(issueRefPattern, text)
}

// detectMRIIDs extracts GitLab-style merge request references (!N) from text.
func detectMRIIDs(text string) []int64 {
	return uniqueIIDs(mrRefPattern, text)
}

func uniqueIIDs(pattern *regexp.Regexp, text string) []int64 {
	var out []int64
	seen := map[int64]bool{}
	for _, m := range pattern.FindAllStringSubmatch(text, -1) {
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// RelatedCollector gathers issues, linked MRs and the discussion thread for an MR.
type RelatedCollector struct {
	log *slog.Logger
}

func NewRelatedCollector(log *slog.Logger) *RelatedCollector {
	return &RelatedCollector{log: log}
}

// Collect best-effort assembles the related context. Returns nil when the
// platform client does not support it or nothing was gathered.
func (rc *RelatedCollector) Collect(client domain.PlatformClient, mr *domain.MergeRequest) *domain.RelatedContext {
	provider, ok := client.(domain.RelatedContextProvider)
	if !ok {
		return nil
	}

	out := &domain.RelatedContext{}
	seenIssue := map[int64]bool{}

	// Issues the MR closes (most reliable source).
	if closes, err := provider.GetClosesIssues(mr.ProjectRef, mr.IID); err != nil {
		rc.log.Warn("fetching closes_issues", slog.String("error", err.Error()))
	} else {
		for _, is := range closes {
			if !seenIssue[is.IID] {
				seenIssue[is.IID] = true
				out.Issues = append(out.Issues, is)
			}
		}
	}

	// Issues referenced in the description (deduped against closes).
	for _, iid := range detectIssueIIDs(mr.Description) {
		if seenIssue[iid] {
			continue
		}
		is, err := provider.GetIssue(mr.ProjectRef, iid)
		if err != nil || is == nil {
			if err != nil {
				rc.log.Warn("fetching issue", slog.Int64("iid", iid), slog.String("error", err.Error()))
			}
			continue
		}
		seenIssue[iid] = true
		out.Issues = append(out.Issues, *is)
	}

	// Linked MRs referenced in the description (excluding the MR itself).
	for _, iid := range detectMRIIDs(mr.Description) {
		if iid == mr.IID {
			continue
		}
		linked, err := client.GetMergeRequest(mr.ProjectRef, iid)
		if err != nil || linked == nil {
			if err != nil {
				rc.log.Warn("fetching linked MR", slog.Int64("iid", iid), slog.String("error", err.Error()))
			}
			continue
		}
		out.LinkedMRs = append(out.LinkedMRs, domain.MergeRequestRef{
			IID: linked.IID, Title: linked.Title, State: linked.State, WebURL: linked.WebURL,
		})
	}

	// Discussion thread.
	if disc, err := provider.GetMergeRequestDiscussions(mr.ProjectRef, mr.IID); err != nil {
		rc.log.Warn("fetching discussions", slog.String("error", err.Error()))
	} else {
		out.Discussions = disc
	}

	if out.IsEmpty() {
		return nil
	}
	return out
}
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/usecase/ -run 'TestDetect|TestCollect' -v`
Expected: PASS (4 testes).

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/related_context.go internal/usecase/related_context_test.go
git commit -m "feat(usecase): RelatedCollector com detecção de refs e montagem best-effort"
```

---

## Task 3: Usecase — coletar quando `profile.IncludeRelated`

**Files:**
- Modify: `internal/usecase/process_merge_request.go`
- Modify: `internal/usecase/process_merge_request_test.go`

- [ ] **Step 1: Escrever o teste que falha**

Em `internal/usecase/process_merge_request_test.go`, adicionar:

```go
func TestExecute_RelatedCollectedWhenProfileIncludesRelated(t *testing.T) {
	mr := happyMR("alice")
	mr.Description = "Closes #7"
	client := &mockRelatedClient{
		mockClient:  mockClient{name: "test-client", username: "alice", mr: mr, diff: "--- a/x.go\n+x"},
		closes:      []domain.Issue{{IID: 7, Title: "Bug"}},
		issuesByIID: map[int64]*domain.Issue{7: {IID: 7, Title: "Bug"}},
	}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	profiles := &mockProfileRepo{resolved: &domain.ReviewProfile{Name: "related", IncludeRelated: true}}
	uc := newUsecaseWithProfiles(&client.mockClient, &mockRepository{}, ai, profiles)
	// the registry must return the related-capable client:
	uc.registry = &mockRegistry{client: client}

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Related == nil || len(ai.lastInput.Related.Issues) != 1 {
		t.Fatalf("expected related context with 1 issue, got %+v", ai.lastInput.Related)
	}
}

func TestExecute_NoRelatedWhenProfileDiffOnly(t *testing.T) {
	mr := happyMR("alice")
	mr.Description = "Closes #7"
	client := &mockRelatedClient{
		mockClient: mockClient{name: "test-client", username: "alice", mr: mr, diff: "--- a/x.go\n+x"},
		closes:     []domain.Issue{{IID: 7, Title: "Bug"}},
	}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	uc := newUsecaseWithProfiles(&client.mockClient, &mockRepository{}, ai, &mockProfileRepo{resolved: nil})
	uc.registry = &mockRegistry{client: client}

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Related != nil {
		t.Fatalf("expected no related context for diff-only profile, got %+v", ai.lastInput.Related)
	}
}
```

NOTA: o teste precisa que `uc.registry` seja sobrescrevível pelo pacote de teste. O campo `registry` já é acessível (mesmo pacote `usecase`). Se preferir, adicione um setter; mas atribuição direta ao campo no teste é suficiente (mesmo package).

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/usecase/ -run TestExecute_RelatedCollected -v`
Expected: FAIL — `ReviewInput.Related` nunca é preenchido (collector não existe no pipeline).

- [ ] **Step 3: Implementar**

Em `internal/usecase/process_merge_request.go`:

(a) Adicionar o campo na struct (após `profileRepo domain.ReviewProfileRepository`):

```go
	profileRepo    domain.ReviewProfileRepository
	related        *RelatedCollector
```

(b) Inicializar no construtor. No corpo do `NewProcessMergeRequest`, na criação do struct, adicionar:

```go
		profileRepo:    profileRepo,
		related:        NewRelatedCollector(log),
```

(c) No `Execute`, depois de `profile := uc.resolveProfile(log, event)` e antes de montar o `ReviewInput`, coletar quando habilitado:

```go
	profile := uc.resolveProfile(log, event)

	var related *domain.RelatedContext
	if profile.IncludeRelated {
		related = uc.related.Collect(client, mr)
		if related != nil {
			log.Info("related context collected",
				slog.Int("issues", len(related.Issues)),
				slog.Int("linked_mrs", len(related.LinkedMRs)),
				slog.Int("discussions", len(related.Discussions)),
			)
		}
	}

	log.Info("sending diff to AI reviewer", slog.String("profile", profile.Name))

	result, err := uc.reviewer.ReviewDiff(domain.ReviewInput{
		MergeRequest: mr,
		Diff:         diff,
		ClientName:   event.ClientName,
		ProjectRef:   event.ProjectRef,
		Profile:      &profile,
		Related:      related,
	})
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/usecase/ -v`
Expected: PASS (todos, incluindo os 2 novos).

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/process_merge_request.go internal/usecase/process_merge_request_test.go
git commit -m "feat(usecase): coleta contexto relacionado quando profile.IncludeRelated"
```

---

## Task 4: Prompt — renderizar o bloco "Contexto relacionado"

**Files:**
- Modify: `internal/infrastructure/ai/prompt.go`
- Modify: `internal/infrastructure/ai/prompt_test.go`

- [ ] **Step 1: Escrever o teste que falha**

Em `internal/infrastructure/ai/prompt_test.go`, adicionar:

```go
func TestBuildPromptIncludesRelatedContext(t *testing.T) {
	input := domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "feat", ProjectRef: "42", Description: "d"},
		Diff:         "--- a/x.go\n+++ b/x.go\n@@ -1 +1 @@\n+x",
		Related: &domain.RelatedContext{
			Issues:      []domain.Issue{{IID: 7, Title: "Bug do login", Description: "usuário não loga"}},
			LinkedMRs:   []domain.MergeRequestRef{{IID: 34, Title: "Refactor", State: "merged"}},
			Discussions: []domain.Discussion{{Author: "ana", Body: "revisem o caso X"}},
		},
	}
	out := buildPrompt(input, "")
	for _, want := range []string{"Contexto relacionado", "#7", "Bug do login", "!34", "ana", "compat"} {
		if !strings.Contains(out, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPromptWithoutRelatedContextOmitsBlock(t *testing.T) {
	input := domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "feat", ProjectRef: "42"},
		Diff:         "--- a/x.go\n+++ b/x.go\n@@ -1 +1 @@\n+x",
	}
	if strings.Contains(buildPrompt(input, ""), "Contexto relacionado") {
		t.Error("expected no related block when Related is nil")
	}
}
```

(Confirme que `prompt_test.go` já importa `strings` e `domain`; se não, adicione.)

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/ai/ -run TestBuildPrompt -v`
Expected: FAIL — bloco "Contexto relacionado" ausente.

- [ ] **Step 3: Implementar**

Em `internal/infrastructure/ai/prompt.go`, adicionar a função de render (antes de `buildPrompt`):

```go
// renderRelatedContext formats the related layer (issues, linked MRs, discussion)
// into a Markdown block, including an explicit compatibility question for issues.
func renderRelatedContext(rc *domain.RelatedContext) string {
	if rc.IsEmpty() {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n## Contexto relacionado\n")

	if len(rc.Issues) > 0 {
		b.WriteString("\n### Issues vinculadas\n")
		for _, is := range rc.Issues {
			fmt.Fprintf(&b, "\n**#%d — %s** (%s)\n%s\n", is.IID, is.Title, is.State, is.Description)
		}
		b.WriteString("\nAo revisar, verifique a **compatibilidade**: o MR cumpre o que cada issue acima pede? Aponte divergências explicitamente.\n")
	}

	if len(rc.LinkedMRs) > 0 {
		b.WriteString("\n### MRs relacionados\n")
		for _, m := range rc.LinkedMRs {
			fmt.Fprintf(&b, "- !%d — %s (%s) %s\n", m.IID, m.Title, m.State, m.WebURL)
		}
	}

	if len(rc.Discussions) > 0 {
		b.WriteString("\n### Discussão do MR\n")
		for _, d := range rc.Discussions {
			fmt.Fprintf(&b, "- **%s:** %s\n", d.Author, d.Body)
		}
	}

	return b.String()
}
```

Em `buildPrompt`, montar o bloco e inseri-lo no prompt. Logo após a linha `diff = annotateDiffLineNumbers(diff)` (e antes de `outputFmt := ...`), adicionar:

```go
	related := ""
	if input.Related != nil {
		related = renderRelatedContext(input.Related)
	}
```

E no `fmt.Sprintf` final do prompt, inserir o `%s` do bloco related logo após a seção do diff. Localize o trecho do template:

```go
## Diff

%s

---

Por favor, revise o diff acima.
```

e altere para:

```go
## Diff

%s
%s

---

Por favor, revise o diff acima.
```

adicionando `related` como o argumento correspondente no `fmt.Sprintf`, na posição imediatamente após `diff`:

```go
		diff,
		related,
		maxInlineComments,
		outputFmt,
```

ATENÇÃO: a ordem dos argumentos do `fmt.Sprintf` deve casar exatamente com a ordem dos `%s`/`%d` no template. O novo `%s` do `related` vem logo após o `%s` do `diff`. Verifique todos os verbos após a inserção.

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/ai/ -v`
Expected: PASS (testes antigos + os 2 novos).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/ai/prompt.go internal/infrastructure/ai/prompt_test.go
git commit -m "feat(ai): renderiza bloco de contexto relacionado no prompt"
```

---

## Task 5: GitLab — implementar `RelatedContextProvider`

**Files:**
- Modify: `internal/infrastructure/platform/gitlab/client.go`

- [ ] **Step 1: Implementar os métodos HTTP**

Em `internal/infrastructure/platform/gitlab/client.go`, adicionar (seguindo o estilo de `GetMergeRequest`, usando `c.get` e `encodeProjectRef`):

```go
// ---------- RelatedContextProvider ----------

func (c *Client) GetIssue(projectRef string, issueIID int64) (*domain.Issue, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/issues/%d", c.baseURL, encodeProjectRef(projectRef), issueIID)

	var raw struct {
		IID         int64  `json:"iid"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		WebURL      string `json:"web_url"`
	}
	if err := c.get(url, &raw); err != nil {
		return nil, fmt.Errorf("gitlab GetIssue: %w", err)
	}
	return &domain.Issue{
		IID: raw.IID, Title: raw.Title, Description: raw.Description,
		State: raw.State, WebURL: raw.WebURL,
	}, nil
}

func (c *Client) GetClosesIssues(projectRef string, mrIID int64) ([]domain.Issue, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/closes_issues", c.baseURL, encodeProjectRef(projectRef), mrIID)

	var raw []struct {
		IID         int64  `json:"iid"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		WebURL      string `json:"web_url"`
	}
	if err := c.get(url, &raw); err != nil {
		return nil, fmt.Errorf("gitlab GetClosesIssues: %w", err)
	}
	out := make([]domain.Issue, 0, len(raw))
	for _, r := range raw {
		out = append(out, domain.Issue{
			IID: r.IID, Title: r.Title, Description: r.Description, State: r.State, WebURL: r.WebURL,
		})
	}
	return out, nil
}

func (c *Client) GetMergeRequestDiscussions(projectRef string, mrIID int64) ([]domain.Discussion, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/discussions", c.baseURL, encodeProjectRef(projectRef), mrIID)

	var raw []struct {
		Notes []struct {
			Body   string `json:"body"`
			System bool   `json:"system"`
			Author struct {
				Username string `json:"username"`
			} `json:"author"`
		} `json:"notes"`
	}
	if err := c.get(url, &raw); err != nil {
		return nil, fmt.Errorf("gitlab GetMergeRequestDiscussions: %w", err)
	}
	var out []domain.Discussion
	for _, d := range raw {
		for _, n := range d.Notes {
			if n.System || n.Body == "" {
				continue // skip system notes (label changes, etc.)
			}
			out = append(out, domain.Discussion{Author: n.Author.Username, Body: n.Body})
		}
	}
	return out, nil
}
```

- [ ] **Step 2: Verificar que o GitLab satisfaz a interface**

Adicionar um assert de compilação no fim de `internal/infrastructure/platform/gitlab/client.go`:

```go
// Compile-time check: GitLab supports the optional related-context capability.
var _ domain.RelatedContextProvider = (*Client)(nil)
```

- [ ] **Step 3: Build + testes**

Run: `go build ./... && go test ./internal/infrastructure/platform/gitlab/...`
Expected: sucesso (os métodos HTTP são testados manualmente, como o resto do client; o assert garante a interface).

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/platform/gitlab/client.go
git commit -m "feat(gitlab): implementa RelatedContextProvider (issues, closes_issues, discussions)"
```

---

## Verificação final do milestone

- [ ] **Build:** `go build ./...` → sucesso.
- [ ] **Testes:** `go test ./...` → tudo verde, sem regressão.
- [ ] **Sanidade:** com perfil `related`/`full`, o pipeline coleta e o prompt ganha o bloco "Contexto relacionado"; com perfil diff-only, nada muda. GitHub/Gitea não implementam o provider → `Collect` retorna nil (sem erro).

## Notas para o executor

- **Interface opcional, não core:** NÃO adicione os métodos a `domain.PlatformClient`. Use a interface separada `RelatedContextProvider` + type-assertion no collector. Isso evita mexer em github/gitea/mocks.
- **Best-effort de verdade:** toda falha de fetch é logada e ignorada; o review continua. Nunca propague erro da coleta de related.
- **Dedup de issues:** `closes_issues` tem prioridade; refs da descrição só entram se a IID ainda não apareceu.
- **Self-ref de MR:** ao detectar `!N`, pule `N == mr.IID`.
- **Ordem dos verbos no `fmt.Sprintf`** do `buildPrompt`: ao inserir o `%s` do related, confira que TODOS os argumentos seguintes continuam alinhados com seus verbos.
