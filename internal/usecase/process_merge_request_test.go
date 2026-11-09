package usecase

import (
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ---------- fakes ----------

type fakeRegistry map[string]domain.PlatformClient

func (r fakeRegistry) Register(c domain.PlatformClient) { r[c.ClientName()] = c }
func (r fakeRegistry) Unregister(name string)           { delete(r, name) }
func (r fakeRegistry) Replace(oldName string, c domain.PlatformClient) {
	delete(r, oldName)
	r[c.ClientName()] = c
}
func (r fakeRegistry) Get(name string) (domain.PlatformClient, bool) {
	c, ok := r[name]
	return c, ok
}

// mockClient is a PlatformClient whose diff and note calls can fail the first N times.
type mockClient struct {
	name, username string
	mr             *domain.MergeRequest
	mrErr          error
	diff           string
	diffFails      int
	noteFails      int
	inlineErr      error

	diffCalls int
	notes     []string // every note attempt, failed ones included
	inline    []domain.InlineComment
}

func (m *mockClient) Platform() domain.PlatformType          { return domain.PlatformGitLab }
func (m *mockClient) ClientName() string                     { return m.name }
func (m *mockClient) WatchedUsername() string                { return m.username }
func (m *mockClient) ValidateWebhook(domain.RawWebhook) bool { return true }
func (m *mockClient) ParseWebhook(domain.RawWebhook) (*domain.WebhookEvent, error) {
	return nil, nil
}
func (m *mockClient) GetMergeRequest(string, int64) (*domain.MergeRequest, error) {
	return m.mr, m.mrErr
}
func (m *mockClient) GetMergeRequestDiff(string, int64) (string, error) {
	m.diffCalls++
	if m.diffCalls <= m.diffFails {
		return "", errors.New("diff unavailable")
	}
	return m.diff, nil
}
func (m *mockClient) CreateMergeRequestNote(_ string, _ int64, body string) error {
	m.notes = append(m.notes, body)
	if len(m.notes) <= m.noteFails {
		return errors.New("cannot post")
	}
	return nil
}
func (m *mockClient) CreateMergeRequestInlineComment(_ *domain.MergeRequest, c domain.InlineComment) error {
	m.inline = append(m.inline, c)
	return m.inlineErr
}

type incrementalClient struct {
	mockClient
	from, to string
}

func (c *incrementalClient) GetMergeRequestDiffSince(_, from, to string) (string, error) {
	c.from, c.to = from, to
	return "incremental", nil
}

type locatorClient struct{ mockClient }

func (*locatorClient) RepoSource(string) (domain.RepoSource, error) {
	return domain.RepoSource{RemoteURL: "https://git.example.com/x.git"}, nil
}

type fakeClaims struct {
	mu            sync.Mutex
	claimed       bool
	claimErr      error
	history       *domain.ReviewHistory
	completedSHA  string
	completedBody string
}

func (f *fakeClaims) ClaimReview(_, _ string, _, _ int64, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.claimErr != nil {
		return false, f.claimErr
	}
	if f.claimed {
		return false, nil
	}
	f.claimed = true
	return true, nil
}
func (f *fakeClaims) GetReviewHistory(_, _ string, _ int64, _ string) (*domain.ReviewHistory, error) {
	return f.history, nil
}
func (f *fakeClaims) CompleteReview(_, _ string, _ int64, _, headSHA, body string) error {
	f.completedSHA, f.completedBody = headSHA, body
	return nil
}
func (f *fakeClaims) ClearReview(_, _ string, _ int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claimed = false
	return nil
}

// eventLog is both the event store and the live-feed publisher.
type eventLog struct {
	mu        sync.Mutex
	saved     []domain.ReviewEvent
	published int
}

func (l *eventLog) SaveEvent(e domain.ReviewEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.saved = append(l.saved, e)
	return nil
}
func (l *eventLog) Publish(domain.ReviewEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.published++
}
func (l *eventLog) last() domain.ReviewEvent { return l.saved[len(l.saved)-1] }

type fakeUsers map[string]bool

func (u fakeUsers) GetByUsername(name string) (*domain.User, error) {
	if !u[name] {
		return nil, domain.ErrNotFound
	}
	return &domain.User{Username: name}, nil
}

type fakeProfiles struct {
	p   *domain.ReviewProfile
	err error
}

func (f fakeProfiles) Resolve(string, string) (*domain.ReviewProfile, error) { return f.p, f.err }

type fakeFetcher struct {
	err     error
	sha     string
	cleaned bool
}

func (f *fakeFetcher) Checkout(_ domain.RepoSource, sha string) (domain.CodeWorkspace, func(), error) {
	f.sha = sha
	if f.err != nil {
		return nil, nil, f.err
	}
	return stubWorkspace{}, func() { f.cleaned = true }, nil
}

type stubWorkspace struct{}

func (stubWorkspace) ReadFile(string) ([]byte, error)               { return nil, nil }
func (stubWorkspace) ListDir(string) ([]domain.DirEntry, error)     { return nil, nil }
func (stubWorkspace) Grep(string, string) ([]domain.GrepHit, error) { return nil, nil }
func (stubWorkspace) Root() string                                  { return "/stub" }

type mockAI struct {
	result *domain.ReviewResult
	fails  int
	calls  int
	last   domain.ReviewInput
}

func (m *mockAI) ReviewDiff(in domain.ReviewInput) (*domain.ReviewResult, error) {
	m.calls++
	m.last = in
	if m.calls <= m.fails {
		return nil, errors.New("AI unavailable")
	}
	return m.result, nil
}

type blockingAI struct {
	reached chan struct{}
	release chan struct{}
}

func (m *blockingAI) ReviewDiff(domain.ReviewInput) (*domain.ReviewResult, error) {
	m.reached <- struct{}{}
	<-m.release
	return &domain.ReviewResult{Body: "LGTM"}, nil
}

// ---------- helpers ----------

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// newClient watches "alice", who reviews an MR authored by "carol" at head "new".
func newClient() *mockClient {
	return &mockClient{
		name:     "acme",
		username: "alice",
		diff:     "+x",
		mr: &domain.MergeRequest{
			ID: 1, IID: 1, ProjectRef: "42", Title: "feat", AuthorUsername: "carol",
			HeadSHA: "new", Reviewers: []string{"alice"},
		},
	}
}

func baseEvent() *domain.WebhookEvent {
	return &domain.WebhookEvent{ClientName: "acme", ProjectRef: "42", MRIID: 1, Action: "open"}
}

func okAI() *mockAI { return &mockAI{result: &domain.ReviewResult{Body: "LGTM"}} }

// newPipeline wires the use case with fakes; mod may replace any dependency.
func newPipeline(client domain.PlatformClient, ai domain.AIReviewer, mod func(*ReviewDeps)) (*ProcessMergeRequest, *fakeClaims, *eventLog) {
	events := &eventLog{}
	d := ReviewDeps{
		Registry:  fakeRegistry{client.ClientName(): client},
		Claims:    &fakeClaims{},
		Events:    events,
		Users:     fakeUsers{},
		Profiles:  fakeProfiles{},
		Fetcher:   &fakeFetcher{},
		Reviewer:  ai,
		Publisher: events,
		Retry:     RetryConfig{MaxAttempts: 1},
		Log:       discardLogger(),
	}
	if mod != nil {
		mod(&d)
	}
	return NewProcessMergeRequest(d), d.Claims.(*fakeClaims), events
}

// ---------- tests ----------

func TestExecute_PostsReviewAndRecordsOutcome(t *testing.T) {
	client := newClient()
	ai := &mockAI{result: &domain.ReviewResult{
		Body:    "LGTM",
		Usage:   domain.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		CostUSD: 0.42,
	}}
	uc, claims, events := newPipeline(client, ai, nil)

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if ai.last.Diff != "+x" || ai.last.MergeRequest != client.mr {
		t.Errorf("reviewer got diff %q, mr %+v", ai.last.Diff, ai.last.MergeRequest)
	}
	if !slices.Equal(client.notes, []string{"LGTM"}) {
		t.Errorf("notes = %q, want the review body once", client.notes)
	}
	if claims.completedSHA != "new" || claims.completedBody != "LGTM" {
		t.Errorf("CompleteReview(%q, %q), want head SHA and body", claims.completedSHA, claims.completedBody)
	}
	e := events.last()
	if e.Status != domain.ReviewStatusSuccess || e.CostUSD != 0.42 || e.TotalTokens != 15 || e.MRTitle != "feat" {
		t.Errorf("event = %+v", e)
	}
	if events.published != len(events.saved) {
		t.Errorf("published %d of %d events", events.published, len(events.saved))
	}
}

func TestExecute_SkipsWithoutReviewing(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*mockClient, *fakeClaims)
	}{
		{"watched user not assigned", func(c *mockClient, _ *fakeClaims) { c.mr.Reviewers = []string{"bob"} }},
		{"already claimed", func(_ *mockClient, cl *fakeClaims) { cl.claimed = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, claims, ai := newClient(), &fakeClaims{}, okAI()
			tc.setup(client, claims)
			uc, _, events := newPipeline(client, ai, func(d *ReviewDeps) { d.Claims = claims })

			if err := uc.Execute(baseEvent()); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if ai.calls != 0 || len(client.notes) != 0 {
				t.Errorf("reviewed anyway: ai calls %d, notes %q", ai.calls, client.notes)
			}
			if got := events.last().Status; got != domain.ReviewStatusSkipped {
				t.Errorf("status = %q, want skipped", got)
			}
		})
	}
}

