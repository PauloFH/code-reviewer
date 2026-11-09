package ai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// capture is a fake provider: it answers each request with the next canned
// response and records what was sent.
type capture struct {
	mu        sync.Mutex
	responses []string
	paths     []string
	headers   []http.Header
	bodies    []map[string]any
}

func (c *capture) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("request is not JSON: %v", err)
		}
		c.mu.Lock()
		i := len(c.bodies)
		c.paths = append(c.paths, r.URL.Path)
		c.headers = append(c.headers, r.Header.Clone())
		c.bodies = append(c.bodies, body)
		c.mu.Unlock()
		_, _ = w.Write([]byte(c.responses[min(i, len(c.responses)-1)]))
	}))
	t.Cleanup(srv.Close)
	return srv
}

const finalJSON = `{\"review_markdown\":\"revisão concluída sem pendências\",\"inline_comments\":[]}`

// provider fixtures: a tool-call response, a final-answer response and the
// key that carries the token budget in the request.
var providerCases = []struct {
	provider  domain.AIProvider
	path      string
	toolCall  string
	final     string
	maxTokens func(body map[string]any) any
}{
	{
		provider:  domain.ProviderClaude,
		path:      "/v1/messages",
		toolCall:  `{"content":[{"type":"tool_use","id":"tu_1","name":"read_file","input":{"path":"main.go"}}],"stop_reason":"tool_use","usage":{"input_tokens":100,"output_tokens":20}}`,
		final:     `{"content":[{"type":"text","text":"` + finalJSON + `"}],"stop_reason":"end_turn","usage":{"input_tokens":50,"output_tokens":10}}`,
		maxTokens: func(b map[string]any) any { return b["max_tokens"] },
	},
	{
		provider:  domain.ProviderOpenAI,
		path:      "/chat/completions",
		toolCall:  `{"choices":[{"message":{"content":null,"tool_calls":[{"id":"tc_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120}}`,
		final:     `{"choices":[{"message":{"content":"` + finalJSON + `"},"finish_reason":"stop"}],"usage":{"prompt_tokens":50,"completion_tokens":10,"total_tokens":60}}`,
		maxTokens: func(b map[string]any) any { return b["max_tokens"] },
	},
	{
		provider: domain.ProviderGemini,
		path:     "/models/gemini-flash-latest:generateContent",
		toolCall: `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"read_file","args":{"path":"main.go"}}}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":20,"totalTokenCount":120}}`,
		final:    `{"candidates":[{"content":{"role":"model","parts":[{"text":"` + finalJSON + `"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":50,"candidatesTokenCount":10,"totalTokenCount":60}}`,
		maxTokens: func(b map[string]any) any {
			gc, _ := b["generationConfig"].(map[string]any)
			return gc["maxOutputTokens"]
		},
	},
}

func testConfig(p domain.AIProvider, baseURL string) domain.AIConfig {
	return domain.AIConfig{Name: "t", Provider: p, Mode: domain.AIModeAPI, BaseURL: baseURL, APIKey: "k", TimeoutSeconds: 5}
}

func TestSingleShotSendsNoToolsAndParsesAnswer(t *testing.T) {
	for _, pc := range providerCases {
		t.Run(string(pc.provider), func(t *testing.T) {
			c := &capture{responses: []string{pc.final}}
			srv := c.server(t)
			res, err := reviewOnce(newConversation(testConfig(pc.provider, srv.URL), "sys", "user", nil))
			if err != nil {
				t.Fatalf("reviewOnce: %v", err)
			}
			if res.Body != "revisão concluída sem pendências" || res.Usage.TotalTokens != 60 {
				t.Fatalf("result = %+v", res)
			}
			if c.paths[0] != pc.path {
				t.Fatalf("path = %q, want %q", c.paths[0], pc.path)
			}
			if _, ok := c.bodies[0]["tools"]; ok {
				t.Fatalf("single-shot request must not carry a tools field: %v", c.bodies[0])
			}
			h, body := c.headers[0], c.bodies[0]
			switch pc.provider {
			case domain.ProviderOpenAI:
				if h.Get("Authorization") != "Bearer k" || body["temperature"] != temperature {
					t.Fatalf("auth/temperature: %v %v", h, body)
				}
			case domain.ProviderGemini:
				gc, _ := body["generationConfig"].(map[string]any)
				if h.Get("X-goog-api-key") != "k" || gc["temperature"] != temperature {
					t.Fatalf("auth/temperature: %v %v", h, body)
				}
			}
		})
	}
}

