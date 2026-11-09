package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const (
	temperature = 0.2
	// lastTurnMinTokens is the floor for the agentic loop's forced last turn
	// (tools disabled): the full JSON payload needs more headroom than the
	// usual cap and there is no further turn to retry a cut-off answer.
	lastTurnMinTokens = 8192
)

// conversation is one provider-specific chat with the model. step sends the
// previous tool results (nil on the first call) plus an optional nudge and
// returns the model's next move. A conversation built without tools is a
// single-shot review: requests carry no tools field. With tools, allowTools
// false is the forced last turn: tools are withdrawn so the model must answer.
type conversation interface {
	step(results []toolResult, nudge string, allowTools bool) (agenticStep, error)
}

// connection is the immutable per-review view of an AIConfig.
type connection struct {
	baseURL   string
	model     string
	token     string
	mode      domain.AIConfigMode
	maxTokens int
	http      *http.Client
}

func newConnection(cfg domain.AIConfig) connection {
	token := cfg.APIKey
	if cfg.Mode == domain.AIModeAccount {
		token = cfg.OAuthAccessToken
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = domain.DefaultAITimeoutSeconds
	}
	return connection{
		baseURL:   strings.TrimRight(cfg.BaseURLOrDefault(), "/"),
		model:     cfg.ModelOrDefault(),
		token:     token,
		mode:      cfg.Mode,
		maxTokens: cfg.MaxTokensOrDefault(),
		http:      &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}
}

// newConversation picks the wire protocol for the provider; openai, deepseek and
// any other OpenAI-compatible vendor speak chat completions.
func newConversation(cfg domain.AIConfig, sysPrompt, userPrompt string, tools []toolSpec) conversation {
	conn := newConnection(cfg)
	switch cfg.Provider {
	case domain.ProviderClaude:
		return newAnthropicConversation(conn, sysPrompt, userPrompt, tools)
	case domain.ProviderGemini:
		return newGeminiConversation(conn, sysPrompt, userPrompt, tools)
	default:
		return newChatConversation(conn, sysPrompt, userPrompt, tools)
	}
}

// reviewOnce runs a single-shot review: one step, no tools.
func reviewOnce(conv conversation) (*domain.ReviewResult, error) {
	st, err := conv.step(nil, "", false)
	if err != nil {
		return nil, err
	}
	body, comments := parseReviewPayload(st.Final)
	return &domain.ReviewResult{Body: body, InlineComments: comments, Usage: st.Usage}, nil
}

// lastTurnMaxTokens raises an explicit cap below lastTurnMinTokens; 0 (no cap) stays 0.
func lastTurnMaxTokens(maxTokens int) int {
	if maxTokens > 0 && maxTokens < lastTurnMinTokens {
		return lastTurnMinTokens
	}
	return maxTokens
}

func emptyResponseError(reason string) error {
	if reason != "" {
		return fmt.Errorf("ai: empty response from API (finish reason: %s)", reason)
	}
	return errors.New("ai: empty response from API")
}

func postJSON(client *http.Client, url string, headers map[string]string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("ai: marshalling request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ai: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ai: calling API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ai: API returned status %d: %s", resp.StatusCode, rb)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("ai: decoding response: %w", err)
	}
	return nil
}
