package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type geminiTransport struct {
	baseURL   string
	apiKey    string
	model     string
	maxTokens int
	http      *http.Client
}

type geminiRequest struct {
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	Contents          []geminiContent        `json:"contents"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
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

func (t *geminiTransport) review(sysPrompt, userPrompt string) (*domain.ReviewResult, error) {
	reqBody, err := json.Marshal(geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: sysPrompt}},
		},
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: userPrompt}}},
		},
		GenerationConfig: geminiGenerationConfig{Temperature: 0.2, MaxOutputTokens: t.maxTokens},
	})
	if err != nil {
		return nil, fmt.Errorf("ai: marshalling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, t.generateContentURL(), bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("ai: building request: %w", err)
	}
	req.Header.Set("X-goog-api-key", t.apiKey)
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

func (t *geminiTransport) generateContentURL() string {
	model := strings.TrimPrefix(t.model, "models/")
	return t.baseURL + "/models/" + url.PathEscape(model) + ":generateContent"
}

func collectText(parts []geminiPart) string {
	var out strings.Builder
	for _, part := range parts {
		out.WriteString(part.Text)
	}
	return strings.TrimSpace(out.String())
}
