package ai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestGeminiAgenticToolThenFinal(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		call++
		if call == 1 {
			if !strings.Contains(string(body), "functionDeclarations") {
				t.Errorf("first request missing functionDeclarations: %s", body)
			}
			_, _ = w.Write([]byte(`{
				"candidates":[{"content":{"role":"model","parts":[
					{"functionCall":{"name":"read_file","args":{"path":"main.go"}}}
				]},"finishReason":"STOP"}],
				"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":20,"totalTokenCount":120}
			}`))
			return
		}
		if !strings.Contains(string(body), "functionResponse") || !strings.Contains(string(body), "read_file") {
			t.Errorf("second request missing functionResponse: %s", body)
		}
		_, _ = w.Write([]byte(`{
			"candidates":[{"content":{"role":"model","parts":[{"text":"{\"review_markdown\":\"ok gemini\",\"inline_comments\":[]}"}]},"finishReason":"STOP"}],
			"usageMetadata":{"promptTokenCount":50,"candidatesTokenCount":10,"totalTokenCount":60}
		}`))
	}))
	defer srv.Close()

	c := &Client{
		transportKind: providerGemini,
		mode:          domain.AIModeAPI,
		baseURL:       srv.URL,
		token:         "k",
		model:         "gemini-flash-latest",
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
	if res.Body != "ok gemini" {
		t.Fatalf("body = %q", res.Body)
	}
	if res.Usage.TotalTokens != 180 {
		t.Fatalf("usage = %d, want 180", res.Usage.TotalTokens)
	}
}
