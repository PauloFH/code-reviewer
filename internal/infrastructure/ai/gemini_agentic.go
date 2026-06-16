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

type geminiAgenticDriver struct {
	baseURL   string
	apiKey    string
	model     string
	maxTokens int
	http      *http.Client

	system   *geminiAgContent
	tools    []geminiTool
	contents []geminiAgContent
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type geminiAgPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiAgContent struct {
	Role  string         `json:"role,omitempty"`
	Parts []geminiAgPart `json:"parts"`
}

type geminiAgRequest struct {
	SystemInstruction *geminiAgContent       `json:"systemInstruction,omitempty"`
	Contents          []geminiAgContent      `json:"contents"`
	Tools             []geminiTool           `json:"tools,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig"`
}

type geminiAgResponse struct {
	Candidates []struct {
		Content      geminiAgContent `json:"content"`
		FinishReason string          `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func newGeminiAgenticDriver(c *Client, sysPrompt, userPrompt string, tools []toolSpec) *geminiAgenticDriver {
	decls := make([]geminiFunctionDecl, 0, len(tools))
	for _, s := range tools {
		decls = append(decls, geminiFunctionDecl{Name: s.Name, Description: s.Description, Parameters: s.Schema})
	}
	return &geminiAgenticDriver{
		baseURL:   c.baseURL,
		apiKey:    c.token,
		model:     c.model,
		maxTokens: c.maxTokens,
		http:      c.http,
		system:    &geminiAgContent{Parts: []geminiAgPart{{Text: sysPrompt}}},
		tools:     []geminiTool{{FunctionDeclarations: decls}},
		contents: []geminiAgContent{
			{Role: "user", Parts: []geminiAgPart{{Text: userPrompt}}},
		},
	}
}

func (d *geminiAgenticDriver) step(results []toolResult) (agenticStep, error) {
	if len(results) > 0 {
		parts := make([]geminiAgPart, 0, len(results))
		for _, r := range results {
			parts = append(parts, geminiAgPart{FunctionResponse: &geminiFunctionResponse{
				Name:     r.Name,
				Response: map[string]any{"output": r.Output},
			}})
		}
		// Gemini expects tool results in a content with role "function".
		d.contents = append(d.contents, geminiAgContent{Role: "function", Parts: parts})
	}

	reqBody, err := json.Marshal(geminiAgRequest{
		SystemInstruction: d.system,
		Contents:          d.contents,
		Tools:             d.tools,
		GenerationConfig:  geminiGenerationConfig{Temperature: 0.2, MaxOutputTokens: d.maxTokens},
	})
	if err != nil {
		return agenticStep{}, fmt.Errorf("ai: marshalling agentic request: %w", err)
	}

	model := strings.TrimPrefix(d.model, "models/")
	endpoint := d.baseURL + "/models/" + url.PathEscape(model) + ":generateContent"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return agenticStep{}, err
	}
	req.Header.Set("X-goog-api-key", d.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.http.Do(req)
	if err != nil {
		return agenticStep{}, fmt.Errorf("ai: calling API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return agenticStep{}, fmt.Errorf("ai: API returned status %d: %s", resp.StatusCode, rb)
	}

	var out geminiAgResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return agenticStep{}, fmt.Errorf("ai: decoding response: %w", err)
	}
	if len(out.Candidates) == 0 {
		return agenticStep{}, fmt.Errorf("ai: empty response from API")
	}

	usage := domain.TokenUsage{
		PromptTokens:     out.UsageMetadata.PromptTokenCount,
		CompletionTokens: out.UsageMetadata.CandidatesTokenCount,
		TotalTokens:      out.UsageMetadata.TotalTokenCount,
	}

	cand := out.Candidates[0].Content
	// Record the model turn verbatim BEFORE appending the next functionResponse.
	// IMPORTANT: this must happen BEFORE the next tool result content is appended,
	// so the API sees the correct turn sequence.
	d.contents = append(d.contents, geminiAgContent{Role: "model", Parts: cand.Parts})

	var finalText strings.Builder
	var calls []toolCall
	for _, p := range cand.Parts {
		if p.FunctionCall != nil {
			calls = append(calls, toolCall{ID: p.FunctionCall.Name, Name: p.FunctionCall.Name, Input: p.FunctionCall.Args})
		} else if p.Text != "" {
			finalText.WriteString(p.Text)
		}
	}

	if len(calls) > 0 {
		return agenticStep{Calls: calls, Usage: usage}, nil
	}
	return agenticStep{Final: finalText.String(), Usage: usage}, nil
}
