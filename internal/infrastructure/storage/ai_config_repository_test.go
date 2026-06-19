package storage

import (
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// testEncKey is base64 of "0123456789abcdef0123456789abcdef" (32 bytes).
// The shared newTestRepo (review_profile_repository_test.go) applies it so
// secret-bearing configs can be created in tests.
const testEncKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

// TestResolveConfigCascade guards against the "ambiguous column name: id"
// regression: ResolveConfig JOINs ai_configs with ai_config_assignments (both
// have an "id" column), so the SELECT must use table-qualified columns.
func TestResolveConfigCascade(t *testing.T) {
	repo := newTestRepo(t)

	// Empty secrets → encryptField("") == "", so this works with any key.
	cfg, err := repo.CreateConfig(domain.AIConfig{
		Name:     "Claude API",
		Provider: domain.ProviderClaude,
		Mode:     domain.AIModeAPI,
	})
	if err != nil {
		t.Fatalf("CreateConfig: %v", err)
	}

	if err := repo.SetConfigAssignment(domain.AIConfigAssignment{
		ConfigID: cfg.ID, Scope: "client", ClientName: "acme",
	}); err != nil {
		t.Fatalf("SetConfigAssignment: %v", err)
	}

	// Before the fix this returns an "ambiguous column name: id" error.
	got, err := repo.ResolveConfig("acme", "999")
	if err != nil {
		t.Fatalf("ResolveConfig returned error (ambiguous column bug?): %v", err)
	}
	if got == nil || got.Name != "Claude API" {
		t.Fatalf("ResolveConfig = %+v, want config 'Claude API'", got)
	}

	// No assignment matches → nil, no error.
	none, err := repo.ResolveConfig("other", "1")
	if err != nil {
		t.Fatalf("ResolveConfig (no match) error: %v", err)
	}
	if none != nil {
		t.Fatalf("expected nil for unmatched scope, got %+v", none)
	}
}

// TestResolveConfigScopes reproduces the real webhook resolution path (which the
// "test configuration" button bypasses via GetConfig). It guards against the
// ambiguous "id" column bug: ResolveConfig JOINs ai_config_assignments, which
// also has an "id" column, so an unqualified SELECT errored and silently fell
// back to the env client (Gemini).
func TestResolveConfigScopes(t *testing.T) {
	repo := newTestRepo(t)

	admin, err := repo.CreateConfig(domain.AIConfig{
		Name: "admin", Provider: domain.ProviderClaude, Mode: domain.AIModeAccount,
		OAuthAccessToken: "tok", OAuthRefreshToken: "ref",
	})
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	geral, err := repo.CreateConfig(domain.AIConfig{
		Name: "geral", Provider: domain.ProviderGemini, Mode: domain.AIModeAPI, APIKey: "k",
	})
	if err != nil {
		t.Fatalf("create geral: %v", err)
	}

	assign := func(cfgID int64, scope, client, project string) {
		t.Helper()
		if err := repo.SetConfigAssignment(domain.AIConfigAssignment{
			ConfigID: cfgID, Scope: scope, ClientName: client, ProjectRef: project,
		}); err != nil {
			t.Fatalf("assign %s: %v", scope, err)
		}
	}
	assign(geral.ID, "client", "gitlab", "")
	assign(admin.ID, "global", "", "")
	assign(admin.ID, "project", "gitlab", "federation-agent")

	cases := []struct {
		name, client, project, wantName string
	}{
		{"project scope wins", "gitlab", "federation-agent", "admin"},
		{"client scope when no project match", "gitlab", "other-repo", "geral"},
		{"global fallback for unknown client", "github", "whatever", "admin"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.ResolveConfig(tc.client, tc.project)
			if err != nil {
				t.Fatalf("ResolveConfig errored (would fall back to env/Gemini): %v", err)
			}
			if got == nil {
				t.Fatalf("ResolveConfig returned nil; wanted %q", tc.wantName)
			}
			if got.Name != tc.wantName {
				t.Fatalf("resolved %q, wanted %q", got.Name, tc.wantName)
			}
		})
	}
}