func TestExecute_ConcurrentWebhooksPostOnlyOneReview(t *testing.T) {
	client := newClient()
	ai := &blockingAI{reached: make(chan struct{}, 2), release: make(chan struct{})}
	uc, _, _ := newPipeline(client, ai, nil)

	errs := make(chan error, 2)
	go func() { errs <- uc.Execute(baseEvent()) }()
	<-ai.reached
	go func() { errs <- uc.Execute(baseEvent()) }()
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("duplicate webhook failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("duplicate webhook reached the AI reviewer instead of being skipped")
	}
	close(ai.release)
	if err := <-errs; err != nil {
		t.Fatalf("first webhook failed: %v", err)
	}
	if len(client.notes) != 1 {
		t.Fatalf("posted %d reviews, want 1", len(client.notes))
	}
}

func TestExecute_ReviewCommandAuthorization(t *testing.T) {
	cases := []struct {
		name, author string
		users        fakeUsers
		wantReview   bool
	}{
		{"MR author", "carol", nil, true},
		{"dashboard user", "dave", fakeUsers{"dave": true}, true},
		{"stranger", "mallory", fakeUsers{"dave": true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, ai := newClient(), okAI()
			uc, _, _ := newPipeline(client, ai, func(d *ReviewDeps) { d.Users = tc.users })
			ev := baseEvent()
			ev.IsReanalyze, ev.CommentAuthor = true, tc.author

			if err := uc.Execute(ev); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if got := ai.calls > 0 && len(client.notes) == 1; got != tc.wantReview {
				t.Errorf("reviewed = %v, want %v", got, tc.wantReview)
			}
		})
	}
}

