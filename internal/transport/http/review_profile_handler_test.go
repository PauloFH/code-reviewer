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
