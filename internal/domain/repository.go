package domain

// ReviewRepository tracks which MRs have already been reviewed per client,
// guaranteeing idempotency across service restarts.
type ReviewRepository interface {
	AlreadyReviewed(clientName, projectRef string, mrIID int64, username string) (bool, error)
	MarkAsReviewed(clientName, projectRef string, mrIID, mrID int64, username string) error
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
	UpdateClient(c Client) (*Client, error)
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
}
