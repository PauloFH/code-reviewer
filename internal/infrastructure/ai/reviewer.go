package ai

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type configStore interface {
	Resolve(clientName, projectRef string) (*domain.AIConfig, error)
	Get(id int64) (*domain.AIConfig, error)
	UpdateOAuthTokens(id int64, access, refresh string, expiresAt time.Time) error
}

type promptStore interface {
	Resolve(clientName, projectRef string) (*domain.Prompt, error)
}

type tokenRefresher interface {
	Refresh(refreshToken string) (*domain.OAuthTokens, error)
}

// Reviewer implements domain.AIReviewer. Each review resolves the AIConfig of
// the most specific scope (project → client → global), or uses the env
// fallback, and talks to the provider through a per-review conversation.
type Reviewer struct {
	configs  configStore
	prompts  promptStore
	oauth    tokenRefresher
	fallback domain.AIConfig
	sandbox  Sandbox
	log      *slog.Logger
}

// NewReviewer builds the reviewer. prompts and sandbox may be nil; fallback is
// the env-configured AIConfig, whose prices also stand in for configs priced at 0.
func NewReviewer(configs configStore, prompts promptStore, oauth tokenRefresher, fallback domain.AIConfig, sandbox Sandbox, log *slog.Logger) *Reviewer {
	return &Reviewer{configs: configs, prompts: prompts, oauth: oauth, fallback: fallback, sandbox: sandbox, log: log}
}

func (r *Reviewer) ReviewDiff(input domain.ReviewInput) (*domain.ReviewResult, error) {
	cfg, err := r.configs.Resolve(input.ClientName, input.ProjectRef)
	if err != nil {
		// Never fall back here: that would send the code to another provider.
		return nil, fmt.Errorf("ai: resolving config: %w", err)
	}
	if cfg == nil {
		cfg = &r.fallback
	}
	return r.review(*cfg, input)
}

// sampleTestDiff is a tiny diff with an obvious bug, used by TestConfig.
const sampleTestDiff = `diff --git a/math.go b/math.go
--- a/math.go
+++ b/math.go
@@ -1,1 +1,6 @@
 package math
+
+// Somar deveria retornar a soma de a e b.
+func Somar(a, b int) int {
+	return a - b
+}
`

// TestConfig reviews sampleTestDiff with one specific config through the real
// auth path (API key or OAuth, including refresh), to validate it from the UI.
func (r *Reviewer) TestConfig(id int64) (*domain.ReviewResult, error) {
	cfg, err := r.configs.Get(id)
	if err != nil {
		return nil, fmt.Errorf("ai: loading config %d: %w", id, err)
	}
	return r.review(*cfg, domain.ReviewInput{
		MergeRequest: &domain.MergeRequest{
			Title:       "Teste de configuração de IA",
			Description: "Diff de exemplo para validar a conexão e a geração de review.",
		},
		Diff: sampleTestDiff,
	})
}

// review works on its own copy of cfg, so concurrent reviews share no state.
func (r *Reviewer) review(cfg domain.AIConfig, input domain.ReviewInput) (*domain.ReviewResult, error) {
	if cfg.Mode == domain.AIModeAccount {
		if err := r.ensureValidToken(&cfg); err != nil {
			return nil, err
		}
	}

	sysPrompt, outputTpl := r.resolvePrompt(input)
	diff := reviewDiff(input.Diff)
	userPrompt := buildPrompt(input, diff, outputTpl)

	var result *domain.ReviewResult
	var err error
	if input.Workspace != nil && input.Profile != nil && input.Profile.Agentic {
		var sb Sandbox
		if input.Profile.Execution {
			sb = r.sandbox
		}
		conv := newConversation(cfg, sysPrompt, userPrompt, toolSpecsFor(sb != nil))
		result, err = runAgenticLoop(conv, newToolExecutor(input.Workspace, sb), input.Profile.MaxIterations)
	} else {
		result, err = reviewOnce(newConversation(cfg, sysPrompt, userPrompt, nil))
	}
	if err != nil {
		return nil, err
	}

	result.InlineComments = validateInlineComments(result.InlineComments, diff)
	result.CostUSD = r.cost(cfg, result.Usage)
	return result, nil
}

// cost prices usage with the config's per-million-token prices; a price of 0
// means "unknown" and falls back to the env price.
func (r *Reviewer) cost(cfg domain.AIConfig, usage domain.TokenUsage) float64 {
	in, out := cfg.PricePerMInputTokens, cfg.PricePerMOutputTokens
	if in <= 0 {
		in = r.fallback.PricePerMInputTokens
	}
	if out <= 0 {
		out = r.fallback.PricePerMOutputTokens
	}
	return float64(usage.PromptTokens)/1_000_000*in + float64(usage.CompletionTokens)/1_000_000*out
}

// resolvePrompt returns the system prompt and output template for the review,
// falling back to the built-in defaults when no prompt is assigned.
func (r *Reviewer) resolvePrompt(input domain.ReviewInput) (sysPrompt, outputTpl string) {
	sysPrompt = systemPrompt
	if r.prompts == nil {
		return sysPrompt, ""
	}
	p, err := r.prompts.Resolve(input.ClientName, input.ProjectRef)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		r.log.Warn("resolving prompt, using the default prompt", slog.String("error", err.Error()))
	}
	if err != nil || p == nil {
		return sysPrompt, ""
	}
	if text := htmlToPlainText(p.SystemPrompt); text != "" {
		sysPrompt = text
	}
	if p.OutputTemplate != "" {
		outputTpl = htmlToPlainText(p.OutputTemplate)
	}
	return sysPrompt, outputTpl
}

// ensureValidToken refreshes the OAuth access token when it is missing or within
// a minute of expiry, persisting the new tokens.
func (r *Reviewer) ensureValidToken(cfg *domain.AIConfig) error {
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

	if err := r.configs.UpdateOAuthTokens(cfg.ID, cfg.OAuthAccessToken, cfg.OAuthRefreshToken, cfg.OAuthExpiresAt); err != nil {
		r.log.Error("persisting refreshed oauth tokens", slog.String("error", err.Error()))
	}
	return nil
}
