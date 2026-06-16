package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// aiConfigColumns is the canonical column list for ai_configs selects.
const aiConfigColumns = `id, name, provider, mode, base_url, model,
	api_key_enc, oauth_access_enc, oauth_refresh_enc, oauth_expires_at,
	price_in, price_out, timeout_seconds, max_tokens, created_at, updated_at`

// aiConfigColumnsAliased is the same list with the "c." table alias, for the
// ResolveConfig JOINs where ai_config_assignments also has an "id" column
// (an unqualified "id" there is ambiguous and fails at runtime).
const aiConfigColumnsAliased = `c.id, c.name, c.provider, c.mode, c.base_url, c.model,
	c.api_key_enc, c.oauth_access_enc, c.oauth_refresh_enc, c.oauth_expires_at,
	c.price_in, c.price_out, c.timeout_seconds, c.max_tokens, c.created_at, c.updated_at`

func (r *SQLiteRepository) ListConfigs() ([]domain.AIConfig, error) {
	rows, err := r.db.Query(`SELECT ` + aiConfigColumns + ` FROM ai_configs ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing ai configs: %w", err)
	}
	defer rows.Close()

	var out []domain.AIConfig
	for rows.Next() {
		c, err := r.scanAIConfig(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) GetConfig(id int64) (*domain.AIConfig, error) {
	row := r.db.QueryRow(`SELECT `+aiConfigColumns+` FROM ai_configs WHERE id=?`, id)
	c, err := r.scanAIConfig(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *SQLiteRepository) CreateConfig(c domain.AIConfig) (*domain.AIConfig, error) {
	apiKeyEnc, accessEnc, refreshEnc, err := r.encryptSecrets(c)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO ai_configs(
		name, provider, mode, base_url, model,
		api_key_enc, oauth_access_enc, oauth_refresh_enc, oauth_expires_at,
		price_in, price_out, timeout_seconds, max_tokens, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.Name, string(c.Provider), string(c.Mode), c.BaseURL, c.Model,
		apiKeyEnc, accessEnc, refreshEnc, nullableTime(c.OAuthExpiresAt),
		c.PricePerMInputTokens, c.PricePerMOutputTokens, c.TimeoutSeconds, c.MaxTokens, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("ai config %q already exists", c.Name)
		}
		return nil, fmt.Errorf("creating ai config: %w", err)
	}
	id, _ := res.LastInsertId()
	c.ID = id
	c.CreatedAt = now
	c.UpdatedAt = now
	return &c, nil
}

func (r *SQLiteRepository) UpdateConfig(id int64, c domain.AIConfig) (*domain.AIConfig, error) {
	apiKeyEnc, accessEnc, refreshEnc, err := r.encryptSecrets(c)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	_, err = r.db.Exec(`UPDATE ai_configs SET
		name=?, provider=?, mode=?, base_url=?, model=?,
		api_key_enc=?, oauth_access_enc=?, oauth_refresh_enc=?, oauth_expires_at=?,
		price_in=?, price_out=?, timeout_seconds=?, max_tokens=?, updated_at=?
		WHERE id=?`,
		c.Name, string(c.Provider), string(c.Mode), c.BaseURL, c.Model,
		apiKeyEnc, accessEnc, refreshEnc, nullableTime(c.OAuthExpiresAt),
		c.PricePerMInputTokens, c.PricePerMOutputTokens, c.TimeoutSeconds, c.MaxTokens, now, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("ai config %q already exists", c.Name)
		}
		return nil, fmt.Errorf("updating ai config: %w", err)
	}
	c.ID = id
	c.UpdatedAt = now
	return &c, nil
}

func (r *SQLiteRepository) DeleteConfig(id int64) error {
	if _, err := r.db.Exec(`DELETE FROM ai_configs WHERE id=?`, id); err != nil {
		return fmt.Errorf("deleting ai config: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) UpdateOAuthTokens(id int64, access, refresh string, expiresAt time.Time) error {
	accessEnc, err := r.encryptField(access)
	if err != nil {
		return err
	}
	refreshEnc, err := r.encryptField(refresh)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`UPDATE ai_configs SET oauth_access_enc=?, oauth_refresh_enc=?, oauth_expires_at=?, updated_at=? WHERE id=?`,
		accessEnc, refreshEnc, nullableTime(expiresAt), time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("updating oauth tokens: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) SetConfigAssignment(a domain.AIConfigAssignment) error {
	_, err := r.db.Exec(`INSERT OR REPLACE INTO ai_config_assignments(config_id, scope, client_name, project_ref) VALUES(?,?,?,?)`,
		a.ConfigID, a.Scope, a.ClientName, a.ProjectRef)
	if err != nil {
		return fmt.Errorf("setting ai config assignment: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) RemoveConfigAssignment(scope, clientName, projectRef string) error {
	_, err := r.db.Exec(`DELETE FROM ai_config_assignments WHERE scope=? AND client_name=? AND project_ref=?`,
		scope, clientName, projectRef)
	if err != nil {
		return fmt.Errorf("removing ai config assignment: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListConfigAssignments() ([]domain.AIConfigAssignment, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.config_id, c.name, a.scope, a.client_name, a.project_ref
		FROM ai_config_assignments a
		JOIN ai_configs c ON c.id = a.config_id
		ORDER BY a.scope, a.client_name, a.project_ref`)
	if err != nil {
		return nil, fmt.Errorf("listing ai config assignments: %w", err)
	}
	defer rows.Close()

	var out []domain.AIConfigAssignment
	for rows.Next() {
		var a domain.AIConfigAssignment
		if err := rows.Scan(&a.ID, &a.ConfigID, &a.ConfigName, &a.Scope, &a.ClientName, &a.ProjectRef); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ResolveConfig returns the AI config assigned to the most specific scope that
// matches: project → client → global. Returns nil when nothing is assigned.
func (r *SQLiteRepository) ResolveConfig(clientName, projectRef string) (*domain.AIConfig, error) {
	queries := []struct {
		sql  string
		args []any
	}{
		{`SELECT ` + aiConfigColumnsAliased + ` FROM ai_configs c
			JOIN ai_config_assignments a ON a.config_id = c.id
			WHERE a.scope='project' AND a.client_name=? AND a.project_ref=?`, []any{clientName, projectRef}},
		{`SELECT ` + aiConfigColumnsAliased + ` FROM ai_configs c
			JOIN ai_config_assignments a ON a.config_id = c.id
			WHERE a.scope='client' AND a.client_name=? AND a.project_ref=''`, []any{clientName}},
		{`SELECT ` + aiConfigColumnsAliased + ` FROM ai_configs c
			JOIN ai_config_assignments a ON a.config_id = c.id
			WHERE a.scope='global'`, nil},
	}

	for _, q := range queries {
		row := r.db.QueryRow(q.sql, q.args...)
		c, err := r.scanAIConfig(row)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		return &c, nil
	}
	return nil, nil
}

func (r *SQLiteRepository) encryptSecrets(c domain.AIConfig) (apiKeyEnc, accessEnc, refreshEnc string, err error) {
	if apiKeyEnc, err = r.encryptField(c.APIKey); err != nil {
		return "", "", "", err
	}
	if accessEnc, err = r.encryptField(c.OAuthAccessToken); err != nil {
		return "", "", "", err
	}
	if refreshEnc, err = r.encryptField(c.OAuthRefreshToken); err != nil {
		return "", "", "", err
	}
	return apiKeyEnc, accessEnc, refreshEnc, nil
}

func (r *SQLiteRepository) scanAIConfig(s scanner) (domain.AIConfig, error) {
	var c domain.AIConfig
	var provider, mode string
	var apiKeyEnc, accessEnc, refreshEnc string
	var expiresAt sql.NullString
	var createdAt, updatedAt string

	err := s.Scan(&c.ID, &c.Name, &provider, &mode, &c.BaseURL, &c.Model,
		&apiKeyEnc, &accessEnc, &refreshEnc, &expiresAt,
		&c.PricePerMInputTokens, &c.PricePerMOutputTokens, &c.TimeoutSeconds, &c.MaxTokens,
		&createdAt, &updatedAt)
	if err != nil {
		return c, err
	}

	c.Provider = domain.AIProvider(provider)
	c.Mode = domain.AIConfigMode(mode)
	if c.APIKey, err = r.decryptField(apiKeyEnc); err != nil {
		return c, err
	}
	if c.OAuthAccessToken, err = r.decryptField(accessEnc); err != nil {
		return c, err
	}
	if c.OAuthRefreshToken, err = r.decryptField(refreshEnc); err != nil {
		return c, err
	}
	if expiresAt.Valid {
		c.OAuthExpiresAt = parseDBTime(expiresAt.String)
	}
	c.CreatedAt = parseDBTime(createdAt)
	c.UpdatedAt = parseDBTime(updatedAt)
	return c, nil
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}

func parseDBTime(s string) time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05Z", s)
	if t.IsZero() {
		t, _ = time.Parse("2006-01-02 15:04:05", s)
	}
	return t
}
