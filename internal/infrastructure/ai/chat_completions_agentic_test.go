package ai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestChatCompletionsAgenticToolThenFinal(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		call++
		if call == 1 {
			var req map[string]any
			_ = json.Unmarshal(body, &req)
			if _, ok := req["tools"]; !ok {
				t.Errorf("first request missing tools")
			}
			_, _ = w.Write([]byte(`{
				"choices":[{"message":{"content":null,"tool_calls":[
					{"id":"tc_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}
				]},"finish_reason":"tool_calls"}],
				"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120}
			}`))
			return
		}
		if !strings.Contains(string(body), `"role":"tool"`) || !strings.Contains(string(body), "tc_1") {
			t.Errorf("second request missing tool result for tc_1: %s", body)
		}
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"{\"review_markdown\":\"ok chat, revisão concluída sem pendências\",\"inline_comments\":[]}"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":50,"completion_tokens":10,"total_tokens":60}
		}`))
	}))
	defer srv.Close()

	c := &Client{
		transportKind: providerChatCompletions,
		mode:          domain.AIModeAPI,
		baseURL:       srv.URL,
		token:         "k",
		model:         "gpt-4o",
		http:          &http.Client{Timeout: 10 * time.Second},
	}
	driver, err := c.agenticDriver("sys", "user prompt", readToolSpecs())
	if err != nil {
		t.Fatalf("agenticDriver: %v", err)
	}
	exec := newToolExecutor(&fakeWorkspace{files: map[string]string{"main.go": "package main\n"}}, nil)

	res, err := runAgenticLoop(driver, exec, 12)
	if err != nil {
		t.Fatalf("loop: %v", err)
	}
	if res.Body != "ok chat, revisão concluída sem pendências" {
		t.Fatalf("body = %q", res.Body)
	}
	if res.Usage.TotalTokens != 180 {
		t.Fatalf("usage = %d, want 180", res.Usage.TotalTokens)
	}
}

func TestChatCompletionsAgenticDriverLastTurnDisablesTools(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if tools, ok := req["tools"].([]any); !ok || len(tools) != 0 {
			t.Errorf("expected empty tools on the forced last turn, got: %s", body)
		}
		if !strings.Contains(string(body), "último turno") {
			t.Errorf("expected the last-turn nudge in the request: %s", body)
		}
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"{\"review_markdown\":\"revisão forçada no último turno\",\"inline_comments\":[]}"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
		}`))
	}))
	defer srv.Close()

	c := &Client{
		transportKind: providerChatCompletions,
		mode:          domain.AIModeAPI,
		baseURL:       srv.URL,
		token:         "k",
		model:         "gpt-4o",
		http:          &http.Client{Timeout: 10 * time.Second},
	}
	driver, err := c.agenticDriver("sys", "user prompt", readToolSpecs())
	if err != nil {
		t.Fatalf("agenticDriver: %v", err)
	}
	exec := newToolExecutor(&fakeWorkspace{}, nil)

	res, err := runAgenticLoop(driver, exec, 1)
	if err != nil {
		t.Fatalf("loop: %v", err)
	}
	if res.Body != "revisão forçada no último turno" {
		t.Fatalf("body = %q", res.Body)
	}
}
