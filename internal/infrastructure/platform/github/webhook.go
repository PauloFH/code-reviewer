package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ValidateWebhook verifies the X-Hub-Signature-256 header using HMAC-SHA256.
func (c *Client) ValidateWebhook(w domain.RawWebhook) bool {
	sig := w.Headers["x-hub-signature-256"]
	if !strings.HasPrefix(sig, "sha256=") {
		return false
	}
	expected := "sha256=" + computeHMAC(c.secret, w.Body)
	return hmac.Equal([]byte(sig), []byte(expected))
}

// ParseWebhook normalises a GitHub pull_request webhook into a domain.WebhookEvent.
func (c *Client) ParseWebhook(w domain.RawWebhook) (*domain.WebhookEvent, error) {
	eventType := w.Headers["x-github-event"]
	if eventType != "pull_request" {
		return nil, fmt.Errorf("github: unsupported event %q", eventType)
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
		return nil, fmt.Errorf("github: parsing webhook body: %w", err)
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
