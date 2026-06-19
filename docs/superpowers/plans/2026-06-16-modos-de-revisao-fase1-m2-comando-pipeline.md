# Modos de Revisão — Fase 1 / Milestone 2: Override por comando + integração no pipeline — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrar a resolução de perfil no pipeline de revisão e permitir override por comando `/review <preset>` no GitLab, mantilhando o default diff-only e 100% do comportamento legado.

**Architecture:** O `usecase.ProcessMergeRequest` passa a resolver o `ReviewProfile` por escopo (via `domain.ReviewProfileRepository`, injetado), aplicar override quando o comentário `/review <preset>` vem de um autor autorizado, e plumar o perfil resolvido em `ReviewInput.Profile`. O parsing de presets é função pura no domínio. Nesta etapa o perfil ainda não muda o comportamento da IA (related/agentic chegam em M3/M4) — é resolução + override + plumbing.

**Tech Stack:** Go (stdlib), testes com `testing` e mocks por interface.

**Pré-requisito:** Milestone 1 concluído (entidades `ReviewProfile`, `DefaultReviewProfile`, `ReviewProfileRepository` com `ResolveProfile`).

**Referências de padrão (ler antes):**
- `internal/infrastructure/platform/gitlab/webhook.go` — `parseNoteWebhook` (hoje exige exatamente `/review`).
- `internal/infrastructure/platform/gitlab/webhook_test.go` — estilo dos testes de webhook (`rawWebhook`, payloads via `map[string]any`).
- `internal/usecase/process_merge_request.go` — pipeline `Execute`, autorização de `/review` (linhas ~104-112).
- `internal/usecase/process_merge_request_test.go` — mocks (`mockClient`, `mockAI`, `newUsecase`, `NewProcessMergeRequest` direto).
- `cmd/server/main.go:149-153` — chamada de `NewProcessMergeRequest`.

**IMPORTANTE — nomes reais do M1:** os métodos de atribuição no repositório/intf chamam-se `SetProfileAssignment`/`RemoveProfileAssignment`/`ListProfileAssignments` (renomeados no M1 para evitar colisão com `PromptRepository`). O método de resolução é `ResolveProfile`.

---

## File Structure

- **Modify:** `internal/domain/platform.go` — campo `RequestedProfile string` em `WebhookEvent`.
- **Modify:** `internal/domain/review.go` — campo `Profile *ReviewProfile` em `ReviewInput`.
- **Modify:** `internal/domain/review_profile.go` — função pura `ProfileFromPreset`.
- **Modify:** `internal/domain/review_profile_test.go` — teste de `ProfileFromPreset`.
- **Modify:** `internal/infrastructure/platform/gitlab/webhook.go` — `parseNoteWebhook` aceita `/review <preset>`.
- **Modify:** `internal/infrastructure/platform/gitlab/webhook_test.go` — testes de parsing de preset.
- **Modify:** `internal/usecase/process_merge_request.go` — injeção de `profileRepo` + `resolveProfile` + plumbing.
- **Modify:** `internal/usecase/process_merge_request_test.go` — `mockProfileRepo`, captura de input no `mockAI`, novos testes.
- **Modify:** `cmd/server/main.go:149-153` — passar `repo` como `profileRepo`.

---

## Task 1: Domínio — campos e parsing de preset (puro)

**Files:**
- Modify: `internal/domain/platform.go`
- Modify: `internal/domain/review.go`
- Modify: `internal/domain/review_profile.go`
- Modify: `internal/domain/review_profile_test.go`

- [ ] **Step 1: Escrever o teste que falha**

Adicionar ao fim de `internal/domain/review_profile_test.go`:

```go
func TestProfileFromPreset(t *testing.T) {
	cases := []struct {
		preset                   string
		ok                       bool
		includeRelated, agentic  bool
	}{
		{"diff", true, false, false},
		{"related", true, true, false},
		{"deep", true, false, true},
		{"full", true, true, true},
		{"DEEP", true, false, true}, // case-insensitive
		{"bogus", false, false, false},
		{"", false, false, false},
	}
	for _, tc := range cases {
		got, ok := ProfileFromPreset(tc.preset)
		if ok != tc.ok {
			t.Errorf("ProfileFromPreset(%q) ok = %v, want %v", tc.preset, ok, tc.ok)
			continue
		}
		if ok && (got.IncludeRelated != tc.includeRelated || got.Agentic != tc.agentic) {
			t.Errorf("ProfileFromPreset(%q) = {related:%v agentic:%v}, want {related:%v agentic:%v}",
				tc.preset, got.IncludeRelated, got.Agentic, tc.includeRelated, tc.agentic)
		}
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/domain/ -run TestProfileFromPreset -v`
Expected: FAIL — `undefined: ProfileFromPreset`.

