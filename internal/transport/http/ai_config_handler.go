package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ClaudeOAuthFlow is the subset of the Claude OAuth driver the handler needs.
// Implemented in the ai package and adapted at wiring time.
type ClaudeOAuthFlow interface {
	BuildAuthorizeURL() (authURL, state string, err error)
	Exchange(state, code string) (access, refresh string, expiresAt time.Time, err error)
}

// ConfigTester runs a sample review against a stored config to validate it.
// Implemented by the ai.ResolvingReviewer.
type ConfigTester interface {
	TestConfig(id int64) (*domain.ReviewResult, error)
}

// AIConfigHandler serves /api/v1/ai-configs routes.
type AIConfigHandler struct {
	repo   domain.AIConfigRepository
	oauth  ClaudeOAuthFlow
	tester ConfigTester
}

func NewAIConfigHandler(repo domain.AIConfigRepository, oauth ClaudeOAuthFlow, tester ConfigTester) *AIConfigHandler {
	return &AIConfigHandler{repo: repo, oauth: oauth, tester: tester}
}

// aiConfigDTO is the safe representation returned to clients: it never carries
// the API key or OAuth tokens, only whether they are present.
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

func toAIConfigDTO(c domain.AIConfig) aiConfigDTO {
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

func (h *AIConfigHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := h.repo.ListConfigs()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	out := make([]aiConfigDTO, 0, len(configs))
	for _, c := range configs {
		out = append(out, toAIConfigDTO(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AIConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	c, err := h.repo.GetConfig(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if c == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, toAIConfigDTO(*c))
}

func (h *AIConfigHandler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	var c domain.AIConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if msg, ok := validateConfig(c); !ok {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 120
	}
	// OAuth tokens are never set on create; they arrive through the connect flow.
	c.OAuthAccessToken = ""
	c.OAuthRefreshToken = ""
	c.OAuthExpiresAt = time.Time{}

	created, err := h.repo.CreateConfig(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, toAIConfigDTO(*created))
}

func (h *AIConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	existing, err := h.repo.GetConfig(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var c domain.AIConfig
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if msg, ok := validateConfig(c); !ok {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 120
	}

	// Preserve secrets that the form does not (and must not) resend: an empty
	// api_key keeps the stored one, and OAuth tokens are always managed by the
	// connect flow rather than this endpoint.
	if c.APIKey == "" {
		c.APIKey = existing.APIKey
	}
	c.OAuthAccessToken = existing.OAuthAccessToken
	c.OAuthRefreshToken = existing.OAuthRefreshToken
	c.OAuthExpiresAt = existing.OAuthExpiresAt

	updated, err := h.repo.UpdateConfig(id, c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, toAIConfigDTO(*updated))
}

func (h *AIConfigHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.DeleteConfig(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AIConfigHandler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	assignments, err := h.repo.ListConfigAssignments()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if assignments == nil {
		assignments = []domain.AIConfigAssignment{}
	}
	writeJSON(w, http.StatusOK, assignments)
}

func (h *AIConfigHandler) SetAssignment(w http.ResponseWriter, r *http.Request) {
	var a domain.AIConfigAssignment
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if a.Scope == "" || a.ConfigID == 0 {
		http.Error(w, "scope and config_id are required", http.StatusBadRequest)
		return
	}
	if a.Scope != "global" && a.Scope != "client" && a.Scope != "project" {
		http.Error(w, "scope must be global, client, or project", http.StatusBadRequest)
		return
	}
	if err := h.repo.SetConfigAssignment(a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AIConfigHandler) RemoveAssignment(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	clientName := r.URL.Query().Get("client_name")
	projectRef := r.URL.Query().Get("project_ref")
	if scope == "" {
		http.Error(w, "scope query param is required", http.StatusBadRequest)
		return
	}
	if err := h.repo.RemoveConfigAssignment(scope, clientName, projectRef); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// StartClaudeOAuth begins the Claude subscription connect flow, returning the
// authorize URL the user must open and the state to echo back on exchange.
func (h *AIConfigHandler) StartClaudeOAuth(w http.ResponseWriter, r *http.Request) {
	authURL, state, err := h.oauth.BuildAuthorizeURL()
	if err != nil {
		http.Error(w, "could not start oauth flow", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"authorize_url": authURL, "state": state})
}

// ExchangeClaudeOAuth swaps the pasted authorization code for tokens and stores
// them against the target config.
func (h *AIConfigHandler) ExchangeClaudeOAuth(w http.ResponseWriter, r *http.Request) {
	var body struct {
		State    string `json:"state"`
		Code     string `json:"code"`
		ConfigID int64  `json:"config_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.State == "" || body.Code == "" || body.ConfigID == 0 {
		http.Error(w, "state, code and config_id are required", http.StatusBadRequest)
		return
	}

	access, refresh, expiresAt, err := h.oauth.Exchange(body.State, body.Code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateOAuthTokens(body.ConfigID, access, refresh, expiresAt); err != nil {
		http.Error(w, "could not store tokens", http.StatusInternalServerError)
		return
	}

	c, err := h.repo.GetConfig(body.ConfigID)
	if err != nil || c == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, toAIConfigDTO(*c))
}

// TestConfig runs a sample review against the config to validate credentials and
// connectivity, returning the generated review and token usage.
func (h *AIConfigHandler) TestConfig(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	result, err := h.tester.TestConfig(id)
	if err != nil {
		// A failed test is an expected outcome (bad key, expired/invalid token,
		// model not allowed), not a gateway failure. Answer 200 with the cause so
		// the operator sees it: a 5xx here is intercepted by the CDN (Cloudflare),
		// which discards the body and shows a generic "502 Bad gateway" instead.
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"review_markdown": result.Body,
		"inline_comments": result.InlineComments,
		"usage":           result.Usage,
	})
}

func validateConfig(c domain.AIConfig) (string, bool) {
	if c.Name == "" {
		return "name is required", false
	}
	switch c.Provider {
	case domain.ProviderOpenAI, domain.ProviderClaude, domain.ProviderGemini, domain.ProviderDeepSeek:
	default:
		return "provider must be openai, claude, gemini or deepseek", false
	}
	switch c.Mode {
	case domain.AIModeAPI:
	case domain.AIModeAccount:
		if c.Provider != domain.ProviderClaude {
			return "account mode is only supported for claude", false
		}
	default:
		return "mode must be api or account", false
	}
	return "", true
}
