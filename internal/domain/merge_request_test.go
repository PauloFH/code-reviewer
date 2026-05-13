package domain

import "testing"

func TestIsUserAssigned_InReviewers(t *testing.T) {
	mr := &MergeRequest{Reviewers: []string{"alice", "bob"}}
	if !mr.IsUserAssigned("alice") {
		t.Error("expected alice to be detected as reviewer")
	}
}

func TestIsUserAssigned_InAssignees(t *testing.T) {
	mr := &MergeRequest{Assignees: []string{"carol"}}
	if !mr.IsUserAssigned("carol") {
		t.Error("expected carol to be detected as assignee")
	}
}

func TestIsUserAssigned_NotPresent(t *testing.T) {
	mr := &MergeRequest{
		Reviewers: []string{"alice"},
		Assignees: []string{"bob"},
	}
	if mr.IsUserAssigned("charlie") {
		t.Error("charlie should not be detected")
	}
}

func TestIsUserAssigned_EmptyLists(t *testing.T) {
	mr := &MergeRequest{}
	if mr.IsUserAssigned("anyone") {
		t.Error("empty MR should never match")
	}
}

func TestIsUserAssigned_CaseSensitive(t *testing.T) {
	mr := &MergeRequest{Reviewers: []string{"Alice"}}
	if mr.IsUserAssigned("alice") {
		t.Error("matching should be case-sensitive")
	}
}
