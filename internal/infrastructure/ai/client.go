package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// Client implements domain.AIReviewer using a chat/completions-compatible HTTP API.
type Client struct {
	baseURL    string
	token      string
	model      string
	http       *http.Client
	promptRepo domain.PromptRepository
}

func New(baseURL, token, model string, timeout time.Duration, promptRepo domain.PromptRepository) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		model:      model,
		http:       &http.Client{Timeout: timeout},
		promptRepo: promptRepo,
	}
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
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

func (c *Client) ReviewDiff(input domain.ReviewInput) (*domain.ReviewResult, error) {
	sysPrompt := systemPrompt
	outputTpl := ""

	if c.promptRepo != nil {
		resolved, err := c.promptRepo.ResolvePrompt(input.ClientName, input.ProjectRef)
		if err == nil && resolved != nil {
			if resolved.SystemPrompt != "" {
				sysPrompt = htmlToPlainText(resolved.SystemPrompt)
			}
			if resolved.OutputTemplate != "" {
				outputTpl = htmlToPlainText(resolved.OutputTemplate)
			}
		}
	}

	prompt := buildPrompt(input, outputTpl)

	body, err := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("ai: marshalling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ai: building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
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

	return &domain.ReviewResult{
		Body: out.Choices[0].Message.Content,
		Usage: domain.TokenUsage{
			PromptTokens:     out.Usage.PromptTokens,
			CompletionTokens: out.Usage.CompletionTokens,
			TotalTokens:      out.Usage.TotalTokens,
		},
	}, nil
}
