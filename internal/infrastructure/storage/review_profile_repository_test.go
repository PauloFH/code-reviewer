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

	if err := repo.SetProfileAssignment(domain.ReviewProfileAssignment{
		ProfileID: clientP.ID, Scope: "client", ClientName: "acme",
	}); err != nil {
		t.Fatalf("SetProfileAssignment client: %v", err)
	}
	if err := repo.SetProfileAssignment(domain.ReviewProfileAssignment{
		ProfileID: projectP.ID, Scope: "project", ClientName: "acme", ProjectRef: "42",
	}); err != nil {
		t.Fatalf("SetProfileAssignment project: %v", err)
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