func TestAgenticToolThenFinal(t *testing.T) {
	for _, pc := range providerCases {
		t.Run(string(pc.provider), func(t *testing.T) {
			c := &capture{responses: []string{pc.toolCall, pc.final}}
			srv := c.server(t)
			conv := newConversation(testConfig(pc.provider, srv.URL), "sys", "user", readToolSpecs())
			exec := newToolExecutor(&fakeWorkspace{files: map[string]string{"main.go": "package main\n"}}, nil)

			res, err := runAgenticLoop(conv, exec, 12)
			if err != nil {
				t.Fatalf("loop: %v", err)
			}
			if res.Body != "revisão concluída sem pendências" || res.Usage.TotalTokens != 180 {
				t.Fatalf("result = %+v", res)
			}
			if _, ok := c.bodies[0]["tools"]; !ok {
				t.Fatalf("first agentic request must carry tools: %v", c.bodies[0])
			}
			second, _ := json.Marshal(c.bodies[1])
			if !strings.Contains(string(second), "package main") {
				t.Fatalf("second request does not carry the tool result: %s", second)
			}
		})
	}
}

func TestAgenticLastTurnWithdrawsToolsAndRaisesMaxTokens(t *testing.T) {
	cases := []struct {
		provider      domain.AIProvider
		maxTokens     int
		wantTools     any // []any{} = "tools": [], nil = field absent
		wantMaxTokens any // nil = field absent
	}{
		{domain.ProviderClaude, 0, []any{}, float64(lastTurnMinTokens)}, // default 4096 raised
		{domain.ProviderClaude, 16000, []any{}, float64(16000)},
		{domain.ProviderOpenAI, 1000, []any{}, float64(lastTurnMinTokens)},
		{domain.ProviderOpenAI, 0, []any{}, nil}, // no cap stays no cap
		{domain.ProviderGemini, 1000, nil, float64(lastTurnMinTokens)},
		{domain.ProviderGemini, 0, nil, nil},
	}
	for _, tc := range cases {
		var pc = providerCases[0]
		for _, p := range providerCases {
			if p.provider == tc.provider {
				pc = p
			}
		}
		c := &capture{responses: []string{pc.final}}
		srv := c.server(t)
		cfg := testConfig(tc.provider, srv.URL)
		cfg.MaxTokens = tc.maxTokens

		if _, err := runAgenticLoop(newConversation(cfg, "sys", "user", readToolSpecs()), newToolExecutor(&fakeWorkspace{}, nil), 1); err != nil {
			t.Fatalf("%s: loop: %v", tc.provider, err)
		}
		body := c.bodies[0]
		tools, hasTools := body["tools"]
		switch want := tc.wantTools.(type) {
		case nil:
			if hasTools {
				t.Errorf("%s/%d: tools = %v, want absent", tc.provider, tc.maxTokens, tools)
			}
		case []any:
			if got, ok := tools.([]any); !ok || len(got) != len(want) {
				t.Errorf("%s/%d: tools = %v, want []", tc.provider, tc.maxTokens, tools)
			}
		}
		if got := pc.maxTokens(body); got != tc.wantMaxTokens {
			t.Errorf("%s/%d: max tokens = %v, want %v", tc.provider, tc.maxTokens, got, tc.wantMaxTokens)
		}
		raw, _ := json.Marshal(body)
		if !strings.Contains(string(raw), "último turno") {
			t.Errorf("%s: last-turn nudge missing: %s", tc.provider, raw)
		}
	}
}

func TestSingleShotKeepsConfiguredMaxTokens(t *testing.T) {
	for _, pc := range providerCases {
		c := &capture{responses: []string{pc.final}}
		srv := c.server(t)
		cfg := testConfig(pc.provider, srv.URL)
		cfg.MaxTokens = 1000
		if _, err := reviewOnce(newConversation(cfg, "sys", "user", nil)); err != nil {
			t.Fatalf("%s: %v", pc.provider, err)
		}
		if got := pc.maxTokens(c.bodies[0]); got != float64(1000) {
			t.Errorf("%s: max tokens = %v, want 1000 (no last-turn floor in single-shot)", pc.provider, got)
		}
	}
}

func TestGeminiBlockReasonAndFinishReasonInBothModes(t *testing.T) {
	cases := map[string]string{
		`{"candidates":[],"promptFeedback":{"blockReason":"SAFETY"}}`:                          "prompt blocked: SAFETY",
		`{"candidates":[{"content":{"parts":[],"role":"model"},"finishReason":"MAX_TOKENS"}]}`: "finish reason: MAX_TOKENS",
	}
	for resp, want := range cases {
		for _, tools := range [][]toolSpec{nil, readToolSpecs()} {
			c := &capture{responses: []string{resp}}
			srv := c.server(t)
			conv := newConversation(testConfig(domain.ProviderGemini, srv.URL), "sys", "user", tools)
			var err error
			if tools == nil {
				_, err = reviewOnce(conv)
			} else {
				_, err = runAgenticLoop(conv, newToolExecutor(&fakeWorkspace{}, nil), 3)
			}
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("agentic=%v: err = %v, want %q", tools != nil, err, want)
			}
		}
	}
}

