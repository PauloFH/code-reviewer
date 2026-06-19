# Modos de Revisão — Fase 1 / Milestone 4b: Loop agêntico + driver Anthropic — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Entregar a revisão agêntica end-to-end para Claude: definição das tools de leitura, um executor sobre `CodeWorkspace`, um loop de tool-use genérico (driver-agnóstico) e o driver Anthropic (modos api + account), com seleção automática do caminho agêntico no `Client.ReviewDiff` e degradação graciosa para single-shot quando não há driver/workspace.

**Architecture:** Camada genérica em `ai/agentic.go`: `toolSpec` (nome+descrição+schema JSON), `toolExecutor` que roda `read_file`/`list_dir`/`grep` contra `domain.CodeWorkspace`, e `runAgenticLoop(driver, exec, maxIter)` que alterna entre pedir o próximo passo ao `agenticDriver` e executar as tools, até o modelo emitir o review final (JSON) ou estourar o teto de iterações. Cada provider implementa `agenticDriver` (mantém o histórico da conversa + HTTP + tradução do formato de tool-use). Neste milestone só o **Anthropic** tem driver; os demais retornam "unsupported" e o `Client` cai no single-shot.

**Tech Stack:** Go (stdlib): `encoding/json`, `net/http`, `bytes`. Testes com `testing`, fake driver/workspace e `httptest`.

**Pré-requisitos:** M4a concluído (`domain.CodeWorkspace`, `ReviewInput.Workspace`). M1–M3 concluídos (`ReviewInput.Profile`, `parseReviewPayload`).

**Referências de padrão (ler antes):**
- `internal/infrastructure/ai/client.go` — interface `transport`, `Client`, `transport()`, `transportKind*`, `parseReviewPayload`, `buildPrompt`, `defaultMaxTokens`.
- `internal/infrastructure/ai/anthropic.go` — `anthropicTransport`, headers (api vs account), `claudeCodeIdentity`, `anthropicBetaHeader`, structs de request/response.
- `internal/infrastructure/ai/anthropic_test.go` — estilo de teste com `httptest` para o transporte Anthropic.
- `internal/infrastructure/platform/gitfetch/workspace.go` — `CodeWorkspace` (ReadFile/ListDir/Grep/Root).

---

## File Structure

- **Create:** `internal/infrastructure/ai/agentic.go` — tool specs, executor, `agenticDriver`, `runAgenticLoop`.
- **Create:** `internal/infrastructure/ai/agentic_test.go` — testes do executor + loop com fakes.
- **Create:** `internal/infrastructure/ai/anthropic_agentic.go` — `anthropicAgenticDriver`.
- **Create:** `internal/infrastructure/ai/anthropic_agentic_test.go` — teste do driver com `httptest`.
- **Modify:** `internal/infrastructure/ai/client.go` — caminho agêntico em `ReviewDiff` + factory `agenticDriver()`.

---

## Task 1: Tool specs + executor sobre CodeWorkspace

**Files:**
- Create: `internal/infrastructure/ai/agentic.go`
- Create: `internal/infrastructure/ai/agentic_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Criar `internal/infrastructure/ai/agentic_test.go`:

```go
package ai