func TestExecute_JustifiedReanalyze(t *testing.T) {
	cases := []struct {
		name, previousSHA, wantDiff, wantFrom string
	}{
		{"new commits get an incremental diff", "old", "incremental", "old"},
		{"same head keeps the full diff", "new", "+x", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &incrementalClient{mockClient: *newClient()}
			ai := okAI()
			claims := &fakeClaims{claimed: true, history: &domain.ReviewHistory{HeadSHA: tc.previousSHA, Body: "A1: validar setor."}}
			uc, _, _ := newPipeline(client, ai, func(d *ReviewDeps) { d.Claims = claims })
			ev := baseEvent()
			ev.IsReanalyze, ev.CommentAuthor, ev.ReviewJustification = true, "carol", "setor vazio é permitido"

			if err := uc.Execute(ev); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			in := ai.last
			if in.PreviousReview != "A1: validar setor." || in.ReviewJustification != "setor vazio é permitido" {
				t.Errorf("PreviousReview %q, justification %q", in.PreviousReview, in.ReviewJustification)
			}
			if in.Diff != tc.wantDiff || client.from != tc.wantFrom {
				t.Errorf("diff %q (since %q), want %q (since %q)", in.Diff, client.from, tc.wantDiff, tc.wantFrom)
			}
			if tc.wantFrom != "" && client.to != "new" {
				t.Errorf("incremental diff up to %q, want head SHA", client.to)
			}
		})
	}
}

