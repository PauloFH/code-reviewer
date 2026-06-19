package ai

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// tokenRefresher is the subset of ClaudeOAuth the resolver depends on. Defined
// as an interface so tests can inject a stub.
type tokenRefresher interface {
	Refresh(refreshToken string) (*OAuthTokens, error)
}

// ResolvingReviewer implements domain.AIReviewer by selecting the AI config
// assigned to the most specific scope (project → client → global) at review
// time. When no config is assigned it falls back to the env-configured client.
type ResolvingReviewer struct {
	configRepo  domain.AIConfigRepository
	promptRepo  domain.PromptRepository
	oauth       tokenRefresher
	envClient   *Client
	envPriceIn  float64
	envPriceOut float64
	sandbox     Sandbox
	log         *slog.Logger
}

func NewResolvingReviewer(
	configRepo domain.AIConfigRepository,
	promptRepo domain.PromptRepository,
	oauth tokenRefresher,
	envClient *Client,
	envPriceIn, envPriceOut float64,
	sandbox Sandbox,
	log *slog.Logger,
) *ResolvingReviewer {
	return &ResolvingReviewer{
		configRepo:  configRepo,
		promptRepo:  promptRepo,
		oauth:       oauth,
		envClient:   envClient,
		envPriceIn:  envPriceIn,
		envPriceOut: envPriceOut,
		sandbox:     sandbox,
		log:         log,
	}
}

func (r *ResolvingReviewer) ReviewDiff(input domain.ReviewInput) (*domain.ReviewResult, error) {
	cfg, err := r.configRepo.ResolveConfig(input.ClientName, input.ProjectRef)
	if err != nil {
		r.log.Warn("resolving AI config, falling back to env config", slog.String("error", err.Error()))
		cfg = nil
	}

	if cfg == nil {
		r.envClient.sandbox = r.sandbox
		result, err := r.envClient.ReviewDiff(input)
		if err != nil {
			return nil, err
		}
		result.PricePerMInputTokens = r.envPriceIn
		result.PricePerMOutputTokens = r.envPriceOut
		return result, nil
	}

	if cfg.Mode == domain.AIModeAccount {
		if err := r.ensureValidToken(cfg); err != nil {
			return nil, err
		}
	}

	client := newClientFromConfig(*cfg, r.promptRepo)
	client.sandbox = r.sandbox
	result, err := client.ReviewDiff(input)
	if err != nil {
		return nil, err
	}
	result.PricePerMInputTokens = cfg.PricePerMInputTokens
	result.PricePerMOutputTokens = cfg.PricePerMOutputTokens
	return result, nil
}

// sampleTestDiff is a tiny diff with an obvious bug, used by TestConfig to
// validate a configuration end-to-end without a real merge request.
const sampleTestDiff = `diff --git a/math.go b/math.go
--- a/math.go
+++ b/math.go
@@ -1,3 +1,7 @@
 package math
+
+// Somar deveria retornar a soma de a e b.
+func Somar(a, b int) int {
+	return a - b
+}
`

// TestConfig runs a review of a built-in sample diff against a specific config,
// exercising the real auth path (API key or OAuth, including refresh). It is used
// by the "test configuration" endpoint to validate a config from the UI.
func (r *ResolvingReviewer) TestConfig(id int64) (*domain.ReviewResult, error) {
	cfg, err := r.configRepo.GetConfig(id)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("ai: config %d not found", id)
	}

	if cfg.Mode == domain.AIModeAccount {
		if err := r.ensureValidToken(cfg); err != nil {
			return nil, err
		}
	}

	client := newClientFromConfig(*cfg, r.promptRepo)
	client.sandbox = r.sandbox
	result, err := client.ReviewDiff(domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{
			Title:       "Teste de configuração de IA",
			Description: "Diff de exemplo para validar a conexão e a geração de review.",
		},
		Diff: sampleTestDiff,
	})
	if err != nil {
		return nil, err
	}
	result.PricePerMInputTokens = cfg.PricePerMInputTokens
	result.PricePerMOutputTokens = cfg.PricePerMOutputTokens
	return result, nil
}

// ensureValidToken refreshes the OAuth access token when it is missing or within
// a minute of expiry, persisting the new tokens back to the repository.
func (r *ResolvingReviewer) ensureValidToken(cfg *domain.AIConfig) error {
	if cfg.OAuthAccessToken != "" && time.Until(cfg.OAuthExpiresAt) > time.Minute {
		return nil
	}
	if cfg.OAuthRefreshToken == "" {
		return fmt.Errorf("ai: account token expired and no refresh token available for config %q", cfg.Name)
	}

	tokens, err := r.oauth.Refresh(cfg.OAuthRefreshToken)
	if err != nil {
		return fmt.Errorf("ai: refreshing account token: %w", err)
	}

	cfg.OAuthAccessToken = tokens.AccessToken
	cfg.OAuthExpiresAt = tokens.ExpiresAt
	if tokens.RefreshToken != "" {
		cfg.OAuthRefreshToken = tokens.RefreshToken
	}

	if cfg.ID != 0 {
		if err := r.configRepo.UpdateOAuthTokens(cfg.ID, cfg.OAuthAccessToken, cfg.OAuthRefreshToken, cfg.OAuthExpiresAt); err != nil {
			r.log.Error("persisting refreshed oauth tokens", slog.String("error", err.Error()))
		}
	}
	return nil
}
