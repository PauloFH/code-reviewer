package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// ClientConfig describes a single monitored Git platform instance.
// Multiple entries in clients.json allow monitoring GitLab, GitHub and
// Gitea (including self-hosted instances) simultaneously.
type ClientConfig struct {
	Name          string `json:"name"`           // unique identifier used in webhook URL
	Platform      string `json:"platform"`       // "gitlab" | "github" | "gitea"
	BaseURL       string `json:"base_url"`       // e.g. "https://gitlab.com"
	Token         string `json:"token"`          // API token
	WebhookSecret string `json:"webhook_secret"` // secret for validating incoming webhooks
	Username      string `json:"username"`       // username to watch for reviewer/assignee events
}

// LoadClients reads the JSON file at path and returns the list of client configs.
func LoadClients(path string) ([]ClientConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening clients config %q: %w", path, err)
	}
	defer f.Close()

	var clients []ClientConfig
	if err := json.NewDecoder(f).Decode(&clients); err != nil {
		return nil, fmt.Errorf("parsing clients config %q: %w", path, err)
	}

	if err := validateClients(clients); err != nil {
		return nil, err
	}

	return clients, nil
}

func validateClients(clients []ClientConfig) error {
	seen := make(map[string]struct{})
	for i, c := range clients {
		if c.Name == "" {
			return fmt.Errorf("clients[%d]: name is required", i)
		}
		if _, dup := seen[c.Name]; dup {
			return fmt.Errorf("clients[%d]: duplicate name %q", i, c.Name)
		}
		seen[c.Name] = struct{}{}

		switch c.Platform {
		case "gitlab", "github", "gitea":
		default:
			return fmt.Errorf("clients[%d] %q: unknown platform %q (must be gitlab, github or gitea)", i, c.Name, c.Platform)
		}
		if c.BaseURL == "" {
			return fmt.Errorf("clients[%d] %q: base_url is required", i, c.Name)
		}
		if c.Token == "" {
			return fmt.Errorf("clients[%d] %q: token is required", i, c.Name)
		}
		if c.WebhookSecret == "" {
			return fmt.Errorf("clients[%d] %q: webhook_secret is required", i, c.Name)
		}
		if c.Username == "" {
			return fmt.Errorf("clients[%d] %q: username is required", i, c.Name)
		}
	}
	return nil
}
