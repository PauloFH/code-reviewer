package gitlab

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestRepoSourceKeepsTokenOutOfURL(t *testing.T) {
	c := New(domain.Client{BaseURL: "https://gitlab.example.com", Token: "secret-token"}, time.Second, "")

	src, err := c.RepoSource("grp/sub/repo")
	if err != nil {
		t.Fatalf("RepoSource: %v", err)
	}
	if src.RemoteURL != "https://gitlab.example.com/grp/sub/repo.git" {
		t.Errorf("RemoteURL = %q", src.RemoteURL)
	}
	if want := "Basic " + base64.StdEncoding.EncodeToString([]byte("oauth2:secret-token")); src.AuthHeader != want {
		t.Errorf("AuthHeader = %q, want %q", src.AuthHeader, want)
	}
	if src.LocalPath != "" {
		t.Errorf("LocalPath should be empty without reposPath, got %q", src.LocalPath)
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

func TestRepoSourceResolvesNumericRefAndLocalPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/projects/1" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"id":1,"path_with_namespace":"grp/repo"}`))
	}))
	defer srv.Close()

	src, err := New(domain.Client{BaseURL: srv.URL, Token: "tok"}, time.Second, "/repos").RepoSource("1")
	if err != nil {
		t.Fatalf("RepoSource: %v", err)
	}
	if src.RemoteURL != srv.URL+"/grp/repo.git" {
		t.Errorf("RemoteURL = %q", src.RemoteURL)
	}
	if want := "/repos/@hashed/6b/86/6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b.git"; src.LocalPath != want {
		t.Errorf("LocalPath = %q, want %q", src.LocalPath, want)
	}
}

func TestGetMergeRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/v4/projects/grp%2Frepo/merge_requests/7" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "tok" {
			t.Errorf("PRIVATE-TOKEN = %q, want %q", got, "tok")
		}
		_, _ = w.Write([]byte(`{"iid":7,"author":{"username":"alice"},
			"diff_refs":{"base_sha":"b","start_sha":"s","head_sha":"h"},
			"reviewers":[{"username":"bot"}],"assignees":[]}`))
	}))
	defer srv.Close()

	mr, err := New(domain.Client{BaseURL: srv.URL, Token: "tok"}, time.Second, "").GetMergeRequest("grp/repo", 7)
	if err != nil {
		t.Fatalf("GetMergeRequest: %v", err)
	}
	if mr.IID != 7 || mr.AuthorUsername != "alice" || mr.HeadSHA != "h" || mr.DiffBaseSHA != "b" || mr.DiffStartSHA != "s" {
		t.Errorf("unexpected MR: %+v", mr)
	}
	if !mr.IsUserAssigned("bot") {
		t.Errorf("reviewer not mapped: %v", mr.Reviewers)
	}
}

func TestGetMergeRequestDiffSince(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("from") != "old" || r.URL.Query().Get("to") != "new" {
			t.Errorf("unexpected compare query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"diffs":[{"old_path":"a.go","new_path":"a.go","diff":"+new"}]}`))
	}))
	defer srv.Close()

	diff, err := New(domain.Client{BaseURL: srv.URL}, time.Second, "").GetMergeRequestDiffSince("group/repo", "old", "new")
	if err != nil {
		t.Fatalf("GetMergeRequestDiffSince: %v", err)
	}
	if !strings.Contains(diff, "+++ b/a.go") || !strings.Contains(diff, "+new") {
		t.Fatalf("unexpected diff: %q", diff)
	}
}