func TestExecute_PostsInlineCommentsAsReturned(t *testing.T) {
	comments := []domain.InlineComment{
		{Path: "main.go", Line: 2, Body: "Adicionar teste."},
		{Path: "main.go", Line: 99, Body: "Outro achado."},
	}
	cases := []struct {
		name      string
		inlineErr error
		wantCalls int
	}{
		{"all posted", nil, 2},
		{"a failure does not stop the rest", errors.New("boom"), 2},
		{"unsupported platform stops after the first", domain.ErrInlineCommentsUnsupported, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newClient()
			client.inlineErr = tc.inlineErr
			uc, claims, _ := newPipeline(client, &mockAI{result: &domain.ReviewResult{Body: "LGTM", InlineComments: comments}}, nil)

			if err := uc.Execute(baseEvent()); err != nil {
				t.Fatalf("inline comments are best-effort, got %v", err)
			}
			if !slices.Equal(client.inline, comments[:tc.wantCalls]) {
				t.Errorf("inline = %+v, want %+v", client.inline, comments[:tc.wantCalls])
			}
			if claims.completedBody != "LGTM" {
				t.Error("review not completed")
			}
		})
	}
}

func TestExecute_FailsBeforeClaimWithoutCommenting(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*mockClient, *fakeClaims, *domain.WebhookEvent)
	}{
		{"unknown client", func(_ *mockClient, _ *fakeClaims, ev *domain.WebhookEvent) { ev.ClientName = "other" }},
		{"MR fetch fails", func(c *mockClient, _ *fakeClaims, _ *domain.WebhookEvent) { c.mrErr = errors.New("unreachable") }},
		{"claim fails", func(_ *mockClient, cl *fakeClaims, _ *domain.WebhookEvent) { cl.claimErr = errors.New("db locked") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, claims, ai, ev := newClient(), &fakeClaims{}, okAI(), baseEvent()
			tc.setup(client, claims, ev)
			uc, _, _ := newPipeline(client, ai, func(d *ReviewDeps) { d.Claims = claims })

			if err := uc.Execute(ev); err == nil {
				t.Fatal("expected error")
			}
			if ai.calls != 0 || len(client.notes) != 0 {
				t.Errorf("ai calls %d, notes %q; want none", ai.calls, client.notes)
			}
		})
	}
}

func TestExecute_RetriesTransientDiffError(t *testing.T) {
	client := newClient()
	client.diffFails = 2
	uc, _, _ := newPipeline(client, okAI(), func(d *ReviewDeps) {
		d.Retry = RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond}
	})

	if err := uc.Execute(baseEvent()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.diffCalls != 3 || !slices.Equal(client.notes, []string{"LGTM"}) {
		t.Errorf("diff calls %d, notes %q", client.diffCalls, client.notes)
	}
}

func TestExecute_RetriesExhaustedPostsNoticeAndReleasesClaim(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(*mockClient, *mockAI)
		wantNotes int
	}{
		{"diff", func(c *mockClient, _ *mockAI) { c.diffFails = 99 }, 1},
		{"AI", func(_ *mockClient, ai *mockAI) { ai.fails = 99 }, 1},
		// two review attempts plus the best-effort notice
		{"posting", func(c *mockClient, _ *mockAI) { c.noteFails = 99 }, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, ai := newClient(), okAI()
			tc.setup(client, ai)
			uc, claims, events := newPipeline(client, ai, func(d *ReviewDeps) {
				d.Retry = RetryConfig{MaxAttempts: 2, BaseDelay: time.Millisecond}
			})

			if err := uc.Execute(baseEvent()); err == nil {
				t.Fatal("expected error")
			}
			if len(client.notes) != tc.wantNotes {
				t.Fatalf("notes = %q, want %d", client.notes, tc.wantNotes)
			}
			if notice := client.notes[len(client.notes)-1]; !strings.Contains(notice, "/review") || !strings.Contains(notice, "2 tentativa") {
				t.Errorf("notice = %q, want attempt count and a /review hint", notice)
			}
			if claims.claimed || claims.completedBody != "" {
				t.Error("claim not released; a manual /review would be blocked")
			}
			if got := events.last().Status; got != domain.ReviewStatusError {
				t.Errorf("status = %q, want error", got)
			}
		})
	}
}

