package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type chatCompletionsAgenticDriver struct {
	baseURL   string
	token     string
	model     string
	maxTokens int
	http      *http.Client

	tools    []chatTool
	messages []chatMsg
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

type chatMsg struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type chatAgenticRequest struct {
	Model       string     `json:"model"`
	Messages    []chatMsg  `json:"messages"`
	Tools       []chatTool `json:"tools"`
	Temperature float64    `json:"temperature"`
	MaxTokens   int        `json:"max_tokens,omitempty"`
}

type chatAgenticResponse struct {
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

func newChatCompletionsAgenticDriver(c *Client, sysPrompt, userPrompt string, tools []toolSpec) *chatCompletionsAgenticDriver {
	ct := make([]chatTool, 0, len(tools))
	for _, s := range tools {
		ct = append(ct, chatTool{Type: "function", Function: chatFunctionDef{
			Name: s.Name, Description: s.Description, Parameters: s.Schema,
		}})
	}
	return &chatCompletionsAgenticDriver{
		baseURL:   c.baseURL,
		token:     c.token,
		model:     c.model,
		maxTokens: c.maxTokens,
		http:      c.http,
		tools:     ct,
		messages: []chatMsg{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
}

func (d *chatCompletionsAgenticDriver) step(results []toolResult, nudge string, allowTools bool) (agenticStep, error) {
	for _, r := range results {
		d.messages = append(d.messages, chatMsg{Role: "tool", ToolCallID: r.ID, Content: r.Output})
	}
	if nudge != "" {
		d.messages = append(d.messages, chatMsg{Role: "user", Content: nudge})
	}

	tools := d.tools
	maxTokens := d.maxTokens
	if !allowTools {
		tools = []chatTool{}
		// This is the forced final turn: no further chance to retry if the
		// JSON payload gets cut off, so guarantee enough headroom. A zero
		// maxTokens (field omitted) already means "no explicit cap", so it
		// is left alone — only an explicit, too-low cap is raised.
		if maxTokens > 0 && maxTokens < lastTurnMinTokens {
			maxTokens = lastTurnMinTokens
		}
	}

	reqBody, err := json.Marshal(chatAgenticRequest{
		Model:       d.model,
		Messages:    d.messages,
		Tools:       tools,
		Temperature: 0.2,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return agenticStep{}, fmt.Errorf("ai: marshalling agentic request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, d.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return agenticStep{}, err
	}
	req.Header.Set("Authorization", "Bearer "+d.token)
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

	var out chatAgenticResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return agenticStep{}, fmt.Errorf("ai: decoding response: %w", err)
	}
	if len(out.Choices) == 0 {
		return agenticStep{}, fmt.Errorf("ai: empty response from API")
	}

	usage := domain.TokenUsage{
		PromptTokens:     out.Usage.PromptTokens,
		CompletionTokens: out.Usage.CompletionTokens,
		TotalTokens:      out.Usage.TotalTokens,
	}

	msg := out.Choices[0].Message
	// Record the assistant turn verbatim BEFORE appending tool results next round.
	// IMPORTANT: this must happen BEFORE the next tool result message is appended,
	// so the API sees the correct alternating sequence.
	d.messages = append(d.messages, chatMsg{Role: "assistant", Content: msg.Content, ToolCalls: msg.ToolCalls})

	if len(msg.ToolCalls) > 0 {
		calls := make([]toolCall, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			input := map[string]any{}
			if tc.Function.Arguments != "" {
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
			}
			calls = append(calls, toolCall{ID: tc.ID, Name: tc.Function.Name, Input: input})
		}
		return agenticStep{Calls: calls, Usage: usage}, nil
	}
	return agenticStep{Final: msg.Content, Usage: usage}, nil
}
