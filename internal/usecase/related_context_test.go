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
func (m *mockRelatedClient) GetMergeRequest(ref string, iid int64) (*domain.MergeRequest, error) {
	if mr, ok := m.mrsByIID[iid]; ok {
		return mr, nil
	}
	// Fall back to the embedded mockClient so pipeline's main MR lookup still works.
	return m.mockClient.GetMergeRequest(ref, iid)
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