func TestExecute_ProfileResolution(t *testing.T) {
	related := &domain.ReviewProfile{Name: "related", IncludeRelated: true}
	cases := []struct {
		name                     string
		profiles                 fakeProfiles
		requested                string
		wantRelated, wantAgentic bool
	}{
		{"diff-only default", fakeProfiles{}, "", false, false},
		{"resolved by scope", fakeProfiles{p: related}, "", true, false},
		{"resolver error falls back to default", fakeProfiles{p: related, err: errors.New("db")}, "", false, false},
		{"preset overrides scope", fakeProfiles{p: &domain.ReviewProfile{Name: "diff"}}, "full", true, true},
		{"unknown preset is ignored", fakeProfiles{p: related}, "bogus", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ai := okAI()
			uc, _, _ := newPipeline(newClient(), ai, func(d *ReviewDeps) { d.Profiles = tc.profiles })
			ev := baseEvent()
			ev.RequestedProfile = tc.requested

			if err := uc.Execute(ev); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			p := ai.last.Profile
			if p == nil || p.IncludeRelated != tc.wantRelated || p.Agentic != tc.wantAgentic {
				t.Errorf("profile = %+v, want related=%v agentic=%v", p, tc.wantRelated, tc.wantAgentic)
			}
		})
	}
}

func TestExecute_RelatedContextFollowsProfile(t *testing.T) {
	for _, includeRelated := range []bool{true, false} {
		client := &mockRelatedClient{
			mockClient:  *newClient(),
			closes:      []domain.Issue{{IID: 7, Title: "Bug"}},
			issuesByIID: map[int64]*domain.Issue{7: {IID: 7, Title: "Bug"}},
		}
		ai := okAI()
		uc, _, _ := newPipeline(client, ai, func(d *ReviewDeps) {
			d.Profiles = fakeProfiles{p: &domain.ReviewProfile{Name: "p", IncludeRelated: includeRelated}}
		})

		if err := uc.Execute(baseEvent()); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if got := ai.last.Related != nil && len(ai.last.Related.Issues) == 1; got != includeRelated {
			t.Errorf("include_related=%v: related = %+v", includeRelated, ai.last.Related)
		}
	}
}

func TestExecute_AgenticWorkspace(t *testing.T) {
	cases := []struct {
		name          string
		client        domain.PlatformClient
		checkoutErr   error
		wantWorkspace bool
	}{
		{"checked out at head", &locatorClient{*newClient()}, nil, true},
		{"platform without locator", newClient(), nil, false},
		{"checkout failure degrades to diff", &locatorClient{*newClient()}, errors.New("clone failed"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ai, fetcher := okAI(), &fakeFetcher{err: tc.checkoutErr}
			uc, claims, _ := newPipeline(tc.client, ai, func(d *ReviewDeps) {
				d.Profiles = fakeProfiles{p: &domain.ReviewProfile{Name: "deep", Agentic: true}}
				d.Fetcher = fetcher
			})

			if err := uc.Execute(baseEvent()); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if got := ai.last.Workspace != nil; got != tc.wantWorkspace {
				t.Errorf("workspace set = %v, want %v", got, tc.wantWorkspace)
			}
			if tc.wantWorkspace && (fetcher.sha != "new" || !fetcher.cleaned) {
				t.Errorf("checkout sha %q, cleaned %v; want head SHA and cleanup", fetcher.sha, fetcher.cleaned)
			}
			if claims.completedBody != "LGTM" {
				t.Error("review not completed")
			}
		})
	}
}
