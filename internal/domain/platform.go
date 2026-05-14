package domain

// PlatformType identifies the Git hosting platform.
type PlatformType string

const (
	PlatformGitLab           PlatformType = "gitlab"
	PlatformGitLabSelfHosted PlatformType = "gitlab-selfhosted"
	PlatformGitHub           PlatformType = "github"
	PlatformGitea            PlatformType = "gitea"
	PlatformGiteaSelfHosted  PlatformType = "gitea-selfhosted"
)

// RawWebhook carries the raw HTTP data from a webhook request,
// decoupling the domain from net/http.
type RawWebhook struct {
	Headers map[string]string
	Body    []byte
}

// WebhookEvent is the normalised representation of an incoming platform event.
type WebhookEvent struct {
	ClientName    string
	ProjectRef    string // numeric string (GitLab) or "owner/repo" (GitHub/Gitea)
	MRIID         int64
	MRID          int64
	Action        string
	IsReanalyze   bool
	CommentAuthor string
}

// PlatformClient is the Strategy interface. Every Git hosting platform
// (GitLab, GitHub, Gitea, …) must implement this contract.
type PlatformClient interface {
	// Identity
	Platform() PlatformType
	ClientName() string
	WatchedUsername() string

	// Webhook handling
	ValidateWebhook(w RawWebhook) bool
	ParseWebhook(w RawWebhook) (*WebhookEvent, error)

	// GitLab / GitHub / Gitea API
	GetMergeRequest(projectRef string, mrIID int64) (*MergeRequest, error)
	GetMergeRequestDiff(projectRef string, mrIID int64) (string, error)
	CreateMergeRequestNote(projectRef string, mrIID int64, body string) error
}

// ClientRegistry holds all registered platform clients.
type ClientRegistry interface {
	Register(client PlatformClient)
	Unregister(name string)
	Get(name string) (PlatformClient, bool)
	All() []PlatformClient
}
