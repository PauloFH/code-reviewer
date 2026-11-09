package ai

import (
	"encoding/json"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// chatConversation talks to an OpenAI-compatible POST /chat/completions.
type chatConversation struct {
	conn     connection
	tools    []chatTool
	messages []chatMessage
}

type chatTool struct {
	Type     string          `json:"type"`
	Function chatFunctionDef `json:"function"`
}

type chatFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Tools       any           `json:"tools,omitempty"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content   string         `json:"content"`
			ToolCalls []chatToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func newChatConversation(conn connection, sysPrompt, userPrompt string, tools []toolSpec) *chatConversation {
	var ct []chatTool
	for _, s := range tools {
		ct = append(ct, chatTool{Type: "function", Function: chatFunctionDef{
			Name: s.Name, Description: s.Description, Parameters: s.Schema,
		}})
	}
	return &chatConversation{
		conn:  conn,
		tools: ct,
		messages: []chatMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
}

func (c *chatConversation) step(results []toolResult, nudge string, allowTools bool) (agenticStep, error) {
	for _, r := range results {
		c.messages = append(c.messages, chatMessage{Role: "tool", ToolCallID: r.ID, Content: r.Output})
	}
	if nudge != "" {
		c.messages = append(c.messages, chatMessage{Role: "user", Content: nudge})
	}

	req := chatRequest{Model: c.conn.model, Messages: c.messages, Temperature: temperature, MaxTokens: c.conn.maxTokens}
	if c.tools != nil {
		if allowTools {
			req.Tools = c.tools
		} else {
			req.Tools = []chatTool{}
			req.MaxTokens = lastTurnMaxTokens(req.MaxTokens)
		}
	}

	var out chatResponse
	headers := map[string]string{"Authorization": "Bearer " + c.conn.token}
	if err := postJSON(c.conn.http, c.conn.baseURL+"/chat/completions", headers, req, &out); err != nil {
		return agenticStep{}, err
	}
	if len(out.Choices) == 0 {
		return agenticStep{}, emptyResponseError("")
	}

	usage := domain.TokenUsage{
		PromptTokens:     out.Usage.PromptTokens,
		CompletionTokens: out.Usage.CompletionTokens,
		TotalTokens:      out.Usage.TotalTokens,
	}
	choice := out.Choices[0]
	msg := choice.Message
	// The assistant turn must be recorded before the next tool result messages.
	c.messages = append(c.messages, chatMessage{Role: "assistant", Content: msg.Content, ToolCalls: msg.ToolCalls})

	if len(msg.ToolCalls) > 0 {
		calls := make([]toolCall, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			call := toolCall{ID: tc.ID, Name: tc.Function.Name, Input: map[string]any{}}
			if tc.Function.Arguments != "" {
				call.ArgsErr = json.Unmarshal([]byte(tc.Function.Arguments), &call.Input)
			}
			calls = append(calls, call)
		}
		return agenticStep{Calls: calls, Usage: usage}, nil
	}
	if strings.TrimSpace(msg.Content) == "" {
		return agenticStep{}, emptyResponseError(choice.FinishReason)
	}
	return agenticStep{Final: msg.Content, Usage: usage}, nil
}
