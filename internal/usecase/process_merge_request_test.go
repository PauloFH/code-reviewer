package usecase

import (
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

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
func (m *mockEventRepo) GetStats(_, _, _ string) (*domain.ReviewStats, error)  { return nil, nil }
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
	noteCount      int
	noteMu         sync.Mutex
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
	m.noteMu.Lock()
	defer m.noteMu.Unlock()
	m.noteCalled = true
	m.noteCount++
	return m.noteErr
}
func (m *mockClient) CreateMergeRequestInlineComment(_ string, _ *domain.MergeRequest, c domain.InlineComment) error {
	m.inlineComments = append(m.inlineComments, c)
	return m.inlineErr
}

type mockRepository struct {
	already    bool
	alreadyErr error
	claimErr   error
	marked     bool
	cleared    bool
	history    *domain.ReviewHistory
	mu         sync.Mutex
}

func (m *mockRepository) ClaimReview(_, _ string, _, _ int64, _ string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.alreadyErr != nil {
		return false, m.alreadyErr
	}
	if m.already || m.marked {
		return false, nil
	}
	m.marked = true
	return true, m.claimErr
}
func (m *mockRepository) GetReviewHistory(_, _ string, _ int64, _ string) (*domain.ReviewHistory, error) {
	return m.history, nil
}
func (m *mockRepository) CompleteReview(_, _ string, _ int64, _, _, _ string) error { return nil }
func (m *mockRepository) ClearReview(_, _ string, _ int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.marked = false
	m.cleared = true
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
	result    *domain.ReviewResult
	err       error
	lastInput domain.ReviewInput
}

func (m *mockAI) ReviewDiff(in domain.ReviewInput) (*domain.ReviewResult, error) {
	m.lastInput = in
	return m.result, m.err
}

type blockingAI struct {
	reached chan struct{}
	release <-chan struct{}
	result  *domain.ReviewResult
}

func (m *blockingAI) ReviewDiff(domain.ReviewInput) (*domain.ReviewResult, error) {
	m.reached <- struct{}{}
	<-m.release
	return m.result, nil
}

type mockProfileRepo struct {
	resolved *domain.ReviewProfile
	err      error
}

