package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)


// transportKind identifies the wire protocol used to talk to a provider.
const (
	providerGemini          = "gemini"
	providerChatCompletions = "chat_completions"
	providerAnthropic       = "anthropic"
)

// defaultMaxTokens caps the AI response length for providers that require it.
const defaultMaxTokens = 4096

// transport executes a single review round-trip against one provider.
type transport interface {
	review(sysPrompt, userPrompt string) (*domain.ReviewResult, error)
}

// Client implements domain.AIReviewer for a single, fixed provider configuration.
// It resolves the prompt, builds the user message and delegates the HTTP call to
// the provider-specific transport.
type Client struct {
	transportKind string
	mode          domain.AIConfigMode
	baseURL       string
	token         string
	model         string
	maxTokens     int
	http          *http.Client
	promptRepo    domain.PromptRepository
	sandbox       Sandbox
}

// New builds a Client from the legacy env-based configuration. provider is
// "gemini" or "chat_completions" (with "openai"/"claude" accepted as aliases).
func New(provider, baseURL, token, model string, timeout time.Duration, promptRepo domain.PromptRepository) *Client {
	return &Client{
		transportKind: normalizeProvider(provider),
		mode:          domain.AIModeAPI,
		baseURL:       strings.TrimRight(baseURL, "/"),
		token:         token,
		model:         model,
		http:          &http.Client{Timeout: timeout},
		promptRepo:    promptRepo,
	}
}

// newClientFromConfig builds a Client from a frontend-managed AIConfig, applying
// per-provider defaults for base URL and model when unset.
func newClientFromConfig(cfg domain.AIConfig, promptRepo domain.PromptRepository) *Client {
	token := cfg.APIKey
	if cfg.Mode == domain.AIModeAccount {
		token = cfg.OAuthAccessToken
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Client{
		transportKind: transportKindForProvider(cfg.Provider),
		mode:          cfg.Mode,
		baseURL:       strings.TrimRight(baseURLForConfig(cfg), "/"),
		token:         token,
		model:         modelForConfig(cfg),
		maxTokens:     cfg.MaxTokens,
		http:          &http.Client{Timeout: timeout},
		promptRepo:    promptRepo,
	}
}

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
	execution := input.Profile != nil && input.Profile.Execution && c.sandbox != nil
	driver, err := c.agenticDriver(sysPrompt, userPrompt, toolSpecsFor(execution))
	if err != nil {
		return nil, nil, false
	}
	var sb Sandbox
	if execution {
		sb = c.sandbox
	}
	maxIter := 0
	if input.Profile != nil {
		maxIter = input.Profile.MaxIterations
	}
	result, runErr := runAgenticLoop(driver, newToolExecutor(input.Workspace, sb), maxIter)
	return result, runErr, true
}

// agenticDriver builds the provider-specific tool-use driver. Returns an error
// for providers that don't yet support agentic review.
func (c *Client) agenticDriver(sysPrompt, userPrompt string, tools []toolSpec) (agenticDriver, error) {
	switch c.transportKind {
	case providerAnthropic:
		return newAnthropicAgenticDriver(c, sysPrompt, userPrompt, tools), nil
	case providerChatCompletions:
		return newChatCompletionsAgenticDriver(c, sysPrompt, userPrompt, tools), nil
	case providerGemini:
		return newGeminiAgenticDriver(c, sysPrompt, userPrompt, tools), nil
	default:
		return nil, fmt.Errorf("ai: agentic mode not supported for provider %q", c.transportKind)
	}
}

// resolvePrompt returns the system prompt and output template for the review,
// falling back to the built-in defaults when no prompt is assigned.
func (c *Client) resolvePrompt(input domain.ReviewInput) (sysPrompt, outputTpl string) {
	sysPrompt = systemPrompt
	if c.promptRepo == nil {
		return sysPrompt, ""
	}
	resolved, err := c.promptRepo.ResolvePrompt(input.ClientName, input.ProjectRef)
	if err != nil || resolved == nil {
		return sysPrompt, ""
	}
	if resolved.SystemPrompt != "" {
		sysPrompt = htmlToPlainText(resolved.SystemPrompt)
	}
	if resolved.OutputTemplate != "" {
		outputTpl = htmlToPlainText(resolved.OutputTemplate)
	}
	return sysPrompt, outputTpl
}

