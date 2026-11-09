package ai

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type stubConfigs struct {
	resolve    *domain.AIConfig
	resolveErr error
	byID       map[int64]*domain.AIConfig

	mu            sync.Mutex
	updatedID     int64
	updatedAccess string
}

func (s *stubConfigs) Resolve(string, string) (*domain.AIConfig, error) {
	return s.resolve, s.resolveErr
}

func (s *stubConfigs) Get(id int64) (*domain.AIConfig, error) {
	if c, ok := s.byID[id]; ok {
		return c, nil
	}
	return nil, domain.ErrNotFound
}

func (s *stubConfigs) UpdateOAuthTokens(id int64, access, _ string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updatedID, s.updatedAccess = id, access
	return nil
}

type stubPrompts struct {
	prompt *domain.Prompt
	err    error
}

func (s stubPrompts) Resolve(string, string) (*domain.Prompt, error) { return s.prompt, s.err }

type stubRefresher struct{ tokens *domain.OAuthTokens }

func (s stubRefresher) Refresh(string) (*domain.OAuthTokens, error) { return s.tokens, nil }

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func reviewInput(diff string) domain.ReviewInput {
	return domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "MR"},
		Diff:         diff,
		ClientName:   "acme",
		ProjectRef:   "acme/repo",
	}
}

const chatFinal = `{"choices":[{"message":{"content":"%s"}}],"usage":{"prompt_tokens":1000000,"completion_tokens":2000000,"total_tokens":3000000}}`

