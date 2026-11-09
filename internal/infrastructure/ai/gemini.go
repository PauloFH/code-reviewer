package ai

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// geminiConversation talks to POST /models/{model}:generateContent.
type geminiConversation struct {
	conn     connection
	system   *geminiContent
	tools    []geminiTool
	contents []geminiContent
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type geminiPart struct {
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

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiRequest struct {
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	Contents          []geminiContent        `json:"contents"`
	Tools             []geminiTool           `json:"tools,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig"`
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

func newGeminiConversation(conn connection, sysPrompt, userPrompt string, tools []toolSpec) *geminiConversation {
	g := &geminiConversation{
		conn:     conn,
		system:   &geminiContent{Parts: []geminiPart{{Text: sysPrompt}}},
		contents: []geminiContent{{Role: "user", Parts: []geminiPart{{Text: userPrompt}}}},
	}
	if len(tools) > 0 {
		decls := make([]geminiFunctionDecl, 0, len(tools))
		for _, s := range tools {
			decls = append(decls, geminiFunctionDecl{Name: s.Name, Description: s.Description, Parameters: s.Schema})
		}
		g.tools = []geminiTool{{FunctionDeclarations: decls}}
	}
	return g
}

func (g *geminiConversation) step(results []toolResult, nudge string, allowTools bool) (agenticStep, error) {
	if len(results) > 0 {
		parts := make([]geminiPart, 0, len(results))
		for _, r := range results {
			parts = append(parts, geminiPart{FunctionResponse: &geminiFunctionResponse{
				Name:     r.Name,
				Response: map[string]any{"output": r.Output},
			}})
		}
		// Gemini expects tool results in a content with role "function".
		g.contents = append(g.contents, geminiContent{Role: "function", Parts: parts})
	}
	if nudge != "" {
		// A separate user turn, so the nudge doesn't read as part of a tool's output.
		g.contents = append(g.contents, geminiContent{Role: "user", Parts: []geminiPart{{Text: nudge}}})
	}

	req := geminiRequest{
		SystemInstruction: g.system,
		Contents:          g.contents,
		GenerationConfig:  geminiGenerationConfig{Temperature: temperature, MaxOutputTokens: g.conn.maxTokens},
	}
	if g.tools != nil {
		if allowTools {
			req.Tools = g.tools
		} else {
			req.GenerationConfig.MaxOutputTokens = lastTurnMaxTokens(g.conn.maxTokens)
		}
	}

	model := strings.TrimPrefix(g.conn.model, "models/")
	endpoint := g.conn.baseURL + "/models/" + url.PathEscape(model) + ":generateContent"
	var out geminiResponse
	if err := postJSON(g.conn.http, endpoint, map[string]string{"X-goog-api-key": g.conn.token}, req, &out); err != nil {
		return agenticStep{}, err
	}
	if len(out.Candidates) == 0 {
		if out.PromptFeedback.BlockReason != "" {
			return agenticStep{}, fmt.Errorf("ai: Gemini returned no candidates: prompt blocked: %s", out.PromptFeedback.BlockReason)
		}
		return agenticStep{}, emptyResponseError("")
	}

	usage := domain.TokenUsage{
		PromptTokens:     out.UsageMetadata.PromptTokenCount,
		CompletionTokens: out.UsageMetadata.CandidatesTokenCount,
		TotalTokens:      out.UsageMetadata.TotalTokenCount,
	}
	cand := out.Candidates[0]
	// The model turn must be recorded before the next functionResponse content.
	g.contents = append(g.contents, geminiContent{Role: "model", Parts: cand.Content.Parts})

	var text strings.Builder
	var calls []toolCall
	for _, p := range cand.Content.Parts {
		if p.FunctionCall != nil {
			// Gemini has no call ids; the function name stands in for one.
			calls = append(calls, toolCall{ID: p.FunctionCall.Name, Name: p.FunctionCall.Name, Input: p.FunctionCall.Args})
		} else {
			text.WriteString(p.Text)
		}
	}
	if len(calls) > 0 {
		return agenticStep{Calls: calls, Usage: usage}, nil
	}
	final := strings.TrimSpace(text.String())
	if final == "" {
		return agenticStep{}, emptyResponseError(cand.FinishReason)
	}
	return agenticStep{Final: final, Usage: usage}, nil
}