- [ ] **Step 3: Implementar**

Em `internal/domain/review_profile.go`, adicionar (após `DefaultReviewProfile`):

```go
// ProfileFromPreset maps a /review command preset to the corresponding layered
// profile. The second return value is false for unknown presets. The empty
// string is not a preset (it means "no override").
func ProfileFromPreset(preset string) (ReviewProfile, bool) {
	switch strings.ToLower(preset) {
	case "diff":
		return ReviewProfile{Name: "diff"}, true
	case "related":
		return ReviewProfile{Name: "related", IncludeRelated: true}, true
	case "deep":
		return ReviewProfile{Name: "deep", Agentic: true}, true
	case "full":
		return ReviewProfile{Name: "full", IncludeRelated: true, Agentic: true}, true
	default:
		return ReviewProfile{}, false
	}
}
```

Adicionar `"strings"` ao import de `internal/domain/review_profile.go`:

```go
import (
	"strings"
	"time"
)
```

Em `internal/domain/platform.go`, adicionar o campo ao struct `WebhookEvent` (após `CommentAuthor string`):

```go
	CommentAuthor string
	// RequestedProfile is the preset token after a "/review <preset>" command
	// (e.g. "deep", "full"). Empty means no override was requested.
	RequestedProfile string
```

Em `internal/domain/review.go`, adicionar o campo ao struct `ReviewInput` (após `ProjectRef string`):

```go
	ProjectRef   string
	// Profile is the resolved review profile (layers). Nil means diff-only.
	Profile *ReviewProfile
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/domain/ -v`
Expected: PASS (inclui `TestDefaultReviewProfile` e `TestProfileFromPreset`).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/platform.go internal/domain/review.go internal/domain/review_profile.go internal/domain/review_profile_test.go
git commit -m "feat(domain): RequestedProfile, ReviewInput.Profile e ProfileFromPreset"
```

---

## Task 2: GitLab webhook — aceitar `/review <preset>`

**Files:**
- Modify: `internal/infrastructure/platform/gitlab/webhook.go`
- Modify: `internal/infrastructure/platform/gitlab/webhook_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Adicionar ao fim de `internal/infrastructure/platform/gitlab/webhook_test.go`:

```go
func notePayload(note string) []byte {
	payload := map[string]any{
		"object_kind": "note",
		"project":     map[string]any{"id": 42},
		"object_attributes": map[string]any{
			"note":          note,
			"noteable_type": "MergeRequest",
		},
		"merge_request": map[string]any{"id": 100, "iid": 7},
		"user":          map[string]any{"username": "carol"},
	}
	b, _ := json.Marshal(payload)
	return b
}

func TestParseWebhook_ReviewCommandNoPreset(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{"x-gitlab-token": "mysecret"}, notePayload("/review"))

	event, err := c.ParseWebhook(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !event.IsReanalyze || event.CommentAuthor != "carol" {
		t.Fatalf("expected reanalyze by carol, got %+v", event)
	}
	if event.RequestedProfile != "" {
		t.Errorf("RequestedProfile = %q, want empty", event.RequestedProfile)
	}
}

func TestParseWebhook_ReviewCommandWithPreset(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{"x-gitlab-token": "mysecret"}, notePayload("/review deep"))

	event, err := c.ParseWebhook(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.RequestedProfile != "deep" {
		t.Errorf("RequestedProfile = %q, want %q", event.RequestedProfile, "deep")
	}
}

func TestParseWebhook_NonReviewNoteIgnored(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{"x-gitlab-token": "mysecret"}, notePayload("just a comment"))

	if _, err := c.ParseWebhook(w); err == nil {
		t.Error("expected non-/review note to be ignored with an error")
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/platform/gitlab/ -run TestParseWebhook_Review -v`
Expected: FAIL — `RequestedProfile` sempre vazio (parser atual exige exatamente `/review` e não preenche o campo) e/ou `/review deep` é rejeitado.