func TestReviewer_FallsBackToEnvConfigOnlyWhenNothingIsAssigned(t *testing.T) {
	c := &capture{responses: []string{fmt.Sprintf(chatFinal, "env review")}}
	srv := c.server(t)
	fallback := testConfig(domain.ProviderOpenAI, srv.URL)
	fallback.APIKey = "env-token"

	res, err := NewReviewer(&stubConfigs{}, nil, stubRefresher{}, fallback, nil, discardLogger()).ReviewDiff(reviewInput("+change"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Body != "env review" || c.headers[0].Get("Authorization") != "Bearer env-token" {
		t.Fatalf("env fallback not used: %+v %v", res, c.headers[0])
	}

	// A store failure must not send the code to the env provider instead.
	_, err = NewReviewer(&stubConfigs{resolveErr: errors.New("db down")}, nil, stubRefresher{}, fallback, nil, discardLogger()).ReviewDiff(reviewInput("+change"))
	if err == nil || len(c.headers) != 1 {
		t.Fatalf("store error: err=%v, requests=%d; want error and no extra request", err, len(c.headers))
	}
}

func TestReviewer_UsesResolvedConfigAndCostsWithEnvPriceFallback(t *testing.T) {
	c := &capture{responses: []string{fmt.Sprintf(chatFinal, "cfg review")}}
	srv := c.server(t)
	cfg := testConfig(domain.ProviderDeepSeek, srv.URL)
	cfg.APIKey = "cfg-key"
	cfg.PricePerMInputTokens = 1.5 // output price 0 = unknown → env price
	fallback := domain.AIConfig{PricePerMInputTokens: 3, PricePerMOutputTokens: 15}

	res, err := NewReviewer(&stubConfigs{resolve: &cfg}, nil, stubRefresher{}, fallback, nil, discardLogger()).ReviewDiff(reviewInput("+change"))
	if err != nil {
		t.Fatalf("ReviewDiff: %v", err)
	}
	if c.headers[0].Get("Authorization") != "Bearer cfg-key" || res.Body != "cfg review" {
		t.Fatalf("resolved config not used: %+v", res)
	}
	// 1M prompt tokens at 1.5 + 2M completion tokens at the env 15.
	if want := 1.5 + 2*15.0; math.Abs(res.CostUSD-want) > 1e-9 {
		t.Fatalf("CostUSD = %v, want %v", res.CostUSD, want)
	}
}

func TestReviewer_RefreshesExpiredAccountToken(t *testing.T) {
	c := &capture{responses: []string{providerCases[0].final}}
	srv := c.server(t)
	cfg := testConfig(domain.ProviderClaude, srv.URL)
	cfg.ID = 1
	cfg.Mode = domain.AIModeAccount
	cfg.OAuthAccessToken = "old-token"
	cfg.OAuthRefreshToken = "refresh-token"
	cfg.OAuthExpiresAt = time.Now().Add(-time.Hour)
	configs := &stubConfigs{resolve: &cfg}
	refresher := stubRefresher{tokens: &domain.OAuthTokens{AccessToken: "fresh-token", ExpiresAt: time.Now().Add(time.Hour)}}

	if _, err := NewReviewer(configs, nil, refresher, domain.AIConfig{}, nil, discardLogger()).ReviewDiff(reviewInput("+change")); err != nil {
		t.Fatalf("ReviewDiff: %v", err)
	}
	if configs.updatedID != 1 || configs.updatedAccess != "fresh-token" {
		t.Fatalf("refreshed tokens not persisted: id=%d access=%q", configs.updatedID, configs.updatedAccess)
	}
	if got := c.headers[0].Get("Authorization"); got != "Bearer fresh-token" {
		t.Fatalf("Authorization = %q", got)
	}
	if cfg.OAuthAccessToken != "old-token" {
		t.Fatal("the resolved config must not be mutated in place")
	}
}

func TestReviewer_ValidatesInlineCommentsAgainstReviewedDiff(t *testing.T) {
	answer := `{\"review_markdown\":\"ok\",\"inline_comments\":[` +
		`{\"path\":\"b/main.go\",\"line\":2,\"body\":\"bom\"},` +
		`{\"path\":\"main.go\",\"line\":1,\"body\":\"linha de contexto\"},` +
		`{\"path\":\"go.sum\",\"line\":1,\"body\":\"arquivo filtrado\"}]}`
	c := &capture{responses: []string{fmt.Sprintf(chatFinal, answer)}}
	srv := c.server(t)
	diff := "--- a/main.go\n+++ b/main.go\n@@ -1 +1,2 @@\n package main\n+func run() {}\n" +
		"--- a/go.sum\n+++ b/go.sum\n@@ -0,0 +1 @@\n+x v1 h1:abc\n"

	res, err := NewReviewer(&stubConfigs{}, nil, stubRefresher{}, testConfig(domain.ProviderOpenAI, srv.URL), nil, discardLogger()).ReviewDiff(reviewInput(diff))
	if err != nil {
		t.Fatalf("ReviewDiff: %v", err)
	}
	want := domain.InlineComment{Path: "main.go", Line: 2, Body: "bom"}
	if len(res.InlineComments) != 1 || res.InlineComments[0] != want {
		t.Fatalf("InlineComments = %+v, want [%+v]", res.InlineComments, want)
	}
}

func TestReviewer_PromptStore(t *testing.T) {
	cases := map[string]struct {
		store   stubPrompts
		wantSys string
		wantLog bool
	}{
		"assigned":  {stubPrompts{prompt: &domain.Prompt{SystemPrompt: "<p>Seja breve.</p>"}}, "Seja breve.", false},
		"none":      {stubPrompts{}, systemPrompt, false},
		"not found": {stubPrompts{err: domain.ErrNotFound}, systemPrompt, false},
		"error":     {stubPrompts{err: errors.New("db down")}, systemPrompt, true},
	}
	for name, tc := range cases {
		c := &capture{responses: []string{fmt.Sprintf(chatFinal, "ok")}}
		srv := c.server(t)
		var logs bytes.Buffer
		log := slog.New(slog.NewTextHandler(&logs, nil))
		if _, err := NewReviewer(&stubConfigs{}, tc.store, stubRefresher{}, testConfig(domain.ProviderOpenAI, srv.URL), nil, log).ReviewDiff(reviewInput("+x")); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		msgs := c.bodies[0]["messages"].([]any)
		if got := msgs[0].(map[string]any)["content"]; got != tc.wantSys {
			t.Errorf("%s: system prompt = %q, want %q", name, got, tc.wantSys)
		}
		if got := strings.Contains(logs.String(), "db down"); got != tc.wantLog {
			t.Errorf("%s: logged error = %v, want %v (%s)", name, got, tc.wantLog, logs.String())
		}
	}
}

func TestReviewer_AgenticOnlyWithWorkspace(t *testing.T) {
	c := &capture{responses: []string{fmt.Sprintf(chatFinal, "single shot sem workspace")}}
	srv := c.server(t)
	input := reviewInput("+x")
	input.Profile = &domain.ReviewProfile{Agentic: true}

	res, err := NewReviewer(&stubConfigs{}, nil, stubRefresher{}, testConfig(domain.ProviderOpenAI, srv.URL), nil, discardLogger()).ReviewDiff(input)
	if err != nil {
		t.Fatalf("ReviewDiff: %v", err)
	}
	if _, ok := c.bodies[0]["tools"]; ok || res.Body != "single shot sem workspace" {
		t.Fatalf("expected a single-shot request without tools: %v", c.bodies[0])
	}
}

func TestReviewer_TestConfig(t *testing.T) {
	c := &capture{responses: []string{fmt.Sprintf(chatFinal, "teste ok")}}
	srv := c.server(t)
	cfg := testConfig(domain.ProviderOpenAI, srv.URL)
	r := NewReviewer(&stubConfigs{byID: map[int64]*domain.AIConfig{7: &cfg}}, nil, stubRefresher{}, domain.AIConfig{}, nil, discardLogger())

	res, err := r.TestConfig(7)
	if err != nil || res.Body != "teste ok" {
		t.Fatalf("TestConfig(7) = %+v, %v", res, err)
	}
	if _, err := r.TestConfig(8); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("TestConfig(8) err = %v, want ErrNotFound", err)
	}
}

func TestReviewer_ConcurrentReviewsAreRaceFree(t *testing.T) {
	c := &capture{responses: []string{fmt.Sprintf(chatFinal, "ok")}}
	srv := c.server(t)
	sb := &fakeSandbox{}
	r := NewReviewer(&stubConfigs{}, nil, stubRefresher{}, testConfig(domain.ProviderOpenAI, srv.URL), sb, discardLogger())

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(agentic bool) {
			defer wg.Done()
			input := reviewInput("+x")
			if agentic {
				input.Workspace = &fakeWorkspace{}
				input.Profile = &domain.ReviewProfile{Agentic: true, Execution: true}
			}
			if _, err := r.ReviewDiff(input); err != nil {
				t.Errorf("ReviewDiff: %v", err)
			}
		}(i%2 == 0)
	}
	wg.Wait()
}
