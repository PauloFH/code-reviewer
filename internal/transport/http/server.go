package http

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/paulofh/code-reviewer/internal/domain"
	"github.com/paulofh/code-reviewer/internal/usecase"
)

type reviewUsecase interface {
	Execute(event *domain.WebhookEvent) error
}

// Deps is everything the HTTP layer serves.
type Deps struct {
	Reviews        reviewUsecase
	Registry       domain.ClientRegistry
	Clients        *usecase.ClientManagementService
	Users          domain.UserRepository
	Events         domain.ReviewEventRepository
	Prompts        domain.PromptRepository
	ReviewProfiles domain.ReviewProfileRepository
	AIConfigs      domain.AIConfigRepository
	ClaudeOAuth    ClaudeOAuthFlow
	ConfigTester   ConfigTester
	Broadcaster    *EventBroadcaster
	JWTSecret      []byte
	Log            *slog.Logger
}

type Server struct {
	Handler http.Handler
	jobs    *jobs
}

// Shutdown waits for in-flight reviews, which run detached from their request,
// until ctx is done.
func (s *Server) Shutdown(ctx context.Context) { s.jobs.wait(ctx) }

func NewServer(d Deps) *Server {
	j := &jobs{reviews: d.Reviews, log: d.Log}
	api := &apiHandler{events: d.Events, clients: d.Clients, jobs: j}
	auth := &authHandler{users: d.Users, secret: d.JWTSecret}
	users := &userHandler{repo: d.Users}
	aiConfigs := NewAIConfigHandler(d.AIConfigs, d.ClaudeOAuth, d.ConfigTester)
	hooks := &webhookHandler{registry: d.Registry, jobs: j, log: d.Log}

	signedIn := jwtAuth(d.JWTSecret, false)
	manager := func(h http.HandlerFunc) http.Handler {
		return signedIn(requireRole(domain.RoleSuper, domain.RoleGestor)(h))
	}
	super := func(h http.HandlerFunc) http.Handler {
		return signedIn(requireRole(domain.RoleSuper)(h))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook/{client}", hooks.handle)
	mux.HandleFunc("POST /webhook/{client}/", hooks.handle)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	mux.HandleFunc("POST /api/v1/auth/login", auth.login)
	mux.Handle("GET /api/v1/auth/me", signedIn(http.HandlerFunc(auth.me)))

	mux.Handle("GET /api/v1/events", signedIn(http.HandlerFunc(api.listEvents)))
	mux.Handle("GET /api/v1/events/stats", signedIn(http.HandlerFunc(api.stats)))
	mux.Handle("GET /api/v1/events/daily", signedIn(http.HandlerFunc(api.dailyCosts)))
	// EventSource cannot set headers, so the stream also accepts ?token=.
	mux.Handle("GET /api/v1/events/stream", jwtAuth(d.JWTSecret, true)(http.HandlerFunc(d.Broadcaster.ServeSSE)))
	mux.Handle("POST /api/v1/reanalyze", signedIn(http.HandlerFunc(api.reanalyze)))
	mux.Handle("POST /api/v1/vitals", signedIn(vitalsHandler(d.Log)))

	mux.Handle("GET /api/v1/clients", manager(api.listClients))
	mux.Handle("POST /api/v1/clients", manager(api.createClient))
	mux.Handle("GET /api/v1/clients/{name}", manager(api.getClient))
	mux.Handle("PUT /api/v1/clients/{name}", manager(api.updateClient))
	mux.Handle("DELETE /api/v1/clients/{name}", manager(api.deleteClient))

	mountLibrary(mux, "/api/v1/prompts", manager, libraryHandler[domain.Prompt]{lib: d.Prompts}, d.Prompts)
	mountLibrary(mux, "/api/v1/review-profiles", manager, libraryHandler[domain.ReviewProfile]{lib: d.ReviewProfiles}, d.ReviewProfiles)
	mountLibrary(mux, "/api/v1/ai-configs", manager, aiConfigs.library(), d.AIConfigs)
	mux.Handle("GET /api/v1/ai-configs/providers", manager(aiConfigs.ListProviders))
	mux.Handle("POST /api/v1/ai-configs/{id}/test", manager(aiConfigs.TestConfig))
	mux.Handle("POST /api/v1/ai-configs/oauth/claude/start", manager(aiConfigs.StartClaudeOAuth))
	mux.Handle("POST /api/v1/ai-configs/oauth/claude/exchange", manager(aiConfigs.ExchangeClaudeOAuth))

	mux.Handle("GET /api/v1/users", super(users.list))
	mux.Handle("POST /api/v1/users", super(users.create))
	mux.Handle("GET /api/v1/users/{id}", super(users.get))
	mux.Handle("PUT /api/v1/users/{id}", super(users.update))
	mux.Handle("DELETE /api/v1/users/{id}", super(users.delete))

	// Unknown API paths must not fall through to the SPA's 200 index.html.
	mux.Handle("/api/", http.NotFoundHandler())
	mux.Handle("/", frontendHandler())

	return &Server{Handler: logging(d.Log)(mux), jobs: j}
}

type crud interface {
	list(http.ResponseWriter, *http.Request)
	get(http.ResponseWriter, *http.Request)
	create(http.ResponseWriter, *http.Request)
	update(http.ResponseWriter, *http.Request)
	delete(http.ResponseWriter, *http.Request)
}

func mountLibrary(mux *http.ServeMux, prefix string, guard func(http.HandlerFunc) http.Handler, h crud, a domain.Assignments) {
	as := assignmentHandler{a: a}
	mux.Handle("GET "+prefix, guard(h.list))
	mux.Handle("POST "+prefix, guard(h.create))
	mux.Handle("GET "+prefix+"/{id}", guard(h.get))
	mux.Handle("PUT "+prefix+"/{id}", guard(h.update))
	mux.Handle("DELETE "+prefix+"/{id}", guard(h.delete))
	mux.Handle("GET "+prefix+"/assignments", guard(as.list))
	mux.Handle("PUT "+prefix+"/assignments", guard(as.set))
	mux.Handle("DELETE "+prefix+"/assignments", guard(as.remove))
}

// jobs runs reviews in the background and lets shutdown wait for them.
type jobs struct {
	reviews reviewUsecase
	wg      sync.WaitGroup
	log     *slog.Logger
}

func (j *jobs) run(event *domain.WebhookEvent) {
	j.wg.Add(1)
	go func() {
		defer j.wg.Done()
		if err := j.reviews.Execute(event); err != nil {
			j.log.Error("review failed", slog.String("client", event.ClientName), slog.String("error", err.Error()))
		}
	}()
}

func (j *jobs) wait(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		j.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		j.log.Warn("shutdown timed out waiting for in-flight reviews")
	}
}
