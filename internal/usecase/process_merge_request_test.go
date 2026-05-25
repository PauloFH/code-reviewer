package usecase

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ---------- mocks ----------

type mockRegistry struct {
	client domain.PlatformClient
}

func (m *mockRegistry) Register(c domain.PlatformClient) { m.client = c }
func (m *mockRegistry) Unregister(_ string)              {}
func (m *mockRegistry) Get(name string) (domain.PlatformClient, bool) {
	if m.client != nil && m.client.ClientName() == name {
		return m.client, true
	}
	return nil, false
}
func (m *mockRegistry) All() []domain.PlatformClient {
	if m.client != nil {
		return []domain.PlatformClient{m.client}
	}
	return nil
}

type mockEventRepo struct{}

func (m *mockEventRepo) SaveEvent(_ domain.ReviewEvent) error { return nil }
func (m *mockEventRepo) ListEvents(_ domain.ReviewEventFilter) ([]domain.ReviewEvent, error) {
	return nil, nil
}
func (m *mockEventRepo) CountEvents(_ domain.ReviewEventFilter) (int64, error) { return 0, nil }
func (m *mockEventRepo) GetStats(_ string) (*domain.ReviewStats, error)        { return nil, nil }
func (m *mockEventRepo) GetDailyCosts(_ string, _ int) ([]domain.DailyCostPoint, error) {
	return nil, nil
}

type mockClient struct {
	name           string
	username       string
	mr             *domain.MergeRequest
	mrErr          error
	diff           string
	diffErr        error
	noteErr        error
	noteCalled     bool
	inlineErr      error
	inlineComments []domain.InlineComment
}

func (m *mockClient) Platform() domain.PlatformType          { return domain.PlatformGitLab }
func (m *mockClient) ClientName() string                     { return m.name }
func (m *mockClient) WatchedUsername() string                { return m.username }
func (m *mockClient) ValidateWebhook(domain.RawWebhook) bool { return true }
func (m *mockClient) ParseWebhook(domain.RawWebhook) (*domain.WebhookEvent, error) {
	return nil, nil
}
func (m *mockClient) GetMergeRequest(_ string, _ int64) (*domain.MergeRequest, error) {
	return m.mr, m.mrErr
}
func (m *mockClient) GetMergeRequestDiff(_ string, _ int64) (string, error) {
	return m.diff, m.diffErr
}
func (m *mockClient) CreateMergeRequestNote(_ string, _ int64, _ string) error {
	m.noteCalled = true
	return m.noteErr
}
func (m *mockClient) CreateMergeRequestInlineComment(_ string, _ *domain.MergeRequest, c domain.InlineComment) error {
	m.inlineComments = append(m.inlineComments, c)
	return m.inlineErr
}

type mockRepository struct {
	already    bool
	alreadyErr error
	markErr    error
	marked     bool
}

func (m *mockRepository) AlreadyReviewed(_, _ string, _ int64, _ string) (bool, error) {
	return m.already, m.alreadyErr
}
func (m *mockRepository) MarkAsReviewed(_, _ string, _, _ int64, _ string) error {
	m.marked = true
	return m.markErr
}
func (m *mockRepository) ClearReview(_, _ string, _ int64) error {
	return nil
}

type mockUserRepoUC struct{}

func (m *mockUserRepoUC) GetByUsername(_ string) (*domain.User, error)            { return nil, nil }
func (m *mockUserRepoUC) GetByID(_ int64) (*domain.User, error)                   { return nil, nil }
func (m *mockUserRepoUC) ListUsers() ([]domain.User, error)                       { return nil, nil }
func (m *mockUserRepoUC) CreateUser(u domain.User) (*domain.User, error)          { return &u, nil }
func (m *mockUserRepoUC) UpdateUser(_ int64, u domain.User) (*domain.User, error) { return &u, nil }
func (m *mockUserRepoUC) DeleteUser(_ int64) error                                { return nil }
func (m *mockUserRepoUC) CountUsers() (int, error)                                { return 0, nil }
func (m *mockUserRepoUC) UserExists(_ string) (bool, error)                       { return false, nil }

type mockAI struct {
	result *domain.ReviewResult
	err    error
}

func (m *mockAI) ReviewDiff(_ domain.ReviewInput) (*domain.ReviewResult, error) {
	return m.result, m.err
}

// ---------- helpers ----------

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func happyMR(username string) *domain.MergeRequest {
	return &domain.MergeRequest{
		ID:         1,
		IID:        1,
		ProjectRef: "42",
		Title:      "feat: add feature",
		WebURL:     "https://gitlab.com/proj/-/merge_requests/1",
		Reviewers:  []string{username},
	}
}

func baseEvent() *domain.WebhookEvent {
	return &domain.WebhookEvent{
		ClientName: "test-client",
		ProjectRef: "42",
		MRIID:      1,
		MRID:       100,
		Action:     "open",
	}
}

func newUsecase(client *mockClient, repo *mockRepository, ai *mockAI) *ProcessMergeRequest {
	reg := &mockRegistry{}
	if client != nil {
		reg.client = client
	}
	return NewProcessMergeRequest(reg, repo, &mockEventRepo{}, &mockUserRepoUC{}, ai, nil, 3.0, 15.0, discardLogger())
}

// ---------- tests ----------

func TestExecute_UserNotAssigned_Skips(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       &domain.MergeRequest{Reviewers: []string{"bob"}},
	}
	repo := &mockRepository{}
	ai := &mockAI{}
	uc := newUsecase(client, repo, ai)

	err := uc.Execute(baseEvent())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if client.noteCalled {
		t.Error("expected no comment to be posted when user is not assigned")
	}
}

