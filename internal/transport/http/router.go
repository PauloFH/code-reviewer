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
//	POST /webhook/{clientName}   — receive a platform webhook
//	GET  /healthz                — liveness probe
func NewRouter(
	uc mrUsecase,
	registry domain.ClientRegistry,
	log *slog.Logger,
) http.Handler {
	adapter := &registryAdapter{inner: registry}
	handler := NewWebhookHandler(uc, adapter, log)

	mux := http.NewServeMux()

	mux.Handle("POST /webhook/", chain(
		http.HandlerFunc(handler.Handle),
		webhookAuth(adapter, log),
	))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

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
