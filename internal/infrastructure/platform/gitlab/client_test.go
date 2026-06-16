package gitlab

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRepoSourcePathRef(t *testing.T) {
	c := New("test", "https://gitlab.example.com", "secret-token", "wh", "bot", 30*time.Second, "")

	src, err := c.RepoSource("grp/sub/repo")
	if err != nil {
		t.Fatalf("RepoSource: %v", err)
	}
	// must be an authenticated HTTPS URL ending in the repo path + .git
	if !strings.HasPrefix(src.RemoteURL, "https://oauth2:") {
		t.Fatalf("RemoteURL missing oauth2 userinfo: %q", src.RemoteURL)
	}
	if !strings.Contains(src.RemoteURL, "@gitlab.example.com/") {
		t.Fatalf("RemoteURL missing host: %q", src.RemoteURL)
	}
	if !strings.HasSuffix(src.RemoteURL, "/grp/sub/repo.git") {
		t.Fatalf("RemoteURL missing repo path: %q", src.RemoteURL)
	}
	if !strings.Contains(src.RemoteURL, "secret-token") {
		t.Fatalf("RemoteURL missing token: %q", src.RemoteURL)
	}
}

func TestHashedRepoPath(t *testing.T) {
	// SHA256("1") = 6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b
	got := hashedRepoPath("/var/opt/gitlab/git-data/repositories", 1)
	want := "/var/opt/gitlab/git-data/repositories/@hashed/6b/86/6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b.git"
	if got != want {
		t.Fatalf("hashedRepoPath = %q, want %q", got, want)
	}
}

// Sanity-check: the hardcoded hash in tests matches stdlib.
func TestHashedRepoPathMatchesStdlib(t *testing.T) {
	sum := sha256.Sum256([]byte("1"))
	if hex.EncodeToString(sum[:]) != "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b" {
		t.Fatal("sha256 of '1' does not match expected constant")
	}
}

func TestRepoSourceWithLocalReposPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":1,"path_with_namespace":"grp/repo"}`))
	}))
	defer srv.Close()

	c := New("test", srv.URL, "tok", "wh", "bot", 30*time.Second, "/repos")
	src, err := c.RepoSource("grp/repo")
	if err != nil {
		t.Fatalf("RepoSource: %v", err)
	}
	if src.RemoteURL == "" || !strings.HasSuffix(src.RemoteURL, "/grp/repo.git") {
		t.Fatalf("RemoteURL inesperado: %q", src.RemoteURL)
	}
	wantLocal := "/repos/@hashed/6b/86/6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b.git"
	if src.LocalPath != wantLocal {
		t.Fatalf("LocalPath = %q, want %q", src.LocalPath, wantLocal)
	}
}

func TestRepoSourceWithoutReposPathHasNoLocal(t *testing.T) {
	c := New("test", "https://gitlab.example.com", "tok", "wh", "bot", 30*time.Second, "")
	src, err := c.RepoSource("grp/repo")
	if err != nil {
		t.Fatalf("RepoSource: %v", err)
	}
	if src.LocalPath != "" {
		t.Fatalf("LocalPath should be empty without reposPath, got %q", src.LocalPath)
	}
}
