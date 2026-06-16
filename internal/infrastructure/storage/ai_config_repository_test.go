package storage

import (
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// TestResolveConfigCascade guards against the "ambiguous column name: id"
// regression: ResolveConfig JOINs ai_configs with ai_config_assignments (both
// have an "id" column), so the SELECT must use table-qualified columns.
func TestResolveConfigCascade(t *testing.T) {
	repo := newTestRepo(t)

	// Empty secrets → no encryption key needed (encryptField("") == "").
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
