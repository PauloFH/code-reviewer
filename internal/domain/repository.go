package domain

import "time"

// ReviewRepository tracks which MRs have already been reviewed per client,
// guaranteeing idempotency across service restarts.
type ReviewRepository interface {
	AlreadyReviewed(clientName, projectRef string, mrIID int64, username string) (bool, error)
	MarkAsReviewed(clientName, projectRef string, mrIID, mrID int64, username string) error
	ClearReview(clientName, projectRef string, mrIID int64) error
}

// UserRepository manages user accounts and authentication.
type UserRepository interface {
	GetByUsername(username string) (*User, error)
	GetByID(id int64) (*User, error)
	ListUsers() ([]User, error)
	CreateUser(u User) (*User, error)
	UpdateUser(id int64, u User) (*User, error)
	DeleteUser(id int64) error
	CountUsers() (int, error)
	UserExists(username string) (bool, error)
}

// AIReviewer sends a diff to an AI API and returns the review text.
type AIReviewer interface {
	ReviewDiff(input ReviewInput) (*ReviewResult, error)
}

// ClientRepository manages persistent platform client configurations.
// It is the source of truth after the initial seed from clients.json.
type ClientRepository interface {
	ListClients() ([]Client, error)
	GetClient(name string) (*Client, error)
	CreateClient(c Client) (*Client, error)
	UpdateClient(oldName string, c Client) (*Client, error)
	DeleteClient(name string) error
	// SeedFromConfigs inserts clients that don't already exist (one-time migration).
	SeedFromConfigs(clients []Client) error
	// CountClients returns the number of persisted clients.
	CountClients() (int, error)
}

// ReviewEventRepository records and queries the per-review event log.
type ReviewEventRepository interface {
	SaveEvent(event ReviewEvent) error
	ListEvents(filter ReviewEventFilter) ([]ReviewEvent, error)
	CountEvents(filter ReviewEventFilter) (int64, error)
	GetStats(clientName string) (*ReviewStats, error)
	GetDailyCosts(clientName string, days int) ([]DailyCostPoint, error)
}

// AIConfigRepository manages the AI provider configuration library and
// scope-based assignments. Credentials are encrypted at rest by the implementation.
type AIConfigRepository interface {
	ListConfigs() ([]AIConfig, error)
	GetConfig(id int64) (*AIConfig, error)
	CreateConfig(c AIConfig) (*AIConfig, error)
	UpdateConfig(id int64, c AIConfig) (*AIConfig, error)
	DeleteConfig(id int64) error
	SetConfigAssignment(a AIConfigAssignment) error
	RemoveConfigAssignment(scope, clientName, projectRef string) error
	ListConfigAssignments() ([]AIConfigAssignment, error)
	ResolveConfig(clientName, projectRef string) (*AIConfig, error)
	UpdateOAuthTokens(id int64, access, refresh string, expiresAt time.Time) error
}

// PromptRepository manages the prompt library and scope-based assignments.
type PromptRepository interface {
	ListPrompts() ([]Prompt, error)
	GetPrompt(id int64) (*Prompt, error)
	CreatePrompt(p Prompt) (*Prompt, error)
	UpdatePrompt(id int64, p Prompt) (*Prompt, error)
	DeletePrompt(id int64) error
	SetAssignment(a PromptAssignment) error
	RemoveAssignment(scope, clientName, projectRef string) error
	ListAssignments() ([]PromptAssignment, error)
	ResolvePrompt(clientName, projectRef string) (*Prompt, error)
}
