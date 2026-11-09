package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

func TestLibraryCRUD(t *testing.T) {
	cases := []struct {
		name, method, path, body string
		writeErr                 error
		want                     int
		wantBody                 string
	}{
		{"get missing", http.MethodGet, "/api/v1/prompts/99", "", nil, http.StatusNotFound, ""},
		{"update missing", http.MethodPut, "/api/v1/prompts/99", `{"name":"x"}`, nil, http.StatusNotFound, ""},
		{"delete missing", http.MethodDelete, "/api/v1/prompts/99", "", nil, http.StatusNotFound, ""},
		{"non-numeric id", http.MethodGet, "/api/v1/prompts/abc", "", nil, http.StatusBadRequest, ""},
		{"invalid record", http.MethodPost, "/api/v1/prompts", `{"name":""}`, nil, http.StatusBadRequest, "name is required"},
		{"duplicate name", http.MethodPost, "/api/v1/prompts", `{"name":"p"}`, domain.ErrConflict, http.StatusConflict, ""},
		{"create", http.MethodPost, "/api/v1/prompts", `{"name":"new"}`, nil, http.StatusCreated, `"name":"new"`},
		{"update", http.MethodPut, "/api/v1/prompts/1", `{"name":"renamed"}`, nil, http.StatusOK, `"name":"renamed"`},
		{"delete", http.MethodDelete, "/api/v1/prompts/1", "", nil, http.StatusNoContent, ""},
		{"unexpected error hides details", http.MethodPost, "/api/v1/prompts", `{"name":"p"}`, errors.New("sqlite: disk I/O"), http.StatusInternalServerError, "internal error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			e.prompts.items[1] = domain.Prompt{ID: 1, Name: "p"}
			e.prompts.writeErr = tc.writeErr

			rec := e.do(domain.RoleGestor, tc.method, tc.path, tc.body)
			if rec.Code != tc.want || !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Errorf("%d %q, want %d containing %q", rec.Code, rec.Body.String(), tc.want, tc.wantBody)
			}
			if strings.Contains(rec.Body.String(), "sqlite") {
				t.Errorf("internal error leaked: %q", rec.Body.String())
			}
		})
	}
}

func TestAssignments(t *testing.T) {
	cases := []struct {
		name, method, path, body string
		want                     int
		wantBody                 string
	}{
		{"set global", http.MethodPut, "/api/v1/prompts/assignments", `{"target_id":3,"target_name":"p","scope":"global"}`, http.StatusNoContent, ""},
		{"missing target", http.MethodPut, "/api/v1/prompts/assignments", `{"scope":"global"}`, http.StatusBadRequest, "target 0 does not exist"},
		{"unknown target", http.MethodPut, "/api/v1/prompts/assignments", `{"target_id":9,"scope":"global"}`, http.StatusBadRequest, "target 9 does not exist"},
		{"client scope without client", http.MethodPut, "/api/v1/prompts/assignments", `{"target_id":3,"scope":"client"}`, http.StatusBadRequest, "client_name is required"},
		{"remove unknown scope", http.MethodDelete, "/api/v1/prompts/assignments?scope=bogus", "", http.StatusBadRequest, "scope must be"},
		{"remove client scope", http.MethodDelete, "/api/v1/prompts/assignments?scope=client&client_name=acme", "", http.StatusNoContent, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			e.prompts.items[3] = domain.Prompt{ID: 3, Name: "p"}
			rec := e.do(domain.RoleGestor, tc.method, tc.path, tc.body)
			if rec.Code != tc.want || !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Errorf("%d %q, want %d containing %q", rec.Code, rec.Body.String(), tc.want, tc.wantBody)
			}
		})
	}

	e := newEnv()
	e.prompts.items[3] = domain.Prompt{ID: 3, Name: "p"}
	e.do(domain.RoleGestor, http.MethodPut, "/api/v1/prompts/assignments", `{"target_id":3,"target_name":"p","scope":"client","client_name":"acme"}`)
	body := e.do(domain.RoleGestor, http.MethodGet, "/api/v1/prompts/assignments", "").Body.String()
	for _, field := range []string{`"target_id":3`, `"target_name":"p"`, `"scope":"client"`, `"client_name":"acme"`} {
		if !strings.Contains(body, field) {
			t.Errorf("assignment listing %s lacks %s", body, field)
		}
	}
}

