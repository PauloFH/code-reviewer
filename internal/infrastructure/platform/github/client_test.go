package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// Known vector from GitHub's "Validating webhook deliveries" docs.
const (
	docsSecret    = "It's a Secret to Everybody"
	docsPayload   = "Hello, World!"
	docsSignature = "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17"
)

func TestValidateWebhook(t *testing.T) {
	c := New(domain.Client{WebhookSecret: docsSecret}, time.Second)
	cases := []struct {
		name    string
		headers map[string][]string
		body    string
		want    bool
	}{
		{"docs vector", map[string][]string{"X-Hub-Signature-256": {docsSignature}}, docsPayload, true},
		{"header name is case-insensitive", map[string][]string{"x-hub-signature-256": {docsSignature}}, docsPayload, true},
		{"tampered body", map[string][]string{"X-Hub-Signature-256": {docsSignature}}, docsPayload + " ", false},
		{"missing sha256= prefix", map[string][]string{"X-Hub-Signature-256": {docsSignature[len("sha256="):]}}, docsPayload, false},
		{"missing header", nil, docsPayload, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.ValidateWebhook(domain.NewRawWebhook(tc.headers, []byte(tc.body))); got != tc.want {
				t.Errorf("ValidateWebhook = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseWebhook(t *testing.T) {
	c := New(domain.Client{Name: "gh"}, time.Second)
	body := []byte(`{"action":"review_requested","number":5,"repository":{"full_name":"acme/backend"}}`)

	ev, err := c.ParseWebhook(domain.NewRawWebhook(map[string][]string{"X-GitHub-Event": {"pull_request"}}, body))
	if err != nil {
		t.Fatalf("ParseWebhook: %v", err)
	}
	want := domain.WebhookEvent{ClientName: "gh", ProjectRef: "acme/backend", MRIID: 5, Action: "review_requested"}
	if *ev != want {
		t.Errorf("event = %+v, want %+v", *ev, want)
	}

	for name, w := range map[string]domain.RawWebhook{
		"other event":  domain.NewRawWebhook(map[string][]string{"X-GitHub-Event": {"push"}}, body),
		"invalid json": domain.NewRawWebhook(map[string][]string{"X-GitHub-Event": {"pull_request"}}, []byte("{")),
	} {
		if _, err := c.ParseWebhook(w); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestGetMergeRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/backend/pulls/5" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer tok")
		}
		_, _ = w.Write([]byte(`{"id":99,"number":5,"title":"T","user":{"login":"alice"},
			"head":{"sha":"abc123"},
			"requested_reviewers":[{"login":"bot"}],"assignees":[{"login":"carol"}]}`))
	}))
	defer srv.Close()

	mr, err := New(domain.Client{BaseURL: srv.URL, Token: "tok"}, time.Second).GetMergeRequest("acme/backend", 5)
	if err != nil {
		t.Fatalf("GetMergeRequest: %v", err)
	}
	if mr.ID != 99 || mr.IID != 5 || mr.ProjectRef != "acme/backend" || mr.AuthorUsername != "alice" || mr.HeadSHA != "abc123" {
		t.Errorf("unexpected MR: %+v", mr)
	}
	if !slices.Equal(mr.Reviewers, []string{"bot"}) || !slices.Equal(mr.Assignees, []string{"carol"}) {
		t.Errorf("reviewers=%v assignees=%v", mr.Reviewers, mr.Assignees)
	}
}

// A PR with more files than one page holds must be reviewed whole.
func TestGetMergeRequestDiffFollowsPages(t *testing.T) {
	const total = 230
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		per, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
		var files []map[string]string
		for i := (page - 1) * per; i < min(page*per, total); i++ {
			files = append(files, map[string]string{"filename": fmt.Sprintf("f%d.go", i), "patch": "@@ -0,0 +1 @@\n+x"})
		}
		_ = json.NewEncoder(w).Encode(files)
	}))
	defer srv.Close()

	diff, err := New(domain.Client{BaseURL: srv.URL, Token: "tok"}, time.Second).GetMergeRequestDiff("acme/backend", 5)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(diff, "+++ b/"); got != total {
		t.Fatalf("diff has %d files, want %d", got, total)
	}
}
