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

func TestAnthropicAgenticDriverToolThenFinal(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)

		call++
		if call == 1 {
			// must include tools
			if _, ok := req["tools"]; !ok {
				t.Errorf("first request missing tools")
			}
			_, _ = w.Write([]byte(`{
				"content":[
					{"type":"text","text":"vou olhar o arquivo"},
					{"type":"tool_use","id":"tu_1","name":"read_file","input":{"path":"main.go"}}
				],
				"stop_reason":"tool_use",
				"usage":{"input_tokens":100,"output_tokens":20}
			}`))
			return
		}
		// second request must carry the tool_result for tu_1
		if !strings.Contains(string(body), "tool_result") || !strings.Contains(string(body), "tu_1") {
			t.Errorf("second request missing tool_result for tu_1: %s", body)
		}
		_, _ = w.Write([]byte(`{
			"content":[{"type":"text","text":"{\"review_markdown\":\"ok agentic\",\"inline_comments\":[]}"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":50,"output_tokens":10}
		}`))
	}))
	defer srv.Close()

	c := &Client{
		transportKind: providerAnthropic,
		mode:          domain.AIModeAPI,
		baseURL:       srv.URL,
		token:         "k",
		model:         "claude-sonnet-4-5",
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
	if res.Body != "ok agentic" {
		t.Fatalf("body = %q", res.Body)
	}
	if res.Usage.TotalTokens != 180 { // 120 + 60
		t.Fatalf("usage = %d, want 180", res.Usage.TotalTokens)
	}
}
