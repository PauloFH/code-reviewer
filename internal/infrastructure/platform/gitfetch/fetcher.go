package gitfetch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
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
// (LocalPath, fast, no network) and falls back to RemoteURL.
// Returns the workspace and a cleanup that removes the working tree.
func (f *GitWorkspaceFetcher) Checkout(src domain.RepoSource, sha string) (domain.CodeWorkspace, func(), error) {
	if src.LocalPath != "" {
		ws, cleanup, err := f.cloneFrom("file://"+src.LocalPath, nil, 0, sha)
		if err == nil {
			return ws, cleanup, nil
		}
		f.log.Warn("local clone failed, falling back to remote", slog.String("error", err.Error()))
	}
	if src.RemoteURL == "" {
		return nil, nil, fmt.Errorf("gitfetch: no usable repo source (no local path, empty remote URL)")
	}
	var env []string
	if src.AuthHeader != "" {
		// Passed through the environment so the credential is neither visible
		// in argv nor written to the clone's .git/config.
		env = []string{"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=http.extraHeader", "GIT_CONFIG_VALUE_0=Authorization: " + src.AuthHeader}
	}
	return f.cloneFrom(src.RemoteURL, env, f.cloneDepth, sha)
}

// validSHA matches a git object name (short or full). cloneFrom rejects anything
// else so an attacker-influenced ref can't smuggle a leading-dash git flag.
var validSHA = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)

// cloneFrom clones repoURL (depth 0 = full) into an ephemeral dir and checks out sha.
func (f *GitWorkspaceFetcher) cloneFrom(repoURL string, env []string, depth int, sha string) (domain.CodeWorkspace, func(), error) {
	if sha != "" && !validSHA.MatchString(sha) {
		return nil, nil, fmt.Errorf("gitfetch: refusing suspicious sha %q", sha)
	}
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
	// "--" stops git from treating a repoURL/dir starting with "-" as a flag.
	args = append(args, "--", repoURL, dir)
	if out, err := f.runGit(ctx, env, "", args...); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("gitfetch: clone failed: %w (%s)", err, out)
	}

	if sha != "" {
		if _, err := f.runGit(ctx, env, dir, "checkout", "--quiet", sha); err != nil {
			// A shallow clone may not contain an arbitrary sha; fetch it explicitly.
			if _, ferr := f.runGit(ctx, env, dir, "fetch", "--quiet", "--depth", "1", "origin", sha); ferr == nil {
				_, err = f.runGit(ctx, env, dir, "checkout", "--quiet", sha)
			}
			if err != nil {
				cleanup()
				return nil, nil, fmt.Errorf("gitfetch: checkout %s: %w", sha, err)
			}
		}
	}

	ws, err := openLocalWorkspace(dir)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("gitfetch: opening workspace: %w", err)
	}
	return ws, func() { _ = ws.close(); cleanup() }, nil
}

func (f *GitWorkspaceFetcher) runGit(ctx context.Context, env []string, dir string, args ...string) (string, error) {
	// Disable the ext:: transport (an arbitrary-command RCE vector) and system
	// config; combined with the "--" separators this blocks flag/URL smuggling.
	full := append([]string{"-c", "protocol.ext.allow=never"}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	if dir != "" {
		cmd.Dir = dir
	}
	// Never prompt for credentials — fail fast instead of hanging.
	cmd.Env = append(append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1"), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
