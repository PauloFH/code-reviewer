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
// working tree. All paths are relative to the workspace root; implementations
// MUST reject paths that escape the root (path traversal).
type CodeWorkspace interface {
	ReadFile(path string) ([]byte, error)
	ListDir(path string) ([]DirEntry, error)
	Grep(pattern, pathGlob string) ([]GrepHit, error)
	Root() string
}

// RepoSource describes where a project's repository can be cloned from.
type RepoSource struct {
	RemoteURL string // authenticated HTTPS clone URL (Fase 1)
	LocalPath string // local bare-repo path (Fase 2; empty if N/A)
}

// RepoLocator is an OPTIONAL capability a PlatformClient may implement to
// describe where a project's repository can be obtained. Mirrors the
// RelatedContextProvider optional-interface pattern.
type RepoLocator interface {
	RepoSource(projectRef string) (RepoSource, error)
}
