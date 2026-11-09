package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/paulofh/code-reviewer/internal/domain"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform"
	"github.com/paulofh/code-reviewer/internal/usecase"
)

// ---------- fakes ----------

// fakeLib is an in-memory domain.ScopedLibrary; SetAssignment normalizes like the real store.
type fakeLib[T any] struct {
	items    map[int64]T
	writeErr error
	saved    *T
	assigned []domain.Assignment
}

func newFakeLib[T any]() *fakeLib[T] { return &fakeLib[T]{items: map[int64]T{}} }

func (f *fakeLib[T]) List() ([]T, error) {
	var out []T
	for _, v := range f.items {
		out = append(out, v)
	}
	return out, nil
}
func (f *fakeLib[T]) Get(id int64) (*T, error) {
	v, ok := f.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &v, nil
}
func (f *fakeLib[T]) Create(v T) (*T, error) {
	if f.writeErr != nil {
		return nil, f.writeErr
	}
	f.saved = &v
	return &v, nil
}
func (f *fakeLib[T]) Update(id int64, v T) (*T, error) {
	if f.writeErr != nil {
		return nil, f.writeErr
	}
	if _, ok := f.items[id]; !ok {
		return nil, domain.ErrNotFound
	}
	f.items[id] = v
	f.saved = &v
	return &v, nil
}
func (f *fakeLib[T]) Delete(id int64) error {
	if _, ok := f.items[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.items, id)
	return nil
}
func (f *fakeLib[T]) ListAssignments() ([]domain.Assignment, error) { return f.assigned, nil }
func (f *fakeLib[T]) SetAssignment(a domain.Assignment) error {
	if _, ok := f.items[a.TargetID]; !ok {
		return domain.Invalid("target %d does not exist", a.TargetID)
	}
	k, err := a.ScopeKey.Normalize()
	if err != nil {
		return err
	}
	a.ScopeKey = k
	f.assigned = append(f.assigned, a)
	return nil
}
func (f *fakeLib[T]) RemoveAssignment(k domain.ScopeKey) error {
	_, err := k.Normalize()
	return err
}
func (f *fakeLib[T]) Resolve(string, string) (*T, error) { return nil, nil }

type fakeAIConfigs struct{ *fakeLib[domain.AIConfig] }

func (fakeAIConfigs) UpdateOAuthTokens(int64, string, string, time.Time) error { return nil }

type fakeUsers map[int64]domain.User

