package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestAnthropicTransport_APIKeyMode(t *testing.T) {
	var gotAPIKey, gotAuth, gotVersion, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotPath = req.URL.Path
		gotAPIKey = req.Header.Get("x-api-key")
		gotAuth = req.Header.Get("Authorization")
		gotVersion = req.Header.Get("anthropic-version")
		_, _ = w.Write([]byte(`{
			"content": [{"type":"text","text":"review via anthropic"}],
			"usage": {"input_tokens": 12, "output_tokens": 34}
		}`))
	}))
	defer server.Close()

	tr := &anthropicTransport{
		baseURL: server.URL,
		model:   "claude-sonnet-4-5",
		token:   "sk-ant-key",
		mode:    domain.AIModeAPI,
		http:    &http.Client{Timeout: time.Second},
	}

	result, err := tr.review("system", "user prompt")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path = %q, want /v1/messages", gotPath)
	}
	if gotAPIKey != "sk-ant-key" {
		t.Fatalf("x-api-key = %q, want sk-ant-key", gotAPIKey)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization should be empty in api mode, got %q", gotAuth)
	}
	if gotVersion == "" {
		t.Fatal("anthropic-version header missing")
	}
	if result.Body != "review via anthropic" {
		t.Fatalf("body = %q", result.Body)
	}
	if result.Usage.PromptTokens != 12 || result.Usage.CompletionTokens != 34 || result.Usage.TotalTokens != 46 {
		t.Fatalf("unexpected usage: %+v", result.Usage)
	}
}

func TestAnthropicTransport_AccountModeSendsClaudeCodeOAuthProfile(t *testing.T) {
	var gotAPIKey, gotAuth, gotBeta string
	var gotReq anthropicRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotAPIKey = req.Header.Get("x-api-key")
		gotAuth = req.Header.Get("Authorization")
		gotBeta = req.Header.Get("anthropic-beta")
		if err := json.NewDecoder(req.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{
			"content": [{"type":"text","text":"review via account"}],
			"usage": {"input_tokens": 1, "output_tokens": 2}
		}`))
	}))
	defer server.Close()

	tr := &anthropicTransport{
		baseURL: server.URL,
		model:   "claude-sonnet-4-5",
		token:   "oauth-token",
		mode:    domain.AIModeAccount,
		http:    &http.Client{Timeout: time.Second},
	}

	result, err := tr.review("system", "user prompt")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if gotAPIKey != "" {
		t.Fatalf("x-api-key should be empty in account mode, got %q", gotAPIKey)
	}
	if gotAuth != "Bearer oauth-token" {
		t.Fatalf("Authorization = %q, want Bearer oauth-token", gotAuth)
	}
	for _, beta := range []string{anthropicClaudeCodeBeta, anthropicOAuthBeta} {
		if !strings.Contains(gotBeta, beta) {
			t.Fatalf("anthropic-beta = %q, want %q", gotBeta, beta)
		}
	}
	if len(gotReq.System) < 2 || gotReq.System[0].Text != claudeCodeIdentity || gotReq.System[1].Text != "system" {
		t.Fatalf("unexpected system blocks: %+v", gotReq.System)
	}
	if result.Body != "review via account" {
		t.Fatalf("body = %q", result.Body)
	}
}
