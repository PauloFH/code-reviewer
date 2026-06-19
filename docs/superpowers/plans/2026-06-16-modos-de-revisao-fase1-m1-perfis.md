# Modos de Revisão — Fase 1 / Milestone 1: Perfis de revisão (fundação) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Criar a fundação de dados dos perfis de revisão — entidades de domínio, repositório SQLite com CRUD/atribuições e resolução por escopo (projeto→client→global), com default `diff`-only quando nada resolve.

**Architecture:** Espelha exatamente o que já existe para `Prompt`/`AIConfig`: entidade + assignment no `domain`, interface `ReviewProfileRepository` no `domain`, tabela versionada via migration e implementação no `SQLiteRepository`. Nenhuma criptografia (perfis não guardam segredo). A regra "sem perfil = diff-only" vive numa função pura `DefaultReviewProfile()` no domínio.

**Tech Stack:** Go (stdlib + `github.com/mattn/go-sqlite3`), `database/sql`, testes com `testing` e `t.TempDir()`.

**Referências de padrão (ler antes de começar):**
- `internal/domain/prompt.go` — formato de entidade + assignment espelhado.
- `internal/infrastructure/storage/ai_config_repository.go` — CRUD + `ResolveConfig` cascata + scanner.
- `internal/infrastructure/storage/migrations.go` — formato das migrations (`migration003` cria prompts).
- `internal/infrastructure/storage/sqlite_repository.go:54-93` — lista `pending` de migrations e `runMigrations`; `internal/infrastructure/storage/sqlite_repository.go:384` — `type scanner interface`.

---

## File Structure

- **Create:** `internal/domain/review_profile.go` — entidades `ReviewProfile`, `ReviewProfileAssignment` e a função pura `DefaultReviewProfile()`.
- **Create:** `internal/domain/review_profile_test.go` — teste puro de `DefaultReviewProfile()`.
- **Modify:** `internal/domain/repository.go` — adiciona a interface `ReviewProfileRepository`.
- **Modify:** `internal/infrastructure/storage/migrations.go` — adiciona `migration007` (tabelas `review_profiles` + `review_profile_assignments` + seed do perfil global "Diff").
- **Modify:** `internal/infrastructure/storage/sqlite_repository.go` — registra `{7, migration007}` na lista `pending`.
- **Create:** `internal/infrastructure/storage/review_profile_repository.go` — CRUD + atribuições + `ResolveProfile`.
- **Create:** `internal/infrastructure/storage/review_profile_repository_test.go` — testes de round-trip CRUD e cascata de `ResolveProfile` com DB temporário.

---

## Task 1: Entidades de domínio + default puro

**Files:**
- Create: `internal/domain/review_profile.go`
- Test: `internal/domain/review_profile_test.go`

- [ ] **Step 1: Escrever o teste que falha**

```go
package domain

import "testing"

func TestDefaultReviewProfile(t *testing.T) {
	p := DefaultReviewProfile()
	if p.IncludeRelated {
		t.Errorf("default profile must not include related context")
	}
	if p.Agentic {
		t.Errorf("default profile must not be agentic")
	}
	if p.Execution {
		t.Errorf("default profile must not enable execution")
	}
	if p.Name != "Diff" {
		t.Errorf("default profile name = %q, want %q", p.Name, "Diff")
	}
}
```

- [ ] **Step 2: Rodar o teste e ver falhar**

Run: `go test ./internal/domain/ -run TestDefaultReviewProfile -v`
Expected: FAIL — `undefined: DefaultReviewProfile`.

- [ ] **Step 3: Implementar as entidades e o default**

Criar `internal/domain/review_profile.go`:

```go
package domain

import "time"

// ReviewProfile descreve quais camadas de contexto a revisão usa.
// A base (diff) está sempre presente; IncludeRelated e Agentic são aditivas.
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

// ReviewProfileAssignment vincula um ReviewProfile a um escopo na hierarquia:
// "global" (fallback), "client" (por provider Git) ou "project" (por repo).
// Espelha PromptAssignment / AIConfigAssignment.
type ReviewProfileAssignment struct {
	ID          int64  `json:"id"`
	ProfileID   int64  `json:"profile_id"`
	ProfileName string `json:"profile_name,omitempty"`
	Scope       string `json:"scope"`
	ClientName  string `json:"client_name"`
	ProjectRef  string `json:"project_ref"`
}

// DefaultReviewProfile é o perfil efetivo quando nenhum perfil é resolvido por
// escopo: somente diff, preservando o comportamento legado.
func DefaultReviewProfile() ReviewProfile {
	return ReviewProfile{Name: "Diff"}
}
```

- [ ] **Step 4: Rodar o teste e ver passar**

Run: `go test ./internal/domain/ -run TestDefaultReviewProfile -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/review_profile.go internal/domain/review_profile_test.go
git commit -m "feat(domain): ReviewProfile, assignment e default diff-only"
```

---

## Task 2: Interface do repositório no domínio

**Files:**
- Modify: `internal/domain/repository.go`

- [ ] **Step 1: Adicionar a interface**

Em `internal/domain/repository.go`, após o bloco `AIConfigRepository`, adicionar:

```go
// ReviewProfileRepository manages the review profile library and scope-based
// assignments, mirroring PromptRepository.
type ReviewProfileRepository interface {
	ListProfiles() ([]ReviewProfile, error)
	GetProfile(id int64) (*ReviewProfile, error)
	CreateProfile(p ReviewProfile) (*ReviewProfile, error)
	UpdateProfile(id int64, p ReviewProfile) (*ReviewProfile, error)
	DeleteProfile(id int64) error
	SetAssignment(a ReviewProfileAssignment) error
	RemoveAssignment(scope, clientName, projectRef string) error
	ListAssignments() ([]ReviewProfileAssignment, error)
	// ResolveProfile returns the profile assigned to the most specific scope
	// (project → client → global). Returns nil when nothing is assigned.
	ResolveProfile(clientName, projectRef string) (*ReviewProfile, error)
}
```

- [ ] **Step 2: Verificar compilação do domínio**

Run: `go build ./internal/domain/`
Expected: sem saída (sucesso). A interface não tem implementação ainda, mas o `domain` não exige isso.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/repository.go
git commit -m "feat(domain): interface ReviewProfileRepository"
```

---

## Task 3: Migration das tabelas

**Files:**
- Modify: `internal/infrastructure/storage/migrations.go`
- Modify: `internal/infrastructure/storage/sqlite_repository.go:67-74`

- [ ] **Step 1: Adicionar a migration**

No fim de `internal/infrastructure/storage/migrations.go`, adicionar:

```go
// migration007 adds the review profiles library and scope-based assignments,
// mirroring the prompts schema. Seeds a global "Diff" profile (legacy behavior).
const migration007 = `
CREATE TABLE IF NOT EXISTS review_profiles (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    name            TEXT     NOT NULL UNIQUE,
    include_related INTEGER  NOT NULL DEFAULT 0,
    agentic         INTEGER  NOT NULL DEFAULT 0,
    execution       INTEGER  NOT NULL DEFAULT 0,
    max_iterations  INTEGER  NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS review_profile_assignments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id   INTEGER NOT NULL REFERENCES review_profiles(id) ON DELETE CASCADE,
    scope        TEXT    NOT NULL CHECK(scope IN ('global','client','project')),
    client_name  TEXT    NOT NULL DEFAULT '',
    project_ref  TEXT    NOT NULL DEFAULT '',
    UNIQUE(scope, client_name, project_ref)
);

CREATE INDEX IF NOT EXISTS idx_review_profile_assignments_lookup
    ON review_profile_assignments(client_name, project_ref);

INSERT INTO review_profiles(name, include_related, agentic, execution, max_iterations, created_at, updated_at)
VALUES('Diff', 0, 0, 0, 0, datetime('now'), datetime('now'));

INSERT INTO review_profile_assignments(profile_id, scope, client_name, project_ref)
VALUES((SELECT id FROM review_profiles WHERE name='Diff'), 'global', '', '');
`
```

- [ ] **Step 2: Registrar a migration na lista**

Em `internal/infrastructure/storage/sqlite_repository.go`, no slice `pending` (linhas 67-74), adicionar a entrada após `{6, migration006},`:

```go
	pending := []migration{
		{1, migration001},
		{2, migration002},
		{3, migration003},
		{4, migration004},
		{5, migration005},
		{6, migration006},
		{7, migration007},
	}