import (
	"strings"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// fakeWorkspace is an in-memory CodeWorkspace for tests.
type fakeWorkspace struct {
	files map[string]string
}

func (f *fakeWorkspace) Root() string { return "/fake" }
func (f *fakeWorkspace) ReadFile(path string) ([]byte, error) {
	if c, ok := f.files[path]; ok {
		return []byte(c), nil
	}
	return nil, &notFoundErr{path}
}
func (f *fakeWorkspace) ListDir(_ string) ([]domain.DirEntry, error) {
	return []domain.DirEntry{{Name: "main.go"}, {Name: "internal", IsDir: true}}, nil
}
func (f *fakeWorkspace) Grep(pattern, _ string) ([]domain.GrepHit, error) {
	var hits []domain.GrepHit
	for p, c := range f.files {
		for i, line := range strings.Split(c, "\n") {
			if strings.Contains(line, pattern) {
				hits = append(hits, domain.GrepHit{Path: p, Line: i + 1, Text: line})
			}
		}
	}
	return hits, nil
}

type notFoundErr struct{ p string }

func (e *notFoundErr) Error() string { return "not found: " + e.p }

func TestToolExecutorReadFile(t *testing.T) {
	exec := newToolExecutor(&fakeWorkspace{files: map[string]string{"main.go": "package main\n"}})
	out := exec.run("read_file", map[string]any{"path": "main.go"})
	if !strings.Contains(out, "package main") {
		t.Fatalf("read_file output = %q", out)
	}
}

func TestToolExecutorUnknownTool(t *testing.T) {
	exec := newToolExecutor(&fakeWorkspace{})
	out := exec.run("frobnicate", map[string]any{})
	if !strings.Contains(strings.ToLower(out), "unknown tool") {
		t.Fatalf("expected unknown-tool message, got %q", out)
	}
}

func TestToolExecutorReadFileErrorIsReported(t *testing.T) {
	exec := newToolExecutor(&fakeWorkspace{files: map[string]string{}})
	out := exec.run("read_file", map[string]any{"path": "missing.go"})
	if !strings.Contains(strings.ToLower(out), "error") && !strings.Contains(out, "not found") {
		t.Fatalf("expected error text in output, got %q", out)
	}
}

func TestToolSpecsCoverReadTools(t *testing.T) {
	names := map[string]bool{}
	for _, s := range readToolSpecs() {
		names[s.Name] = true
	}
	for _, want := range []string{"read_file", "list_dir", "grep"} {
		if !names[want] {
			t.Errorf("missing tool spec %q", want)
		}
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/ai/ -run 'TestTool' -v`
Expected: FAIL — `newToolExecutor`/`readToolSpecs` indefinidos.

- [ ] **Step 3: Implementar**

Criar `internal/infrastructure/ai/agentic.go`:

```go
package ai

import (
	"fmt"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// maxToolOutputBytes caps a single tool result fed back to the model.
const maxToolOutputBytes = 16_000

// defaultAgenticMaxIterations bounds the tool-use loop when the profile/env
// does not specify a limit.
const defaultAgenticMaxIterations = 12

// toolSpec is a provider-agnostic tool definition (JSON schema for the input).
type toolSpec struct {
	Name        string
	Description string
	Schema      map[string]any
}

// toolCall is a request from the model to run a tool.
type toolCall struct {
	ID    string
	Name  string
	Input map[string]any
}

// toolResult is the outcome of executing a toolCall, fed back to the model.
type toolResult struct {
	ID     string
	Name   string
	Output string
}

// readToolSpecs returns the read-only tools exposed to the agentic reviewer.
func readToolSpecs() []toolSpec {
	stringProp := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}
	return []toolSpec{
		{
			Name:        "read_file",
			Description: "Lê o conteúdo de um arquivo do repositório (caminho relativo à raiz).",
			Schema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"path": stringProp("Caminho do arquivo, relativo à raiz do repositório.")},
				"required":   []string{"path"},
			},
		},
		{
			Name:        "list_dir",
			Description: "Lista os arquivos e subdiretórios de um diretório (caminho relativo; use \".\" para a raiz).",
			Schema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"path": stringProp("Diretório, relativo à raiz. Use \".\" para a raiz.")},
				"required":   []string{"path"},
			},
		},
		{
			Name:        "grep",
			Description: "Busca uma regex nas linhas dos arquivos. Opcionalmente filtra por glob de caminho.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern":  stringProp("Regex (sintaxe Go) a buscar."),
					"path_glob": stringProp("Glob opcional de caminho (ex.: *.go). Vazio = todos."),
				},
				"required": []string{"pattern"},
			},
		},
	}
}

// toolExecutor runs tool calls against a CodeWorkspace.
type toolExecutor struct {
	ws domain.CodeWorkspace
}

func newToolExecutor(ws domain.CodeWorkspace) *toolExecutor {
	return &toolExecutor{ws: ws}
}

