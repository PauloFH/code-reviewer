package domain

import (
	"strings"
	"time"
)

// AIConfigMode distinguishes how the service authenticates against the AI provider.
type AIConfigMode string

const (
	// AIModeAPI uses an API key (header auth) provided by the user.
	AIModeAPI AIConfigMode = "api"
	// AIModeAccount uses an OAuth token tied to a provider subscription.
	AIModeAccount AIConfigMode = "account"
)

// AIProvider enumerates the supported AI vendors.
type AIProvider string

const (
	ProviderOpenAI   AIProvider = "openai"
	ProviderClaude   AIProvider = "claude"
	ProviderGemini   AIProvider = "gemini"
	ProviderDeepSeek AIProvider = "deepseek"
)

// DefaultAITimeoutSeconds applies when an AIConfig leaves TimeoutSeconds unset.
const DefaultAITimeoutSeconds = 120

// AIProviderSpec describes a supported provider and the defaults used when an
// AIConfig leaves BaseURL or Model empty.
type AIProviderSpec struct {
	ID             AIProvider `json:"id"`
	Label          string     `json:"label"`
	DefaultBaseURL string     `json:"default_base_url"`
	DefaultModel   string     `json:"default_model"`
	// DefaultMaxTokens caps the answer when the config sets none; 0 lets the provider decide.
	DefaultMaxTokens int  `json:"default_max_tokens"`
	SupportsAccount  bool `json:"supports_account"`
}

// AIProviders is the single table of supported providers.
var AIProviders = []AIProviderSpec{
	{ID: ProviderOpenAI, Label: "OpenAI", DefaultBaseURL: "https://api.openai.com/v1", DefaultModel: "gpt-4o"},
	{ID: ProviderClaude, Label: "Claude (Anthropic)", DefaultBaseURL: "https://api.anthropic.com", DefaultModel: "claude-sonnet-4-5", DefaultMaxTokens: 4096, SupportsAccount: true},
	{ID: ProviderGemini, Label: "Google Gemini", DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta", DefaultModel: "gemini-flash-latest"},
	{ID: ProviderDeepSeek, Label: "DeepSeek", DefaultBaseURL: "https://api.deepseek.com/v1", DefaultModel: "deepseek-chat"},
}

// LookupAIProvider returns the spec of a supported provider.
func LookupAIProvider(id AIProvider) (AIProviderSpec, bool) {
	for _, p := range AIProviders {
		if p.ID == id {
			return p, true
		}
	}
	return AIProviderSpec{}, false
}

// OAuthTokens is the result of an OAuth authorization-code exchange or refresh.
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// AIConfig is a persisted, frontend-managed AI provider configuration.
// Credentials are held in plaintext in memory but encrypted at rest by the
// storage layer; the HTTP layer must never serialise the secret fields.
type AIConfig struct {
	ID                    int64        `json:"id"`
	Name                  string       `json:"name"`
	Provider              AIProvider   `json:"provider"`
	Mode                  AIConfigMode `json:"mode"`
	BaseURL               string       `json:"base_url"`
	Model                 string       `json:"model"`
	APIKey                string       `json:"api_key,omitempty"`
	OAuthAccessToken      string       `json:"-"`
	OAuthRefreshToken     string       `json:"-"`
	OAuthExpiresAt        time.Time    `json:"oauth_expires_at"`
	PricePerMInputTokens  float64      `json:"price_per_m_input_tokens"`
	PricePerMOutputTokens float64      `json:"price_per_m_output_tokens"`
	MaxTokens             int          `json:"max_tokens"`
	TimeoutSeconds        int          `json:"timeout_seconds"`
	CreatedAt             time.Time    `json:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at"`
}

// Validate checks the fields a config needs to be usable; failures are *InvalidError.
func (c AIConfig) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return Invalid("name is required")
	}
	spec, ok := LookupAIProvider(c.Provider)
	if !ok {
		ids := make([]string, len(AIProviders))
		for i, p := range AIProviders {
			ids[i] = string(p.ID)
		}
		return Invalid("provider must be one of: %s", strings.Join(ids, ", "))
	}
	switch c.Mode {
	case AIModeAPI:
	case AIModeAccount:
		if !spec.SupportsAccount {
			return Invalid("account mode is not supported for provider %s", c.Provider)
		}
	default:
		return Invalid("mode must be api or account")
	}
	return nil
}

// SameEndpoint reports whether o sends requests where c does. Stored
// credentials may only be reused for the same endpoint: otherwise editing the
// base URL would hand the saved key or OAuth token to whatever host it names.
func (c AIConfig) SameEndpoint(o AIConfig) bool {
	return c.Provider == o.Provider &&
		strings.TrimRight(c.BaseURLOrDefault(), "/") == strings.TrimRight(o.BaseURLOrDefault(), "/")
}

// BaseURLOrDefault returns BaseURL, or the provider's default when unset.
func (c AIConfig) BaseURLOrDefault() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	spec, _ := LookupAIProvider(c.Provider)
	return spec.DefaultBaseURL
}

// MaxTokensOrDefault returns MaxTokens, or the provider's default when unset.
func (c AIConfig) MaxTokensOrDefault() int {
	if c.MaxTokens > 0 {
		return c.MaxTokens
	}
	spec, _ := LookupAIProvider(c.Provider)
	return spec.DefaultMaxTokens
}

// ModelOrDefault returns Model, or the provider's default when unset.
func (c AIConfig) ModelOrDefault() string {
	if c.Model != "" {
		return c.Model
	}
	spec, _ := LookupAIProvider(c.Provider)
	return spec.DefaultModel
}
