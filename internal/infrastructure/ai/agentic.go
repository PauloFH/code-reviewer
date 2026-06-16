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

// Sandbox runs a shell command against the checked-out working tree in isolation.
// Implemented by internal/infrastructure/sandbox. exitCode is the command's exit
// status; err is non-nil only for sandbox/infra failures (not command failure).
type Sandbox interface {
	Run(workdir, command string) (output string, exitCode int, err error)
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
					"pattern":   stringProp("Regex (sintaxe Go) a buscar."),
					"path_glob": stringProp("Glob opcional de caminho (ex.: *.go). Vazio = todos."),
				},
				"required": []string{"pattern"},
			},
		},
	}
}

func runToolSpec() toolSpec {
	return toolSpec{
		Name:        "run",
		Description: "Executa um comando shell no repositório dentro de um sandbox isolado (sem rede, sem segredos, código somente-leitura). Use para rodar testes, build ou linters.",
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"command": map[string]any{"type": "string", "description": "Comando shell, ex.: 'go test ./...'"}},
			"required":   []string{"command"},
		},
	}
}

// toolSpecsFor returns the tools available to the agentic reviewer; the run tool
// is included only when sandboxed execution is enabled.
func toolSpecsFor(execution bool) []toolSpec {
	specs := readToolSpecs()
	if execution {
		specs = append(specs, runToolSpec())
	}
	return specs
}

// toolExecutor runs tool calls against a CodeWorkspace.
type toolExecutor struct {
	ws      domain.CodeWorkspace
	sandbox Sandbox
}

func newToolExecutor(ws domain.CodeWorkspace, sandbox Sandbox) *toolExecutor {
	return &toolExecutor{ws: ws, sandbox: sandbox}
}

func (e *toolExecutor) run(name string, input map[string]any) string {
	switch name {
	case "run":
		if e.sandbox == nil {
			return "error: execução em sandbox não habilitada"
		}
		out, code, err := e.sandbox.Run(e.ws.Root(), asString(input["command"]))
		if err != nil {
			return "error: " + err.Error()
		}
		return truncate(fmt.Sprintf("exit=%d\n%s", code, out))
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
