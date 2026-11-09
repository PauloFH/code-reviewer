package storage

import (
	"fmt"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const clientColumns = `id,name,platform,base_url,token,webhook_secret,username,created_at,updated_at,secrets_encrypted`

func (r *SQLiteRepository) CountClients() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(1) FROM clients`).Scan(&n)
	return n, err
}

func (r *SQLiteRepository) ListClients() ([]domain.Client, error) {
	rows, err := r.db.Query(`SELECT ` + clientColumns + ` FROM clients ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Client
	for rows.Next() {
		c, err := r.scanClient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) GetClient(name string) (*domain.Client, error) {
	c, err := r.scanClient(r.db.QueryRow(`SELECT `+clientColumns+` FROM clients WHERE name=?`, name))
	if err != nil {
		return nil, notFound(err)
	}
	return &c, nil
}

func (r *SQLiteRepository) CreateClient(c domain.Client) (*domain.Client, error) {
	if err := r.insertClient(`INSERT`, c); err != nil {
		return nil, err
	}
	return r.GetClient(c.Name)
}

// SeedClients inserts the clients whose name is not stored yet.
func (r *SQLiteRepository) SeedClients(clients []domain.Client) error {
	for _, c := range clients {
		if err := r.insertClient(`INSERT OR IGNORE`, c); err != nil {
			return fmt.Errorf("seeding client %q: %w", c.Name, err)
		}
	}
	return nil
}

func (r *SQLiteRepository) insertClient(verb string, c domain.Client) error {
	token, secret := c.Token, c.WebhookSecret
	if err := r.encryptEach(&token, &secret); err != nil {
		return fmt.Errorf("encrypting client secrets: %w", err)
	}
	t := now()
	_, err := r.db.Exec(verb+` INTO clients(name,platform,base_url,token,webhook_secret,username,created_at,updated_at,secrets_encrypted)
		VALUES(?,?,?,?,?,?,?,?,1)`,
		c.Name, c.Platform, c.BaseURL, token, secret, c.Username, t, t)
	if isUnique(err) {
		return fmt.Errorf("client %q: %w", c.Name, domain.ErrConflict)
	}
	return err
}

func (r *SQLiteRepository) UpdateClient(oldName string, c domain.Client) (*domain.Client, error) {
	token, secret := c.Token, c.WebhookSecret
	if err := r.encryptEach(&token, &secret); err != nil {
		return nil, fmt.Errorf("encrypting client secrets: %w", err)
	}
	err := affectedOne(r.db.Exec(`
		UPDATE clients SET name=?,platform=?,base_url=?,token=?,webhook_secret=?,username=?,updated_at=?,secrets_encrypted=1
		WHERE name=?`,
		c.Name, c.Platform, c.BaseURL, token, secret, c.Username, now(), oldName))
	if isUnique(err) {
		return nil, fmt.Errorf("client %q: %w", c.Name, domain.ErrConflict)
	}
	if err != nil {
		return nil, err
	}
	return r.GetClient(c.Name)
}

func (r *SQLiteRepository) DeleteClient(name string) error {
	return affectedOne(r.db.Exec(`DELETE FROM clients WHERE name=?`, name))
}

func (r *SQLiteRepository) scanClient(s scanner) (domain.Client, error) {
	var c domain.Client
	var encrypted bool
	err := s.Scan(&c.ID, &c.Name, &c.Platform, &c.BaseURL, &c.Token, &c.WebhookSecret, &c.Username, &c.CreatedAt, &c.UpdatedAt, &encrypted)
	if err != nil || !encrypted {
		return c, err
	}
	if err := r.decryptEach(&c.Token, &c.WebhookSecret); err != nil {
		return c, fmt.Errorf("decrypting secrets of client %q: %w", c.Name, err)
	}
	return c, nil
}

// encryptPlaintextClientSecrets encrypts the token and webhook_secret of rows
// saved before migration010; a no-op once every row is encrypted.
func (r *SQLiteRepository) encryptPlaintextClientSecrets() (int, error) {
	rows, err := r.db.Query(`SELECT name,token,webhook_secret FROM clients WHERE secrets_encrypted=0`)
	if err != nil {
		return 0, fmt.Errorf("listing plaintext client secrets: %w", err)
	}
	var pending []domain.Client
	for rows.Next() {
		var c domain.Client
		if err := rows.Scan(&c.Name, &c.Token, &c.WebhookSecret); err != nil {
			rows.Close()
			return 0, err
		}
		pending = append(pending, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, c := range pending {
		if err := r.encryptEach(&c.Token, &c.WebhookSecret); err != nil {
			return 0, fmt.Errorf("client %q: %w", c.Name, err)
		}
		if _, err := r.db.Exec(`UPDATE clients SET token=?,webhook_secret=?,secrets_encrypted=1 WHERE name=? AND secrets_encrypted=0`,
			c.Token, c.WebhookSecret, c.Name); err != nil {
			return 0, fmt.Errorf("encrypting client %q: %w", c.Name, err)
		}
	}
	if len(pending) > 0 {
		// O texto puro antigo ainda vive no WAL e em páginas liberadas antes do secure_delete: reescreve o arquivo.
		if _, err := r.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			return 0, fmt.Errorf("checkpointing after client encryption: %w", err)
		}
		if _, err := r.db.Exec(`VACUUM`); err != nil {
			return 0, fmt.Errorf("vacuuming after client encryption: %w", err)
		}
	}
	return len(pending), nil
}
