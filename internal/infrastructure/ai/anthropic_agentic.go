package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// anthropicAgenticDriver runs a tool-use conversation against the Messages API.
// It keeps the full message history (with content blocks) between steps.
type anthropicAgenticDriver struct {
	baseURL   string
	model     string
	token     string
	mode      domain.AIConfigMode
	maxTokens int
	http      *http.Client

	system   []anthropicSystemBlock
	tools    []anthropicTool
	messages []anthropicBlockMessage
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// anthropicBlockMessage uses content blocks (text/tool_use/tool_result) instead
// of a plain string, as required for tool use.
type anthropicBlockMessage struct {
	Role    string           `json:"role"`
	Content []anthropicBlock `json:"content"`
}

type anthropicBlock struct {
	Type string `json:"type"`
	// text
	Text string `json:"text,omitempty"`
	// tool_use
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
	// tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

type anthropicBlockRequest struct {
	Model     string                  `json:"model"`
	MaxTokens int                     `json:"max_tokens"`
	System    []anthropicSystemBlock  `json:"system"`
	Tools     []anthropicTool         `json:"tools"`
	Messages  []anthropicBlockMessage `json:"messages"`
}

type anthropicBlockResponse struct {
	Content []struct {
		Type  string         `json:"type"`
		Text  string         `json:"text"`
		ID    string         `json:"id"`
		Name  string         `json:"name"`
		Input map[string]any `json:"input"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func newAnthropicAgenticDriver(c *Client, sysPrompt, userPrompt string, tools []toolSpec) *anthropicAgenticDriver {
	system := []anthropicSystemBlock{{Type: "text", Text: sysPrompt}}
	if c.mode == domain.AIModeAccount {
		system = []anthropicSystemBlock{
			{Type: "text", Text: claudeCodeIdentity},
			{Type: "text", Text: sysPrompt},
		}
	}
	at := make([]anthropicTool, 0, len(tools))
	for _, s := range tools {
		at = append(at, anthropicTool{Name: s.Name, Description: s.Description, InputSchema: s.Schema})
	}
	return &anthropicAgenticDriver{
		baseURL:   c.baseURL,
		model:     c.model,
		token:     c.token,
		mode:      c.mode,
		maxTokens: c.maxTokens,
		http:      c.http,
		system:    system,
		tools:     at,
		messages: []anthropicBlockMessage{
			{Role: "user", Content: []anthropicBlock{{Type: "text", Text: userPrompt}}},
		},
	}
}

func (d *anthropicAgenticDriver) step(results []toolResult, nudge string, allowTools bool) (agenticStep, error) {
	// Append previous tool results (if any) plus an optional nudge as a
	// single user message of content blocks.
	if len(results) > 0 || nudge != "" {
		blocks := make([]anthropicBlock, 0, len(results)+1)
		for _, r := range results {
			blocks = append(blocks, anthropicBlock{Type: "tool_result", ToolUseID: r.ID, Content: r.Output})
		}
		if nudge != "" {
			blocks = append(blocks, anthropicBlock{Type: "text", Text: nudge})
		}
		d.messages = append(d.messages, anthropicBlockMessage{Role: "user", Content: blocks})
	}

	maxTokens := d.maxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	tools := d.tools
	if !allowTools {
		tools = []anthropicTool{}
		// This is the forced final turn: no further chance to retry if the
		// JSON payload gets cut off, so guarantee enough headroom.
		if maxTokens < lastTurnMinTokens {
			maxTokens = lastTurnMinTokens
		}
	}

	reqBody, err := json.Marshal(anthropicBlockRequest{
		Model:     d.model,
		MaxTokens: maxTokens,
		System:    d.system,
		Tools:     tools,
		Messages:  d.messages,
	})
	if err != nil {
		return agenticStep{}, fmt.Errorf("ai: marshalling agentic request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, d.baseURL+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return agenticStep{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", anthropicVersion)
	if d.mode == domain.AIModeAccount {
		req.Header.Set("Authorization", "Bearer "+d.token)
		req.Header.Set("anthropic-beta", anthropicBetaHeader())
	} else {
		req.Header.Set("x-api-key", d.token)
	}

	resp, err := d.http.Do(req)
	if err != nil {
		return agenticStep{}, fmt.Errorf("ai: calling API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return agenticStep{}, fmt.Errorf("ai: API returned status %d: %s", resp.StatusCode, rb)
	}

	var out anthropicBlockResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return agenticStep{}, fmt.Errorf("ai: decoding response: %w", err)
	}

	// Record the assistant turn verbatim (text + tool_use blocks) for history.
	// IMPORTANT: this must happen BEFORE the next tool_result user message is
	// appended, so the API sees the correct alternating sequence.
	assistant := anthropicBlockMessage{Role: "assistant"}
	var finalText bytes.Buffer
	var calls []toolCall
	for _, b := range out.Content {
		switch b.Type {
		case "text":
			finalText.WriteString(b.Text)
			assistant.Content = append(assistant.Content, anthropicBlock{Type: "text", Text: b.Text})
		case "tool_use":
			assistant.Content = append(assistant.Content, anthropicBlock{Type: "tool_use", ID: b.ID, Name: b.Name, Input: b.Input})
			calls = append(calls, toolCall{ID: b.ID, Name: b.Name, Input: b.Input})
		}
	}
	d.messages = append(d.messages, assistant)

	usage := domain.TokenUsage{
		PromptTokens:     out.Usage.InputTokens,
		CompletionTokens: out.Usage.OutputTokens,
		TotalTokens:      out.Usage.InputTokens + out.Usage.OutputTokens,
	}

	// If the model requested tools, continue the loop; otherwise it's the final answer.
	if out.StopReason == "tool_use" && len(calls) > 0 {
		return agenticStep{Calls: calls, Usage: usage}, nil
	}
	return agenticStep{Final: finalText.String(), Usage: usage}, nil
}
