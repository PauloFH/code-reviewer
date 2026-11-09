// Package gitea implements Gitea, whose REST API and webhooks follow GitHub's
// dialect under /api/v1. projectRef is "owner/repo".
package gitea

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/forge"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/github"
)

type Client struct {
	forge.Identity
	api *forge.API
}

func New(c domain.Client, timeout time.Duration) *Client {
	return &Client{
		Identity: forge.Identity{Client: c},
		api: forge.NewAPI(strings.TrimRight(c.BaseURL, "/")+"/api/v1", timeout, func(req *http.Request) {
			req.Header.Set("Authorization", "token "+c.Token)
		}),
	}
}

func (c *Client) ValidateWebhook(w domain.RawWebhook) bool {
	return forge.ValidHMAC(c.Client.WebhookSecret, w.Body, w.Header("X-Gitea-Signature"), "")
}

func (c *Client) ParseWebhook(w domain.RawWebhook) (*domain.WebhookEvent, error) {
	return github.ParsePullRequestEvent(c.Client.Name, w.Header("X-Gitea-Event"), w.Body)
}

func (c *Client) GetMergeRequest(projectRef string, mrIID int64) (*domain.MergeRequest, error) {
	return github.GetPullRequest(c.api, projectRef, mrIID)
}

func (c *Client) GetMergeRequestDiff(projectRef string, mrIID int64) (string, error) {
	if _, _, err := forge.SplitOwnerRepo(projectRef); err != nil {
		return "", err
	}
	diff, err := c.api.GetText(fmt.Sprintf("/repos/%s/pulls/%d.diff", projectRef, mrIID))
	if err != nil {
		return "", fmt.Errorf("gitea GetMergeRequestDiff: %w", err)
	}
	return diff, nil
}

func (c *Client) CreateMergeRequestNote(projectRef string, mrIID int64, body string) error {
	return github.CreateIssueComment(c.api, projectRef, mrIID, body)
}

func (c *Client) CreateMergeRequestInlineComment(*domain.MergeRequest, domain.InlineComment) error {
	return domain.ErrInlineCommentsUnsupported
}
