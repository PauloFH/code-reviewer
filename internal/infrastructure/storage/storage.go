// Package storage implements the domain repositories on a single SQLite file.
package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mattn/go-sqlite3"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type SQLiteRepository struct {
	db  *sql.DB
	enc *encryptor
}

// Open opens (or creates) the database, migrates it and makes the stored
// secrets usable with encKey (base64 AES-256 key; empty = no secrets allowed):
// it fails when the key cannot decrypt what is stored, then encrypts client
// secrets still kept in plaintext by older versions.
func Open(path, encKey string, log *slog.Logger) (*SQLiteRepository, error) {
	// secure_delete zera o conteúdo apagado ou reescrito: o histórico expurgado e o token em texto puro
	// substituído pela versão cifrada não ficam legíveis nas páginas livres do arquivo.
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&_secure_delete=true")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite allows only one writer at a time

	r := &SQLiteRepository{db: db}
	if err := r.open(encKey, log); err != nil {
		db.Close()
		return nil, err
	}
	return r, nil
}

func (r *SQLiteRepository) open(encKey string, log *slog.Logger) error {
	if err := runMigrations(r.db); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	if encKey != "" {
		enc, err := newEncryptor(encKey)
		if err != nil {
			return err
		}
		r.enc = enc
	}
	if err := r.verifyKey(); err != nil {
		return err
	}
	n, err := r.encryptPlaintextClientSecrets()
	if err != nil {
		return err
	}
	if n > 0 {
		log.Info("encrypted plaintext client secrets", slog.Int("count", n))
	}
	return nil
}

// verifyKey decrypts one stored secret, if any. Checking before encrypting the
// plaintext client secrets matters: a wrong key would otherwise seal them with
// it, leaving them unreadable even after the key is fixed.
func (r *SQLiteRepository) verifyKey() error {
	var sample string
	err := r.db.QueryRow(`
		SELECT v FROM (
			SELECT api_key_enc AS v FROM ai_configs WHERE api_key_enc <> ''
			UNION ALL SELECT oauth_access_enc FROM ai_configs WHERE oauth_access_enc <> ''
			UNION ALL SELECT token FROM clients WHERE secrets_encrypted = 1 AND token <> ''
		) LIMIT 1`).Scan(&sample)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("sampling stored secrets: %w", err)
	}
	if _, err := r.decryptField(sample); err != nil {
		return fmt.Errorf("stored secrets cannot be decrypted with the configured key: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) Close() error { return r.db.Close() }

func (r *SQLiteRepository) Prompts() domain.PromptRepository { return newPromptLibrary(r.db) }

func (r *SQLiteRepository) ReviewProfiles() domain.ReviewProfileRepository {
	return newReviewProfileLibrary(r.db)
}

func (r *SQLiteRepository) AIConfigs() domain.AIConfigRepository { return newAIConfigLibrary(r) }

type scanner interface {
	Scan(dest ...any) error
}

func isUnique(err error) bool { return isConstraint(err, sqlite3.ErrConstraintUnique) }

func isConstraint(err error, code sqlite3.ErrNoExtended) bool {
	var se sqlite3.Error
	return errors.As(err, &se) && se.ExtendedCode == code
}

// affectedOne turns a write that matched no row into domain.ErrNotFound.
func affectedOne(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func now() time.Time { return time.Now().UTC() }