```

- [ ] **Step 3: Verificar compilação**

Run: `go build ./internal/infrastructure/storage/`
Expected: sucesso.

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/storage/migrations.go internal/infrastructure/storage/sqlite_repository.go
git commit -m "feat(storage): migration007 com review_profiles e seed global Diff"
```

---

## Task 4: Repositório SQLite (CRUD + atribuições + ResolveProfile)

**Files:**
- Create: `internal/infrastructure/storage/review_profile_repository.go`
- Test: `internal/infrastructure/storage/review_profile_repository_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Criar `internal/infrastructure/storage/review_profile_repository_test.go`:

```go
package storage

import (
	"path/filepath"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func newTestRepo(t *testing.T) *SQLiteRepository {
	t.Helper()
	repo, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return repo
}

func TestReviewProfileCRUDRoundTrip(t *testing.T) {
	repo := newTestRepo(t)

	created, err := repo.CreateProfile(domain.ReviewProfile{
		Name:           "Full",
		IncludeRelated: true,
		Agentic:        true,
		MaxIterations:  12,
	})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("CreateProfile returned ID 0")
	}

	got, err := repo.GetProfile(created.ID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got == nil || !got.IncludeRelated || !got.Agentic || got.MaxIterations != 12 {
		t.Fatalf("GetProfile round-trip mismatch: %+v", got)
	}

	got.Agentic = false
	if _, err := repo.UpdateProfile(got.ID, *got); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	after, _ := repo.GetProfile(got.ID)
	if after.Agentic {
		t.Fatalf("UpdateProfile did not persist Agentic=false")
	}

	if err := repo.DeleteProfile(got.ID); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
	gone, _ := repo.GetProfile(got.ID)
	if gone != nil {
		t.Fatalf("DeleteProfile did not remove the profile")
	}
}

func TestResolveProfileCascade(t *testing.T) {
	repo := newTestRepo(t)

	clientP, _ := repo.CreateProfile(domain.ReviewProfile{Name: "Related", IncludeRelated: true})
	projectP, _ := repo.CreateProfile(domain.ReviewProfile{Name: "Deep", Agentic: true})

	if err := repo.SetAssignment(domain.ReviewProfileAssignment{
		ProfileID: clientP.ID, Scope: "client", ClientName: "acme",
	}); err != nil {
		t.Fatalf("SetAssignment client: %v", err)
	}
	if err := repo.SetAssignment(domain.ReviewProfileAssignment{
		ProfileID: projectP.ID, Scope: "project", ClientName: "acme", ProjectRef: "42",
	}); err != nil {
		t.Fatalf("SetAssignment project: %v", err)
	}

	// project scope wins for the exact match
	got, err := repo.ResolveProfile("acme", "42")
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}
	if got == nil || got.Name != "Deep" {
		t.Fatalf("project resolve = %+v, want Deep", got)
	}

	// client scope wins when no project match
	got, _ = repo.ResolveProfile("acme", "999")
	if got == nil || got.Name != "Related" {
		t.Fatalf("client resolve = %+v, want Related", got)
	}

	// falls back to the seeded global "Diff" profile
	got, _ = repo.ResolveProfile("other", "1")
	if got == nil || got.Name != "Diff" {
		t.Fatalf("global resolve = %+v, want seeded Diff", got)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/storage/ -run 'TestReviewProfile|TestResolveProfile' -v`
Expected: FAIL — métodos `CreateProfile`/`GetProfile`/etc. indefinidos no `SQLiteRepository`.

- [ ] **Step 3: Implementar o repositório**

Criar `internal/infrastructure/storage/review_profile_repository.go`:

```go
package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// reviewProfileColumns is the canonical column list for review_profiles selects.
const reviewProfileColumns = `id, name, include_related, agentic, execution,
	max_iterations, created_at, updated_at`

func (r *SQLiteRepository) ListProfiles() ([]domain.ReviewProfile, error) {
	rows, err := r.db.Query(`SELECT ` + reviewProfileColumns + ` FROM review_profiles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing review profiles: %w", err)
	}
	defer rows.Close()

	var out []domain.ReviewProfile
	for rows.Next() {
		p, err := scanReviewProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) GetProfile(id int64) (*domain.ReviewProfile, error) {
	row := r.db.QueryRow(`SELECT `+reviewProfileColumns+` FROM review_profiles WHERE id=?`, id)
	p, err := scanReviewProfile(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *SQLiteRepository) CreateProfile(p domain.ReviewProfile) (*domain.ReviewProfile, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO review_profiles(
		name, include_related, agentic, execution, max_iterations, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?)`,
		p.Name, p.IncludeRelated, p.Agentic, p.Execution, p.MaxIterations, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("review profile %q already exists", p.Name)
		}
		return nil, fmt.Errorf("creating review profile: %w", err)
	}
	id, _ := res.LastInsertId()
	p.ID = id
	p.CreatedAt = now
	p.UpdatedAt = now
	return &p, nil
}

func (r *SQLiteRepository) UpdateProfile(id int64, p domain.ReviewProfile) (*domain.ReviewProfile, error) {
	now := time.Now().UTC()
	_, err := r.db.Exec(`UPDATE review_profiles SET
		name=?, include_related=?, agentic=?, execution=?, max_iterations=?, updated_at=?
		WHERE id=?`,
		p.Name, p.IncludeRelated, p.Agentic, p.Execution, p.MaxIterations, now, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("review profile %q already exists", p.Name)
		}
		return nil, fmt.Errorf("updating review profile: %w", err)
	}
	p.ID = id
	p.UpdatedAt = now
	return &p, nil
}