func TestEmptyAnswerIsAnError(t *testing.T) {
	empty := map[domain.AIProvider]string{
		domain.ProviderClaude: `{"content":[],"stop_reason":"end_turn"}`,
		domain.ProviderOpenAI: `{"choices":[{"message":{"content":""},"finish_reason":"length"}]}`,
	}
	for p, resp := range empty {
		c := &capture{responses: []string{resp}}
		srv := c.server(t)
		if _, err := reviewOnce(newConversation(testConfig(p, srv.URL), "sys", "user", nil)); err == nil || !strings.Contains(err.Error(), "empty response") {
			t.Errorf("%s: err = %v, want empty response error", p, err)
		}
	}
}

func TestHTTPErrorStatusIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "quota exceeded", http.StatusTooManyRequests)
	}))
	defer srv.Close()
	_, err := reviewOnce(newConversation(testConfig(domain.ProviderOpenAI, srv.URL), "sys", "user", nil))
	if err == nil || !strings.Contains(err.Error(), "status 429") || !strings.Contains(err.Error(), "quota exceeded") {
		t.Fatalf("err = %v", err)
	}
}

func TestChatInvalidToolArgumentsAreFlagged(t *testing.T) {
	bad := `{"choices":[{"message":{"tool_calls":[{"id":"tc_1","type":"function","function":{"name":"read_file","arguments":"{not json"}}]},"finish_reason":"tool_calls"}]}`
	c := &capture{responses: []string{bad}}
	srv := c.server(t)
	st, err := newConversation(testConfig(domain.ProviderOpenAI, srv.URL), "sys", "user", readToolSpecs()).step(nil, "", true)
	if err != nil {
		t.Fatalf("step: %v", err)
	}
	if len(st.Calls) != 1 || st.Calls[0].ArgsErr == nil {
		t.Fatalf("invalid arguments not flagged: %+v", st.Calls)
	}
}

func TestAnthropicAuthModes(t *testing.T) {
	c := &capture{responses: []string{providerCases[0].final}}
	srv := c.server(t)

	api := testConfig(domain.ProviderClaude, srv.URL)
	api.APIKey = "sk-ant-key" //nolint:gosec // G101: test fixture, not a real credential
	account := testConfig(domain.ProviderClaude, srv.URL)
	account.Mode = domain.AIModeAccount
	account.OAuthAccessToken = "oauth-token"

	for _, cfg := range []domain.AIConfig{api, account} {
		if _, err := reviewOnce(newConversation(cfg, "sys", "user", nil)); err != nil {
			t.Fatalf("%s: %v", cfg.Mode, err)
		}
	}

	h := c.headers[0]
	if h.Get("x-api-key") != "sk-ant-key" || h.Get("Authorization") != "" || h.Get("anthropic-version") != anthropicVersion {
		t.Fatalf("api mode headers = %v", h)
	}
	if _, ok := c.bodies[0]["temperature"]; ok {
		t.Fatal("anthropic requests must not send temperature")
	}

	h = c.headers[1]
	if h.Get("x-api-key") != "" || h.Get("Authorization") != "Bearer oauth-token" ||
		h.Get("anthropic-beta") != "claude-code-20250219,oauth-2025-04-20" {
		t.Fatalf("account mode headers = %v", h)
	}
	system, _ := json.Marshal(c.bodies[1]["system"])
	want := `[{"text":"You are Claude Code, Anthropic's official CLI for Claude.","type":"text"},{"text":"sys","type":"text"}]`
	if string(system) != want {
		t.Fatalf("system = %s, want %s", system, want)
	}
	if c.bodies[1]["max_tokens"] != float64(4096) {
		t.Fatalf("max_tokens = %v, want the Claude default 4096", c.bodies[1]["max_tokens"])
	}
}

func TestNewConnectionUsesProviderTableDefaults(t *testing.T) {
	for _, spec := range domain.AIProviders {
		conn := newConnection(domain.AIConfig{Provider: spec.ID})
		if conn.baseURL != spec.DefaultBaseURL || conn.model != spec.DefaultModel {
			t.Errorf("%s: got (%q, %q), want table defaults (%q, %q)", spec.ID, conn.baseURL, conn.model, spec.DefaultBaseURL, spec.DefaultModel)
		}
		if conn.http.Timeout.Seconds() != domain.DefaultAITimeoutSeconds {
			t.Errorf("%s: timeout = %v, want default", spec.ID, conn.http.Timeout)
		}
	}
	conn := newConnection(domain.AIConfig{Provider: domain.ProviderOpenAI, BaseURL: "https://proxy/v1/", Model: "m"})
	if conn.baseURL != "https://proxy/v1" || conn.model != "m" {
		t.Fatalf("explicit values not kept: %+v", conn)
	}
}
