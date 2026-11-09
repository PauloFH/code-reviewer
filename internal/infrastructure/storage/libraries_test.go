package storage

import (
	"errors"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// checkResolution exercises the scope cascade of any scoped library: lib must
// hold records named "a" (id a) and "b" (id b) and no assignments.
func checkResolution[T any](t *testing.T, lib domain.ScopedLibrary[T], a, b int64, name func(T) string) {
	t.Helper()
	set := func(target int64, k domain.ScopeKey) {
		t.Helper()
		if err := lib.SetAssignment(domain.Assignment{TargetID: target, ScopeKey: k}); err != nil {
			t.Fatalf("SetAssignment %+v: %v", k, err)
		}
	}
	resolve := func(client, project string) string {
		t.Helper()
		got, err := lib.Resolve(client, project)
		if err != nil {
			t.Fatalf("Resolve(%s,%s): %v", client, project, err)
		}
		if got == nil {
			return ""
		}
		return name(*got)
	}

	if got := resolve("acme", "x"); got != "" {
		t.Fatalf("nothing assigned resolved to %q", got)
	}
	// Fields the scope does not use are dropped, so this is THE global slot.
	set(a, domain.ScopeKey{Scope: domain.ScopeGlobal, ClientName: "junk", ProjectRef: "junk"})
	set(b, domain.ScopeKey{Scope: domain.ScopeClient, ClientName: "acme", ProjectRef: "ignored"})
	set(a, domain.ScopeKey{Scope: domain.ScopeProject, ClientName: "acme", ProjectRef: "core"})

	for _, tc := range []struct{ client, project, want string }{
		{"acme", "core", "a"},  // project beats client
		{"acme", "other", "b"}, // client beats global
		{"globex", "core", "a"},
	} {
		if got := resolve(tc.client, tc.project); got != tc.want {
			t.Errorf("Resolve(%s,%s) = %q, want %q", tc.client, tc.project, got, tc.want)
		}
	}

	set(b, domain.ScopeKey{Scope: domain.ScopeGlobal})
	list, err := lib.ListAssignments()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("assignments = %+v, want 3 (re-setting global replaces it)", list)
	}
	if err := lib.RemoveAssignment(domain.ScopeKey{Scope: domain.ScopeClient, ClientName: "acme"}); err != nil {
		t.Fatal(err)
	}
	if got := resolve("acme", "other"); got != "b" {
		t.Errorf("after removing client slot resolved %q, want global b", got)
	}
	if err := lib.SetAssignment(domain.Assignment{TargetID: a, ScopeKey: domain.ScopeKey{Scope: domain.ScopeClient}}); err == nil {
		t.Error("client scope without client_name was accepted")
	}
}

// clearAssignments drops the global defaults the migrations seed.
func clearAssignments(t *testing.T, repo *SQLiteRepository, deleteSQL string) {
	t.Helper()
	if _, err := repo.db.Exec(deleteSQL); err != nil {
		t.Fatal(err)
	}
}

func TestPromptResolution(t *testing.T) {
	repo := newTestRepo(t)
	clearAssignments(t, repo, `DELETE FROM prompt_assignments`)
	lib := repo.Prompts()
	a, _ := lib.Create(domain.Prompt{Name: "a"})
	b, _ := lib.Create(domain.Prompt{Name: "b"})
	checkResolution(t, lib, a.ID, b.ID, func(p domain.Prompt) string { return p.Name })
}

func TestReviewProfileResolution(t *testing.T) {
	repo := newTestRepo(t)
	clearAssignments(t, repo, `DELETE FROM review_profile_assignments`)
	lib := repo.ReviewProfiles()
	a, _ := lib.Create(domain.ReviewProfile{Name: "a", Agentic: true})
	b, _ := lib.Create(domain.ReviewProfile{Name: "b"})
	checkResolution(t, lib, a.ID, b.ID, func(p domain.ReviewProfile) string { return p.Name })
}

func TestAIConfigResolution(t *testing.T) {
	lib := newTestRepo(t).AIConfigs()
	a, err := lib.Create(domain.AIConfig{Name: "a", Provider: domain.ProviderClaude, Mode: domain.AIModeAccount, OAuthAccessToken: "tok", OAuthRefreshToken: "ref"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := lib.Create(domain.AIConfig{Name: "b", Provider: domain.ProviderGemini, Mode: domain.AIModeAPI, APIKey: "k"})
	checkResolution(t, lib, a.ID, b.ID, func(c domain.AIConfig) string { return c.Name })

	got, err := lib.Resolve("acme", "core")
	if err != nil || got.OAuthAccessToken != "tok" || got.OAuthRefreshToken != "ref" {
		t.Fatalf("resolved config lost its secrets: %+v %v", got, err)
	}
}

func TestLibraryWriteErrors(t *testing.T) {
	lib := newTestRepo(t).ReviewProfiles()
	p, err := lib.Create(domain.ReviewProfile{Name: "Full", IncludeRelated: true, MaxIterations: 12})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := lib.Get(p.ID); !got.IncludeRelated || got.MaxIterations != 12 || got.CreatedAt.IsZero() {
		t.Fatalf("round trip: %+v", got)
	}
	if _, err := lib.Create(domain.ReviewProfile{Name: "Full"}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("duplicate name: %v, want ErrConflict", err)
	}
	if _, err := lib.Update(999, domain.ReviewProfile{Name: "x"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("update missing: %v, want ErrNotFound", err)
	}
	if err := lib.Delete(999); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("delete missing: %v, want ErrNotFound", err)
	}
	if err := lib.Delete(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := lib.Get(p.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("get deleted: %v, want ErrNotFound", err)
	}
}

func TestAssignmentToMissingTargetIsInvalid(t *testing.T) {
	err := newTestRepo(t).Prompts().SetAssignment(domain.Assignment{TargetID: 999, ScopeKey: domain.ScopeKey{Scope: domain.ScopeGlobal}})
	var invalid *domain.InvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("SetAssignment to a missing target = %v, want InvalidError", err)
	}
}