func (r *SQLiteRepository) DeleteProfile(id int64) error {
	if _, err := r.db.Exec(`DELETE FROM review_profiles WHERE id=?`, id); err != nil {
		return fmt.Errorf("deleting review profile: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) SetAssignment(a domain.ReviewProfileAssignment) error {
	_, err := r.db.Exec(`INSERT OR REPLACE INTO review_profile_assignments(profile_id, scope, client_name, project_ref) VALUES(?,?,?,?)`,
		a.ProfileID, a.Scope, a.ClientName, a.ProjectRef)
	if err != nil {
		return fmt.Errorf("setting review profile assignment: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) RemoveAssignment(scope, clientName, projectRef string) error {
	_, err := r.db.Exec(`DELETE FROM review_profile_assignments WHERE scope=? AND client_name=? AND project_ref=?`,
		scope, clientName, projectRef)
	if err != nil {
		return fmt.Errorf("removing review profile assignment: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListAssignments() ([]domain.ReviewProfileAssignment, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.profile_id, p.name, a.scope, a.client_name, a.project_ref
		FROM review_profile_assignments a
		JOIN review_profiles p ON p.id = a.profile_id
		ORDER BY a.scope, a.client_name, a.project_ref`)
	if err != nil {
		return nil, fmt.Errorf("listing review profile assignments: %w", err)
	}
	defer rows.Close()

	var out []domain.ReviewProfileAssignment
	for rows.Next() {
		var a domain.ReviewProfileAssignment
		if err := rows.Scan(&a.ID, &a.ProfileID, &a.ProfileName, &a.Scope, &a.ClientName, &a.ProjectRef); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ResolveProfile returns the review profile assigned to the most specific scope
// that matches: project → client → global. Returns nil when nothing is assigned.
func (r *SQLiteRepository) ResolveProfile(clientName, projectRef string) (*domain.ReviewProfile, error) {
	queries := []struct {
		sql  string
		args []any
	}{
		{`SELECT ` + reviewProfileColumns + ` FROM review_profiles p
			JOIN review_profile_assignments a ON a.profile_id = p.id
			WHERE a.scope='project' AND a.client_name=? AND a.project_ref=?`, []any{clientName, projectRef}},
		{`SELECT ` + reviewProfileColumns + ` FROM review_profiles p
			JOIN review_profile_assignments a ON a.profile_id = p.id
			WHERE a.scope='client' AND a.client_name=? AND a.project_ref=''`, []any{clientName}},
		{`SELECT ` + reviewProfileColumns + ` FROM review_profiles p
			JOIN review_profile_assignments a ON a.profile_id = p.id
			WHERE a.scope='global'`, nil},
	}

	for _, q := range queries {
		row := r.db.QueryRow(q.sql, q.args...)
		p, err := scanReviewProfile(row)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		return &p, nil
	}
	return nil, nil
}

func scanReviewProfile(s scanner) (domain.ReviewProfile, error) {
	var p domain.ReviewProfile
	var includeRelated, agentic, execution int
	var createdAt, updatedAt string

	err := s.Scan(&p.ID, &p.Name, &includeRelated, &agentic, &execution,
		&p.MaxIterations, &createdAt, &updatedAt)
	if err != nil {
		return p, err
	}

	p.IncludeRelated = includeRelated != 0
	p.Agentic = agentic != 0
	p.Execution = execution != 0
	p.CreatedAt = parseDBTime(createdAt)
	p.UpdatedAt = parseDBTime(updatedAt)
	return p, nil
}
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/storage/ -run 'TestReviewProfile|TestResolveProfile' -v`
Expected: PASS (3 testes: CRUD round-trip + cascata).

- [ ] **Step 5: Rodar a suíte completa do storage para garantir não-regressão**

Run: `go test ./internal/infrastructure/storage/...`
Expected: PASS (inclui os testes de `crypto.go` já existentes).

- [ ] **Step 6: Commit**

```bash
git add internal/infrastructure/storage/review_profile_repository.go internal/infrastructure/storage/review_profile_repository_test.go
git commit -m "feat(storage): repositório de review profiles com ResolveProfile por escopo"
```

---

## Verificação final do milestone

- [ ] **Build geral:** `go build ./...` → sucesso.
- [ ] **Testes gerais:** `go test ./...` → tudo verde (sem regressão; `*SQLiteRepository` agora satisfaz `domain.ReviewProfileRepository`).
- [ ] **Sanidade da interface (opcional):** confirmar em revisão que `*SQLiteRepository` implementa `domain.ReviewProfileRepository` — todos os 9 métodos da interface existem com as assinaturas exatas.

## Notas para o executor

- **Booleans no SQLite:** as colunas são `INTEGER` (0/1). O `scanReviewProfile` lê em `int` e converte para `bool` — não escanear direto em `*bool`.
- **Sem criptografia:** diferente de `ai_config_repository.go`, perfis não chamam `encryptField`/`decryptField`.
- **`parseDBTime` e `type scanner`** já existem no pacote `storage` (em `ai_config_repository.go` e `sqlite_repository.go`); reutilize, não redefina.
- Este milestone **não** faz wiring no `main.go` nem no pipeline — só a fundação de dados. A integração no `usecase` é o Milestone 2.