func (f fakeUsers) GetByUsername(name string) (*domain.User, error) {
	for _, u := range f {
		if u.Username == name {
			return &u, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (f fakeUsers) GetByID(id int64) (*domain.User, error) {
	u, ok := f[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &u, nil
}
func (f fakeUsers) ListUsers() ([]domain.User, error) {
	var out []domain.User
	for _, u := range f {
		out = append(out, u)
	}
	return out, nil
}
func (f fakeUsers) CreateUser(u domain.User) (*domain.User, error) {
	u.ID = int64(len(f) + 100)
	f[u.ID] = u
	return &u, nil
}

// lastSuper mirrors the repository contract: no write may leave zero supers.
func (f fakeUsers) lastSuper(id int64) bool {
	supers := 0
	for _, u := range f {
		if u.Role == domain.RoleSuper {
			supers++
		}
	}
	return f[id].Role == domain.RoleSuper && supers == 1
}

func (f fakeUsers) UpdateUser(id int64, u domain.User) (*domain.User, error) {
	if _, ok := f[id]; !ok {
		return nil, domain.ErrNotFound
	}
	if u.Role != domain.RoleSuper && f.lastSuper(id) {
		return nil, domain.ErrLastSuper
	}
	u.ID = id
	f[id] = u
	return &u, nil
}
func (f fakeUsers) DeleteUser(id int64) error {
	if _, ok := f[id]; !ok {
		return domain.ErrNotFound
	}
	if f.lastSuper(id) {
		return domain.ErrLastSuper
	}
	delete(f, id)
	return nil
}
func (f fakeUsers) CountUsers() (int, error) { return len(f), nil }

type fakeClientRepo map[string]domain.Client

func (f fakeClientRepo) ListClients() ([]domain.Client, error) {
	var out []domain.Client
	for _, c := range f {
		out = append(out, c)
	}
	return out, nil
}
func (f fakeClientRepo) GetClient(name string) (*domain.Client, error) {
	c, ok := f[name]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &c, nil
}
func (f fakeClientRepo) CreateClient(c domain.Client) (*domain.Client, error) {
	if _, ok := f[c.Name]; ok {
		return nil, domain.ErrConflict
	}
	f[c.Name] = c
	return &c, nil
}
func (f fakeClientRepo) UpdateClient(oldName string, c domain.Client) (*domain.Client, error) {
	if _, ok := f[oldName]; !ok {
		return nil, domain.ErrNotFound
	}
	delete(f, oldName)
	f[c.Name] = c
	return &c, nil
}
func (f fakeClientRepo) DeleteClient(name string) error    { delete(f, name); return nil }
func (f fakeClientRepo) SeedClients([]domain.Client) error { return nil }
func (f fakeClientRepo) CountClients() (int, error)        { return len(f), nil }

type fakeEvents struct{}

func (fakeEvents) SaveEvent(domain.ReviewEvent) error                                { return nil }
func (fakeEvents) ListEvents(domain.ReviewEventFilter) ([]domain.ReviewEvent, error) { return nil, nil }
func (fakeEvents) CountEvents(domain.ReviewEventFilter) (int64, error)               { return 0, nil }
func (fakeEvents) GetStats(domain.ReviewEventFilter) (*domain.ReviewStats, error) {
	return &domain.ReviewStats{}, nil
}
func (fakeEvents) GetDailyCosts(string, int) ([]domain.DailyCostPoint, error) { return nil, nil }

type fakeTester struct{ err error }

func (f *fakeTester) TestConfig(int64) (*domain.ReviewResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.ReviewResult{Body: "looks fine"}, nil
}

// fakeOAuth rejects every code with an upstream-looking error.
type fakeOAuth struct{}

func (fakeOAuth) BuildAuthorizeURL() (string, string, error) { return "https://auth", "st", nil }
func (fakeOAuth) Exchange(string, string) (*domain.OAuthTokens, error) {
	return nil, errors.New(`token endpoint: 400 {"error":"invalid_grant","request_id":"req_123"}`)
}

// fakeReviews finishes after the HTTP response, so only Server.Shutdown can observe it.
type fakeReviews struct {
	mu  sync.Mutex
	got []domain.WebhookEvent
}

func (f *fakeReviews) Execute(e *domain.WebhookEvent) error {
	time.Sleep(20 * time.Millisecond)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = append(f.got, *e)
	return nil
}

func (f *fakeReviews) executed() []domain.WebhookEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.got)
}

// hookClient accepts webhooks carrying "X-Token: ok" and ignores the body "ignored".
type hookClient struct{ domain.PlatformClient }

func (hookClient) ClientName() string { return "hooked" }
func (hookClient) ValidateWebhook(w domain.RawWebhook) bool {
	return w.Header("X-Token") == "ok"
}
func (hookClient) ParseWebhook(w domain.RawWebhook) (*domain.WebhookEvent, error) {
	if string(w.Body) == "ignored" {
		return nil, errors.New("unsupported event")
	}
	return &domain.WebhookEvent{ClientName: "hooked", ProjectRef: "g/r", MRIID: 7}, nil
}

// ---------- harness ----------

var testSecret = []byte("test-jwt-secret")

type testEnv struct {
	srv       *Server
	users     fakeUsers
	clients   fakeClientRepo
	prompts   *fakeLib[domain.Prompt]
	aiConfigs *fakeLib[domain.AIConfig]
	tester    *fakeTester
	reviews   *fakeReviews
}

func newEnv() *testEnv {
	log := slog.New(slog.DiscardHandler)
	e := &testEnv{
		users:     fakeUsers{},
		clients:   fakeClientRepo{},
		prompts:   newFakeLib[domain.Prompt](),
		aiConfigs: newFakeLib[domain.AIConfig](),
		tester:    &fakeTester{},
		reviews:   &fakeReviews{},
	}
	registry := platform.NewRegistry()
	registry.Register(hookClient{})
	e.srv = NewServer(Deps{
		Reviews:        e.reviews,
		Registry:       registry,
		Clients:        usecase.NewClientManagementService(e.clients, registry, platform.Factory{Timeout: time.Second}, log),
		Users:          e.users,
		Events:         fakeEvents{},
		Prompts:        e.prompts,
		ReviewProfiles: newFakeLib[domain.ReviewProfile](),
		AIConfigs:      fakeAIConfigs{e.aiConfigs},
		ConfigTester:   e.tester,
		ClaudeOAuth:    fakeOAuth{},
		Broadcaster:    NewEventBroadcaster(),
		JWTSecret:      testSecret,
		Log:            log,
	})
	return e
}

func tokenFor(role domain.Role) string {
	return issueToken(&domain.User{ID: 1, Username: "u", Role: role}, testSecret)
}

// do sends a request as a user with the given role; an empty role is anonymous.
func (e *testEnv) do(role domain.Role, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if role != "" {
		req.Header.Set("Authorization", "Bearer "+tokenFor(role))
	}
	rec := httptest.NewRecorder()
	e.srv.Handler.ServeHTTP(rec, req)
	return rec
}

// waitJobs returns once the background reviews started by requests are done.
func (e *testEnv) waitJobs(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	e.srv.Shutdown(ctx)
}

// signJWT signs an HS256 JWT independently of the production code.
func signJWT(secret []byte, payload string) string {
	in := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(in))
	return in + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func claims(role domain.Role, exp time.Time) string {
	return fmt.Sprintf(`{"id":1,"sub":"root","role":%q,"exp":%d}`, role, exp.Unix())
}

// ---------- auth ----------

func TestJWTValidation(t *testing.T) {
	future := time.Now().Add(time.Hour)
	padrao := signJWT(testSecret, claims(domain.RolePadrao, future))
	parts := strings.Split(padrao, ".")
	escalated := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(claims(domain.RoleSuper, future))) + "." + parts[2]

	cases := []struct {
		name, token string
		want        int
	}{
		{"spec-compliant HS256 token", signJWT(testSecret, claims(domain.RoleSuper, future)), http.StatusOK},
		{"payload tampered, original signature", escalated, http.StatusUnauthorized},
		{"signed with another secret", signJWT([]byte("other"), claims(domain.RoleSuper, future)), http.StatusUnauthorized},
		{"expired", signJWT(testSecret, claims(domain.RoleSuper, time.Now().Add(-time.Minute))), http.StatusUnauthorized},
		{"malformed", "not-a-jwt", http.StatusUnauthorized},
		{"missing", "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			newEnv().srv.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, body string
		want       int
	}{
		{"valid credentials", `{"username":"admin","password":"secret"}`, http.StatusOK},
		{"wrong password", `{"username":"admin","password":"nope"}`, http.StatusUnauthorized},
		{"unknown user", `{"username":"ghost","password":"secret"}`, http.StatusUnauthorized},
		{"bad body", `not json`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			e.users[1] = domain.User{ID: 1, Username: "admin", Password: string(hash), Role: domain.RoleGestor}
			rec := e.do("", http.MethodPost, "/api/v1/auth/login", tc.body)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
			if tc.want != http.StatusOK {
				return
			}
			var resp struct{ Token string }
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
			req.Header.Set("Authorization", "Bearer "+resp.Token)
			me := httptest.NewRecorder()
			e.srv.Handler.ServeHTTP(me, req)
			if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), `"username":"admin"`) || strings.Contains(me.Body.String(), string(hash)) {
				t.Errorf("/auth/me with issued token: %d %s", me.Code, me.Body.String())
			}
		})
	}
}

