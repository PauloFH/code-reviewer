package domain

import (
	"errors"
	"slices"
	"strings"
)

type PlatformType string

const (
	PlatformGitLab           PlatformType = "gitlab"
	PlatformGitLabSelfHosted PlatformType = "gitlab-selfhosted"
	PlatformGitHub           PlatformType = "github"
	PlatformGitea            PlatformType = "gitea"
	PlatformGiteaSelfHosted  PlatformType = "gitea-selfhosted"
)

// Platforms lists every supported platform; validation and the client factory
// both derive from it.
var Platforms = []PlatformType{
	PlatformGitLab, PlatformGitLabSelfHosted, PlatformGitHub, PlatformGitea, PlatformGiteaSelfHosted,
}

func (p PlatformType) Valid() bool { return slices.Contains(Platforms, p) }

var ErrInlineCommentsUnsupported = errors.New("inline comments unsupported by platform")

// RawWebhook carries the raw HTTP data from a webhook request, decoupling the
// domain from net/http. Header lookups are case-insensitive.
type RawWebhook struct {
	headers map[string]string
	Body    []byte
}

func NewRawWebhook(headers map[string][]string, body []byte) RawWebhook {
	h := make(map[string]string, len(headers))
	for k, v := range headers {
		if len(v) > 0 {
			h[strings.ToLower(k)] = v[0]
		}
	}
	return RawWebhook{headers: h, Body: body}
}

func (w RawWebhook) Header(name string) string { return w.headers[strings.ToLower(name)] }

// WebhookEvent is the normalised representation of an incoming platform event.
type WebhookEvent struct {
	ClientName          string
	ProjectRef          string // "group/repo" path (GitLab falls back to the numeric id) or "owner/repo"
	MRIID               int64
	Action              string
	IsReanalyze         bool
	CommentAuthor       string
	ReviewJustification string
	// RequestedProfile is the preset token after a "/review <preset>" command
	// (e.g. "deep", "full"). Empty means no override was requested.
	RequestedProfile string
}

// PlatformClient is the Strategy every Git hosting platform implements.
type PlatformClient interface {
	Platform() PlatformType
	ClientName() string
	WatchedUsername() string

	ValidateWebhook(w RawWebhook) bool
	ParseWebhook(w RawWebhook) (*WebhookEvent, error)

	GetMergeRequest(projectRef string, mrIID int64) (*MergeRequest, error)
	GetMergeRequestDiff(projectRef string, mrIID int64) (string, error)
	CreateMergeRequestNote(projectRef string, mrIID int64, body string) error
	CreateMergeRequestInlineComment(mr *MergeRequest, comment InlineComment) error
}

type IncrementalDiffProvider interface {
	GetMergeRequestDiffSince(projectRef, fromSHA, toSHA string) (string, error)
}

type ClientRegistry interface {
	Register(client PlatformClient)
	Unregister(name string)
	// Replace swaps the client registered as oldName for client, handling renames.
	Replace(oldName string, client PlatformClient)
	Get(name string) (PlatformClient, bool)
}
