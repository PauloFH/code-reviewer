package http

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
	"github.com/paulofh/code-reviewer/internal/usecase"
)

// clientRepoStub is a minimal ClientRepository that stores a single client and
// records the value passed to UpdateClient so tests can assert what was persisted.
type clientRepoStub struct {
	stored  domain.Client
	updated domain.Client
}

func (r *clientRepoStub) ListClients() ([]domain.Client, error) {
	return []domain.Client{r.stored}, nil
}
func (r *clientRepoStub) GetClient(name string) (*domain.Client, error) {
	if name != r.stored.Name {
		return nil, nil
	}
	c := r.stored
	return &c, nil
}
func (r *clientRepoStub) CreateClient(c domain.Client) (*domain.Client, error) { return &c, nil }
func (r *clientRepoStub) UpdateClient(oldName string, c domain.Client) (*domain.Client, error) {
	r.updated = c
	r.stored = c
	return &c, nil
}
func (r *clientRepoStub) DeleteClient(string) error             { return nil }
func (r *clientRepoStub) SeedFromConfigs([]domain.Client) error { return nil }
func (r *clientRepoStub) CountClients() (int, error)            { return 1, nil }

// registryStub is a no-op ClientRegistry.
type registryStub struct{}

func (registryStub) Register(domain.PlatformClient)           {}
func (registryStub) Unregister(string)                        {}
func (registryStub) Get(string) (domain.PlatformClient, bool) { return nil, false }
func (registryStub) All() []domain.PlatformClient             { return nil }

// factoryStub returns a nil PlatformClient; the service only forwards it to the
// (no-op) registry, so nil is harmless and keeps the test free of a full mock.
type factoryStub struct{}

func (factoryStub) BuildFromDomain(domain.Client, time.Duration) (domain.PlatformClient, error) {
	return nil, nil
}

func newClientHandler(stored domain.Client) (*APIHandler, *clientRepoStub) {
	repo := &clientRepoStub{stored: stored}
	svc := usecase.NewClientManagementService(repo, registryStub{}, factoryStub{}, time.Second, slog.Default())
	return NewAPIHandler(nil, nil, svc, nil), repo
}

func putClient(t *testing.T, h *APIHandler, name, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/clients/"+name, strings.NewReader(body))
	req.SetPathValue("name", name)
	rec := httptest.NewRecorder()
	h.UpdateClient(rec, req)
	return rec
}

// Editing a client without resupplying the token (the dashboard redacts it on
// read) must not wipe the stored token. Regression for the "404 Project Not
// Found" outage caused by an emptied token.
func TestUpdateClientPreservesTokenWhenOmitted(t *testing.T) {
	stored := domain.Client{
		Name: "inter", Platform: domain.PlatformGitLabSelfHosted,
		BaseURL: "https://gitlab.example.com", Token: "secret-token",
		WebhookSecret: "whsec", Username: "Holanda",
	}
	h, repo := newClientHandler(stored)

	// Body changes only the username; token/webhook_secret omitted.
	putClient(t, h, "inter", `{"name":"inter","platform":"gitlab-selfhosted","base_url":"https://gitlab.example.com","username":"willian"}`)

	if repo.updated.Token != "secret-token" {
		t.Fatalf("token = %q, want preserved %q", repo.updated.Token, "secret-token")
	}
	if repo.updated.WebhookSecret != "whsec" {
		t.Fatalf("webhook_secret = %q, want preserved %q", repo.updated.WebhookSecret, "whsec")
	}
	if repo.updated.Username != "willian" {
		t.Fatalf("username = %q, want updated %q", repo.updated.Username, "willian")
	}
}

// The redaction sentinel echoed back by the dashboard is also treated as "keep".
func TestUpdateClientPreservesTokenWhenSentinel(t *testing.T) {
	stored := domain.Client{Name: "inter", Token: "secret-token", WebhookSecret: "whsec"}
	h, repo := newClientHandler(stored)

	putClient(t, h, "inter", `{"name":"inter","token":"***","webhook_secret":"***","username":"willian"}`)

	if repo.updated.Token != "secret-token" {
		t.Fatalf("token = %q, want preserved %q", repo.updated.Token, "secret-token")
	}
	if repo.updated.WebhookSecret != "whsec" {
		t.Fatalf("webhook_secret = %q, want preserved %q", repo.updated.WebhookSecret, "whsec")
	}
}

// A genuinely new token must overwrite the stored one.
func TestUpdateClientUpdatesTokenWhenProvided(t *testing.T) {
	stored := domain.Client{Name: "inter", Token: "old-token", WebhookSecret: "whsec"}
	h, repo := newClientHandler(stored)

	putClient(t, h, "inter", `{"name":"inter","token":"new-token","username":"willian"}`)

	if repo.updated.Token != "new-token" {
		t.Fatalf("token = %q, want updated %q", repo.updated.Token, "new-token")
	}
}
