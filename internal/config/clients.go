package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// LoadClients reads the clients.json seed file.
func LoadClients(path string) ([]domain.Client, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("opening clients config %q: %w", path, err)
	}
	var clients []domain.Client
	if err := json.Unmarshal(data, &clients); err != nil {
		return nil, fmt.Errorf("parsing clients config %q: %w", path, err)
	}
	seen := make(map[string]bool)
	for i, c := range clients {
		if err := c.Validate(); err != nil {
			return nil, fmt.Errorf("clients[%d] %q: %w", i, c.Name, err)
		}
		if seen[c.Name] {
			return nil, fmt.Errorf("clients[%d]: duplicate name %q", i, c.Name)
		}
		seen[c.Name] = true
	}
	return clients, nil
}
