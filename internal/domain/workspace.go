package domain

// DirEntry is a single entry returned by CodeWorkspace.ListDir.
type DirEntry struct {
	Name  string
	IsDir bool
}

// GrepHit is a single line match returned by CodeWorkspace.Grep.
type GrepHit struct {
	Path string
	Line int
	Text string
}

// CodeWorkspace gives the agentic review read-only access to a checked-out
// working tree. Paths are relative to the root and must not escape it.
type CodeWorkspace interface {
	ReadFile(path string) ([]byte, error)
	ListDir(path string) ([]DirEntry, error)
	Grep(pattern, pathGlob string) ([]GrepHit, error)
	Root() string
}

// RepoSource describes where a project's repository can be cloned from.
type RepoSource struct {
	RemoteURL  string // HTTPS clone URL, without credentials
	AuthHeader string // Authorization header value for RemoteURL
	LocalPath  string // local bare-repo path; empty when unavailable
}

// RepoLocator is an optional PlatformClient capability.
type RepoLocator interface {
	RepoSource(projectRef string) (RepoSource, error)
}
