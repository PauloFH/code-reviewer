# Modos de Revisão — Fase 1 / Milestone 4b-2: Drivers agênticos OpenAI/DeepSeek + Gemini — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Estender a revisão agêntica para além do Claude: implementar `agenticDriver` para `chat_completions` (OpenAI/DeepSeek, formato `tool_calls`) e para Gemini (`functionDeclarations`/`functionCall`), plugando-os no mesmo loop e no switch `Client.agenticDriver`.

**Architecture:** Reusa toda a camada genérica do M4b (`toolSpec`, `toolExecutor`, `runAgenticLoop`, interface `agenticDriver`). Cada driver mantém o histórico da conversa no formato do seu provider e traduz `toolCall`/`toolResult`. Após este milestone, o switch `agenticDriver()` cobre anthropic + chat_completions + gemini; nada mais cai no fallback por falta de driver.

**Tech Stack:** Go (stdlib): `encoding/json`, `net/http`, `bytes`. Testes com `httptest`.

**Pré-requisitos:** M4b concluído (loop + driver Anthropic + `agenticDriver()` switch em `client.go`).

**Referências de padrão (ler antes):**
- `internal/infrastructure/ai/anthropic_agentic.go` — estrutura de um driver (histórico, step, mapeamento de tools).
- `internal/infrastructure/ai/anthropic_agentic_test.go` — estilo do teste httptest (tool → final, asserção do tool_result no 2º request).
- `internal/infrastructure/ai/chat_completions.go` e `gemini.go` — formato base de cada API (URL, headers, usage).
- `internal/infrastructure/ai/agentic.go` — `toolSpec`, `toolCall`, `toolResult`, `agenticStep`, `agenticDriver`, `defaultMaxTokens`.
- `internal/infrastructure/ai/client.go` — `agenticDriver()` switch.

---

## File Structure

- **Create:** `internal/infrastructure/ai/chat_completions_agentic.go` — `chatCompletionsAgenticDriver`.
- **Create:** `internal/infrastructure/ai/chat_completions_agentic_test.go` — teste httptest.
- **Create:** `internal/infrastructure/ai/gemini_agentic.go` — `geminiAgenticDriver`.
- **Create:** `internal/infrastructure/ai/gemini_agentic_test.go` — teste httptest.
- **Modify:** `internal/infrastructure/ai/client.go` — adicionar os 2 cases em `agenticDriver()`.

---

## Task 1: Driver chat_completions (OpenAI/DeepSeek)

**Files:**
- Create: `internal/infrastructure/ai/chat_completions_agentic.go`
- Create: `internal/infrastructure/ai/chat_completions_agentic_test.go`

- [ ] **Step 1: Escrever o teste que falha**

Criar `internal/infrastructure/ai/chat_completions_agentic_test.go`:

