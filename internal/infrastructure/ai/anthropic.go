package ai

import (
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const (
	anthropicVersion = "2023-06-01"
	// anthropicOAuthBeta is required for subscription (OAuth) tokens.
	anthropicOAuthBeta = "oauth-2025-04-20"
	// anthropicClaudeCodeBeta is part of the Claude Code OAuth request profile.
	anthropicClaudeCodeBeta = "claude-code-20250219"
	// claudeCodeIdentity must be the first system block for OAuth subscription
	// tokens; without it the Messages API rejects the request.
	claudeCodeIdentity = "You are Claude Code, Anthropic's official CLI for Claude."
)

// anthropicConversation talks to the native Messages API (POST /v1/messages).
// No temperature is sent: it is deprecated for the subscription models.
type anthropicConversation struct {
	conn     connection
	system   []anthropicBlock
	tools    []anthropicTool
	messages []anthropicMessage
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicMessage struct {
	Role    string           `json:"role"`
	Content []anthropicBlock `json:"content"`
}

// anthropicBlock covers the text, tool_use and tool_result content blocks.
type anthropicBlock struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   string         `json:"content,omitempty"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    []anthropicBlock   `json:"system"`
	Tools     any                `json:"tools,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content    []anthropicBlock `json:"content"`
	StopReason string           `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func newAnthropicConversation(conn connection, sysPrompt, userPrompt string, tools []toolSpec) *anthropicConversation {
	system := []anthropicBlock{{Type: "text", Text: sysPrompt}}
	if conn.mode == domain.AIModeAccount {
		system = append([]anthropicBlock{{Type: "text", Text: claudeCodeIdentity}}, system...)
	}
	var at []anthropicTool
	for _, s := range tools {
		at = append(at, anthropicTool{Name: s.Name, Description: s.Description, InputSchema: s.Schema})
	}
	return &anthropicConversation{
		conn:     conn,
		system:   system,
		tools:    at,
		messages: []anthropicMessage{{Role: "user", Content: []anthropicBlock{{Type: "text", Text: userPrompt}}}},
	}
}

func (a *anthropicConversation) step(results []toolResult, nudge string, allowTools bool) (agenticStep, error) {
	if len(results) > 0 || nudge != "" {
		blocks := make([]anthropicBlock, 0, len(results)+1)
		for _, r := range results {
			blocks = append(blocks, anthropicBlock{Type: "tool_result", ToolUseID: r.ID, Content: r.Output})
		}
		if nudge != "" {
			blocks = append(blocks, anthropicBlock{Type: "text", Text: nudge})
		}
		a.messages = append(a.messages, anthropicMessage{Role: "user", Content: blocks})
	}

	maxTokens := a.conn.maxTokens
	req := anthropicRequest{Model: a.conn.model, MaxTokens: maxTokens, System: a.system, Messages: a.messages}
	if a.tools != nil {
		if allowTools {
			req.Tools = a.tools
		} else {
			req.Tools = []anthropicTool{}
			req.MaxTokens = lastTurnMaxTokens(maxTokens)
		}
	}

	headers := map[string]string{"anthropic-version": anthropicVersion}
	if a.conn.mode == domain.AIModeAccount {
		headers["Authorization"] = "Bearer " + a.conn.token
		headers["anthropic-beta"] = anthropicClaudeCodeBeta + "," + anthropicOAuthBeta
	} else {
		headers["x-api-key"] = a.conn.token
	}

	var out anthropicResponse
	if err := postJSON(a.conn.http, a.conn.baseURL+"/v1/messages", headers, req, &out); err != nil {
		return agenticStep{}, err
	}

	// The assistant turn must be recorded before the next tool_result message.
	assistant := anthropicMessage{Role: "assistant"}
	var text strings.Builder
	var calls []toolCall
	for _, b := range out.Content {
		switch b.Type {
		case "text":
			text.WriteString(b.Text)
			assistant.Content = append(assistant.Content, anthropicBlock{Type: "text", Text: b.Text})
		case "tool_use":
			assistant.Content = append(assistant.Content, anthropicBlock{Type: "tool_use", ID: b.ID, Name: b.Name, Input: b.Input})
			calls = append(calls, toolCall{ID: b.ID, Name: b.Name, Input: b.Input})
		}
	}
	a.messages = append(a.messages, assistant)

	usage := domain.TokenUsage{
		PromptTokens:     out.Usage.InputTokens,
		CompletionTokens: out.Usage.OutputTokens,
		TotalTokens:      out.Usage.InputTokens + out.Usage.OutputTokens,
	}
	if out.StopReason == "tool_use" && len(calls) > 0 {
		return agenticStep{Calls: calls, Usage: usage}, nil
	}
	if strings.TrimSpace(text.String()) == "" {
		return agenticStep{}, emptyResponseError(out.StopReason)
	}
	return agenticStep{Final: text.String(), Usage: usage}, nil
}
