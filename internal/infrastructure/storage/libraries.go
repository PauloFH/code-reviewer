package storage

import (
	"database/sql"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func newPromptLibrary(db *sql.DB) scopedLibrary[domain.Prompt] {
	return scopedLibrary[domain.Prompt]{
		record: record[domain.Prompt]{
			db: db, table: "prompts",
			columns: []string{"name", "system_prompt", "output_template"},
			values: func(p domain.Prompt) ([]any, error) {
				return []any{p.Name, p.SystemPrompt, p.OutputTemplate}, nil
			},
			scan: func(s scanner) (domain.Prompt, error) {
				var p domain.Prompt
				err := s.Scan(&p.ID, &p.Name, &p.SystemPrompt, &p.OutputTemplate, &p.CreatedAt, &p.UpdatedAt)
				return p, err
			},
		},
		assignments: newAssignments(db, "prompt_assignments", "prompt_id", "prompts"),
	}
}

func newReviewProfileLibrary(db *sql.DB) scopedLibrary[domain.ReviewProfile] {
	return scopedLibrary[domain.ReviewProfile]{
		record: record[domain.ReviewProfile]{
			db: db, table: "review_profiles",
			columns: []string{"name", "include_related", "agentic", "execution", "max_iterations"},
			values: func(p domain.ReviewProfile) ([]any, error) {
				return []any{p.Name, p.IncludeRelated, p.Agentic, p.Execution, p.MaxIterations}, nil
			},
			scan: func(s scanner) (domain.ReviewProfile, error) {
				var p domain.ReviewProfile
				err := s.Scan(&p.ID, &p.Name, &p.IncludeRelated, &p.Agentic, &p.Execution, &p.MaxIterations, &p.CreatedAt, &p.UpdatedAt)
				return p, err
			},
		},
		assignments: newAssignments(db, "review_profile_assignments", "profile_id", "review_profiles"),
	}
}

type aiConfigLibrary struct {
	scopedLibrary[domain.AIConfig]
	r *SQLiteRepository
}

// newAIConfigLibrary stores api_key and the OAuth tokens encrypted.
func newAIConfigLibrary(r *SQLiteRepository) aiConfigLibrary {
	return aiConfigLibrary{
		r: r,
		scopedLibrary: scopedLibrary[domain.AIConfig]{
			record: record[domain.AIConfig]{
				db: r.db, table: "ai_configs",
				columns: []string{"name", "provider", "mode", "base_url", "model",
					"api_key_enc", "oauth_access_enc", "oauth_refresh_enc", "oauth_expires_at",
					"price_in", "price_out", "timeout_seconds", "max_tokens"},
				values: func(c domain.AIConfig) ([]any, error) {
					apiKey, access, refresh := c.APIKey, c.OAuthAccessToken, c.OAuthRefreshToken
					if err := r.encryptEach(&apiKey, &access, &refresh); err != nil {
						return nil, err
					}
					return []any{c.Name, c.Provider, c.Mode, c.BaseURL, c.Model,
						apiKey, access, refresh, nullTime(c.OAuthExpiresAt),
						c.PricePerMInputTokens, c.PricePerMOutputTokens, c.TimeoutSeconds, c.MaxTokens}, nil
				},
				scan: func(s scanner) (domain.AIConfig, error) {
					var c domain.AIConfig
					var apiKey, access, refresh string
					var expiresAt sql.NullTime
					err := s.Scan(&c.ID, &c.Name, &c.Provider, &c.Mode, &c.BaseURL, &c.Model,
						&apiKey, &access, &refresh, &expiresAt,
						&c.PricePerMInputTokens, &c.PricePerMOutputTokens, &c.TimeoutSeconds, &c.MaxTokens,
						&c.CreatedAt, &c.UpdatedAt)
					if err != nil {
						return c, err
					}
					c.OAuthExpiresAt = expiresAt.Time
					c.APIKey, c.OAuthAccessToken, c.OAuthRefreshToken = apiKey, access, refresh
					return c, r.decryptEach(&c.APIKey, &c.OAuthAccessToken, &c.OAuthRefreshToken)
				},
			},
			assignments: newAssignments(r.db, "ai_config_assignments", "config_id", "ai_configs"),
		},
	}
}

func (l aiConfigLibrary) UpdateOAuthTokens(id int64, access, refresh string, expiresAt time.Time) error {
	if err := l.r.encryptEach(&access, &refresh); err != nil {
		return err
	}
	return affectedOne(l.r.db.Exec(`UPDATE ai_configs SET oauth_access_enc=?, oauth_refresh_enc=?, oauth_expires_at=?, updated_at=? WHERE id=?`,
		access, refresh, nullTime(expiresAt), now(), id))
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}
