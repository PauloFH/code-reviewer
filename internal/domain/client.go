package domain

import "time"

// Client is a monitored Git platform instance, whether seeded from clients.json
// or managed through the API.
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

// Validate rejects a client that could not be built or would accept unsigned
// webhooks (an empty secret matches an absent GitLab token header).
func (c Client) Validate() error {
	switch {
	case c.Name == "":
		return Invalid("name is required")
	case !c.Platform.Valid():
		return Invalid("unknown platform %q", c.Platform)
	case c.BaseURL == "":
		return Invalid("base_url is required")
	case c.Token == "":
		return Invalid("token is required")
	case c.WebhookSecret == "":
		return Invalid("webhook_secret is required")
	case c.Username == "":
		return Invalid("username is required")
	}
	return nil
}