func TestExecute_AlreadyReviewed_Skips(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       happyMR("alice"),
	}
	repo := &mockRepository{already: true}
	ai := &mockAI{}
	uc := newUsecase(client, repo, ai)

	err := uc.Execute(baseEvent())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if client.noteCalled {
		t.Error("expected no comment to be posted for already-reviewed MR")
	}
}

func TestExecute_Idempotency_MarksAfterPosting(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       happyMR("alice"),
		diff:     "--- a/main.go\n+++ b/main.go\n+// fix",
	}
	repo := &mockRepository{}
	ai := &mockAI{result: &domain.ReviewResult{Body: "LGTM"}}
	uc := newUsecase(client, repo, ai)

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.marked {
		t.Error("expected MarkAsReviewed to be called after successful review")
	}
}

func TestExecute_PostsInlineCommentsAfterMainReview(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       happyMR("alice"),
		diff:     "--- a/main.go\n+++ b/main.go\n@@ -1,1 +1,2 @@\n package main\n+func run() {}\n",
	}
	repo := &mockRepository{}
	ai := &mockAI{result: &domain.ReviewResult{
		Body: "review body",
		InlineComments: []domain.InlineComment{
			{Path: "main.go", Line: 2, Body: "Adicionar teste para run."},
			{Path: "main.go", Line: 99, Body: "Linha fora do diff."},
		},
	}}
	uc := newUsecase(client, repo, ai)

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.inlineComments) != 1 {
		t.Fatalf("expected 1 inline comment, got %d", len(client.inlineComments))
	}
	if client.inlineComments[0].Path != "main.go" || client.inlineComments[0].Line != 2 {
		t.Fatalf("unexpected inline comment: %+v", client.inlineComments[0])
	}
	if !repo.marked {
		t.Error("expected MarkAsReviewed to be called")
	}
}

func TestExecute_InlineCommentError_StillReturnsNil(t *testing.T) {
	client := &mockClient{
		name:      "test-client",
		username:  "alice",
		mr:        happyMR("alice"),
		diff:      "--- a/main.go\n+++ b/main.go\n@@ -1,1 +1,2 @@\n package main\n+func run() {}\n",
		inlineErr: errors.New("inline API failed"),
	}
	repo := &mockRepository{}
	ai := &mockAI{result: &domain.ReviewResult{
		Body: "review body",
		InlineComments: []domain.InlineComment{
			{Path: "main.go", Line: 2, Body: "Adicionar teste para run."},
		},
	}}
	uc := newUsecase(client, repo, ai)

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("expected nil when inline comments fail best-effort, got %v", err)
	}
	if !repo.marked {
		t.Error("expected MarkAsReviewed to be called")
	}
}

func TestExecute_GetMergeRequestError_ReturnsError(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mrErr:    errors.New("gitlab unreachable"),
	}
	uc := newUsecase(client, &mockRepository{}, &mockAI{})

	if err := uc.Execute(baseEvent()); err == nil {
		t.Error("expected error when GetMergeRequest fails")
	}
}

func TestExecute_GetDiffError_ReturnsError(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       happyMR("alice"),
		diffErr:  errors.New("diff endpoint error"),
	}
	repo := &mockRepository{already: false}
	uc := newUsecase(client, repo, &mockAI{})

	if err := uc.Execute(baseEvent()); err == nil {
		t.Error("expected error when GetMergeRequestDiff fails")
	}
}

func TestExecute_AIError_ReturnsError(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       happyMR("alice"),
		diff:     "--- a/x.go\n+x",
	}
	repo := &mockRepository{}
	ai := &mockAI{err: errors.New("AI API timeout")}
	uc := newUsecase(client, repo, ai)

	if err := uc.Execute(baseEvent()); err == nil {
		t.Error("expected error when AI reviewer fails")
	}
	if client.noteCalled {
		t.Error("expected no comment when AI fails")
	}
}

func TestExecute_CreateNoteError_ReturnsError(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       happyMR("alice"),
		diff:     "--- a/x.go\n+x",
		noteErr:  errors.New("cannot post comment"),
	}
	repo := &mockRepository{}
	ai := &mockAI{result: &domain.ReviewResult{Body: "review body"}}
	uc := newUsecase(client, repo, ai)

	if err := uc.Execute(baseEvent()); err == nil {
		t.Error("expected error when CreateMergeRequestNote fails")
	}
	if repo.marked {
		t.Error("expected MarkAsReviewed NOT to be called when posting comment fails")
	}
}

func TestExecute_MarkAsReviewedError_StillReturnsNil(t *testing.T) {
	// Comment was posted successfully — a mark failure should only log, not error.
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       happyMR("alice"),
		diff:     "--- a/x.go\n+x",
	}
	repo := &mockRepository{markErr: errors.New("db locked")}
	ai := &mockAI{result: &domain.ReviewResult{Body: "review body"}}
	uc := newUsecase(client, repo, ai)

	if err := uc.Execute(baseEvent()); err != nil {
		t.Errorf("expected nil after successful post (mark failure is best-effort), got %v", err)
	}
}

func TestExecute_IdempotencyCheck_Error_ReturnsError(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       happyMR("alice"),
	}
	repo := &mockRepository{alreadyErr: errors.New("db error")}
	uc := newUsecase(client, repo, &mockAI{})

	if err := uc.Execute(baseEvent()); err == nil {
		t.Error("expected error when AlreadyReviewed fails")
	}
}

func TestExecute_UnknownClient_ReturnsError(t *testing.T) {
	// Registry is empty — client lookup will fail.
	reg := &mockRegistry{}
	uc := NewProcessMergeRequest(reg, &mockRepository{}, &mockEventRepo{}, &mockUserRepoUC{}, &mockAI{}, nil, 3.0, 15.0, discardLogger())

	if err := uc.Execute(baseEvent()); err == nil {
		t.Error("expected error for unknown client name")
	}
}
