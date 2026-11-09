package domain

import "time"

// Lookups of a single record return ErrNotFound when it does not exist;
// writes return ErrConflict on a duplicate name and ErrNotFound for a missing id.

// ReviewRepository tracks which MRs have already been reviewed per client,
// guaranteeing idempotency across service restarts.
type ReviewRepository interface {
	ClaimReview(clientName, projectRef string, mrIID, mrID int64, username string) (bool, error)
	// GetReviewHistory returns nil when no review body is stored.
	GetReviewHistory(clientName, projectRef string, mrIID int64, username string) (*ReviewHistory, error)
	CompleteReview(clientName, projectRef string, mrIID int64, username, headSHA, body string) error
	ClearReview(clientName, projectRef string, mrIID int64) error
}

type UserRepository interface {
	GetByUsername(username string) (*User, error)
	GetByID(id int64) (*User, error)
	ListUsers() ([]User, error)
	CreateUser(u User) (*User, error)
	// UpdateUser keeps the stored password when u.Password is empty. UpdateUser
	// and DeleteUser return ErrLastSuper instead of leaving no super user.
	UpdateUser(id int64, u User) (*User, error)
	DeleteUser(id int64) error
	CountUsers() (int, error)
}

type AIReviewer interface {
	ReviewDiff(input ReviewInput) (*ReviewResult, error)
}

// ClientRepository is the source of truth for clients after the first-boot
// seed from clients.json.
type ClientRepository interface {
	ListClients() ([]Client, error)
	GetClient(name string) (*Client, error)
	CreateClient(c Client) (*Client, error)
	UpdateClient(oldName string, c Client) (*Client, error)
	DeleteClient(name string) error
	// SeedClients inserts the clients whose name is not stored yet.
	SeedClients(clients []Client) error
	CountClients() (int, error)
}

type ReviewEventRepository interface {
	SaveEvent(event ReviewEvent) error
	ListEvents(filter ReviewEventFilter) ([]ReviewEvent, error)
	CountEvents(filter ReviewEventFilter) (int64, error)
	// GetStats aggregates successful reviews; filter.Status is ignored.
	GetStats(filter ReviewEventFilter) (*ReviewStats, error)
	GetDailyCosts(clientName string, days int) ([]DailyCostPoint, error)
}

// Library is the CRUD of a named, id-keyed record type.
type Library[T any] interface {
	List() ([]T, error)
	Get(id int64) (*T, error)
	Create(v T) (*T, error)
	Update(id int64, v T) (*T, error)
	Delete(id int64) error
}

// Assignments places library records in the scope hierarchy.
type Assignments interface {
	ListAssignments() ([]Assignment, error)
	// SetAssignment normalizes the scope key and replaces whatever occupied it.
	SetAssignment(a Assignment) error
	RemoveAssignment(k ScopeKey) error
}

// ScopedLibrary is a library whose records are resolved by scope; Resolve
// returns nil when nothing applies to the project.
type ScopedLibrary[T any] interface {
	Library[T]
	Assignments
	Resolve(clientName, projectRef string) (*T, error)
}

type PromptRepository = ScopedLibrary[Prompt]

type ReviewProfileRepository = ScopedLibrary[ReviewProfile]

// AIConfigRepository encrypts credentials at rest.
type AIConfigRepository interface {
	ScopedLibrary[AIConfig]
	UpdateOAuthTokens(id int64, access, refresh string, expiresAt time.Time) error
}
