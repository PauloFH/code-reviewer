package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeJSON(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "clients*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoadClients_ValidConfig(t *testing.T) {
	path := writeJSON(t, `[
		{
			"name": "my-gitlab",
			"platform": "gitlab",
			"base_url": "https://gitlab.com",
			"token": "glpat-xxx",
			"webhook_secret": "secret",
			"username": "alice"
		}
	]`)

	clients, err := LoadClients(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}
	if clients[0].Name != "my-gitlab" {
		t.Errorf("Name = %q, want %q", clients[0].Name, "my-gitlab")
	}
}

func TestLoadClients_MultipleClients(t *testing.T) {
	path := writeJSON(t, `[
		{"name":"gl","platform":"gitlab","base_url":"https://gitlab.com","token":"t","webhook_secret":"s","username":"u"},
		{"name":"gh","platform":"github","base_url":"https://api.github.com","token":"t","webhook_secret":"s","username":"u"},
		{"name":"gt","platform":"gitea","base_url":"https://gitea.example.com","token":"t","webhook_secret":"s","username":"u"}
	]`)

	clients, err := LoadClients(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 3 {
		t.Errorf("expected 3 clients, got %d", len(clients))
	}
}

func TestLoadClients_AllSelfHostedVariants(t *testing.T) {
	path := writeJSON(t, `[
		{"name":"gl-sh","platform":"gitlab-selfhosted","base_url":"https://git.corp.com","token":"t","webhook_secret":"s","username":"u"},
		{"name":"gt-sh","platform":"gitea-selfhosted","base_url":"https://gitea.corp.com","token":"t","webhook_secret":"s","username":"u"}
	]`)

	if _, err := LoadClients(path); err != nil {
		t.Fatalf("unexpected error for self-hosted variants: %v", err)
	}
}

func TestLoadClients_DuplicateName_Error(t *testing.T) {
	path := writeJSON(t, `[
		{"name":"dup","platform":"gitlab","base_url":"https://gitlab.com","token":"t","webhook_secret":"s","username":"u"},
		{"name":"dup","platform":"github","base_url":"https://api.github.com","token":"t","webhook_secret":"s","username":"u"}
	]`)

	if _, err := LoadClients(path); err == nil {
		t.Error("expected error for duplicate client name")
	}
}

func TestLoadClients_UnknownPlatform_Error(t *testing.T) {
	path := writeJSON(t, `[
		{"name":"bad","platform":"bitbucket","base_url":"https://bitbucket.org","token":"t","webhook_secret":"s","username":"u"}
	]`)

	if _, err := LoadClients(path); err == nil {
		t.Error("expected error for unknown platform")
	}
}

func TestLoadClients_MissingName_Error(t *testing.T) {
	path := writeJSON(t, `[
		{"platform":"gitlab","base_url":"https://gitlab.com","token":"t","webhook_secret":"s","username":"u"}
	]`)

	if _, err := LoadClients(path); err == nil {
		t.Error("expected error for missing name")
	}
}

func TestLoadClients_MissingToken_Error(t *testing.T) {
	path := writeJSON(t, `[
		{"name":"gl","platform":"gitlab","base_url":"https://gitlab.com","webhook_secret":"s","username":"u"}
	]`)

	if _, err := LoadClients(path); err == nil {
		t.Error("expected error for missing token")
	}
}

func TestLoadClients_FileNotFound_Error(t *testing.T) {
	if _, err := LoadClients(filepath.Join(t.TempDir(), "nonexistent.json")); err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadClients_InvalidJSON_Error(t *testing.T) {
	path := writeJSON(t, `not valid json`)
	if _, err := LoadClients(path); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
