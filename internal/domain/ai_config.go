package domain

import "time"

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

// AIConfigAssignment links an AIConfig to a scope in the resolution hierarchy:
// "global" (fallback), "client" (per Git provider), or "project" (per repo).
// Mirrors PromptAssignment.
type AIConfigAssignment struct {
	ID         int64  `json:"id"`
	ConfigID   int64  `json:"config_id"`
	ConfigName string `json:"config_name,omitempty"`
	Scope      string `json:"scope"`
	ClientName string `json:"client_name"`
	ProjectRef string `json:"project_ref"`
}