```go
package ai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestChatCompletionsAgenticToolThenFinal(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		call++
		if call == 1 {
			var req map[string]any
			_ = json.Unmarshal(body, &req)
			if _, ok := req["tools"]; !ok {
				t.Errorf("first request missing tools")
			}
			w.Write([]byte(`{
				"choices":[{"message":{"content":null,"tool_calls":[
					{"id":"tc_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}
				]},"finish_reason":"tool_calls"}],
				"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120}
			}`))
			return
		}
		if !strings.Contains(string(body), `"role":"tool"`) || !strings.Contains(string(body), "tc_1") {
			t.Errorf("second request missing tool result for tc_1: %s", body)
		}
		w.Write([]byte(`{
			"choices":[{"message":{"content":"{\"review_markdown\":\"ok chat\",\"inline_comments\":[]}"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":50,"completion_tokens":10,"total_tokens":60}
		}`))
	}))
	defer srv.Close()

	c := &Client{
		transportKind: providerChatCompletions,
		mode:          domain.AIModeAPI,
		baseURL:       srv.URL,
		token:         "k",
		model:         "gpt-4o",
		http:          &http.Client{Timeout: 10 * time.Second},
	}
	driver, err := c.agenticDriver("sys", "user prompt", readToolSpecs())
	if err != nil {
		t.Fatalf("agenticDriver: %v", err)
	}
	exec := newToolExecutor(&fakeWorkspace{files: map[string]string{"main.go": "package main\n"}})

	res, err := runAgenticLoop(driver, exec, 12)
	if err != nil {
		t.Fatalf("loop: %v", err)
	}
	if res.Body != "ok chat" {
		t.Fatalf("body = %q", res.Body)
	}
	if res.Usage.TotalTokens != 180 {
		t.Fatalf("usage = %d, want 180", res.Usage.TotalTokens)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/ai/ -run TestChatCompletionsAgentic -v`
Expected: FAIL — `agenticDriver` retorna erro "not supported" para chat_completions (ainda não há case).

- [ ] **Step 3: Implementar**

Criar `internal/infrastructure/ai/chat_completions_agentic.go`:

```go
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

func (d *chatCompletionsAgenticDriver) step(results []toolResult) (agenticStep, error) {
	for _, r := range results {
		d.messages = append(d.messages, chatMsg{Role: "tool", ToolCallID: r.ID, Content: r.Output})
	}

	reqBody, err := json.Marshal(chatAgenticRequest{
		Model:       d.model,
		Messages:    d.messages,
		Tools:       d.tools,
		Temperature: 0.2,
		MaxTokens:   d.maxTokens,
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
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/ai/ -run TestChatCompletionsAgentic -v`
Expected: PASS.

- [ ] **Step 5: Adicionar o case no switch + rodar**

Em `internal/infrastructure/ai/client.go`, no `agenticDriver()`, adicionar antes do `default`:

```go
	case providerChatCompletions:
		return newChatCompletionsAgenticDriver(c, sysPrompt, userPrompt, tools), nil
```

Run: `go test ./internal/infrastructure/ai/ -run 'TestChatCompletionsAgentic|TestReviewDiffFallsBack' -v`

NOTA: o teste `TestReviewDiffFallsBackWhenAgenticUnsupported` (do M4b) usava `chat_completions` como provider SEM driver para validar o fallback. Agora chat_completions TEM driver — esse teste vai quebrar. Ajuste-o: troque o provider do teste de fallback para um provider sem driver (use `transportKind: providerGemini` ANTES da Task 2; ou, se fizer a Task 2 antes, use um `transportKind` inexistente como `"bogus"`). A forma robusta: no teste de fallback, defina `transportKind: "bogus"` e ajuste a expectativa para cair no `transport()` que retornará erro. PORÉM `transport()` também não conhece "bogus" e retornaria erro — então o fallback resultaria em erro, não em review.

Para manter o teste de fallback válido, reescreva-o assim: use um provider COM transporte single-shot mas SEM driver agêntico. Após este milestone, só sobra... todos têm driver. Então o teste de fallback deixa de fazer sentido com providers reais. **Substitua** `TestReviewDiffFallsBackWhenAgenticUnsupported` por um teste que força o fallback via um `Client` cujo `transportKind` tem single-shot mas cujo `agenticDriver` retorna erro — para isso, simule com `transportKind` setado a um valor que `transport()` conheça e `agenticDriver()` não. Como após este milestone os 3 coincidem, **remova** esse teste de fallback e cubra o fallback por outro caminho: um teste em que `input.Workspace == nil` (agentic sem workspace) cai no single-shot. Adicione:

```go
func TestReviewDiffSingleShotWhenNoWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"review_markdown\":\"single shot\",\"inline_comments\":[]}"}}],"usage":{"total_tokens":7}}`))
	}))
	defer srv.Close()
	c := &Client{transportKind: providerChatCompletions, mode: domain.AIModeAPI, baseURL: srv.URL, token: "k", model: "gpt-4o", http: &http.Client{Timeout: 10 * time.Second}}
	res, err := c.ReviewDiff(domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "t"},
		Diff:         "--- a/x\n+++ b/x\n@@ -1 +1 @@\n+x",
		Profile:      &domain.ReviewProfile{Agentic: true}, // agentic, mas sem Workspace → single-shot
	})
	if err != nil {
		t.Fatalf("expected single-shot, got error: %v", err)
	}
	if res.Body != "single shot" {
		t.Fatalf("body = %q", res.Body)
	}
}
```

Remova o `TestReviewDiffFallsBackWhenAgenticUnsupported` antigo (substituído por este).

Run: `go test ./internal/infrastructure/ai/ -v`
Expected: PASS (todos).

- [ ] **Step 6: Commit**

```bash
git add internal/infrastructure/ai/chat_completions_agentic.go internal/infrastructure/ai/chat_completions_agentic_test.go internal/infrastructure/ai/client.go
git commit -m "feat(ai): driver agêntico chat_completions (OpenAI/DeepSeek, tool_calls)"
```

---

## Task 2: Driver Gemini (functionDeclarations)

**Files:**
- Create: `internal/infrastructure/ai/gemini_agentic.go`
- Create: `internal/infrastructure/ai/gemini_agentic_test.go`

- [ ] **Step 1: Escrever o teste que falha**

Criar `internal/infrastructure/ai/gemini_agentic_test.go`:

```go
package ai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestGeminiAgenticToolThenFinal(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		call++
		if call == 1 {
			if !strings.Contains(string(body), "functionDeclarations") {
				t.Errorf("first request missing functionDeclarations: %s", body)
			}
			w.Write([]byte(`{
				"candidates":[{"content":{"role":"model","parts":[
					{"functionCall":{"name":"read_file","args":{"path":"main.go"}}}
				]},"finishReason":"STOP"}],
				"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":20,"totalTokenCount":120}
			}`))
			return
		}
		if !strings.Contains(string(body), "functionResponse") || !strings.Contains(string(body), "read_file") {
			t.Errorf("second request missing functionResponse: %s", body)
		}
		w.Write([]byte(`{
			"candidates":[{"content":{"role":"model","parts":[{"text":"{\"review_markdown\":\"ok gemini\",\"inline_comments\":[]}"}]},"finishReason":"STOP"}],
			"usageMetadata":{"promptTokenCount":50,"candidatesTokenCount":10,"totalTokenCount":60}
		}`))
	}))
	defer srv.Close()

	c := &Client{
		transportKind: providerGemini,
		mode:          domain.AIModeAPI,
		baseURL:       srv.URL,
		token:         "k",
		model:         "gemini-flash-latest",
		http:          &http.Client{Timeout: 10 * time.Second},
	}
	driver, err := c.agenticDriver("sys", "user prompt", readToolSpecs())
	if err != nil {
		t.Fatalf("agenticDriver: %v", err)
	}
	exec := newToolExecutor(&fakeWorkspace{files: map[string]string{"main.go": "package main\n"}})

	res, err := runAgenticLoop(driver, exec, 12)
	if err != nil {
		t.Fatalf("loop: %v", err)
	}
	if res.Body != "ok gemini" {
		t.Fatalf("body = %q", res.Body)
	}
	if res.Usage.TotalTokens != 180 {
		t.Fatalf("usage = %d, want 180", res.Usage.TotalTokens)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/ai/ -run TestGeminiAgentic -v`
Expected: FAIL — sem case gemini em `agenticDriver`.

- [ ] **Step 3: Implementar**

Criar `internal/infrastructure/ai/gemini_agentic.go`:

```go
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
```

- [ ] **Step 4: Adicionar o case + rodar**

Em `internal/infrastructure/ai/client.go`, no `agenticDriver()`, adicionar antes do `default`:

```go
	case providerGemini:
		return newGeminiAgenticDriver(c, sysPrompt, userPrompt, tools), nil
```

Run: `go test ./internal/infrastructure/ai/ -v`
Expected: PASS (todos).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/ai/gemini_agentic.go internal/infrastructure/ai/gemini_agentic_test.go internal/infrastructure/ai/client.go
git commit -m "feat(ai): driver agêntico Gemini (functionDeclarations/functionCall)"
```

---

## Verificação final do milestone

- [ ] **Build:** `go build ./...` → sucesso.
- [ ] **Testes:** `go test ./...` → tudo verde, sem regressão.
- [ ] **Sanidade:** `agenticDriver()` agora cobre anthropic + chat_completions + gemini. Nenhum provider tool-capable cai mais no fallback por falta de driver.

## Notas para o executor

- **Ordem das mensagens (idêntico ao Anthropic):** registre a resposta do assistant/model (com tool_calls/functionCall) no histórico ANTES de anexar os tool_result/functionResponse do próximo turno.
- **OpenAI arguments é STRING JSON:** parseie `tc.Function.Arguments` com `json.Unmarshal` para `map[string]any`.
- **Gemini role "function":** os `functionResponse` vão num content com role `"function"`. **Caveat:** o teste httptest valida só a NOSSA serialização, não o contrato real do Gemini — verifique o `role` contra a API real do Gemini num teste manual (convenção do projeto: HTTP de provider é testado manualmente). Se a API rejeitar, o valor provável alternativo é `role: "user"`.
- **Teste de fallback do M4b:** após adicionar o driver chat_completions, o antigo `TestReviewDiffFallsBackWhenAgenticUnsupported` deixa de valer (chat_completions passa a ter driver). Substitua-o pelo `TestReviewDiffSingleShotWhenNoWorkspace` (Task 1, Step 5).
- **Reuso:** NÃO duplique o loop nem o executor — só os drivers. `geminiGenerationConfig` já existe em `gemini.go`; reutilize.
