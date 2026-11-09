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

func TestEncryptPlaintextClientSecretsMigratesLegacyRows(t *testing.T) {
	repo := newTestRepo(t)
	// Linha como o código antigo gravava: texto puro, secrets_encrypted=0 (default da migration010).
	if _, err := repo.db.Exec(`INSERT INTO clients(name,platform,base_url,token,webhook_secret,username,created_at,updated_at)
		VALUES('old','github','https://api.github.com','ghp-antigo','whsec-antigo','bot',datetime('now'),datetime('now'))`); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	got, err := repo.GetClient("old")
	if err != nil || got.Token != "ghp-antigo" {
		t.Fatalf("legacy row must stay readable before migration: %+v %v", got, err)
	}

	n, err := repo.EncryptPlaintextClientSecrets()
	if err != nil || n != 1 {
		t.Fatalf("EncryptPlaintextClientSecrets = %d, %v; want 1, nil", n, err)
	}
	if token, _, encrypted := rawClientSecrets(t, repo, "old"); !encrypted || token == "ghp-antigo" {
		t.Fatalf("legacy row still plaintext: token=%q encrypted=%v", token, encrypted)
	}
	got, err = repo.GetClient("old")
	if err != nil || got.Token != "ghp-antigo" || got.WebhookSecret != "whsec-antigo" {
		t.Fatalf("migration lost secrets: %+v %v", got, err)
	}

	if n, err := repo.EncryptPlaintextClientSecrets(); err != nil || n != 0 {
		t.Fatalf("second run = %d, %v; want 0, nil (no double encryption)", n, err)
	}
	if got, _ := repo.GetClient("old"); got.Token != "ghp-antigo" {
		t.Fatalf("second run corrupted token: %q", got.Token)
	}
}

func TestClientSecretsRequireEncryptionKey(t *testing.T) {
	repo, err := New(filepath.Join(t.TempDir(), "nokey.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	_, err = repo.CreateClient(domain.Client{Name: "x", Platform: "gitlab", BaseURL: "u", Token: "t", WebhookSecret: "s", Username: "bot"})
	if !errors.Is(err, errNoEncryptionKey) {
		t.Fatalf("CreateClient without key = %v; want errNoEncryptionKey", err)
	}
}

func TestClientSecretsWithWrongKeyFailInsteadOfReturningGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys.db")
	repo, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := repo.UseEncryptionKey(testEncKey); err != nil {
		t.Fatalf("UseEncryptionKey: %v", err)
	}
	if _, err := repo.CreateClient(domain.Client{Name: "x", Platform: "gitlab", BaseURL: "u", Token: "t", WebhookSecret: "s", Username: "bot"}); err != nil {
		t.Fatalf("CreateClient: %v", err)
	}
	_ = repo.Close()

	other, err := New(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = other.Close() })
	if err := other.UseEncryptionKey("AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA="); err != nil {
		t.Fatalf("UseEncryptionKey(other): %v", err)
	}
	if _, err := other.ListClients(); err == nil {
		t.Fatal("ListClients with the wrong key must fail")
	}
	// A linha já cifrada não pode ser cifrada de novo com a chave errada.
	if n, err := other.EncryptPlaintextClientSecrets(); err != nil || n != 0 {
		t.Fatalf("EncryptPlaintextClientSecrets with wrong key = %d, %v; want 0, nil", n, err)
	}
}