func (e *toolExecutor) run(name string, input map[string]any) string {
	switch name {
	case "read_file":
		b, err := e.ws.ReadFile(asString(input["path"]))
		if err != nil {
			return "error: " + err.Error()
		}
		return truncate(string(b))
	case "list_dir":
		entries, err := e.ws.ListDir(asString(input["path"]))
		if err != nil {
			return "error: " + err.Error()
		}
		var b strings.Builder
		for _, en := range entries {
			if en.IsDir {
				fmt.Fprintf(&b, "%s/\n", en.Name)
			} else {
				fmt.Fprintf(&b, "%s\n", en.Name)
			}
		}
		return truncate(b.String())
	case "grep":
		hits, err := e.ws.Grep(asString(input["pattern"]), asString(input["path_glob"]))
		if err != nil {
			return "error: " + err.Error()
		}
		var b strings.Builder
		for _, h := range hits {
			fmt.Fprintf(&b, "%s:%d: %s\n", h.Path, h.Line, h.Text)
		}
		if b.Len() == 0 {
			return "(sem resultados)"
		}
		return truncate(b.String())
	default:
		return "error: unknown tool " + name
	}
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func truncate(s string) string {
	if len(s) <= maxToolOutputBytes {
		return s
	}
	return s[:maxToolOutputBytes] + "\n[...truncado...]"
}

// agenticStep is one response from a driver: either a final answer or tool calls.
type agenticStep struct {
	Final string
	Calls []toolCall
	Usage domain.TokenUsage
}

// agenticDriver owns the provider-specific conversation. step sends accumulated
// tool results (nil on the first call) and returns the model's next move.
type agenticDriver interface {
	step(results []toolResult) (agenticStep, error)
}

// runAgenticLoop drives a tool-use conversation to a final review.
func runAgenticLoop(driver agenticDriver, exec *toolExecutor, maxIter int) (*domain.ReviewResult, error) {
	if maxIter <= 0 {
		maxIter = defaultAgenticMaxIterations
	}
	var usage domain.TokenUsage
	var results []toolResult

	for i := 0; i < maxIter; i++ {
		st, err := driver.step(results)
		if err != nil {
			return nil, err
		}
		usage.PromptTokens += st.Usage.PromptTokens
		usage.CompletionTokens += st.Usage.CompletionTokens
		usage.TotalTokens += st.Usage.TotalTokens

		if strings.TrimSpace(st.Final) != "" {
			body, comments := parseReviewPayload(st.Final)
			return &domain.ReviewResult{Body: body, InlineComments: comments, Usage: usage}, nil
		}
		if len(st.Calls) == 0 {
			return nil, fmt.Errorf("ai: agentic model returned neither tool calls nor a final answer")
		}

		results = make([]toolResult, 0, len(st.Calls))
		for _, c := range st.Calls {
			results = append(results, toolResult{ID: c.ID, Name: c.Name, Output: exec.run(c.Name, c.Input)})
		}
	}
	return nil, fmt.Errorf("ai: agentic loop exceeded %d iterations", maxIter)
}
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/ai/ -run 'TestTool' -v`
Expected: PASS (4 testes).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/ai/agentic.go internal/infrastructure/ai/agentic_test.go
git commit -m "feat(ai): tool specs + executor sobre CodeWorkspace + loop agêntico genérico"
```

---

## Task 2: Testar o loop genérico com um driver fake

**Files:**
- Modify: `internal/infrastructure/ai/agentic_test.go`

- [ ] **Step 1: Escrever os testes que falham**

Adicionar a `internal/infrastructure/ai/agentic_test.go`:

```go
// scriptedDriver returns a predefined sequence of steps, recording the tool
// results it receives so tests can assert the loop fed them back.
type scriptedDriver struct {
	steps    []agenticStep
	idx      int
	gotInput [][]toolResult
}

func (d *scriptedDriver) step(results []toolResult) (agenticStep, error) {
	d.gotInput = append(d.gotInput, results)
	st := d.steps[d.idx]
	d.idx++
	return st, nil
}

func TestRunAgenticLoopExecutesToolsThenFinal(t *testing.T) {
	driver := &scriptedDriver{steps: []agenticStep{
		{Calls: []toolCall{{ID: "a", Name: "read_file", Input: map[string]any{"path": "main.go"}}}, Usage: domain.TokenUsage{TotalTokens: 10}},
		{Final: `{"review_markdown":"tudo certo","inline_comments":[]}`, Usage: domain.TokenUsage{TotalTokens: 5}},
	}}
	exec := newToolExecutor(&fakeWorkspace{files: map[string]string{"main.go": "package main\n"}})

	res, err := runAgenticLoop(driver, exec, 12)
	if err != nil {
		t.Fatalf("runAgenticLoop: %v", err)
	}
	if res.Body != "tudo certo" {
		t.Fatalf("body = %q", res.Body)
	}
	if res.Usage.TotalTokens != 15 {
		t.Fatalf("usage accumulation = %d, want 15", res.Usage.TotalTokens)
	}
	// second step must have received the tool result from step 1
	if len(driver.gotInput) != 2 || len(driver.gotInput[1]) != 1 || driver.gotInput[1][0].ID != "a" {
		t.Fatalf("loop did not feed tool results back: %+v", driver.gotInput)
	}
}

func TestRunAgenticLoopExceedsIterations(t *testing.T) {
	// always returns a tool call, never finalizes
	loopStep := agenticStep{Calls: []toolCall{{ID: "x", Name: "list_dir", Input: map[string]any{"path": "."}}}}
	driver := &scriptedDriver{steps: []agenticStep{loopStep, loopStep, loopStep}}
	exec := newToolExecutor(&fakeWorkspace{})

	if _, err := runAgenticLoop(driver, exec, 2); err == nil {
		t.Fatal("expected error when loop exceeds max iterations")
	}
}
```

- [ ] **Step 2: Rodar e ver falhar/passar**

Run: `go test ./internal/infrastructure/ai/ -run TestRunAgenticLoop -v`
Expected: PASS (a implementação do loop já existe da Task 1; estes testes a exercitam). Se algum falhar, corrija o loop na Task 1.

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/ai/agentic_test.go
git commit -m "test(ai): cobre o loop agêntico com driver fake (feedback de tools, teto de iterações)"
```

---

## Task 3: Driver Anthropic (api + account)

**Files:**
- Create: `internal/infrastructure/ai/anthropic_agentic.go`
- Create: `internal/infrastructure/ai/anthropic_agentic_test.go`

- [ ] **Step 1: Escrever o teste que falha (httptest)**

Criar `internal/infrastructure/ai/anthropic_agentic_test.go`:

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

func TestAnthropicAgenticDriverToolThenFinal(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)

		call++
		if call == 1 {
			// must include tools
			if _, ok := req["tools"]; !ok {
				t.Errorf("first request missing tools")
			}
			w.Write([]byte(`{
				"content":[
					{"type":"text","text":"vou olhar o arquivo"},
					{"type":"tool_use","id":"tu_1","name":"read_file","input":{"path":"main.go"}}
				],
				"stop_reason":"tool_use",
				"usage":{"input_tokens":100,"output_tokens":20}
			}`))
			return
		}
		// second request must carry the tool_result for tu_1
		if !strings.Contains(string(body), "tool_result") || !strings.Contains(string(body), "tu_1") {
			t.Errorf("second request missing tool_result for tu_1: %s", body)
		}
		w.Write([]byte(`{
			"content":[{"type":"text","text":"{\"review_markdown\":\"ok agentic\",\"inline_comments\":[]}"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":50,"output_tokens":10}
		}`))
	}))
	defer srv.Close()

	c := &Client{
		transportKind: providerAnthropic,
		mode:          domain.AIModeAPI,
		baseURL:       srv.URL,
		token:         "k",
		model:         "claude-sonnet-4-5",
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
	if res.Body != "ok agentic" {
		t.Fatalf("body = %q", res.Body)
	}
	if res.Usage.TotalTokens != 180 { // 120 + 60
		t.Fatalf("usage = %d, want 180", res.Usage.TotalTokens)
	}
}
```

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/ai/ -run TestAnthropicAgentic -v`
Expected: FAIL — `c.agenticDriver` indefinido.

- [ ] **Step 3: Implementar o driver**

Criar `internal/infrastructure/ai/anthropic_agentic.go`:

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

func (d *anthropicAgenticDriver) step(results []toolResult) (agenticStep, error) {
	// Append previous tool results as a user message of tool_result blocks.
	if len(results) > 0 {
		blocks := make([]anthropicBlock, 0, len(results))
		for _, r := range results {
			blocks = append(blocks, anthropicBlock{Type: "tool_result", ToolUseID: r.ID, Content: r.Output})
		}
		d.messages = append(d.messages, anthropicBlockMessage{Role: "user", Content: blocks})
	}

	maxTokens := d.maxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	reqBody, err := json.Marshal(anthropicBlockRequest{
		Model:     d.model,
		MaxTokens: maxTokens,
		System:    d.system,
		Tools:     d.tools,
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
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/ai/ -run TestAnthropicAgentic -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/ai/anthropic_agentic.go internal/infrastructure/ai/anthropic_agentic_test.go
git commit -m "feat(ai): driver agêntico Anthropic (tool_use, api + account)"
```

---

## Task 4: Selecionar o caminho agêntico no Client.ReviewDiff

**Files:**
- Modify: `internal/infrastructure/ai/client.go`

- [ ] **Step 1: Escrever o teste que falha**

Adicionar a `internal/infrastructure/ai/agentic_test.go`:

```go
func TestReviewDiffFallsBackWhenAgenticUnsupported(t *testing.T) {
	// chat_completions has no agentic driver in this milestone → must fall back
	// to single-shot review (not error). Use an httptest server for chat.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"review_markdown\":\"single shot\",\"inline_comments\":[]}"}}],"usage":{"total_tokens":7}}`))
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
	res, err := c.ReviewDiff(domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{Title: "t"},
		Diff:         "--- a/x\n+++ b/x\n@@ -1 +1 @@\n+x",
		Profile:      &domain.ReviewProfile{Agentic: true},
		Workspace:    &fakeWorkspace{},
	})
	if err != nil {
		t.Fatalf("expected graceful fallback, got error: %v", err)
	}
	if res.Body != "single shot" {
		t.Fatalf("body = %q, want single shot", res.Body)
	}
}
```

(Confirme que `agentic_test.go` importa `net/http`, `net/http/httptest`, `time` — adicione ao import se necessário.)

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/infrastructure/ai/ -run TestReviewDiffFallsBack -v`
Expected: FAIL de compilação — `agenticDriver` ainda não existe em `client.go` e `ReviewDiff` não tem caminho agêntico.

