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
	prompts *PromptHandler,
	reviewProfiles *ReviewProfileHandler,
	aiConfigs *AIConfigHandler,
	users *UserHandler,
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
	mux.Handle("GET /api/v1/auth/me", chain(
		http.HandlerFunc(auth.Me),
		jwtMiddleware,
	))

	// ── Events ───────────────────────────────────────────────────────────────
	mux.Handle("GET /api/v1/events/stats", chain(
		http.HandlerFunc(api.GetStats),
		jwtMiddleware,
	))
	mux.Handle("GET /api/v1/events/daily", chain(
		http.HandlerFunc(api.GetDailyCosts),
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

	// ── Clients (super + gestor) ──────────────────────────────────────────────
	mux.Handle("GET /api/v1/clients/{name}", chain(
		http.HandlerFunc(api.GetClient),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("PUT /api/v1/clients/{name}", chain(
		http.HandlerFunc(api.UpdateClient),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("DELETE /api/v1/clients/{name}", chain(
		http.HandlerFunc(api.DeleteClient),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("GET /api/v1/clients", chain(
		http.HandlerFunc(api.ListClients),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("POST /api/v1/clients", chain(
		http.HandlerFunc(api.CreateClient),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))

	// ── Prompts (super + gestor) ──────────────────────────────────────────────
	mux.Handle("GET /api/v1/prompts/assignments", chain(
		http.HandlerFunc(prompts.ListAssignments),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("PUT /api/v1/prompts/assignments", chain(
		http.HandlerFunc(prompts.SetAssignment),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("DELETE /api/v1/prompts/assignments", chain(
		http.HandlerFunc(prompts.RemoveAssignment),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("GET /api/v1/prompts/{id}", chain(
		http.HandlerFunc(prompts.GetPrompt),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("PUT /api/v1/prompts/{id}", chain(
		http.HandlerFunc(prompts.UpdatePrompt),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("DELETE /api/v1/prompts/{id}", chain(
		http.HandlerFunc(prompts.DeletePrompt),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("GET /api/v1/prompts", chain(
		http.HandlerFunc(prompts.ListPrompts),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("POST /api/v1/prompts", chain(
		http.HandlerFunc(prompts.CreatePrompt),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))

	// ── Review profiles (super + gestor) ──────────────────────────────────────
	mux.Handle("GET /api/v1/review-profiles/assignments", chain(
		http.HandlerFunc(reviewProfiles.ListAssignments),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("PUT /api/v1/review-profiles/assignments", chain(
		http.HandlerFunc(reviewProfiles.SetAssignment),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("DELETE /api/v1/review-profiles/assignments", chain(
		http.HandlerFunc(reviewProfiles.RemoveAssignment),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("GET /api/v1/review-profiles/{id}", chain(
		http.HandlerFunc(reviewProfiles.GetProfile),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("PUT /api/v1/review-profiles/{id}", chain(
		http.HandlerFunc(reviewProfiles.UpdateProfile),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("DELETE /api/v1/review-profiles/{id}", chain(
		http.HandlerFunc(reviewProfiles.DeleteProfile),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("GET /api/v1/review-profiles", chain(
		http.HandlerFunc(reviewProfiles.ListProfiles),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("POST /api/v1/review-profiles", chain(
		http.HandlerFunc(reviewProfiles.CreateProfile),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))

	// ── AI configs (super + gestor) ───────────────────────────────────────────
	mux.Handle("POST /api/v1/ai-configs/oauth/claude/start", chain(
		http.HandlerFunc(aiConfigs.StartClaudeOAuth),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("POST /api/v1/ai-configs/oauth/claude/exchange", chain(
		http.HandlerFunc(aiConfigs.ExchangeClaudeOAuth),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("POST /api/v1/ai-configs/{id}/test", chain(
		http.HandlerFunc(aiConfigs.TestConfig),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("GET /api/v1/ai-configs/assignments", chain(
		http.HandlerFunc(aiConfigs.ListAssignments),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("PUT /api/v1/ai-configs/assignments", chain(
		http.HandlerFunc(aiConfigs.SetAssignment),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("DELETE /api/v1/ai-configs/assignments", chain(
		http.HandlerFunc(aiConfigs.RemoveAssignment),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("GET /api/v1/ai-configs/{id}", chain(
		http.HandlerFunc(aiConfigs.GetConfig),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("PUT /api/v1/ai-configs/{id}", chain(
		http.HandlerFunc(aiConfigs.UpdateConfig),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("DELETE /api/v1/ai-configs/{id}", chain(
		http.HandlerFunc(aiConfigs.DeleteConfig),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("GET /api/v1/ai-configs", chain(
		http.HandlerFunc(aiConfigs.ListConfigs),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))
	mux.Handle("POST /api/v1/ai-configs", chain(
		http.HandlerFunc(aiConfigs.CreateConfig),
		jwtMiddleware, requireRole(domain.RoleSuper, domain.RoleGestor),
	))

	// ── Users (super only) ────────────────────────────────────────────────────
	mux.Handle("GET /api/v1/users", chain(
		http.HandlerFunc(users.ListUsers),
		jwtMiddleware, requireRole(domain.RoleSuper),
	))
	mux.Handle("POST /api/v1/users", chain(
		http.HandlerFunc(users.CreateUser),
		jwtMiddleware, requireRole(domain.RoleSuper),
	))
	mux.Handle("GET /api/v1/users/{id}", chain(
		http.HandlerFunc(users.GetUser),
		jwtMiddleware, requireRole(domain.RoleSuper),
	))
	mux.Handle("PUT /api/v1/users/{id}", chain(
		http.HandlerFunc(users.UpdateUser),
		jwtMiddleware, requireRole(domain.RoleSuper),
	))
	mux.Handle("DELETE /api/v1/users/{id}", chain(
		http.HandlerFunc(users.DeleteUser),
		jwtMiddleware, requireRole(domain.RoleSuper),
	))

	// ── Re-analyze (any authenticated user) ───────────────────────────────────
	mux.Handle("POST /api/v1/reanalyze", chain(
		http.HandlerFunc(api.Reanalyze),
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
