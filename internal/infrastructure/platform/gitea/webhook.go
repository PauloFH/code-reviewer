package gitea

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ValidateWebhook verifies the X-Gitea-Signature header using HMAC-SHA256,
// mirroring GitHub's webhook signature scheme.
func (c *Client) ValidateWebhook(w domain.RawWebhook) bool {
	sig := w.Headers["x-gitea-signature"]
	if sig == "" {
		return false
	}
	expected := computeHMAC(c.secret, w.Body)
	return hmac.Equal([]byte(sig), []byte(expected))
}

// ParseWebhook normalises a Gitea pull_request webhook into a domain.WebhookEvent.
func (c *Client) ParseWebhook(w domain.RawWebhook) (*domain.WebhookEvent, error) {
	eventType := w.Headers["x-gitea-event"]
	if eventType != "pull_request" {
		return nil, fmt.Errorf("gitea: unsupported event %q", eventType)
	}

	var payload struct {
		Action     string `json:"action"`
		Number     int64  `json:"number"`
		PullRequest struct {
			ID int64 `json:"id"`
		} `json:"pull_request"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}

	if err := json.Unmarshal(w.Body, &payload); err != nil {
		return nil, fmt.Errorf("gitea: parsing webhook body: %w", err)
	}

	return &domain.WebhookEvent{
		ClientName: c.name,
		ProjectRef: payload.Repository.FullName,
		MRIID:      payload.Number,
		MRID:       payload.PullRequest.ID,
		Action:     payload.Action,
	}, nil
}

func computeHMAC(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