- [ ] **Step 3: Implementar**

Em `internal/infrastructure/ai/client.go`:

(a) Substituir o corpo de `ReviewDiff` por:

```go
func (c *Client) ReviewDiff(input domain.ReviewInput) (*domain.ReviewResult, error) {
	sysPrompt, outputTpl := c.resolvePrompt(input)
	userPrompt := buildPrompt(input, outputTpl)

	if input.Workspace != nil && input.Profile != nil && input.Profile.Agentic {
		if result, err, ok := c.tryAgentic(input, sysPrompt, userPrompt); ok {
			return result, err
		}
		// no agentic driver for this provider → fall back to single-shot
	}

	t, err := c.transport()
	if err != nil {
		return nil, err
	}
	return t.review(sysPrompt, userPrompt)
}

// tryAgentic runs the agentic loop when a driver exists for the provider.
// The third return value is false when this provider has no agentic driver
// (caller should fall back to single-shot). When true, (result, err) is the
// real outcome of the agentic run.
func (c *Client) tryAgentic(input domain.ReviewInput, sysPrompt, userPrompt string) (*domain.ReviewResult, error, bool) {
	driver, err := c.agenticDriver(sysPrompt, userPrompt, readToolSpecs())
	if err != nil {
		return nil, nil, false
	}
	maxIter := 0
	if input.Profile != nil {
		maxIter = input.Profile.MaxIterations
	}
	result, runErr := runAgenticLoop(driver, newToolExecutor(input.Workspace), maxIter)
	return result, runErr, true
}

// agenticDriver builds the provider-specific tool-use driver. Returns an error
// for providers that don't yet support agentic review.
func (c *Client) agenticDriver(sysPrompt, userPrompt string, tools []toolSpec) (agenticDriver, error) {
	switch c.transportKind {
	case providerAnthropic:
		return newAnthropicAgenticDriver(c, sysPrompt, userPrompt, tools), nil
	default:
		return nil, fmt.Errorf("ai: agentic mode not supported for provider %q", c.transportKind)
	}
}
```