- [ ] **Step 3: Implementar**

Em `internal/infrastructure/platform/gitlab/webhook.go`, substituir o bloco de validação do comando dentro de `parseNoteWebhook` (as linhas que hoje fazem `note := strings.TrimSpace(...)` e `if note != "/review" { ... }`) por:

```go
	note := strings.TrimSpace(payload.ObjectAttributes.Note)
	fields := strings.Fields(note)
	if len(fields) == 0 || fields[0] != "/review" {
		return nil, fmt.Errorf("gitlab: note is not /review command, ignoring")
	}
	requestedProfile := ""
	if len(fields) > 1 {
		requestedProfile = strings.ToLower(fields[1])
	}
```

E adicionar o campo no `domain.WebhookEvent` retornado por `parseNoteWebhook` (após `CommentAuthor: payload.User.Username,`):

```go
		CommentAuthor:    payload.User.Username,
		RequestedProfile: requestedProfile,
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/platform/gitlab/ -v`
Expected: PASS (testes antigos + os 3 novos).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/platform/gitlab/webhook.go internal/infrastructure/platform/gitlab/webhook_test.go
git commit -m "feat(gitlab): parse /review <preset> em comentário de MR"
```

---

## Task 3: Usecase — resolução de perfil + override + plumbing

**Files:**
- Modify: `internal/usecase/process_merge_request.go`
- Modify: `internal/usecase/process_merge_request_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Em `internal/usecase/process_merge_request_test.go`:

(a) Tornar o `mockAI` capaz de capturar o input — substituir a struct e o método existentes por:

```go
type mockAI struct {
	result    *domain.ReviewResult
	err       error
	lastInput domain.ReviewInput
}

func (m *mockAI) ReviewDiff(in domain.ReviewInput) (*domain.ReviewResult, error) {
	m.lastInput = in
	return m.result, m.err
}
```

(b) Adicionar um mock de `ReviewProfileRepository` e um construtor de perfil resolvido:

```go
type mockProfileRepo struct {
	resolved *domain.ReviewProfile
	err      error
}

func (m *mockProfileRepo) ListProfiles() ([]domain.ReviewProfile, error)          { return nil, nil }
func (m *mockProfileRepo) GetProfile(_ int64) (*domain.ReviewProfile, error)       { return nil, nil }
func (m *mockProfileRepo) CreateProfile(p domain.ReviewProfile) (*domain.ReviewProfile, error) {
	return &p, nil
}
func (m *mockProfileRepo) UpdateProfile(_ int64, p domain.ReviewProfile) (*domain.ReviewProfile, error) {
	return &p, nil
}
func (m *mockProfileRepo) DeleteProfile(_ int64) error                              { return nil }
func (m *mockProfileRepo) SetProfileAssignment(_ domain.ReviewProfileAssignment) error { return nil }
func (m *mockProfileRepo) RemoveProfileAssignment(_, _, _ string) error             { return nil }
func (m *mockProfileRepo) ListProfileAssignments() ([]domain.ReviewProfileAssignment, error) {
	return nil, nil
}
func (m *mockProfileRepo) ResolveProfile(_, _ string) (*domain.ReviewProfile, error) {
	return m.resolved, m.err
}
```

(c) Atualizar o helper `newUsecase` e o construtor direto para injetar um `mockProfileRepo`. Substituir o `newUsecase` existente por uma versão que aceita o profile repo, e ajustar os call sites:

```go
func newUsecase(client *mockClient, repo *mockRepository, ai *mockAI) *ProcessMergeRequest {
	return newUsecaseWithProfiles(client, repo, ai, &mockProfileRepo{})
}

func newUsecaseWithProfiles(client *mockClient, repo *mockRepository, ai *mockAI, profiles domain.ReviewProfileRepository) *ProcessMergeRequest {
	reg := &mockRegistry{}
	if client != nil {
		reg.client = client
	}
	return NewProcessMergeRequest(reg, repo, &mockEventRepo{}, &mockUserRepoUC{}, profiles, ai, nil, 3.0, 15.0, discardLogger())
}
```