func TestRoleGuards(t *testing.T) {
	cases := []struct {
		role domain.Role
		path string
		want int
	}{
		{"", "/api/v1/prompts", http.StatusUnauthorized},
		{domain.RolePadrao, "/api/v1/events", http.StatusOK},
		{domain.RolePadrao, "/api/v1/prompts", http.StatusForbidden},
		{domain.RolePadrao, "/api/v1/review-profiles", http.StatusForbidden},
		{domain.RolePadrao, "/api/v1/ai-configs", http.StatusForbidden},
		{domain.RolePadrao, "/api/v1/clients", http.StatusForbidden},
		{domain.RoleGestor, "/api/v1/prompts", http.StatusOK},
		{domain.RoleGestor, "/api/v1/clients", http.StatusOK},
		{domain.RoleGestor, "/api/v1/users", http.StatusForbidden},
		{domain.RoleSuper, "/api/v1/users", http.StatusOK},
		{domain.RoleSuper, "/api/v1/nope", http.StatusNotFound}, // not the SPA's index.html
	}
	e := newEnv()
	for _, tc := range cases {
		if got := e.do(tc.role, http.MethodGet, tc.path, "").Code; got != tc.want {
			t.Errorf("%q GET %s = %d, want %d", tc.role, tc.path, got, tc.want)
		}
	}
}

