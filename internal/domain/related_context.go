package domain

// Issue is a minimal view of a tracker issue linked to a merge request.
type Issue struct {
	IID         int64
	Title       string
	Description string
	State       string
	WebURL      string
}

// Discussion is a single comment from a merge request's discussion thread.
type Discussion struct {
	Author string
	Body   string
}

// MergeRequestRef is a lightweight reference to a related merge request.
type MergeRequestRef struct {
	IID    int64
	Title  string
	State  string
	WebURL string
}

// RelatedContext bundles everything the "related" layer gathers around an MR.
type RelatedContext struct {
	Issues      []Issue
	LinkedMRs   []MergeRequestRef
	Discussions []Discussion
}

// IsEmpty reports whether no related context was gathered.
func (rc *RelatedContext) IsEmpty() bool {
	return rc == nil || (len(rc.Issues) == 0 && len(rc.LinkedMRs) == 0 && len(rc.Discussions) == 0)
}

// RelatedContextProvider is an OPTIONAL capability a PlatformClient may implement
// to supply linked issues and discussion threads. Clients that don't implement
// it simply yield no related context (best-effort). Mirrors the RepoLocator
// optional-interface pattern.
type RelatedContextProvider interface {
	GetIssue(projectRef string, issueIID int64) (*Issue, error)
	GetClosesIssues(projectRef string, mrIID int64) ([]Issue, error)
	GetMergeRequestDiscussions(projectRef string, mrIID int64) ([]Discussion, error)
}
