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
// follow: a markdown review plus optional structured inline comments.
type reviewPayload struct {
	ReviewMarkdown string                 `json:"review_markdown"`
	InlineComments []domain.InlineComment `json:"inline_comments"`
}

func parseReviewPayload(content string) (string, []domain.InlineComment) {
	trimmed := strings.TrimSpace(content)
	jsonText := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(trimmed, "```"), "```json"))
	jsonText = strings.TrimSpace(strings.TrimPrefix(jsonText, "```"))

	var payload reviewPayload
	if err := json.Unmarshal([]byte(jsonText), &payload); err != nil {
		return trimmed, nil
	}

	body := strings.TrimSpace(payload.ReviewMarkdown)
	if body == "" {
		body = trimmed
	}

	comments := make([]domain.InlineComment, 0, len(payload.InlineComments))
	for _, comment := range payload.InlineComments {
		comment.Path = strings.TrimSpace(comment.Path)
		comment.Body = strings.TrimSpace(comment.Body)
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
