package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type fakeConfigRepo struct {
	configs []domain.AIConfig
	created domain.AIConfig
}

func (f *fakeConfigRepo) ListConfigs() ([]domain.AIConfig, error) { return f.configs, nil }
func (f *fakeConfigRepo) GetConfig(id int64) (*domain.AIConfig, error) {
	for i := range f.configs {
		if f.configs[i].ID == id {
			return &f.configs[i], nil
		}
	}
	return nil, nil
}
func (f *fakeConfigRepo) CreateConfig(c domain.AIConfig) (*domain.AIConfig, error) {
	c.ID = 1
	f.created = c
	return &c, nil
}
func (f *fakeConfigRepo) UpdateConfig(id int64, c domain.AIConfig) (*domain.AIConfig, error) {
	c.ID = id
	return &c, nil
}
func (f *fakeConfigRepo) DeleteConfig(int64) error                            { return nil }
func (f *fakeConfigRepo) SetConfigAssignment(domain.AIConfigAssignment) error { return nil }
func (f *fakeConfigRepo) RemoveConfigAssignment(string, string, string) error { return nil }
func (f *fakeConfigRepo) ListConfigAssignments() ([]domain.AIConfigAssignment, error) {
	return nil, nil
}
func (f *fakeConfigRepo) ResolveConfig(string, string) (*domain.AIConfig, error) { return nil, nil }
func (f *fakeConfigRepo) UpdateOAuthTokens(int64, string, string, time.Time) error {
	return nil
}

type fakeOAuth struct{}

func (fakeOAuth) BuildAuthorizeURL() (string, string, error) { return "https://example/auth", "st", nil }
func (fakeOAuth) Exchange(string, string) (string, string, time.Time, error) {
	return "acc", "ref", time.Now(), nil
}

type fakeTester struct{}

func (fakeTester) TestConfig(int64) (*domain.ReviewResult, error) {
	return &domain.ReviewResult{Body: "ok"}, nil
}

type fakeFailingTester struct{ err error }

func (f fakeFailingTester) TestConfig(int64) (*domain.ReviewResult, error) {
	return nil, f.err
}

// A test failure must NOT be reported as a 5xx: CDNs (Cloudflare) intercept
// origin 5xx and replace the body with a branded gateway page, hiding the real
// cause. The endpoint must answer 200 with {ok:false,error} so the UI sees it.
func TestTestConfig_FailureReturns200WithErrorBody(t *testing.T) {
	h := NewAIConfigHandler(&fakeConfigRepo{}, fakeOAuth{}, fakeFailingTester{err: errors.New("invalid x-api-key")})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai-configs/1/test", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	h.TestConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (a 5xx gets swallowed by the CDN): %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"ok":false`) {
		t.Fatalf("expected ok:false in body, got: %s", body)
	}
	if !strings.Contains(body, "invalid x-api-key") {
		t.Fatalf("expected the real upstream error surfaced to the UI, got: %s", body)
	}
}

func TestTestConfig_SuccessReturns200WithReview(t *testing.T) {
	h := NewAIConfigHandler(&fakeConfigRepo{}, fakeOAuth{}, fakeTester{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai-configs/1/test", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	h.TestConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"ok":true`) {
		t.Fatalf("expected ok:true, got: %s", body)
	}
	if !strings.Contains(body, "review_markdown") {
		t.Fatalf("expected review payload, got: %s", body)
	}
}

func TestListConfigs_MasksSecrets(t *testing.T) {
	repo := &fakeConfigRepo{configs: []domain.AIConfig{{
		ID: 1, Name: "c", Provider: domain.ProviderClaude, Mode: domain.AIModeAccount,
		APIKey: "sk-secret-key", OAuthAccessToken: "oauth-secret", OAuthRefreshToken: "refresh-secret",
	}}}
	h := NewAIConfigHandler(repo, fakeOAuth{}, fakeTester{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai-configs", nil)
	rec := httptest.NewRecorder()
	h.ListConfigs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, secret := range []string{"sk-secret-key", "oauth-secret", "refresh-secret"} {
		if strings.Contains(body, secret) {
			t.Fatalf("response leaked secret %q: %s", secret, body)
		}
	}
	if !strings.Contains(body, `"has_api_key":true`) || !strings.Contains(body, `"account_connected":true`) {
		t.Fatalf("expected masked presence flags, got: %s", body)
	}
}

func TestCreateConfig_RejectsAccountModeForNonClaude(t *testing.T) {
	h := NewAIConfigHandler(&fakeConfigRepo{}, fakeOAuth{}, fakeTester{})

	body := `{"name":"x","provider":"openai","mode":"account"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai-configs", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateConfig_StoresAPIKeyButMasksResponse(t *testing.T) {
	repo := &fakeConfigRepo{}
	h := NewAIConfigHandler(repo, fakeOAuth{}, fakeTester{})

	body := `{"name":"x","provider":"openai","mode":"api","api_key":"sk-plain"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai-configs", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateConfig(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	if repo.created.APIKey != "sk-plain" {
		t.Fatalf("api key not passed to repo: %q", repo.created.APIKey)
	}
	if strings.Contains(rec.Body.String(), "sk-plain") {
		t.Fatalf("response leaked api key: %s", rec.Body.String())
	}
}
