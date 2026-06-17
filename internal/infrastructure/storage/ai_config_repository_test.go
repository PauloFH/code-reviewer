package storage

import (
	"path/filepath"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// testEncKey is base64 of "0123456789abcdef0123456789abcdef" (32 bytes).
const testEncKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

func newTestRepo(t *testing.T) *SQLiteRepository {
	t.Helper()
	repo, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	if err := repo.UseEncryptionKey(testEncKey); err != nil {
		t.Fatalf("UseEncryptionKey: %v", err)
	}
	return repo
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
