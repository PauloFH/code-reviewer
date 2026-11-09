package platform

import (
	"fmt"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/gitea"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/github"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/gitlab"
)

// Factory builds live platform clients from client records.
type Factory struct {
	Timeout         time.Duration
	GitLabReposPath string // see gitlab.New
}

func (f Factory) Build(c domain.Client) (domain.PlatformClient, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	switch c.Platform {
	case domain.PlatformGitLab, domain.PlatformGitLabSelfHosted:
		return gitlab.New(c, f.Timeout, f.GitLabReposPath), nil
	case domain.PlatformGitHub:
		return github.New(c, f.Timeout), nil
	case domain.PlatformGitea, domain.PlatformGiteaSelfHosted:
		return gitea.New(c, f.Timeout), nil
	}
	return nil, fmt.Errorf("no client implementation for platform %q", c.Platform)
}
