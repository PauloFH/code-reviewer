package http

import (
	"log/slog"
	"net/http"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// registryAdapter adapts domain.ClientRegistry to the local clientRegistry interface.
type registryAdapter struct {
	inner domain.ClientRegistry
}

func (a *registryAdapter) get(name string) (domain.PlatformClient, bool) {
	return a.inner.Get(name)
}

// NewRouter builds and returns the HTTP mux.
//
// Routes:
//
//	POST /webhook/{clientName}         — receive a platform webhook
//	GET  /healthz                      — liveness probe
//	POST /api/v1/auth/login            — issue JWT
//	GET  /api/v1/events                — list review events (JWT required)
//	GET  /api/v1/events/stats          — aggregated stats (JWT required)
//	GET  /api/v1/events/stream         — SSE live feed (JWT required via ?token=)
//	GET  /api/v1/clients               — list clients (JWT required)
//	POST /api/v1/clients               — create client (JWT required)
//	GET  /api/v1/clients/{name}        — get client (JWT required)
//	PUT  /api/v1/clients/{name}        — update client (JWT required)
//	DELETE /api/v1/clients/{name}      — delete client (JWT required)
//	/*                                 — serve Vue SPA
func NewRouter(
	uc mrUsecase,
	registry domain.ClientRegistry,
	auth *AuthHandler,
	api *APIHandler,
	broadcaster *EventBroadcaster,
	log *slog.Logger,
) http.Handler {
	adapter := &registryAdapter{inner: registry}
	webhookHandler := NewWebhookHandler(uc, adapter, log)

	jwtMiddleware := jwtAuth(auth.jwtSecret)

	mux := http.NewServeMux()

	// ── Webhook (no JWT) ──────────────────────────────────────────────────────
	mux.Handle("POST /webhook/", chain(
		http.HandlerFunc(webhookHandler.Handle),
		webhookAuth(adapter, log),
	))

	// ── Health (no auth) ─────────────────────────────────────────────────────
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// ── Auth ─────────────────────────────────────────────────────────────────
	mux.HandleFunc("POST /api/v1/auth/login", auth.Login)

	// ── Events ───────────────────────────────────────────────────────────────
	mux.Handle("GET /api/v1/events/stats", chain(
		http.HandlerFunc(api.GetStats),
		jwtMiddleware,
	))
	mux.Handle("GET /api/v1/events/stream", chain(
		http.HandlerFunc(broadcaster.ServeSSE),
		jwtMiddleware,
	))
	mux.Handle("GET /api/v1/events", chain(
		http.HandlerFunc(api.ListEvents),
		jwtMiddleware,
	))

	// ── Clients ───────────────────────────────────────────────────────────────
	mux.Handle("GET /api/v1/clients/", chain(
		http.HandlerFunc(api.GetClient),
		jwtMiddleware,
	))
	mux.Handle("PUT /api/v1/clients/", chain(
		http.HandlerFunc(api.UpdateClient),
		jwtMiddleware,
	))
	mux.Handle("DELETE /api/v1/clients/", chain(
		http.HandlerFunc(api.DeleteClient),
		jwtMiddleware,
	))
	mux.Handle("GET /api/v1/clients", chain(
		http.HandlerFunc(api.ListClients),
		jwtMiddleware,
	))
	mux.Handle("POST /api/v1/clients", chain(
		http.HandlerFunc(api.CreateClient),
		jwtMiddleware,
	))

	// ── SPA catch-all ─────────────────────────────────────────────────────────
	mux.Handle("/", frontendHandler())

	return logging(log)(mux)
}

// chain applies middlewares right-to-left so the first middleware in the list
// is the outermost wrapper (runs first on the way in).
func chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
