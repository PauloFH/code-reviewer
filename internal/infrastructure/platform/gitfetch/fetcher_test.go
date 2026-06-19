package gitfetch

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// makeOriginRepo creates a real git repo with one commit and returns its path
// and the HEAD sha. Skips the test if git is unavailable.
func makeOriginRepo(t *testing.T) (path, sha string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
		return string(out)
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("oi mundo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "first")
	sha = run("rev-parse", "HEAD")
	return dir, trimNL(sha)
}

func trimNL(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func TestFetcherCheckout(t *testing.T) {
	origin, sha := makeOriginRepo(t)

	f := NewGitWorkspaceFetcher(t.TempDir(), 0, 60*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ws, cleanup, err := f.Checkout(domain.RepoSource{RemoteURL: "file://" + origin}, sha)
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	defer cleanup()

	b, err := ws.ReadFile("hello.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(b) != "oi mundo\n" {
		t.Fatalf("content = %q", b)
	}

	// cleanup removes the working tree
	root := ws.Root()
	cleanup()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected workspace removed after cleanup, stat err = %v", err)
	}
}

func TestFetcherEmptyRemoteURL(t *testing.T) {
	f := NewGitWorkspaceFetcher(t.TempDir(), 0, 10*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, _, err := f.Checkout(domain.RepoSource{}, ""); err == nil {
		t.Fatal("expected error for empty remote URL")
	}
}

func TestFetcherPrefersLocalPath(t *testing.T) {
	origin, sha := makeOriginRepo(t)

	f := NewGitWorkspaceFetcher(t.TempDir(), 0, 60*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	// LocalPath set, RemoteURL empty → must clone from local
	ws, cleanup, err := f.Checkout(domain.RepoSource{LocalPath: origin}, sha)
	if err != nil {
		t.Fatalf("Checkout (local): %v", err)
	}
	defer cleanup()
	if b, err := ws.ReadFile("hello.txt"); err != nil || string(b) != "oi mundo\n" {
		t.Fatalf("local clone content wrong: %q err=%v", b, err)
	}
}

func TestFetcherFallsBackToRemoteWhenLocalFails(t *testing.T) {
	origin, sha := makeOriginRepo(t)

	f := NewGitWorkspaceFetcher(t.TempDir(), 0, 60*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	src := domain.RepoSource{
		LocalPath: "/nonexistent/path/repo.git", // fails → fallback
		RemoteURL: "file://" + origin,
	}
	ws, cleanup, err := f.Checkout(src, sha)
	if err != nil {
		t.Fatalf("Checkout (fallback): %v", err)
	}
	defer cleanup()
	if _, err := ws.ReadFile("hello.txt"); err != nil {
		t.Fatalf("fallback clone failed to read: %v", err)
	}
}
