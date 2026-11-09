package gitea

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestValidateWebhook(t *testing.T) {
	// GitHub's documented HMAC-SHA256 vector; Gitea sends the bare hex digest.
	const sig = "757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17"
	c := New(domain.Client{WebhookSecret: "It's a Secret to Everybody"}, time.Second)
	cases := []struct {
		name    string
		headers map[string][]string
		body    string
		want    bool
	}{
		{"docs vector", map[string][]string{"X-Gitea-Signature": {sig}}, "Hello, World!", true},
		{"tampered body", map[string][]string{"X-Gitea-Signature": {sig}}, "Hello, World?", false},
		{"github-style prefix", map[string][]string{"X-Gitea-Signature": {"sha256=" + sig}}, "Hello, World!", false},
		{"missing header", nil, "Hello, World!", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.ValidateWebhook(domain.NewRawWebhook(tc.headers, []byte(tc.body))); got != tc.want {
				t.Errorf("ValidateWebhook = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseWebhookReadsGiteaEventHeader(t *testing.T) {
	c := New(domain.Client{Name: "gt"}, time.Second)
	body := []byte(`{"action":"opened","number":3,"repository":{"full_name":"acme/api"}}`)

	ev, err := c.ParseWebhook(domain.NewRawWebhook(map[string][]string{"X-Gitea-Event": {"pull_request"}}, body))
	if err != nil {
		t.Fatalf("ParseWebhook: %v", err)
	}
	if ev.ClientName != "gt" || ev.ProjectRef != "acme/api" || ev.MRIID != 3 {
		t.Errorf("unexpected event: %+v", ev)
	}
	if _, err := c.ParseWebhook(domain.NewRawWebhook(map[string][]string{"X-GitHub-Event": {"pull_request"}}, body)); err == nil {
		t.Error("expected error when only the GitHub event header is present")
	}
}

func TestGetMergeRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/acme/api/pulls/3" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "token tok" {
			t.Errorf("Authorization = %q, want %q", got, "token tok")
		}
		_, _ = w.Write([]byte(`{"number":3,"head":{"sha":"def456"},
			"requested_reviewers":[{"login":"bot"}],"assignees":[{"login":"carol"}]}`))
	}))
	defer srv.Close()

	mr, err := New(domain.Client{BaseURL: srv.URL, Token: "tok"}, time.Second).GetMergeRequest("acme/api", 3)
	if err != nil {
		t.Fatalf("GetMergeRequest: %v", err)
	}
	if mr.IID != 3 || mr.HeadSHA != "def456" {
		t.Errorf("unexpected MR: %+v", mr)
	}
	if !slices.Equal(mr.Reviewers, []string{"bot"}) || !slices.Equal(mr.Assignees, []string{"carol"}) {
		t.Errorf("reviewers=%v assignees=%v", mr.Reviewers, mr.Assignees)
	}
}
