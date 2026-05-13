package github

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func newTestClient() *Client {
	return New("test-github", "https://api.github.com", "token", "mysecret", "paulorfholanda", 30*time.Second)
}

func rawWebhook(headers map[string]string, body []byte) domain.RawWebhook {
	return domain.RawWebhook{Headers: headers, Body: body}
}

func signedWebhook(secret string, body []byte, extraHeaders map[string]string) domain.RawWebhook {
	sig := "sha256=" + computeHMAC(secret, body)
	headers := map[string]string{
		"x-hub-signature-256": sig,
		"x-github-event":      "pull_request",
	}
	for k, v := range extraHeaders {
		headers[k] = v
	}
	return domain.RawWebhook{Headers: headers, Body: body}
}

// ---------- ValidateWebhook ----------

func TestValidateWebhook_CorrectHMAC(t *testing.T) {
	c := newTestClient()
	body := []byte(`{"action":"review_requested"}`)
	w := signedWebhook("mysecret", body, nil)
	if !c.ValidateWebhook(w) {
		t.Error("expected correct HMAC to pass validation")
	}
}

func TestValidateWebhook_WrongSecret(t *testing.T) {
	c := newTestClient()
	body := []byte(`{"action":"review_requested"}`)
	w := signedWebhook("wrongsecret", body, nil)
	if c.ValidateWebhook(w) {
		t.Error("expected wrong HMAC to fail validation")
	}
}

func TestValidateWebhook_MissingHeader(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{}, []byte("body"))
	if c.ValidateWebhook(w) {
		t.Error("expected missing signature header to fail validation")
	}
}

func TestValidateWebhook_MalformedSignature(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{"x-hub-signature-256": "notsha256=abc"}, []byte("body"))
	if c.ValidateWebhook(w) {
		t.Error("expected malformed signature to fail validation")
	}
}

// ---------- ParseWebhook ----------

func prPayload(owner, repo string, number, id int64, action string) []byte {
	payload := map[string]any{
		"action": action,
		"number": number,
		"pull_request": map[string]any{
			"id": id,
		},
		"repository": map[string]any{
			"full_name": owner + "/" + repo,
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

func TestParseWebhook_PullRequestEvent(t *testing.T) {
	c := newTestClient()
	body := prPayload("acme", "backend", 5, 999, "review_requested")
	w := signedWebhook("mysecret", body, nil)

	event, err := c.ParseWebhook(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.ClientName != "test-github" {
		t.Errorf("ClientName = %q, want %q", event.ClientName, "test-github")
	}
	if event.ProjectRef != "acme/backend" {
		t.Errorf("ProjectRef = %q, want %q", event.ProjectRef, "acme/backend")
	}
	if event.MRIID != 5 {
		t.Errorf("MRIID = %d, want 5", event.MRIID)
	}
	if event.MRID != 999 {
		t.Errorf("MRID = %d, want 999", event.MRID)
	}
}

func TestParseWebhook_UnsupportedEvent(t *testing.T) {
	c := newTestClient()
	body := []byte(`{}`)
	w := rawWebhook(map[string]string{"x-github-event": "push"}, body)

	if _, err := c.ParseWebhook(w); err == nil {
		t.Error("expected error for unsupported event type")
	}
}

func TestParseWebhook_InvalidJSON(t *testing.T) {
	c := newTestClient()
	w := rawWebhook(map[string]string{"x-github-event": "pull_request"}, []byte("bad json"))

	if _, err := c.ParseWebhook(w); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
