package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const (
	providerGemini          = "gemini"
	providerChatCompletions = "chat_completions"
)

// Client implements domain.AIReviewer using configurable AI providers.
type Client struct {
	provider   string
	baseURL    string
	token      string
	model      string
	http       *http.Client
	promptRepo domain.PromptRepository
}

func New(provider, baseURL, token, model string, timeout time.Duration, promptRepo domain.PromptRepository) *Client {
	return &Client{
		provider:   normalizeProvider(provider),
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		model:      model,
		http:       &http.Client{Timeout: timeout},
		promptRepo: promptRepo,
	}
}

type geminiRequest struct {
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	Contents          []geminiContent        `json:"contents"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig"`
}

type geminiGenerationConfig struct {
	Temperature float64 `json:"temperature"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content      geminiContent `json:"content"`
		FinishReason string        `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
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

type reviewPayload struct {
	ReviewMarkdown string                 `json:"review_markdown"`
	InlineComments []domain.InlineComment `json:"inline_comments"`
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

	switch c.provider {
	case providerGemini:
		return c.reviewDiffGemini(sysPrompt, prompt)
	case providerChatCompletions:
		return c.reviewDiffChatCompletions(sysPrompt, prompt)
	default:
		return nil, fmt.Errorf("ai: unsupported provider %q", c.provider)
	}
}

func (c *Client) reviewDiffGemini(sysPrompt, prompt string) (*domain.ReviewResult, error) {
	reqBody, err := json.Marshal(geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: sysPrompt}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{Temperature: 0.2},
	})
	if err != nil {
		return nil, fmt.Errorf("ai: marshalling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.generateContentURL(), bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("ai: building request: %w", err)
	}
	req.Header.Set("X-goog-api-key", c.token)
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

	var out geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ai: decoding response: %w", err)
	}

	if len(out.Candidates) == 0 {
		if out.PromptFeedback.BlockReason != "" {
			return nil, fmt.Errorf("ai: Gemini returned no candidates: prompt blocked: %s", out.PromptFeedback.BlockReason)
		}
		return nil, fmt.Errorf("ai: empty response from API")
	}
	content := collectText(out.Candidates[0].Content.Parts)
	if content == "" {
		if out.Candidates[0].FinishReason != "" {
			return nil, fmt.Errorf("ai: empty response from API (finish reason: %s)", out.Candidates[0].FinishReason)
		}
		return nil, fmt.Errorf("ai: empty response from API")
	}

	body, comments := parseReviewPayload(content)

	return &domain.ReviewResult{
		Body:           body,
		InlineComments: comments,
		Usage: domain.TokenUsage{
			PromptTokens:     out.UsageMetadata.PromptTokenCount,
			CompletionTokens: out.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      out.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

func (c *Client) reviewDiffChatCompletions(sysPrompt, prompt string) (*domain.ReviewResult, error) {
	reqBody, err := json.Marshal(chatRequest{
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

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(reqBody))
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

func (c *Client) generateContentURL() string {
	model := strings.TrimPrefix(c.model, "models/")
	return c.baseURL + "/models/" + url.PathEscape(model) + ":generateContent"
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(provider) {
	case "", providerChatCompletions, "chat-completions", "openai", "claude":
		return providerChatCompletions
	case providerGemini:
		return providerGemini
	default:
		return provider
	}
}

func collectText(parts []geminiPart) string {
	var out strings.Builder
	for _, part := range parts {
		out.WriteString(part.Text)
	}
	return strings.TrimSpace(out.String())
}

func parseReviewPayload(content string) (string, []domain.InlineComment) {
	trimmed := strings.TrimSpace(content)
	jsonText := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(trimmed, "```"), "```json"))
	jsonText = strings.TrimSpace(strings.TrimPrefix(jsonText, "```"))

	var payload reviewPayload
	if err := json.Unmarshal([]byte(jsonText), &payload); err != nil {
		return trimmed, nil
	}

	body := strings.TrimSpace(payload.ReviewMarkdown)
	if body == "" {
		body = trimmed
	}

	comments := make([]domain.InlineComment, 0, len(payload.InlineComments))
	for _, comment := range payload.InlineComments {
		comment.Path = strings.TrimSpace(comment.Path)
		comment.Body = strings.TrimSpace(comment.Body)
		if comment.Path == "" || comment.Body == "" || comment.Line <= 0 {
			continue
		}
		comments = append(comments, comment)
		if len(comments) >= maxInlineComments {
			break
		}
	}

	return body, comments
}
