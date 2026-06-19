# Modos de Revisão — Fase 1 / Milestone 5a: API REST de perfis — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expor CRUD de perfis de revisão e atribuições por escopo via REST (`/api/v1/review-profiles`), espelhando os endpoints de prompts, com JWT + `requireRole(super, gestor)`.

**Architecture:** `ReviewProfileHandler` (mirror de `PromptHandler`) sobre `domain.ReviewProfileRepository`. Rotas registradas no `router.go`; handler construído e injetado no `main.go`. Atenção aos nomes renomeados do M1: `SetProfileAssignment`/`RemoveProfileAssignment`/`ListProfileAssignments`.

**Tech Stack:** Go (stdlib `net/http`, Go 1.22 ServeMux + `PathValue`), testes com `httptest`.

**Pré-requisitos:** M1 (`domain.ReviewProfileRepository` + entidades).

**Referências de padrão (ler antes):**
- `internal/transport/http/prompt_handler.go` — estrutura do handler a espelhar (`writeJSON`, validação, PathValue).
- `internal/transport/http/router.go` — `NewRouter`, `chain`, `requireRole`, padrão das rotas de prompts (linhas 111-143).
- `cmd/server/main.go:165,177` — criação dos handlers e chamada de `NewRouter`.

---

## File Structure

- **Create:** `internal/transport/http/review_profile_handler.go` — `ReviewProfileHandler`.
- **Create:** `internal/transport/http/review_profile_handler_test.go` — testes httptest.
- **Modify:** `internal/transport/http/router.go` — parâmetro + rotas.
- **Modify:** `cmd/server/main.go` — construir e injetar o handler.

---

## Task 1: ReviewProfileHandler

**Files:**
- Create: `internal/transport/http/review_profile_handler.go`
- Create: `internal/transport/http/review_profile_handler_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Criar `internal/transport/http/review_profile_handler_test.go`:

```go
package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type mockProfileRepoHTTP struct {
	profiles    []domain.ReviewProfile
	assignments []domain.ReviewProfileAssignment
	lastSet     domain.ReviewProfileAssignment
}

func (m *mockProfileRepoHTTP) ListProfiles() ([]domain.ReviewProfile, error) { return m.profiles, nil }
func (m *mockProfileRepoHTTP) GetProfile(id int64) (*domain.ReviewProfile, error) {
	for i := range m.profiles {
		if m.profiles[i].ID == id {
			return &m.profiles[i], nil
		}
	}
	return nil, nil
}
func (m *mockProfileRepoHTTP) CreateProfile(p domain.ReviewProfile) (*domain.ReviewProfile, error) {
	p.ID = 1
	return &p, nil
}
func (m *mockProfileRepoHTTP) UpdateProfile(id int64, p domain.ReviewProfile) (*domain.ReviewProfile, error) {
	p.ID = id
	return &p, nil
}
func (m *mockProfileRepoHTTP) DeleteProfile(int64) error { return nil }
func (m *mockProfileRepoHTTP) SetProfileAssignment(a domain.ReviewProfileAssignment) error {
	m.lastSet = a
	return nil
}
func (m *mockProfileRepoHTTP) RemoveProfileAssignment(_, _, _ string) error { return nil }
func (m *mockProfileRepoHTTP) ListProfileAssignments() ([]domain.ReviewProfileAssignment, error) {
	return m.assignments, nil
}
func (m *mockProfileRepoHTTP) ResolveProfile(_, _ string) (*domain.ReviewProfile, error) {
	return nil, nil
}