func (m *mockProfileRepo) ListProfiles() ([]domain.ReviewProfile, error)     { return nil, nil }
func (m *mockProfileRepo) GetProfile(_ int64) (*domain.ReviewProfile, error) { return nil, nil }
func (m *mockProfileRepo) CreateProfile(p domain.ReviewProfile) (*domain.ReviewProfile, error) {
	return &p, nil
}
func (m *mockProfileRepo) UpdateProfile(_ int64, p domain.ReviewProfile) (*domain.ReviewProfile, error) {
	return &p, nil
}
func (m *mockProfileRepo) DeleteProfile(_ int64) error                                 { return nil }
func (m *mockProfileRepo) SetProfileAssignment(_ domain.ReviewProfileAssignment) error { return nil }
func (m *mockProfileRepo) RemoveProfileAssignment(_, _, _ string) error                { return nil }
func (m *mockProfileRepo) ListProfileAssignments() ([]domain.ReviewProfileAssignment, error) {
	return nil, nil
}
func (m *mockProfileRepo) ResolveProfile(_, _ string) (*domain.ReviewProfile, error) {
	return m.resolved, m.err
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

func newUsecase(client *mockClient, repo *mockRepository, ai domain.AIReviewer) *ProcessMergeRequest {
	return newUsecaseWithProfiles(client, repo, ai, &mockProfileRepo{})
}

// stubWorkspace is a no-op CodeWorkspace for wiring tests.
type stubWorkspace struct{}

func (stubWorkspace) ReadFile(string) ([]byte, error)               { return nil, nil }
func (stubWorkspace) ListDir(string) ([]domain.DirEntry, error)     { return nil, nil }
func (stubWorkspace) Grep(string, string) ([]domain.GrepHit, error) { return nil, nil }
func (stubWorkspace) Root() string                                  { return "/stub" }

type mockRepoFetcher struct {
	ws      domain.CodeWorkspace
	err     error
	cleaned bool
}

func (m *mockRepoFetcher) Checkout(_ domain.RepoSource, _ string) (domain.CodeWorkspace, func(), error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.ws, func() { m.cleaned = true }, nil
}

// mockAgenticClient implements PlatformClient + domain.RepoLocator.
type mockAgenticClient struct {
	mockClient
}

func (m *mockAgenticClient) RepoSource(_ string) (domain.RepoSource, error) {
	return domain.RepoSource{RemoteURL: "https://oauth2:t@h/x.git"}, nil
}

func newUsecaseWithProfiles(client *mockClient, repo *mockRepository, ai domain.AIReviewer, profiles domain.ReviewProfileRepository) *ProcessMergeRequest {
	reg := &mockRegistry{}
	if client != nil {
		reg.client = client
	}
	return NewProcessMergeRequest(reg, repo, &mockEventRepo{}, &mockUserRepoUC{}, profiles, nil, ai, nil, 3.0, 15.0, discardLogger())
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

func TestExecute_ConcurrentWebhooksPostOnlyOneReview(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       happyMR("alice"),
		diff:     "--- a/main.go\n+++ b/main.go\n+// fix",
	}
	repo := &mockRepository{}
	release := make(chan struct{})
	ai := &blockingAI{reached: make(chan struct{}, 2), release: release, result: &domain.ReviewResult{Body: "LGTM"}}
	uc := newUsecaseWithProfiles(client, repo, ai, &mockProfileRepo{})

	errs := make(chan error, 2)
	go func() { errs <- uc.Execute(baseEvent()) }()
	<-ai.reached
	go func() { errs <- uc.Execute(baseEvent()) }()
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("duplicate webhook failed: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("duplicate webhook reached the AI reviewer instead of being skipped")
	}
	close(release)
	if err := <-errs; err != nil {
		t.Fatalf("first webhook failed: %v", err)
	}

	if client.noteCount != 1 {
		t.Fatalf("duplicate webhook processing posted %d reviews; want 1", client.noteCount)
	}
}

func TestExecute_ReanalyzePassesPreviousReviewAndJustification(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       &domain.MergeRequest{ID: 1, IID: 1, ProjectRef: "42", AuthorUsername: "carol", Reviewers: []string{"alice"}},
		diff:     "--- a/main.go\n+++ b/main.go\n+// new change",
	}
	repo := &mockRepository{history: &domain.ReviewHistory{HeadSHA: "old", Body: "A1: validar setor obrigatório."}}
	ai := &mockAI{result: &domain.ReviewResult{Body: "review body"}}
	uc := newUsecase(client, repo, ai)
	event := baseEvent()
	event.IsReanalyze = true
	event.CommentAuthor = "carol"
	event.ReviewJustification = "setor vazio é permitido neste fluxo"

	if err := uc.Execute(event); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if ai.lastInput.PreviousReview != repo.history.Body {
		t.Fatalf("PreviousReview = %q", ai.lastInput.PreviousReview)
	}
	if ai.lastInput.ReviewJustification != event.ReviewJustification {
		t.Fatalf("ReviewJustification = %q", ai.lastInput.ReviewJustification)
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
	if !repo.cleared {
		t.Error("expected the review claim to be released when posting comment fails")
	}
}

func TestExecute_ClaimReviewError_ReturnsError(t *testing.T) {
	client := &mockClient{
		name:     "test-client",
		username: "alice",
		mr:       happyMR("alice"),
		diff:     "--- a/x.go\n+x",
	}
	repo := &mockRepository{claimErr: errors.New("db locked")}
	ai := &mockAI{result: &domain.ReviewResult{Body: "review body"}}
	uc := newUsecase(client, repo, ai)

	if err := uc.Execute(baseEvent()); err == nil {
		t.Error("expected error when claiming review fails")
	}
	if client.noteCalled {
		t.Error("expected no comment when claiming review fails")
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
		t.Error("expected error when ClaimReview fails")
	}
}

func TestExecute_UnknownClient_ReturnsError(t *testing.T) {
	// Registry is empty — client lookup will fail.
	reg := &mockRegistry{}
	uc := NewProcessMergeRequest(reg, &mockRepository{}, &mockEventRepo{}, &mockUserRepoUC{}, &mockProfileRepo{}, nil, &mockAI{}, nil, 3.0, 15.0, discardLogger())

	if err := uc.Execute(baseEvent()); err == nil {
		t.Error("expected error for unknown client name")
	}
}

func TestExecute_ResolvedProfilePassedToReviewer(t *testing.T) {
	client := &mockClient{name: "test-client", username: "alice", mr: happyMR("alice"), diff: "--- a/x.go\n+x"}
	repo := &mockRepository{}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	profiles := &mockProfileRepo{resolved: &domain.ReviewProfile{Name: "related", IncludeRelated: true}}
	uc := newUsecaseWithProfiles(client, repo, ai, profiles)

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Profile == nil || !ai.lastInput.Profile.IncludeRelated {
		t.Fatalf("expected resolved related profile passed to reviewer, got %+v", ai.lastInput.Profile)
	}
}

func TestExecute_DefaultProfileWhenNoneResolved(t *testing.T) {
	client := &mockClient{name: "test-client", username: "alice", mr: happyMR("alice"), diff: "--- a/x.go\n+x"}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	uc := newUsecaseWithProfiles(client, &mockRepository{}, ai, &mockProfileRepo{resolved: nil})

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Profile == nil || ai.lastInput.Profile.IncludeRelated || ai.lastInput.Profile.Agentic {
		t.Fatalf("expected diff-only default profile, got %+v", ai.lastInput.Profile)
	}
}

func TestExecute_RelatedCollectedWhenProfileIncludesRelated(t *testing.T) {
	mr := happyMR("alice")
	mr.Description = "Closes #7"
	client := &mockRelatedClient{
		mockClient:  mockClient{name: "test-client", username: "alice", mr: mr, diff: "--- a/x.go\n+x"},
		closes:      []domain.Issue{{IID: 7, Title: "Bug"}},
		issuesByIID: map[int64]*domain.Issue{7: {IID: 7, Title: "Bug"}},
	}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	profiles := &mockProfileRepo{resolved: &domain.ReviewProfile{Name: "related", IncludeRelated: true}}
	uc := newUsecaseWithProfiles(&client.mockClient, &mockRepository{}, ai, profiles)
	// the registry must return the related-capable client:
	uc.registry = &mockRegistry{client: client}

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Related == nil || len(ai.lastInput.Related.Issues) != 1 {
		t.Fatalf("expected related context with 1 issue, got %+v", ai.lastInput.Related)
	}
}

func TestExecute_NoRelatedWhenProfileDiffOnly(t *testing.T) {
	mr := happyMR("alice")
	mr.Description = "Closes #7"
	client := &mockRelatedClient{
		mockClient: mockClient{name: "test-client", username: "alice", mr: mr, diff: "--- a/x.go\n+x"},
		closes:     []domain.Issue{{IID: 7, Title: "Bug"}},
	}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	uc := newUsecaseWithProfiles(&client.mockClient, &mockRepository{}, ai, &mockProfileRepo{resolved: nil})
	uc.registry = &mockRegistry{client: client}

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Related != nil {
		t.Fatalf("expected no related context for diff-only profile, got %+v", ai.lastInput.Related)
	}
}

func TestExecute_AgenticCheckoutSetsWorkspace(t *testing.T) {
	mr := happyMR("alice")
	client := &mockAgenticClient{mockClient: mockClient{name: "test-client", username: "alice", mr: mr, diff: "--- a/x.go\n+x"}}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	profiles := &mockProfileRepo{resolved: &domain.ReviewProfile{Name: "deep", Agentic: true}}
	fetcher := &mockRepoFetcher{ws: stubWorkspace{}}

	uc := newUsecaseWithProfiles(&client.mockClient, &mockRepository{}, ai, profiles)
	uc.registry = &mockRegistry{client: client}
	uc.repoFetcher = fetcher

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Workspace == nil {
		t.Fatal("expected workspace set for agentic profile")
	}
	if !fetcher.cleaned {
		t.Error("expected cleanup to be called after Execute")
	}
}

func TestExecute_AgenticNoLocatorFallsBack(t *testing.T) {
	// plain mockClient does NOT implement RepoLocator → no workspace, no error
	client := &mockClient{name: "test-client", username: "alice", mr: happyMR("alice"), diff: "--- a/x.go\n+x"}
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	profiles := &mockProfileRepo{resolved: &domain.ReviewProfile{Name: "deep", Agentic: true}}

	uc := newUsecaseWithProfiles(client, &mockRepository{}, ai, profiles)
	uc.repoFetcher = &mockRepoFetcher{ws: stubWorkspace{}}

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Workspace != nil {
		t.Fatal("expected no workspace when client lacks RepoLocator")
	}
}

func TestExecute_CommandPresetOverridesScope(t *testing.T) {
	// MR author issues "/review full" — authorized because they are the author.
	client := &mockClient{name: "test-client", username: "alice", mr: happyMR("alice"), diff: "--- a/x.go\n+x"}
	client.mr.AuthorUsername = "carol"
	ai := &mockAI{result: &domain.ReviewResult{Body: "ok"}}
	profiles := &mockProfileRepo{resolved: &domain.ReviewProfile{Name: "diff"}}
	uc := newUsecaseWithProfiles(client, &mockRepository{}, ai, profiles)

	ev := baseEvent()
	ev.IsReanalyze = true
	ev.CommentAuthor = "carol"
	ev.RequestedProfile = "full"

	if err := uc.Execute(ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ai.lastInput.Profile == nil || !ai.lastInput.Profile.IncludeRelated || !ai.lastInput.Profile.Agentic {
		t.Fatalf("expected /review full to override to related+agentic, got %+v", ai.lastInput.Profile)
	}
}
