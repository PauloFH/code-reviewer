package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type chatCompletionsTransport struct {
	baseURL   string
	token     string
	model     string
	maxTokens int
	http      *http.Client
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (t *chatCompletionsTransport) review(sysPrompt, userPrompt string) (*domain.ReviewResult, error) {
	reqBody, err := json.Marshal(chatRequest{
		Model: t.model,
		Messages: []message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
		MaxTokens:   t.maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("ai: marshalling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, t.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("ai: building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai: calling API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai: API returned status %d: %s", resp.StatusCode, rb)
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ai: decoding response: %w", err)
	}

	if len(out.Choices) == 0 || out.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("ai: empty response from API")
	}

	body, comments := parseReviewPayload(out.Choices[0].Message.Content)

	return &domain.ReviewResult{
		Body:           body,
		InlineComments: comments,
		Usage: domain.TokenUsage{
			PromptTokens:     out.Usage.PromptTokens,
			CompletionTokens: out.Usage.CompletionTokens,
			TotalTokens:      out.Usage.TotalTokens,
		},
	}, nil
}
