package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const (
	anthropicVersion = "2023-06-01"
	// anthropicOAuthBeta is required for subscription (OAuth) tokens.
	anthropicOAuthBeta = "oauth-2025-04-20"
	// anthropicClaudeCodeBeta is part of the Claude Code OAuth request profile.
	anthropicClaudeCodeBeta = "claude-code-20250219"
	// claudeCodeIdentity is the system identity that OAuth subscription tokens
	// are scoped to. Without it as the first system block, the Messages API
	// rejects OAuth-authenticated requests.
	claudeCodeIdentity = "You are Claude Code, Anthropic's official CLI for Claude."
)

// anthropicTransport talks to the native Anthropic Messages API. It supports
// both API-key auth (mode=api) and subscription OAuth auth (mode=account).
type anthropicTransport struct {
	baseURL   string
	model     string
	token     string
	mode      domain.AIConfigMode
	maxTokens int
	http      *http.Client
}

type anthropicSystemBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string                 `json:"model"`
	MaxTokens int                    `json:"max_tokens"`
	System    []anthropicSystemBlock `json:"system"`
	Messages  []anthropicMessage     `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (t *anthropicTransport) review(sysPrompt, userPrompt string) (*domain.ReviewResult, error) {
	system := []anthropicSystemBlock{{Type: "text", Text: sysPrompt}}
	if t.mode == domain.AIModeAccount {
		// OAuth tokens require the Claude Code identity as the first system block.
		system = []anthropicSystemBlock{
			{Type: "text", Text: claudeCodeIdentity},
			{Type: "text", Text: sysPrompt},
		}
	}

	maxTokens := t.maxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	reqBody, err := json.Marshal(anthropicRequest{
		Model:     t.model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  []anthropicMessage{{Role: "user", Content: userPrompt}},
	})
	if err != nil {
		return nil, fmt.Errorf("ai: marshalling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, t.baseURL+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("ai: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", anthropicVersion)
	if t.mode == domain.AIModeAccount {
		req.Header.Set("Authorization", "Bearer "+t.token)
		req.Header.Set("anthropic-beta", anthropicBetaHeader())
	} else {
		req.Header.Set("x-api-key", t.token)
	}

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai: calling API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai: API returned status %d: %s", resp.StatusCode, rb)
	}

	var out anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ai: decoding response: %w", err)
	}

	content := collectAnthropicText(out)
	if content == "" {
		return nil, fmt.Errorf("ai: empty response from API")
	}

	body, comments := parseReviewPayload(content)

	return &domain.ReviewResult{
		Body:           body,
		InlineComments: comments,
		Usage: domain.TokenUsage{
			PromptTokens:     out.Usage.InputTokens,
			CompletionTokens: out.Usage.OutputTokens,
			TotalTokens:      out.Usage.InputTokens + out.Usage.OutputTokens,
		},
	}, nil
}

func anthropicBetaHeader() string {
	return anthropicClaudeCodeBeta + "," + anthropicOAuthBeta
}

func collectAnthropicText(resp anthropicResponse) string {
	var out bytes.Buffer
	for _, block := range resp.Content {
		if block.Type == "text" {
			out.WriteString(block.Text)
		}
	}
	return out.String()
}
