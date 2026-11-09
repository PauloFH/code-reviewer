package storage

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func rawClientSecrets(t *testing.T, repo *SQLiteRepository, name string) (token, secret string, encrypted bool) {
	t.Helper()
	err := repo.db.QueryRow(`SELECT token,webhook_secret,secrets_encrypted FROM clients WHERE name=?`, name).
		Scan(&token, &secret, &encrypted)
	if err != nil {
		t.Fatalf("raw select: %v", err)
	}
	return token, secret, encrypted
}

func TestClientSecretsAreEncryptedAtRest(t *testing.T) {
	repo := newTestRepo(t)
	c := domain.Client{Name: "acme", Platform: "gitlab", BaseURL: "https://gitlab.com", Token: "glpat-segredo", WebhookSecret: "whsec", Username: "bot"}
	if _, err := repo.CreateClient(c); err != nil {
		t.Fatalf("CreateClient: %v", err)
	}

	token, secret, encrypted := rawClientSecrets(t, repo, "acme")
	if !encrypted || strings.Contains(token, "glpat") || secret == "whsec" {
		t.Fatalf("secrets stored in plaintext: token=%q secret=%q encrypted=%v", token, secret, encrypted)
	}

	got, err := repo.GetClient("acme")
	if err != nil || got == nil {
		t.Fatalf("GetClient: %v %v", got, err)
	}
	if got.Token != "glpat-segredo" || got.WebhookSecret != "whsec" {
		t.Fatalf("round trip lost secrets: %+v", got)
	}

	c.Token = "glpat-novo"
	if _, err := repo.UpdateClient("acme", c); err != nil {
		t.Fatalf("UpdateClient: %v", err)
	}
	list, err := repo.ListClients()
	if err != nil || len(list) != 1 || list[0].Token != "glpat-novo" {
		t.Fatalf("ListClients after update: %+v %v", list, err)
	}
}

func TestOpenEncryptsLegacyPlaintextRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	repo := newTestRepoAt(t, path)
	// Linha como o código antigo gravava: texto puro, secrets_encrypted=0 (default da migration010).
	if _, err := repo.db.Exec(`INSERT INTO clients(name,platform,base_url,token,webhook_secret,username,created_at,updated_at)
		VALUES('old','github','https://api.github.com','ghp-antigo','whsec-antigo','bot',datetime('now'),datetime('now'))`); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	if got, err := repo.GetClient("old"); err != nil || got.Token != "ghp-antigo" {
		t.Fatalf("legacy row must stay readable before migration: %+v %v", got, err)
	}
	_ = repo.Close()

	for range 2 { // reopening must not encrypt twice
		repo = newTestRepoAt(t, path)
		if token, _, encrypted := rawClientSecrets(t, repo, "old"); !encrypted || token == "ghp-antigo" { //nolint:gosec // G101: test fixture, not a real credential
			t.Fatalf("legacy row still plaintext: token=%q encrypted=%v", token, encrypted)
		}
		if got, err := repo.GetClient("old"); err != nil || got.Token != "ghp-antigo" || got.WebhookSecret != "whsec-antigo" {
			t.Fatalf("migration lost secrets: %+v %v", got, err)
		}
		_ = repo.Close()
	}
}

func newTestRepoAt(t *testing.T, path string) *SQLiteRepository {
	t.Helper()
	repo, err := openAt(t, path, testEncKey)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return repo
}

func TestClientSecretsRequireEncryptionKey(t *testing.T) {
	repo, err := openAt(t, filepath.Join(t.TempDir(), "nokey.db"), "")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_, err = repo.CreateClient(domain.Client{Name: "x", Platform: "gitlab", BaseURL: "u", Token: "t", WebhookSecret: "s", Username: "bot"})
	if !errors.Is(err, ErrNoEncryptionKey) {
		t.Fatalf("CreateClient without key = %v; want ErrNoEncryptionKey", err)
	}
}

func TestOpenRefusesKeyThatCannotReadStoredSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.db")
	repo := newTestRepoAt(t, path)
	if _, err := repo.CreateClient(domain.Client{Name: "x", Platform: "gitlab", BaseURL: "u", Token: "t", WebhookSecret: "s", Username: "bot"}); err != nil {
		t.Fatalf("CreateClient: %v", err)
	}
	_ = repo.Close()

	if _, err := openAt(t, path, "AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA="); err == nil {
		t.Fatal("Open with the wrong key must fail")
	}
	if _, err := openAt(t, path, ""); err == nil {
		t.Fatal("Open without a key must fail when secrets are stored")
	}
}
