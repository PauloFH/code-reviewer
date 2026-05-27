package ai

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// mockConfigRepo is a minimal domain.AIConfigRepository for resolver tests.
type mockConfigRepo struct {
	resolve    *domain.AIConfig
	resolveErr error

	updateCalled  bool
	updatedID     int64
	updatedAccess string
}

func (m *mockConfigRepo) ResolveConfig(string, string) (*domain.AIConfig, error) {
	return m.resolve, m.resolveErr
}
func (m *mockConfigRepo) UpdateOAuthTokens(id int64, access, _ string, _ time.Time) error {
	m.updateCalled = true
	m.updatedID = id
	m.updatedAccess = access
	return nil
}
func (m *mockConfigRepo) ListConfigs() ([]domain.AIConfig, error)         { return nil, nil }
func (m *mockConfigRepo) GetConfig(int64) (*domain.AIConfig, error)       { return nil, nil }
func (m *mockConfigRepo) CreateConfig(domain.AIConfig) (*domain.AIConfig, error) {
	return nil, nil
}
func (m *mockConfigRepo) UpdateConfig(int64, domain.AIConfig) (*domain.AIConfig, error) {
	return nil, nil
}
func (m *mockConfigRepo) DeleteConfig(int64) error                            { return nil }
func (m *mockConfigRepo) SetConfigAssignment(domain.AIConfigAssignment) error { return nil }
func (m *mockConfigRepo) RemoveConfigAssignment(string, string, string) error { return nil }
func (m *mockConfigRepo) ListConfigAssignments() ([]domain.AIConfigAssignment, error) {
	return nil, nil
}

type stubRefresher struct {
	called bool
	tokens *OAuthTokens
}

func (s *stubRefresher) Refresh(string) (*OAuthTokens, error) {
	s.called = true
	return s.tokens, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func sampleInput() domain.ReviewInput {
	return domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "MR"},
		Diff:         "+change",
		ClientName:   "acme",
		ProjectRef:   "acme/repo",
	}
}

func TestResolvingReviewer_FallsBackToEnvClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"env review"}}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`))
	}))
	defer server.Close()

	repo := &mockConfigRepo{resolve: nil} // nothing assigned
	envClient := New("chat_completions", server.URL, "env-token", "gpt-4o", time.Second, nil)
	r := NewResolvingReviewer(repo, nil, &stubRefresher{}, envClient, 7.0, 9.0, discardLogger())

	result, err := r.ReviewDiff(sampleInput())
	if err != nil {
		t.Fatalf("ReviewDiff: %v", err)
	}
	if result.Body != "env review" {
		t.Fatalf("body = %q, want env review", result.Body)
	}
	if result.PricePerMInputTokens != 7.0 || result.PricePerMOutputTokens != 9.0 {
		t.Fatalf("env prices not applied: %+v", result)
	}
}

func TestResolvingReviewer_UsesResolvedGeminiConfig(t *testing.T) {
	var gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotKey = req.Header.Get("X-goog-api-key")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"gemini review"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":5,"totalTokenCount":9}}`))
	}))
	defer server.Close()

	repo := &mockConfigRepo{resolve: &domain.AIConfig{
		ID:                    1,
		Name:                  "gem",
		Provider:              domain.ProviderGemini,
		Mode:                  domain.AIModeAPI,
		BaseURL:               server.URL,
		Model:                 "gemini-flash-latest",
		APIKey:                "cfg-key",
		PricePerMInputTokens:  1.5,
		PricePerMOutputTokens: 2.5,
		TimeoutSeconds:        5,
	}}
	r := NewResolvingReviewer(repo, nil, &stubRefresher{}, nil, 0, 0, discardLogger())

	result, err := r.ReviewDiff(sampleInput())
	if err != nil {
		t.Fatalf("ReviewDiff: %v", err)
	}
	if gotKey != "cfg-key" {
		t.Fatalf("X-goog-api-key = %q, want cfg-key", gotKey)
	}
	if result.Body != "gemini review" {
		t.Fatalf("body = %q", result.Body)
	}
	if result.PricePerMInputTokens != 1.5 || result.PricePerMOutputTokens != 2.5 {
		t.Fatalf("config prices not applied: %+v", result)
	}
}

func TestResolvingReviewer_RefreshesExpiredAccountToken(t *testing.T) {
	var gotAuth, gotBeta string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotAuth = req.Header.Get("Authorization")
		gotBeta = req.Header.Get("anthropic-beta")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"claude review"}],"usage":{"input_tokens":10,"output_tokens":20}}`))
	}))
	defer server.Close()

	repo := &mockConfigRepo{resolve: &domain.AIConfig{
		ID:                1,
		Name:              "claude-acct",
		Provider:          domain.ProviderClaude,
		Mode:              domain.AIModeAccount,
		BaseURL:           server.URL,
		Model:             "claude-sonnet-4-5",
		OAuthAccessToken:  "old-token",
		OAuthRefreshToken: "refresh-token",
		OAuthExpiresAt:    time.Now().Add(-time.Hour), // expired
		TimeoutSeconds:    5,
	}}
	refresher := &stubRefresher{tokens: &OAuthTokens{
		AccessToken:  "fresh-token",
		RefreshToken: "new-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}}
	r := NewResolvingReviewer(repo, nil, refresher, nil, 0, 0, discardLogger())

	result, err := r.ReviewDiff(sampleInput())
	if err != nil {
		t.Fatalf("ReviewDiff: %v", err)
	}
	if !refresher.called {
		t.Fatal("expected token refresh to be triggered")
	}
	if !repo.updateCalled || repo.updatedID != 1 || repo.updatedAccess != "fresh-token" {
		t.Fatalf("refreshed tokens not persisted: %+v", repo)
	}
	if gotAuth != "Bearer fresh-token" {
		t.Fatalf("Authorization = %q, want Bearer fresh-token", gotAuth)
	}
	if !strings.Contains(gotBeta, "oauth") {
		t.Fatalf("anthropic-beta header = %q, want oauth beta", gotBeta)
	}
	if result.Body != "claude review" {
		t.Fatalf("body = %q", result.Body)
	}
}
