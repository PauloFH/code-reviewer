package gitfetch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// GitWorkspaceFetcher clones the MR branch into an ephemeral directory and
// exposes it as a read-only CodeWorkspace.
type GitWorkspaceFetcher struct {
	workspaceDir string
	cloneDepth   int
	timeout      time.Duration
	log          *slog.Logger
}

func NewGitWorkspaceFetcher(workspaceDir string, cloneDepth int, timeout time.Duration, log *slog.Logger) *GitWorkspaceFetcher {
	return &GitWorkspaceFetcher{
		workspaceDir: workspaceDir,
		cloneDepth:   cloneDepth,
		timeout:      timeout,
		log:          log,
	}
}

// Checkout obtains the MR working tree. It prefers the local bare repo
// (LocalPath, fast, no network) and falls back to the authenticated RemoteURL.
// Returns the workspace and a cleanup that removes the working tree.
func (f *GitWorkspaceFetcher) Checkout(src domain.RepoSource, sha string) (domain.CodeWorkspace, func(), error) {
	if src.LocalPath != "" {
		ws, cleanup, err := f.cloneFrom("file://"+src.LocalPath, 0, sha)
		if err == nil {
			return ws, cleanup, nil
		}
		f.log.Warn("local clone failed, falling back to remote", slog.String("error", err.Error()))
	}
	if src.RemoteURL == "" {
		return nil, nil, fmt.Errorf("gitfetch: no usable repo source (no local path, empty remote URL)")
	}
	return f.cloneFrom(src.RemoteURL, f.cloneDepth, sha)
}

// cloneFrom clones repoURL (depth 0 = full) into an ephemeral dir and checks out sha.
func (f *GitWorkspaceFetcher) cloneFrom(repoURL string, depth int, sha string) (domain.CodeWorkspace, func(), error) {
	if err := os.MkdirAll(f.workspaceDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("gitfetch: creating workspace dir: %w", err)
	}
	dir, err := os.MkdirTemp(f.workspaceDir, "review-*")
	if err != nil {
		return nil, nil, fmt.Errorf("gitfetch: temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	args := []string{"clone", "--quiet"}
	if depth > 0 {
		args = append(args, "--depth", strconv.Itoa(depth))
	}
	args = append(args, repoURL, dir)
	if out, err := f.runGit(ctx, "", args...); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("gitfetch: clone failed: %w (%s)", err, out)
	}

	if sha != "" {
		if _, err := f.runGit(ctx, dir, "checkout", "--quiet", sha); err != nil {
			// A shallow clone may not contain an arbitrary sha; fetch it explicitly.
			if _, ferr := f.runGit(ctx, dir, "fetch", "--quiet", "--depth", "1", "origin", sha); ferr == nil {
				_, err = f.runGit(ctx, dir, "checkout", "--quiet", sha)
			}
			if err != nil {
				cleanup()
				return nil, nil, fmt.Errorf("gitfetch: checkout %s: %w", sha, err)
			}
		}
	}

	return newLocalWorkspace(dir), cleanup, nil
}

func (f *GitWorkspaceFetcher) runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	// Never prompt for credentials — fail fast instead of hanging.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	return string(out), err
}
