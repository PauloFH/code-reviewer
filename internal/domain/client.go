package domain

import "time"

// Client is the domain entity for a managed platform client.
// It mirrors config.ClientConfig but lives in the domain so repositories
// can return it without creating an import cycle with the config package.
type Client struct {
	ID            int64        `json:"id"`
	Name          string       `json:"name"`
	Platform      PlatformType `json:"platform"`
	BaseURL       string       `json:"base_url"`
	Token         string       `json:"token"`
	WebhookSecret string       `json:"webhook_secret"`
	Username      string       `json:"username"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}
