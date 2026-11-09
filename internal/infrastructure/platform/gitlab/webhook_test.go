package gitlab

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestValidateWebhook(t *testing.T) {
	cases := []struct {
		name, secret, header string
		want                 bool
	}{
		{"matching token", "mysecret", "mysecret", true},
		{"wrong token", "mysecret", "wrong", false},
		{"missing header", "mysecret", "", false},
		{"empty configured secret never validates", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := New(domain.Client{WebhookSecret: tc.secret}, time.Second, "")
			headers := map[string][]string{}
			if tc.header != "" {
				headers["X-Gitlab-Token"] = []string{tc.header}
			}
			if got := c.ValidateWebhook(domain.NewRawWebhook(headers, nil)); got != tc.want {
				t.Errorf("ValidateWebhook = %v, want %v", got, tc.want)
			}
		})
	}
}

func parse(t *testing.T, payload any) (*domain.WebhookEvent, error) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return New(domain.Client{Name: "gl"}, time.Second, "").ParseWebhook(domain.NewRawWebhook(nil, body))
}

func TestParseMergeRequestWebhook(t *testing.T) {
	cases := []struct {
		name    string
		project map[string]any
		wantRef string
	}{
		{"numeric id fallback", map[string]any{"id": 42}, "42"},
		{"path preferred over id", map[string]any{"id": 42, "path_with_namespace": "grp/repo"}, "grp/repo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := parse(t, map[string]any{
				"object_kind":       "merge_request",
				"project":           tc.project,
				"object_attributes": map[string]any{"id": 100, "iid": 7, "action": "open"},
			})
			if err != nil {
				t.Fatalf("ParseWebhook: %v", err)
			}
			want := domain.WebhookEvent{ClientName: "gl", ProjectRef: tc.wantRef, MRIID: 7, Action: "open"}
			if *ev != want {
				t.Errorf("event = %+v, want %+v", *ev, want)
			}
		})
	}
}

func TestParseWebhookRejectsUnsupported(t *testing.T) {
	if _, err := parse(t, map[string]any{"object_kind": "push"}); err == nil {
		t.Error("expected error for push event")
	}
	if _, err := New(domain.Client{}, time.Second, "").ParseWebhook(domain.NewRawWebhook(nil, []byte("not json"))); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseNoteWebhook(t *testing.T) {
	cases := []struct {
		name          string
		note          string
		noteableType  string
		wantErr       bool
		wantJustify   string
		wantRequested string
	}{
		{name: "bare /review", note: "/review"},
		{name: "/review with preset", note: "/review Deep", wantRequested: "deep"},
		{name: "bare #revisar", note: "#revisar"},
		{name: "#revisar with justification", note: "#revisar  o setor nulo é esperado ", wantJustify: "o setor nulo é esperado"},
		{name: "#revisar glued to text", note: "#revisarX", wantErr: true},
		{name: "plain comment", note: "just a comment", wantErr: true},
		{name: "/reviewer is not /review", note: "/reviewer", wantErr: true},
		{name: "note not on MR", note: "/review", noteableType: "Issue", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			noteable := tc.noteableType
			if noteable == "" {
				noteable = "MergeRequest"
			}
			ev, err := parse(t, map[string]any{
				"object_kind":       "note",
				"project":           map[string]any{"id": 42, "path_with_namespace": "grp/repo"},
				"object_attributes": map[string]any{"note": tc.note, "noteable_type": noteable},
				"merge_request":     map[string]any{"id": 100, "iid": 7},
				"user":              map[string]any{"username": "carol"},
			})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected note %q to be rejected, got %+v", tc.note, ev)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseWebhook: %v", err)
			}
			want := domain.WebhookEvent{
				ClientName: "gl", ProjectRef: "grp/repo", MRIID: 7, Action: "note",
				IsReanalyze: true, CommentAuthor: "carol",
				ReviewJustification: tc.wantJustify, RequestedProfile: tc.wantRequested,
			}
			if *ev != want {
				t.Errorf("event = %+v, want %+v", *ev, want)
			}
		})
	}
}
