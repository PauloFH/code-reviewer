package gitlab

import (
	"encoding/json"
	"fmt"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ValidateWebhook checks the X-Gitlab-Token header against the configured secret.
func (c *Client) ValidateWebhook(w domain.RawWebhook) bool {
	return w.Headers["x-gitlab-token"] == c.secret
}

// ParseWebhook normalises a GitLab merge_request webhook into a domain.WebhookEvent.
func (c *Client) ParseWebhook(w domain.RawWebhook) (*domain.WebhookEvent, error) {
	var payload struct {
		ObjectKind string `json:"object_kind"`
		Project    struct {
			ID int64 `json:"id"`
		} `json:"project"`
		ObjectAttributes struct {
			ID     int64  `json:"id"`
			IID    int64  `json:"iid"`
			Action string `json:"action"`
		} `json:"object_attributes"`
	}

	if err := json.Unmarshal(w.Body, &payload); err != nil {
		return nil, fmt.Errorf("gitlab: parsing webhook body: %w", err)
	}

	if payload.ObjectKind != "merge_request" {
		return nil, fmt.Errorf("gitlab: unsupported object_kind %q", payload.ObjectKind)
	}

	return &domain.WebhookEvent{
		ClientName: c.name,
		ProjectRef: fmt.Sprintf("%d", payload.Project.ID),
		MRIID:      payload.ObjectAttributes.IID,
		MRID:       payload.ObjectAttributes.ID,
		Action:     payload.ObjectAttributes.Action,
	}, nil
}