func TestReviewProfileHandler_Create(t *testing.T) {
	h := NewReviewProfileHandler(&mockProfileRepoHTTP{})
	body := `{"name":"Full","include_related":true,"agentic":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-profiles", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.CreateProfile(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	var got domain.ReviewProfile
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID == 0 || !got.IncludeRelated || !got.Agentic {
		t.Fatalf("unexpected created profile: %+v", got)
	}
}

func TestReviewProfileHandler_CreateRequiresName(t *testing.T) {
	h := NewReviewProfileHandler(&mockProfileRepoHTTP{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/review-profiles", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	h.CreateProfile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestReviewProfileHandler_List(t *testing.T) {
	repo := &mockProfileRepoHTTP{profiles: []domain.ReviewProfile{{ID: 1, Name: "Diff"}}}
	h := NewReviewProfileHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/review-profiles", nil)
	rec := httptest.NewRecorder()

	h.ListProfiles(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Diff") {
		t.Fatalf("body missing profile: %s", rec.Body.String())
	}
}

func TestReviewProfileHandler_SetAssignmentValidatesScope(t *testing.T) {
	h := NewReviewProfileHandler(&mockProfileRepoHTTP{})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/review-profiles/assignments",
		strings.NewReader(`{"profile_id":1,"scope":"bogus"}`))
	rec := httptest.NewRecorder()

	h.SetAssignment(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for invalid scope", rec.Code)
	}
}

func TestReviewProfileHandler_SetAssignmentOK(t *testing.T) {
	repo := &mockProfileRepoHTTP{}
	h := NewReviewProfileHandler(repo)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/review-profiles/assignments",
		strings.NewReader(`{"profile_id":2,"scope":"client","client_name":"acme"}`))
	rec := httptest.NewRecorder()

	h.SetAssignment(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if repo.lastSet.ProfileID != 2 || repo.lastSet.Scope != "client" {
		t.Fatalf("assignment not persisted: %+v", repo.lastSet)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/transport/http/ -run TestReviewProfileHandler -v`
Expected: FAIL — `NewReviewProfileHandler` indefinido.

- [ ] **Step 3: Implementar**

Criar `internal/transport/http/review_profile_handler.go`:

```go
package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ReviewProfileHandler serves /api/v1/review-profiles routes.
type ReviewProfileHandler struct {
	repo domain.ReviewProfileRepository
}

func NewReviewProfileHandler(repo domain.ReviewProfileRepository) *ReviewProfileHandler {
	return &ReviewProfileHandler{repo: repo}
}

func (h *ReviewProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.repo.ListProfiles()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if profiles == nil {
		profiles = []domain.ReviewProfile{}
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (h *ReviewProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	p, err := h.repo.GetProfile(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if p == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ReviewProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var p domain.ReviewProfile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if p.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	created, err := h.repo.CreateProfile(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *ReviewProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var p domain.ReviewProfile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	updated, err := h.repo.UpdateProfile(id, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *ReviewProfileHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.DeleteProfile(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReviewProfileHandler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	assignments, err := h.repo.ListProfileAssignments()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if assignments == nil {
		assignments = []domain.ReviewProfileAssignment{}
	}
	writeJSON(w, http.StatusOK, assignments)
}

func (h *ReviewProfileHandler) SetAssignment(w http.ResponseWriter, r *http.Request) {
	var a domain.ReviewProfileAssignment
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if a.Scope == "" || a.ProfileID == 0 {
		http.Error(w, "scope and profile_id are required", http.StatusBadRequest)
		return
	}
	if a.Scope != "global" && a.Scope != "client" && a.Scope != "project" {
		http.Error(w, "scope must be global, client, or project", http.StatusBadRequest)
		return
	}
	if err := h.repo.SetProfileAssignment(a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReviewProfileHandler) RemoveAssignment(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	clientName := r.URL.Query().Get("client_name")
	projectRef := r.URL.Query().Get("project_ref")
	if scope == "" {
		http.Error(w, "scope query param is required", http.StatusBadRequest)
		return
	}
	if err := h.repo.RemoveProfileAssignment(scope, clientName, projectRef); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/transport/http/ -run TestReviewProfileHandler -v`
Expected: PASS (5 testes).

- [ ] **Step 5: Commit**

```bash
git add internal/transport/http/review_profile_handler.go internal/transport/http/review_profile_handler_test.go
git commit -m "feat(http): ReviewProfileHandler (CRUD + atribuições)"
```

---

## Task 2: Rotas no router

**Files:**
- Modify: `internal/transport/http/router.go`

- [ ] **Step 1: Adicionar o parâmetro**

Em `NewRouter`, adicionar o parâmetro `reviewProfiles *ReviewProfileHandler` logo após `prompts *PromptHandler,`.

- [ ] **Step 2: Adicionar as rotas**

Após o bloco de rotas de prompts (depois da linha que registra `POST /api/v1/prompts`), adicionar:

```go
	// ── Review profiles (super + gestor) ──────────────────────────────────────
	mux.Handle("GET /api/v1/review-profiles/assignments", chain(
		http.HandlerFunc(reviewProfiles.ListAssignments),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("PUT /api/v1/review-profiles/assignments", chain(
		http.HandlerFunc(reviewProfiles.SetAssignment),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("DELETE /api/v1/review-profiles/assignments", chain(
		http.HandlerFunc(reviewProfiles.RemoveAssignment),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("GET /api/v1/review-profiles/{id}", chain(
		http.HandlerFunc(reviewProfiles.GetProfile),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("PUT /api/v1/review-profiles/{id}", chain(
		http.HandlerFunc(reviewProfiles.UpdateProfile),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("DELETE /api/v1/review-profiles/{id}", chain(
		http.HandlerFunc(reviewProfiles.DeleteProfile),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("GET /api/v1/review-profiles", chain(
		http.HandlerFunc(reviewProfiles.ListProfiles),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("POST /api/v1/review-profiles", chain(
		http.HandlerFunc(reviewProfiles.CreateProfile),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
```

- [ ] **Step 3: Build**

Run: `go build ./internal/transport/http/`
Expected: FAIL no `main.go` ainda não (só este pacote compila; o `main.go` será ajustado na Task 3). `go build ./internal/transport/http/` deve compilar OK (o pacote http não chama NewRouter).

- [ ] **Step 4: Commit**

```bash
git add internal/transport/http/router.go
git commit -m "feat(http): rotas /api/v1/review-profiles (super+gestor)"
```

---

## Task 3: Wiring no main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Construir e injetar**

Em `cmd/server/main.go`:

(a) Após `promptHandler := transport.NewPromptHandler(repo)`, adicionar:

```go
	reviewProfileHandler := transport.NewReviewProfileHandler(repo)
```

(b) Na chamada `transport.NewRouter(...)`, inserir `reviewProfileHandler` logo após `promptHandler`:

```go
	router := transport.NewRouter(uc, registry, authHandler, apiHandler, promptHandler, reviewProfileHandler, aiConfigHandler, userHandler, broadcaster, log)
```

- [ ] **Step 2: Build + testes**

Run: `go build ./... && go test ./...`
Expected: sucesso, tudo verde.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(main): injeta ReviewProfileHandler no router"
```

---

## Verificação final do milestone

- [ ] **Build:** `go build ./...` → sucesso.
- [ ] **Testes:** `go test ./...` → tudo verde, sem regressão.
- [ ] **Sanidade:** `GET/POST /api/v1/review-profiles`, `GET/PUT/DELETE /api/v1/review-profiles/{id}` e `.../assignments` exigem JWT + super/gestor; `{id}` literal "assignments" tem precedência sobre o wildcard `{id}` pela regra do ServeMux Go 1.22.

## Notas para o executor

- **Nomes do M1:** o handler chama `ListProfileAssignments`/`SetProfileAssignment`/`RemoveProfileAssignment` (NÃO `ListAssignments` etc.) no repositório.
- **`writeJSON` já existe** no pacote `http` (usado pelo prompt_handler). Reutilize.
- **NewRouter só é chamado no main.go** — não há callers de teste; a mudança de assinatura só afeta o main.go.
- **Precedência de rotas:** o ServeMux Go 1.22 resolve `assignments` (literal) vs `{id}` (wildcard) preferindo o literal; manter a ordem do plano (assignments antes de {id}) é o padrão já usado nos prompts.