func (c *Client) transport() (transport, error) {
	switch c.transportKind {
	case providerGemini:
		return &geminiTransport{baseURL: c.baseURL, apiKey: c.token, model: c.model, maxTokens: c.maxTokens, http: c.http}, nil
	case providerChatCompletions:
		return &chatCompletionsTransport{baseURL: c.baseURL, token: c.token, model: c.model, maxTokens: c.maxTokens, http: c.http}, nil
	case providerAnthropic:
		return &anthropicTransport{baseURL: c.baseURL, model: c.model, token: c.token, mode: c.mode, maxTokens: c.maxTokens, http: c.http}, nil
	default:
		return nil, fmt.Errorf("ai: unsupported provider %q", c.transportKind)
	}
}

// transportKindForProvider maps a user-facing provider to its wire protocol.
func transportKindForProvider(p domain.AIProvider) string {
	switch p {
	case domain.ProviderGemini:
		return providerGemini
	case domain.ProviderClaude:
		return providerAnthropic
	default: // openai, deepseek and any OpenAI-compatible vendor
		return providerChatCompletions
	}
}

func baseURLForConfig(cfg domain.AIConfig) string {
	if cfg.BaseURL != "" {
		return cfg.BaseURL
	}
	switch cfg.Provider {
	case domain.ProviderGemini:
		return "https://generativelanguage.googleapis.com/v1beta"
	case domain.ProviderClaude:
		return "https://api.anthropic.com"
	case domain.ProviderOpenAI:
		return "https://api.openai.com/v1"
	case domain.ProviderDeepSeek:
		return "https://api.deepseek.com/v1"
	default:
		return ""
	}
}

func modelForConfig(cfg domain.AIConfig) string {
	if cfg.Model != "" {
		return cfg.Model
	}
	switch cfg.Provider {
	case domain.ProviderGemini:
		return "gemini-flash-latest"
	case domain.ProviderClaude:
		return "claude-sonnet-4-5"
	case domain.ProviderOpenAI:
		return "gpt-4o"
	case domain.ProviderDeepSeek:
		return "deepseek-chat"
	default:
		return ""
	}
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(provider) {
	case "", providerChatCompletions, "chat-completions", "openai", "claude":
		return providerChatCompletions
	case providerGemini:
		return providerGemini
	default:
		return provider
	}
}

// reviewPayload is the JSON contract every provider's response is expected to
// follow. The model emits the review as separate text fields and NEVER writes
// code fences itself — the Mermaid fences are added by assembleReviewMarkdown.
// ReviewMarkdown is a legacy fallback for responses that still ship a single
// pre-rendered markdown body.
type reviewPayload struct {
	Guia              string `json:"guia"`
	DiagramaSequencia string `json:"diagrama_sequencia"`
	DiagramaClasses   string `json:"diagrama_classes"`
	Alteracoes        string `json:"alteracoes"`
	PontosAtencao     string `json:"pontos_atencao"`
	Sugestoes         string `json:"sugestoes"`
	Riscos            string `json:"riscos"`
	Testes            string `json:"testes"`
	Conclusao         string `json:"conclusao"`

	// Legacy single-body fallback.
	ReviewMarkdown string `json:"review_markdown"`

	InlineComments []domain.InlineComment `json:"inline_comments"`
}

// mermaidFenceOpen / codeFenceClose are the only place code fences are produced.
// They are added programmatically so the model never has to emit a triple
// backtick (which gets dropped when it builds JSON strings).
const (
	mermaidFenceOpen = "```mermaid"
	codeFenceClose   = "```"
)

// markdownSanitizer strips invisible characters that have broken Mermaid 11
// rendering: U+00A0 (non-breaking space) becomes a normal space, U+200B
// (zero-width space) and U+FEFF (BOM / zero-width no-break space) are removed.
var markdownSanitizer = strings.NewReplacer(
	"\u00a0", " ", // non-breaking space -> normal space
	"\u200b", "", // zero-width space
	"\ufeff", "", // BOM / zero-width no-break space
)

