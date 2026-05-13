package domain

import "time"

// Client is the domain entity for a managed platform client.
// It mirrors config.ClientConfig but lives in the domain so repositories
// can return it without creating an import cycle with the config package.
type Client struct {
	ID            int64
	Name          string
	Platform      PlatformType
	BaseURL       string
	Token         string
	WebhookSecret string
	Username      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
