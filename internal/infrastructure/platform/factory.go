package platform

import (
	"fmt"
	"time"

	"github.com/paulofh/code-reviewer/internal/config"
	"github.com/paulofh/code-reviewer/internal/domain"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/gitea"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/github"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/gitlab"
)

// Build creates a domain.PlatformClient from a ClientConfig entry.
// Adding support for a new platform requires only a new case here.
func Build(cfg config.ClientConfig, timeout time.Duration, reposPath string) (domain.PlatformClient, error) {
	switch cfg.Platform {
	case "gitlab":
		return gitlab.New(cfg.Name, cfg.BaseURL, cfg.Token, cfg.WebhookSecret, cfg.Username, timeout, reposPath), nil
	case "gitlab-selfhosted":
		return gitlab.NewSelfHosted(cfg.Name, cfg.BaseURL, cfg.Token, cfg.WebhookSecret, cfg.Username, timeout, reposPath), nil
	case "github":
		return github.New(cfg.Name, cfg.BaseURL, cfg.Token, cfg.WebhookSecret, cfg.Username, timeout), nil
	case "gitea":
		return gitea.New(cfg.Name, cfg.BaseURL, cfg.Token, cfg.WebhookSecret, cfg.Username, timeout), nil
	case "gitea-selfhosted":
		return gitea.NewSelfHosted(cfg.Name, cfg.BaseURL, cfg.Token, cfg.WebhookSecret, cfg.Username, timeout), nil
	default:
		return nil, fmt.Errorf("unknown platform %q for client %q", cfg.Platform, cfg.Name)
	}
}

// BuildAll constructs and registers all clients defined in configs.
func BuildAll(configs []config.ClientConfig, timeout time.Duration, reposPath string) (*Registry, error) {
	reg := NewRegistry()
	for _, cfg := range configs {
		client, err := Build(cfg, timeout, reposPath)
		if err != nil {
			return nil, err
		}
		reg.Register(client)
	}
	return reg, nil
}

// Factory implements usecase.ClientFactory using the platform-specific builders.
type Factory struct {
	reposPath string
}

func NewFactory(reposPath string) *Factory { return &Factory{reposPath: reposPath} }

// BuildFromDomain creates a domain.PlatformClient from a domain.Client entity.
func (f *Factory) BuildFromDomain(c domain.Client, timeout time.Duration) (domain.PlatformClient, error) {
	cfg := config.ClientConfig{
		Name:          c.Name,
		Platform:      string(c.Platform),
		BaseURL:       c.BaseURL,
		Token:         c.Token,
		WebhookSecret: c.WebhookSecret,
		Username:      c.Username,
	}
	return Build(cfg, timeout, f.reposPath)
}