func TestQueryTokenOnlyOnEventStream(t *testing.T) {
	e := newEnv()
	token := tokenFor(domain.RolePadrao)

	rec := httptest.NewRecorder()
	e.srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/events?token="+token, nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("/events?token= = %d, want 401", rec.Code)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	rec = httptest.NewRecorder()
	e.srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/events/stream?token="+token, nil).WithContext(ctx))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("/events/stream?token= = %d %q, want an open stream", rec.Code, rec.Header().Get("Content-Type"))
	}
}

// ---------- webhook and background reviews ----------

func TestWebhook(t *testing.T) {
	cases := []struct {
		name, path, token, body string
		want                    int
		wantReview              bool
	}{
		{"unknown client", "/webhook/nope", "ok", "{}", http.StatusNotFound, false},
		{"bad signature", "/webhook/hooked", "bad", "{}", http.StatusUnauthorized, false},
		{"ignored event", "/webhook/hooked", "ok", "ignored", http.StatusOK, false},
		{"accepted event", "/webhook/hooked", "ok", "{}", http.StatusOK, true},
		{"trailing slash", "/webhook/hooked/", "ok", "{}", http.StatusOK, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("X-Token", tc.token)
			rec := httptest.NewRecorder()
			e.srv.Handler.ServeHTTP(rec, req)
			e.waitJobs(t)

			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d", rec.Code, tc.want)
			}
			got := e.reviews.executed()
			if tc.wantReview != (len(got) == 1) {
				t.Fatalf("reviews run = %+v, want review: %v", got, tc.wantReview)
			}
			if tc.wantReview && (got[0].ClientName != "hooked" || got[0].MRIID != 7) {
				t.Errorf("review ran for %+v, want the parsed event", got[0])
			}
		})
	}
}

func TestReanalyzeRunsInBackground(t *testing.T) {
	e := newEnv()
	if rec := e.do(domain.RolePadrao, http.MethodPost, "/api/v1/reanalyze", `{"client_name":"acme"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("incomplete request = %d, want 400", rec.Code)
	}
	rec := e.do(domain.RolePadrao, http.MethodPost, "/api/v1/reanalyze", `{"client_name":"acme","project_ref":"g/r","mr_iid":3}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	e.waitJobs(t)

	want := []domain.WebhookEvent{{ClientName: "acme", ProjectRef: "g/r", MRIID: 3, Action: "reanalyze", IsReanalyze: true}}
	if got := e.reviews.executed(); !slices.Equal(got, want) {
		t.Errorf("reviews = %+v, want %+v", got, want)
	}
}