- [ ] **Step 4: Rodar e ver passar**

Run: `go test ./internal/infrastructure/ai/ -v`
Expected: PASS (todos: tools, loop, anthropic agentic, fallback + os testes antigos do pacote).

- [ ] **Step 5: Commit**

```bash
git add internal/infrastructure/ai/client.go
git commit -m "feat(ai): caminho agêntico em ReviewDiff com fallback gracioso para single-shot"
```

---

## Verificação final do milestone

- [ ] **Build:** `go build ./...` → sucesso.
- [ ] **Testes:** `go test ./...` → tudo verde, sem regressão.
- [ ] **Sanidade:** com `Profile.Agentic` + `Workspace` e provider Claude, `ReviewDiff` roda o loop de tool-use; com provider sem driver, cai no single-shot sem erro; sem `Workspace`/sem `Agentic`, o caminho é o single-shot de sempre.

## Notas para o executor

- **Não quebre o single-shot existente:** o `anthropicTransport.review` (string Content) continua existindo e em uso; o driver agêntico usa structs próprios (`anthropicBlock*`) com content em blocos. Não unifique os dois agora.
- **Histórico da conversa:** o driver Anthropic precisa registrar a resposta do assistant (text + tool_use) ANTES de anexar os tool_result, senão a API rejeita a sequência. Mantenha a ordem: user(prompt) → assistant(tool_use) → user(tool_result) → assistant(final).
- **Degradação graciosa vs erro real:** `agenticDriver()` retornar erro = "provider sem driver" → fallback. Já um erro DENTRO de `runAgenticLoop` (API caiu, estourou iterações) é erro real e deve subir (não mascarar).
- **maxIterations:** vem de `Profile.MaxIterations`; 0 → `defaultAgenticMaxIterations`. O override por env é o M4c.
- **Escopo das tools:** só `read_file`/`list_dir`/`grep` neste milestone. Não adicione `git_log`/`git_blame` (extensão futura) nem tools de execução (Fase 2).
- **M4b-2 (próximo):** drivers `chat_completions` (tool_calls) e `gemini` (functionDeclarations) plugam no mesmo `agenticDriver()` switch.
