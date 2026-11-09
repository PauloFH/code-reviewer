package http

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func storedClient() domain.Client {
	return domain.Client{
		Name: "acme", Platform: domain.PlatformGitLab, BaseURL: "https://gitlab.example.com",
		Token: "real-token", WebhookSecret: "real-secret", Username: "bot",
	}
}

func TestClientResponsesAreRedacted(t *testing.T) {
	e := newEnv()
	e.clients["acme"] = storedClient()
	for _, path := range []string{"/api/v1/clients", "/api/v1/clients/acme"} {
		rec := e.do(domain.RoleGestor, http.MethodGet, path, "")
		if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "real-") {
			t.Errorf("GET %s = %d %s", path, rec.Code, rec.Body.String())
		}
	}
	created := e.do(domain.RoleGestor, http.MethodPost, "/api/v1/clients",
		`{"name":"new","platform":"github","base_url":"https://api.github.com","token":"new-token","webhook_secret":"new-secret","username":"bot"}`)
	if created.Code != http.StatusCreated || strings.Contains(created.Body.String(), "new-token") || strings.Contains(created.Body.String(), "new-secret") {
		t.Errorf("POST = %d %s", created.Code, created.Body.String())
	}
}

// The dashboard never receives the real token/secret, so an empty or redacted
// value on update must keep the stored one (an emptied token once caused an outage).
func TestClientUpdateKeepsStoredSecrets(t *testing.T) {
	cases := []struct {
		name, token, secret   string
		wantToken, wantSecret string
	}{
		{"omitted", "", "", "real-token", "real-secret"},
		{"redaction sentinel echoed", "***", "***", "real-token", "real-secret"},
		{"new token", "new-token", "", "new-token", "real-secret"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			e.clients["acme"] = storedClient()
			rec := e.do(domain.RoleGestor, http.MethodPut, "/api/v1/clients/acme",
				`{"name":"acme","platform":"gitlab","base_url":"https://gitlab.example.com","username":"willian",`+
					`"token":"`+tc.token+`","webhook_secret":"`+tc.secret+`"}`)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d %s", rec.Code, rec.Body.String())
			}
			got := e.clients["acme"]
			if got.Token != tc.wantToken || got.WebhookSecret != tc.wantSecret || got.Username != "willian" {
				t.Errorf("stored %+v", got)
			}
			if strings.Contains(rec.Body.String(), tc.wantToken) || strings.Contains(rec.Body.String(), tc.wantSecret) {
				t.Errorf("response leaks secrets: %s", rec.Body.String())
			}
		})
	}
}

func TestUserCreate(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"valid", `{"username":"ana","password":"pw","role":"gestor"}`, http.StatusCreated},
		{"unknown role", `{"username":"ana","password":"pw","role":"admin"}`, http.StatusBadRequest},
		{"missing password", `{"username":"ana","role":"gestor"}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			rec := e.do(domain.RoleSuper, http.MethodPost, "/api/v1/users", tc.body)
			if rec.Code != tc.want {
				t.Fatalf("status = %d %s, want %d", rec.Code, rec.Body.String(), tc.want)
			}
			if tc.want != http.StatusCreated {
				if len(e.users) != 0 {
					t.Error("invalid user was stored")
				}
				return
			}
			u, err := e.users.GetByUsername("ana")
			if err != nil {
				t.Fatal(err)
			}
			if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("pw")) != nil {
				t.Errorf("stored password %q is not a bcrypt hash of the input", u.Password)
			}
			if strings.Contains(rec.Body.String(), "password") || strings.Contains(rec.Body.String(), u.Password) {
				t.Errorf("response exposes the password: %s", rec.Body.String())
			}
		})
	}
}

func TestUserDeleteKeepsLastSuper(t *testing.T) {
	cases := []struct {
		name  string
		users fakeUsers
		path  string
		want  int
	}{
		{"last super", fakeUsers{1: {ID: 1, Role: domain.RoleSuper}, 2: {ID: 2, Role: domain.RolePadrao}}, "/api/v1/users/1", http.StatusForbidden},
		{"another super remains", fakeUsers{1: {ID: 1, Role: domain.RoleSuper}, 3: {ID: 3, Role: domain.RoleSuper}}, "/api/v1/users/1", http.StatusNoContent},
		{"non-super", fakeUsers{1: {ID: 1, Role: domain.RoleSuper}, 2: {ID: 2, Role: domain.RolePadrao}}, "/api/v1/users/2", http.StatusNoContent},
		{"missing", fakeUsers{1: {ID: 1, Role: domain.RoleSuper}}, "/api/v1/users/9", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			for id, u := range tc.users {
				e.users[id] = u
			}
			before := len(e.users)
			rec := e.do(domain.RoleSuper, http.MethodDelete, tc.path, "")
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
			wantDeleted := 0
			if tc.want == http.StatusNoContent {
				wantDeleted = 1
			}
			if deleted := before - len(e.users); deleted != wantDeleted {
				t.Errorf("deleted %d users, want %d", deleted, wantDeleted)
			}
		})
	}
}

func TestUserUpdateCannotDemoteLastSuper(t *testing.T) {
	e := newEnv()
	e.users[1] = domain.User{ID: 1, Username: "root", Role: domain.RoleSuper}
	rec := e.do(domain.RoleSuper, http.MethodPut, "/api/v1/users/1", `{"username":"root","role":"padrao"}`)
	if rec.Code != http.StatusForbidden || e.users[1].Role != domain.RoleSuper {
		t.Fatalf("status = %d, role = %s; want 403 and role kept", rec.Code, e.users[1].Role)
	}
}

func TestOAuthExchangeFailureHidesProviderDetails(t *testing.T) {
	rec := newEnv().do(domain.RoleSuper, http.MethodPost, "/api/v1/ai-configs/oauth/claude/exchange", `{"state":"st","code":"c","config_id":1}`)
	if rec.Code != http.StatusBadRequest || strings.Contains(rec.Body.String(), "req_123") || !strings.Contains(rec.Body.String(), "novo link") {
		t.Fatalf("status = %d, body = %q; want 400 with guidance and no provider details", rec.Code, rec.Body.String())
	}
}

// The dashboard iterates these fields; null instead of [] breaks it.
func TestEmptyListsAreArrays(t *testing.T) {
	cases := []struct{ path, want string }{
		{"/api/v1/events", `"items":[]`},
		{"/api/v1/events/stats", `"by_client":[]`},
		{"/api/v1/events/daily", `[]`},
		{"/api/v1/clients", `[]`},
		{"/api/v1/users", `[]`},
		{"/api/v1/prompts", `[]`},
		{"/api/v1/prompts/assignments", `[]`},
		{"/api/v1/review-profiles", `[]`},
		{"/api/v1/ai-configs/assignments", `[]`},
	}
	e := newEnv()
	for _, tc := range cases {
		rec := e.do(domain.RoleSuper, http.MethodGet, tc.path, "")
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tc.want) {
			t.Errorf("GET %s = %d %s, want %s", tc.path, rec.Code, rec.Body.String(), tc.want)
		}
	}
}

func TestEventFiltersRejectNonRFC3339Dates(t *testing.T) {
	e := newEnv()
	for _, path := range []string{"/api/v1/events?since=yesterday", "/api/v1/events/stats?until=2026-01-01"} {
		if rec := e.do(domain.RolePadrao, http.MethodGet, path, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s = %d, want 400", path, rec.Code)
		}
	}
}