E no teste `TestExecute_UnknownClient_ReturnsError`, atualizar a chamada direta:

```go
	uc := NewProcessMergeRequest(reg, &mockRepository{}, &mockEventRepo{}, &mockUserRepoUC{}, &mockProfileRepo{}, &mockAI{}, nil, 3.0, 15.0, discardLogger())
```

(d) Adicionar os novos testes de comportamento:

```go
func TestExecute_ResolvedProfilePassedToReviewer(t *testing.T) {
	client := &mockClient{name: "test-client", username: "alice", mr: happyMR("alice"), diff: "--- a/x.go\n+x"}
	repo := &mockRepository{}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	profiles := &mockProfileRepo{resolved: &domain.ReviewProfile{Name: "related", IncludeRelated: true}}
	uc := newUsecaseWithProfiles(client, repo, ai, profiles)

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Profile == nil || !ai.lastInput.Profile.IncludeRelated {
		t.Fatalf("expected resolved related profile passed to reviewer, got %+v", ai.lastInput.Profile)
	}
}

func TestExecute_DefaultProfileWhenNoneResolved(t *testing.T) {
	client := &mockClient{name: "test-client", username: "alice", mr: happyMR("alice"), diff: "--- a/x.go\n+x"}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	uc := newUsecaseWithProfiles(client, &mockRepository{}, ai, &mockProfileRepo{resolved: nil})

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Profile == nil || ai.lastInput.Profile.IncludeRelated || ai.lastInput.Profile.Agentic {
		t.Fatalf("expected diff-only default profile, got %+v", ai.lastInput.Profile)
	}
}

func TestExecute_CommandPresetOverridesScope(t *testing.T) {
	// MR author issues "/review full" — authorized because they are the author.
	client := &mockClient{name: "test-client", username: "alice", mr: happyMR("alice"), diff: "--- a/x.go\n+x"}
	client.mr.AuthorUsername = "carol"
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	profiles := &mockProfileRepo{resolved: &domain.ReviewProfile{Name: "diff"}}
	uc := newUsecaseWithProfiles(client, &mockRepository{}, ai, profiles)

	ev := baseEvent()
	ev.IsReanalyze = true
	ev.CommentAuthor = "carol"
	ev.RequestedProfile = "full"

	if err := uc.Execute(ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Profile == nil || !ai.lastInput.Profile.IncludeRelated || !ai.lastInput.Profile.Agentic {
		t.Fatalf("expected /review full to override to related+agentic, got %+v", ai.lastInput.Profile)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/usecase/ -run TestExecute_ResolvedProfile -v`
Expected: FAIL de compilação — `NewProcessMergeRequest` ainda não aceita o `profileRepo` e `ReviewInput.Profile` não é preenchido.

- [ ] **Step 3: Implementar**

Em `internal/usecase/process_merge_request.go`:

(a) Adicionar o campo na struct `ProcessMergeRequest` (após `userRepo domain.UserRepository`):

```go
	userRepo       domain.UserRepository
	profileRepo    domain.ReviewProfileRepository
```

(b) Adicionar o parâmetro no construtor `NewProcessMergeRequest`, na mesma posição (após `userRepo domain.UserRepository,`), e atribuí-lo:

```go
func NewProcessMergeRequest(
	registry domain.ClientRegistry,
	repository domain.ReviewRepository,
	eventRepo domain.ReviewEventRepository,
	userRepo domain.UserRepository,
	profileRepo domain.ReviewProfileRepository,
	reviewer domain.AIReviewer,
	publisher EventPublisher,
	pricePerMInput, pricePerMOut float64,
	log *slog.Logger,
) *ProcessMergeRequest {
	return &ProcessMergeRequest{
		registry:       registry,
		repository:     repository,
		eventRepo:      eventRepo,
		userRepo:       userRepo,
		profileRepo:    profileRepo,
		reviewer:       reviewer,
		publisher:      publisher,
		pricePerMInput: pricePerMInput,
		pricePerMOut:   pricePerMOut,
		log:            log,
	}
}
```

(c) Adicionar o helper `resolveProfile` (no fim do arquivo, junto às outras funções privadas):

