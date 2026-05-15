package gitlab

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ValidateWebhook checks the X-Gitlab-Token header against the configured secret.
func (c *Client) ValidateWebhook(w domain.RawWebhook) bool {
	return w.Headers["x-gitlab-token"] == c.secret
}

// ParseWebhook normalises a GitLab webhook into a domain.WebhookEvent.
// Supports merge_request and note (comment) events.
func (c *Client) ParseWebhook(w domain.RawWebhook) (*domain.WebhookEvent, error) {
	var base struct {
		ObjectKind string `json:"object_kind"`
	}
	if err := json.Unmarshal(w.Body, &base); err != nil {
		return nil, fmt.Errorf("gitlab: parsing webhook body: %w", err)
	}

	switch base.ObjectKind {
	case "merge_request":
		return c.parseMergeRequestWebhook(w)
	case "note":
		return c.parseNoteWebhook(w)
	default:
		return nil, fmt.Errorf("gitlab: unsupported object_kind %q", base.ObjectKind)
	}
}

func (c *Client) parseMergeRequestWebhook(w domain.RawWebhook) (*domain.WebhookEvent, error) {
	var payload struct {
		Project struct {
			ID                int64  `json:"id"`
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
		ObjectAttributes struct {
			ID     int64  `json:"id"`
			IID    int64  `json:"iid"`
			Action string `json:"action"`
		} `json:"object_attributes"`
	}

	if err := json.Unmarshal(w.Body, &payload); err != nil {
		return nil, fmt.Errorf("gitlab: parsing merge_request webhook: %w", err)
	}

	projectRef := payload.Project.PathWithNamespace
	if projectRef == "" {
		projectRef = fmt.Sprintf("%d", payload.Project.ID)
	}

	return &domain.WebhookEvent{
		ClientName: c.name,
		ProjectRef: projectRef,
		MRIID:      payload.ObjectAttributes.IID,
		MRID:       payload.ObjectAttributes.ID,
		Action:     payload.ObjectAttributes.Action,
	}, nil
}

func (c *Client) parseNoteWebhook(w domain.RawWebhook) (*domain.WebhookEvent, error) {
	var payload struct {
		ObjectAttributes struct {
			Note         string `json:"note"`
			NoteableType string `json:"noteable_type"`
		} `json:"object_attributes"`
		MergeRequest struct {
			IID int64 `json:"iid"`
			ID  int64 `json:"id"`
		} `json:"merge_request"`
		Project struct {
			ID                int64  `json:"id"`
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}

	if err := json.Unmarshal(w.Body, &payload); err != nil {
		return nil, fmt.Errorf("gitlab: parsing note webhook: %w", err)
	}

	if payload.ObjectAttributes.NoteableType != "MergeRequest" {
		return nil, fmt.Errorf("gitlab: note not on MR, ignoring")
	}

	note := strings.TrimSpace(payload.ObjectAttributes.Note)
	if note != "/review" {
		return nil, fmt.Errorf("gitlab: note is not /review command, ignoring")
	}

	projectRef := payload.Project.PathWithNamespace
	if projectRef == "" {
		projectRef = fmt.Sprintf("%d", payload.Project.ID)
	}

	return &domain.WebhookEvent{
		ClientName:    c.name,
		ProjectRef:    projectRef,
		MRIID:         payload.MergeRequest.IID,
		MRID:          payload.MergeRequest.ID,
		Action:        "note",
		IsReanalyze:   true,
		CommentAuthor: payload.User.Username,
	}, nil
}
