package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ClaudeOAuthFlow is the Claude subscription connect flow.
type ClaudeOAuthFlow interface {
	BuildAuthorizeURL() (authURL, state string, err error)
	Exchange(state, code string) (*domain.OAuthTokens, error)
}

// ConfigTester runs a sample review against a stored config to validate it.
type ConfigTester interface {
	TestConfig(id int64) (*domain.ReviewResult, error)
}

type AIConfigHandler struct {
	repo   domain.AIConfigRepository
	oauth  ClaudeOAuthFlow
	tester ConfigTester
}

func NewAIConfigHandler(repo domain.AIConfigRepository, oauth ClaudeOAuthFlow, tester ConfigTester) *AIConfigHandler {
	return &AIConfigHandler{repo: repo, oauth: oauth, tester: tester}
}

// aiConfigDTO never carries the API key or OAuth tokens, only whether they are present.
type aiConfigDTO struct {
	ID                    int64               `json:"id"`
	Name                  string              `json:"name"`
	Provider              domain.AIProvider   `json:"provider"`
	Mode                  domain.AIConfigMode `json:"mode"`
	BaseURL               string              `json:"base_url"`
	Model                 string              `json:"model"`
	HasAPIKey             bool                `json:"has_api_key"`
	AccountConnected      bool                `json:"account_connected"`
	OAuthExpiresAt        time.Time           `json:"oauth_expires_at"`
	PricePerMInputTokens  float64             `json:"price_per_m_input_tokens"`
	PricePerMOutputTokens float64             `json:"price_per_m_output_tokens"`
	MaxTokens             int                 `json:"max_tokens"`
	TimeoutSeconds        int                 `json:"timeout_seconds"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

func toAIConfigDTO(c domain.AIConfig) any {
	return aiConfigDTO{
		ID:                    c.ID,
		Name:                  c.Name,
		Provider:              c.Provider,
		Mode:                  c.Mode,
		BaseURL:               c.BaseURL,
		Model:                 c.Model,
		HasAPIKey:             c.APIKey != "",
		AccountConnected:      c.OAuthAccessToken != "",
		OAuthExpiresAt:        c.OAuthExpiresAt,
		PricePerMInputTokens:  c.PricePerMInputTokens,
		PricePerMOutputTokens: c.PricePerMOutputTokens,
		MaxTokens:             c.MaxTokens,
		TimeoutSeconds:        c.TimeoutSeconds,
		CreatedAt:             c.CreatedAt,
		UpdatedAt:             c.UpdatedAt,
	}
}

func (h *AIConfigHandler) library() libraryHandler[domain.AIConfig] {
	return libraryHandler[domain.AIConfig]{lib: h.repo, prepare: h.keepSecrets, present: toAIConfigDTO}
}

// keepSecrets preserves what the form never resends — an empty api_key keeps
// the stored one, and OAuth tokens only change through the connect flow — but
// only while the config still points at the same endpoint.
func (h *AIConfigHandler) keepSecrets(id int64, c *domain.AIConfig) error {
	c.OAuthExpiresAt = time.Time{}
	if id == 0 {
		return nil
	}
	existing, err := h.repo.Get(id)
	if err != nil {
		return err
	}
	if !existing.SameEndpoint(*c) {
		if c.Mode == domain.AIModeAPI && c.APIKey == "" {
			return domain.Invalid("api_key must be entered again when the provider or base_url changes")
		}
		return nil // OAuth tokens stay behind; the account must be connected again
	}
	if c.APIKey == "" {
		c.APIKey = existing.APIKey
	}
	c.OAuthAccessToken = existing.OAuthAccessToken
	c.OAuthRefreshToken = existing.OAuthRefreshToken
	c.OAuthExpiresAt = existing.OAuthExpiresAt
	return nil
}

func (h *AIConfigHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, domain.AIProviders)
}

// StartClaudeOAuth returns the authorize URL the user must open and the state
// to echo back on exchange.
func (h *AIConfigHandler) StartClaudeOAuth(w http.ResponseWriter, r *http.Request) {
	authURL, state, err := h.oauth.BuildAuthorizeURL()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"authorize_url": authURL, "state": state})
}

// ExchangeClaudeOAuth swaps the pasted authorization code for tokens and stores
// them on the target config.
func (h *AIConfigHandler) ExchangeClaudeOAuth(w http.ResponseWriter, r *http.Request) {
	var body struct {
		State    string `json:"state"`
		Code     string `json:"code"`
		ConfigID int64  `json:"config_id"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.State == "" || body.Code == "" || body.ConfigID == 0 {
		writeError(w, domain.Invalid("state, code and config_id are required"))
		return
	}
	tokens, err := h.oauth.Exchange(body.State, body.Code)
	if err != nil {
		// The provider's error text stays in the log; the user gets what to do next.
		slog.Warn("claude oauth exchange failed", slog.Int64("config_id", body.ConfigID), slog.String("error", err.Error()))
		http.Error(w, "não foi possível conectar a conta: o código é inválido, expirou ou já foi usado — gere um novo link e tente de novo", http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateOAuthTokens(body.ConfigID, tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresAt); err != nil {
		writeError(w, err)
		return
	}
	c, err := h.repo.Get(body.ConfigID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAIConfigDTO(*c))
}

func (h *AIConfigHandler) TestConfig(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	result, err := h.tester.TestConfig(id)
	if err != nil {
		// A failed test is an expected outcome (bad key, expired/invalid token,
		// model not allowed), not a gateway failure. Answer 200 with the cause so
		// the operator sees it: a 5xx here is intercepted by the CDN (Cloudflare),
		// which discards the body and shows a generic "502 Bad gateway" instead.
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"review_markdown": result.Body,
		"inline_comments": result.InlineComments,
		"usage":           result.Usage,
	})
}
