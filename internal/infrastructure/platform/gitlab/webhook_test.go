package gitlab

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func newTestClient() *Client {
	return New("test-gitlab", "https://gitlab.com", "token", "mysecret", "paulorfholanda", 30*time.Second, "")
}

func rawWebhook(headers map[string]string, body []byte) domain.RawWebhook {
	return domain.RawWebhook{Headers: headers, Body: body}
}

// ---------- ValidateWebhook ----------

func TestValidateWebhook_CorrectSecret(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{"x-gitlab-token": "mysecret"}, nil)
	if !c.ValidateWebhook(w) {
		t.Error("expected valid webhook to pass validation")
	}
}

func TestValidateWebhook_WrongSecret(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{"x-gitlab-token": "wrong"}, nil)
	if c.ValidateWebhook(w) {
		t.Error("expected wrong secret to fail validation")
	}
}

func TestValidateWebhook_MissingHeader(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{}, nil)
	if c.ValidateWebhook(w) {
		t.Error("expected missing header to fail validation")
	}
}

// ---------- ParseWebhook ----------

func mrPayload(projectID, mrID, mrIID int64, action string) []byte {
	payload := map[string]any{
		"object_kind": "merge_request",
		"project":     map[string]any{"id": projectID},
		"object_attributes": map[string]any{
			"id":     mrID,
			"iid":    mrIID,
			"action": action,
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

func TestParseWebhook_MergeRequestEvent(t *testing.T) {
	c := newTestClient()
	body := mrPayload(42, 100, 7, "open")
	w := rawWebhook(map[string]string{"x-gitlab-token": "mysecret"}, body)

	event, err := c.ParseWebhook(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.ClientName != "test-gitlab" {
		t.Errorf("ClientName = %q, want %q", event.ClientName, "test-gitlab")
	}
	if event.ProjectRef != "42" {
		t.Errorf("ProjectRef = %q, want %q", event.ProjectRef, "42")
	}
	if event.MRIID != 7 {
		t.Errorf("MRIID = %d, want 7", event.MRIID)
	}
	if event.MRID != 100 {
		t.Errorf("MRID = %d, want 100", event.MRID)
	}
	if event.Action != "open" {
		t.Errorf("Action = %q, want %q", event.Action, "open")
	}
}

func TestParseWebhook_UnsupportedObjectKind(t *testing.T) {
	c := newTestClient()
	body, _ := json.Marshal(map[string]any{"object_kind": "push"})
	w := rawWebhook(map[string]string{}, body)

	if _, err := c.ParseWebhook(w); err == nil {
		t.Error("expected error for unsupported object_kind")
	}
}

func TestParseWebhook_InvalidJSON(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{}, []byte("not json"))

	if _, err := c.ParseWebhook(w); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func notePayload(note string) []byte {
	payload := map[string]any{
		"object_kind": "note",
		"project":     map[string]any{"id": 42},
		"object_attributes": map[string]any{
			"note":          note,
			"noteable_type": "MergeRequest",
		},
		"merge_request": map[string]any{"id": 100, "iid": 7},
		"user":          map[string]any{"username": "carol"},
	}
	b, _ := json.Marshal(payload)
	return b
}

func TestParseWebhook_ReviewCommandNoPreset(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{"x-gitlab-token": "mysecret"}, notePayload("/review"))

	event, err := c.ParseWebhook(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !event.IsReanalyze || event.CommentAuthor != "carol" {
		t.Fatalf("expected reanalyze by carol, got %+v", event)
	}
	if event.RequestedProfile != "" {
		t.Errorf("RequestedProfile = %q, want empty", event.RequestedProfile)
	}
}

func TestParseWebhook_ReviewCommandWithPreset(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{"x-gitlab-token": "mysecret"}, notePayload("/review deep"))

	event, err := c.ParseWebhook(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.RequestedProfile != "deep" {
		t.Errorf("RequestedProfile = %q, want %q", event.RequestedProfile, "deep")
	}
}

func TestParseWebhook_NonReviewNoteIgnored(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{"x-gitlab-token": "mysecret"}, notePayload("just a comment"))

	if _, err := c.ParseWebhook(w); err == nil {
		t.Error("expected non-/review note to be ignored with an error")
	}
}