```go
// resolveProfile picks the review profile for this event: the one assigned to
// the most specific scope (project → client → global), or the diff-only default
// when nothing is assigned. A valid "/review <preset>" override replaces it.
func (uc *ProcessMergeRequest) resolveProfile(log *slog.Logger, event *domain.WebhookEvent) domain.ReviewProfile {
	profile := domain.DefaultReviewProfile()
	if uc.profileRepo != nil {
		resolved, err := uc.profileRepo.ResolveProfile(event.ClientName, event.ProjectRef)
		if err != nil {
			log.Error("resolving review profile, using diff-only default", slog.String("error", err.Error()))
		} else if resolved != nil {
			profile = *resolved
		}
	}
	if event.RequestedProfile != "" {
		if preset, ok := domain.ProfileFromPreset(event.RequestedProfile); ok {
			profile = preset
			log.Info("review profile overridden by /review command", slog.String("preset", event.RequestedProfile))
		} else {
			log.Warn("unknown /review preset, ignoring", slog.String("preset", event.RequestedProfile))
		}
	}
	return profile
}
```

(d) No `Execute`, logo antes da chamada `result, err := uc.reviewer.ReviewDiff(...)`, resolver o perfil e plumá-lo no input. Substituir o bloco:

```go
	log.Info("sending diff to AI reviewer")

	result, err := uc.reviewer.ReviewDiff(domain.ReviewInput{
		MergeRequest: mr,
		Diff:         diff,
		ClientName:   event.ClientName,
		ProjectRef:   event.ProjectRef,
	})
```

por:

```go
	profile := uc.resolveProfile(log, event)
	log.Info("sending diff to AI reviewer", slog.String("profile", profile.Name))

	result, err := uc.reviewer.ReviewDiff(domain.ReviewInput{
		MergeRequest: mr,
		Diff:         diff,
		ClientName:   event.ClientName,
		ProjectRef:   event.ProjectRef,
		Profile:      &profile,
	})
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/usecase/ -v`
Expected: PASS (testes antigos + os 3 novos de perfil).

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/process_merge_request.go internal/usecase/process_merge_request_test.go
git commit -m "feat(usecase): resolução de perfil por escopo + override /review no pipeline"
```

---

## Task 4: Wiring no main.go

**Files:**
- Modify: `cmd/server/main.go:149-153`

- [ ] **Step 1: Atualizar a chamada do construtor**

Em `cmd/server/main.go`, na chamada `usecase.NewProcessMergeRequest`, adicionar `repo` como `profileRepo` (4º argumento de repositório, após o `userRepo`). O `*SQLiteRepository` (`repo`) já satisfaz `domain.ReviewProfileRepository` desde o M1:

```go
	uc := usecase.NewProcessMergeRequest(
		registry, repo, repo, repo, repo, reviewer, broadcaster,
		cfg.AI.PricePerMInputTokens, cfg.AI.PricePerMOutputTokens,
		log,
	)
```

- [ ] **Step 2: Verificar compilação**

Run: `go build ./...`
Expected: sucesso.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(main): injeta ReviewProfileRepository no ProcessMergeRequest"
```

---

## Verificação final do milestone

- [ ] **Build:** `go build ./...` → sucesso.
- [ ] **Testes:** `go test ./...` → tudo verde, sem regressão.
- [ ] **Sanidade:** o fluxo legado (webhook de MR sem perfil atribuído e sem `/review`) resolve para o perfil global "Diff" semeado no M1 → `ReviewInput.Profile` = diff-only; comportamento da IA inalterado nesta etapa.

## Notas para o executor

- **Escopo desta etapa:** apenas resolução + override + plumbing. `ReviewInput.Profile` ainda NÃO altera o comportamento da IA — `related` (M3) e `agentic` (M4) virão depois. Não implemente nada de related/agentic aqui (YAGNI).
- **GitHub/Gitea:** não têm parsing de comentário hoje (só evento `pull_request`), então o override `/review <preset>` é só GitLab nesta fase. Não invente parsing de nota para eles.
- **Autorização do override:** já é garantida pelo bloco existente de `/review` (autor do MR ou usuário registrado); não duplicar a checagem — o `resolveProfile` só aplica o preset, a autorização/skip continua acontecendo antes no `Execute`.
- **Posição dos parâmetros:** mantenha `profileRepo` logo após `userRepo` no construtor e propague a ordem em TODOS os call sites (test helper, teste direto, main.go).
