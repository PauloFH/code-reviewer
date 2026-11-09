package gitlab

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func (c *Client) ValidateWebhook(w domain.RawWebhook) bool {
	secret := c.Client.WebhookSecret
	return secret != "" && subtle.ConstantTimeCompare([]byte(w.Header("X-Gitlab-Token")), []byte(secret)) == 1
}

type project struct {
	ID                int64  `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
}

func (p project) ref() string {
	if p.PathWithNamespace != "" {
		return p.PathWithNamespace
	}
	return strconv.FormatInt(p.ID, 10)
}

// ParseWebhook accepts merge_request events and /review or #revisar notes on an MR.
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
		Project          project `json:"project"`
		ObjectAttributes struct {
			IID    int64  `json:"iid"`
			Action string `json:"action"`
		} `json:"object_attributes"`
	}

	if err := json.Unmarshal(w.Body, &payload); err != nil {
		return nil, fmt.Errorf("gitlab: parsing merge_request webhook: %w", err)
	}

	return &domain.WebhookEvent{
		ClientName: c.Client.Name,
		ProjectRef: payload.Project.ref(),
		MRIID:      payload.ObjectAttributes.IID,
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
		} `json:"merge_request"`
		Project project `json:"project"`
		User    struct {
			Username string `json:"username"`
		} `json:"user"`
	}

	if err := json.Unmarshal(w.Body, &payload); err != nil {
		return nil, fmt.Errorf("gitlab: parsing note webhook: %w", err)
	}

	if payload.ObjectAttributes.NoteableType != "MergeRequest" {
		return nil, fmt.Errorf("gitlab: note not on MR, ignoring")
	}

	event := &domain.WebhookEvent{
		ClientName:    c.Client.Name,
		ProjectRef:    payload.Project.ref(),
		MRIID:         payload.MergeRequest.IID,
		Action:        "note",
		IsReanalyze:   true,
		CommentAuthor: payload.User.Username,
	}

	note := strings.TrimSpace(payload.ObjectAttributes.Note)
	// "#revisar" alone or followed by whitespace and a justification.
	if strings.HasPrefix(strings.ToLower(note), "#revisar") {
		suffix := note[len("#revisar"):]
		if suffix == "" || strings.TrimLeft(suffix, " \t\r\n") != suffix {
			event.ReviewJustification = strings.TrimSpace(suffix)
			return event, nil
		}
	}
	fields := strings.Fields(note)
	if len(fields) == 0 || fields[0] != "/review" {
		return nil, fmt.Errorf("gitlab: note is not /review command, ignoring")
	}
	if len(fields) > 1 {
		event.RequestedProfile = strings.ToLower(fields[1])
	}
	return event, nil
}