func sanitizeMarkdown(s string) string {
	return markdownSanitizer.Replace(s)
}

func parseReviewPayload(content string) (string, []domain.InlineComment) {
	trimmed := strings.TrimSpace(content)
	jsonText := stripOuterCodeFence(trimmed)

	var payload reviewPayload
	if err := json.Unmarshal([]byte(jsonText), &payload); err != nil {
		// Not the expected JSON envelope — post the raw text, sanitised.
		return sanitizeMarkdown(trimmed), nil
	}

	body := assembleReviewMarkdown(payload)
	if body == "" {
		body = sanitizeMarkdown(trimmed)
	}

	comments := make([]domain.InlineComment, 0, len(payload.InlineComments))
	for _, comment := range payload.InlineComments {
		comment.Path = strings.TrimSpace(comment.Path)
		comment.Body = sanitizeMarkdown(strings.TrimSpace(comment.Body))
		if comment.Path == "" || comment.Body == "" || comment.Line <= 0 {
			continue
		}
		comments = append(comments, comment)
		if len(comments) >= maxInlineComments {
			break
		}
	}

	return body, comments
}

// assembleReviewMarkdown renders the structured fields into the final markdown,
// wrapping the Mermaid diagrams in code fences here (never by the model). When
// the payload only carries the legacy review_markdown body, that is returned
// as-is (sanitised). The result is always sanitised before posting.
func assembleReviewMarkdown(p reviewPayload) string {
	if !p.hasStructuredContent() {
		return sanitizeMarkdown(strings.TrimSpace(p.ReviewMarkdown))
	}

	var b strings.Builder
	section := func(header, raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(header)
		b.WriteString("\n\n")
		b.WriteString(raw)
	}
	diagram := func(header, raw string) {
		// Defensive: drop any fence the model emitted anyway, leaving pure Mermaid.
		raw = strings.TrimSpace(stripOuterCodeFence(strings.TrimSpace(raw)))
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "mermaid"))
		if raw == "" {
			return
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(header)
		b.WriteString("\n\n")
		b.WriteString(mermaidFenceOpen)
		b.WriteByte('\n')
		b.WriteString(raw)
		b.WriteByte('\n')
		b.WriteString(codeFenceClose)
	}

	section("## Guia do Revisor", p.Guia)
	diagram("### Diagrama de sequência", p.DiagramaSequencia)
	diagram("### Diagrama de classes/componentes", p.DiagramaClasses)
	section("### Alterações por arquivo", p.Alteracoes)
	section("### Pontos de atenção", p.PontosAtencao)
	section("### Sugestões de melhoria", p.Sugestoes)
	section("### Riscos", p.Riscos)
	section("### Testes sugeridos", p.Testes)
	section("### Conclusão", p.Conclusao)

	return sanitizeMarkdown(b.String())
}

func (p reviewPayload) hasStructuredContent() bool {
	return p.Guia != "" || p.DiagramaSequencia != "" || p.DiagramaClasses != "" ||
		p.Alteracoes != "" || p.PontosAtencao != "" || p.Sugestoes != "" ||
		p.Riscos != "" || p.Testes != "" || p.Conclusao != ""
}

// stripOuterCodeFence removes a single OUTER code-fence wrapper (``` or ~~~,
// optionally with a language tag on the opening line) when the whole string is
// wrapped in one. It never touches fences in the interior — only the matching
// first and last fence lines — so nested fences are preserved.
func stripOuterCodeFence(s string) string {
	s = strings.TrimSpace(s)
	for _, fence := range []string{"```", "~~~"} {
		if !strings.HasPrefix(s, fence) || !strings.HasSuffix(s, fence) {
			continue
		}
		nl := strings.IndexByte(s, '\n')
		if nl == -1 {
			continue
		}
		inner := s[nl+1:] // drop the opening fence line (e.g. ```json)
		if !strings.HasSuffix(inner, fence) {
			continue
		}
		return strings.TrimSpace(inner[:len(inner)-len(fence)])
	}
	return s
}