func TestAIConfigNeverExposesSecrets(t *testing.T) {
	e := newEnv()
	e.aiConfigs.items[1] = domain.AIConfig{
		ID: 1, Name: "c", Provider: domain.ProviderClaude, Mode: domain.AIModeAccount,
		APIKey: "sk-stored", OAuthAccessToken: "acc-stored", OAuthRefreshToken: "ref-stored",
	}
	secrets := []string{"sk-stored", "acc-stored", "ref-stored", "sk-new", `"api_key"`}

	responses := map[string]string{
		"list":   e.do(domain.RoleGestor, http.MethodGet, "/api/v1/ai-configs", "").Body.String(),
		"get":    e.do(domain.RoleGestor, http.MethodGet, "/api/v1/ai-configs/1", "").Body.String(),
		"create": e.do(domain.RoleGestor, http.MethodPost, "/api/v1/ai-configs", `{"name":"n","provider":"openai","mode":"api","api_key":"sk-new"}`).Body.String(),
	}
	for name, body := range responses {
		if !strings.Contains(body, `"has_api_key":true`) {
			t.Errorf("%s: %s lacks has_api_key", name, body)
		}
		for _, s := range secrets {
			if strings.Contains(body, s) {
				t.Errorf("%s response leaks %q: %s", name, s, body)
			}
		}
	}
	if !strings.Contains(responses["get"], `"account_connected":true`) {
		t.Errorf("get lacks account_connected: %s", responses["get"])
	}
	if e.aiConfigs.saved.APIKey != "sk-new" {
		t.Errorf("created api key = %q, want stored as sent", e.aiConfigs.saved.APIKey)
	}
}

// Stored credentials follow the config only while it keeps pointing at the
// same endpoint; otherwise a gestor could repoint base_url and have "Testar"
// send the saved key or OAuth token to a host of their choice.
func TestAIConfigUpdateKeepsSecretsOnlyForTheSameEndpoint(t *testing.T) {
	cases := []struct {
		name, body       string
		wantStatus       int
		wantKey          string
		wantOAuthCleared bool
	}{
		{"same endpoint, key kept", `{"name":"c2","provider":"claude","mode":"api","api_key":""}`, 200, "sk-stored", false},
		{"same endpoint, new key", `{"name":"c2","provider":"claude","mode":"api","api_key":"sk-new"}`, 200, "sk-new", false},
		{"default URL spelled out", `{"name":"c2","provider":"claude","mode":"api","base_url":"https://api.anthropic.com/"}`, 200, "sk-stored", false},
		{"base_url changed without key", `{"name":"c2","provider":"claude","mode":"api","base_url":"https://evil.example"}`, 400, "sk-stored", false},
		{"provider changed without key", `{"name":"c2","provider":"openai","mode":"api"}`, 400, "sk-stored", false},
		{"base_url changed with new key", `{"name":"c2","provider":"claude","mode":"api","base_url":"https://proxy.example","api_key":"sk-new"}`, 200, "sk-new", true},
		{"account repointed", `{"name":"c2","provider":"claude","mode":"account","base_url":"https://evil.example"}`, 200, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			e.aiConfigs.items[1] = domain.AIConfig{ID: 1, Name: "c", Provider: domain.ProviderClaude, Mode: domain.AIModeAPI,
				APIKey: "sk-stored", OAuthAccessToken: "acc-stored", OAuthRefreshToken: "ref-stored"}

			rec := e.do(domain.RoleGestor, http.MethodPut, "/api/v1/ai-configs/1", tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d %s, want %d", rec.Code, rec.Body.String(), tc.wantStatus)
			}
			got := e.aiConfigs.items[1]
			if got.APIKey != tc.wantKey {
				t.Errorf("api key = %q, want %q", got.APIKey, tc.wantKey)
			}
			if cleared := got.OAuthAccessToken == "" && got.OAuthRefreshToken == ""; cleared != tc.wantOAuthCleared {
				t.Errorf("oauth tokens = %q/%q, cleared=%v want %v", got.OAuthAccessToken, got.OAuthRefreshToken, cleared, tc.wantOAuthCleared)
			}
		})
	}
}

func TestAIConfigProviders(t *testing.T) {
	rec := newEnv().do(domain.RoleGestor, http.MethodGet, "/api/v1/ai-configs/providers", "")
	var got []domain.AIProviderSpec
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("status %d: %v", rec.Code, err)
	}
	if !slices.Equal(got, domain.AIProviders) {
		t.Errorf("providers = %+v, want domain.AIProviders", got)
	}
}

// A failed test is answered 200 {ok:false,error}: a CDN replaces origin 5xx
// bodies with its own page, hiding the cause from the operator.
func TestAIConfigTest(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantBody map[string]any
	}{
		{"failure", errors.New("invalid x-api-key"), map[string]any{"ok": false, "error": "invalid x-api-key"}},
		{"success", nil, map[string]any{"ok": true, "review_markdown": "looks fine"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			e.tester.err = tc.err
			rec := e.do(domain.RoleGestor, http.MethodPost, "/api/v1/ai-configs/1/test", "")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			var got map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			for k, v := range tc.wantBody {
				if got[k] != v {
					t.Errorf("%s = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
