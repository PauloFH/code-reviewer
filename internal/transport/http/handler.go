package http

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type mrUsecase interface {
	Execute(event *domain.WebhookEvent) error
}

type clientRegistry interface {
	get(name string) (domain.PlatformClient, bool)
}

// WebhookHandler receives platform webhooks, delegates parsing to the
// platform client, and calls the usecase. It intentionally contains no
// business logic — only input transformation and error mapping.
//
// The usecase (including its own retries with backoff) runs in a background
// goroutine so the HTTP response is not held open while it retries; the
// platform gets a 200 as soon as the event is parsed and dispatched. Final
// failures, after the usecase exhausts its retries, surface to the user as a
// comment on the MR instead of an HTTP error.
type WebhookHandler struct {
	usecase  mrUsecase
	registry clientRegistry
	log      *slog.Logger
	wg       sync.WaitGroup
}

func NewWebhookHandler(uc mrUsecase, registry clientRegistry, log *slog.Logger) *WebhookHandler {
	return &WebhookHandler{usecase: uc, registry: registry, log: log}
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	raw := rawWebhookFromCtx(r)
	clientName := clientNameFromPath(r.URL.Path)

	client, ok := h.registry.get(clientName)
	if !ok {
		// Should not reach here — middleware already validated the client.
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}

	event, err := client.ParseWebhook(domain.RawWebhook{
		Headers: raw.headers,
		Body:    raw.body,
	})
	if err != nil {
		h.log.Info("webhook ignored", slog.String("client", clientName), slog.String("reason", err.Error()))
		// Return 200 so the platform does not retry unsupported event types.
		w.WriteHeader(http.StatusOK)
		return
	}

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		if err := h.usecase.Execute(event); err != nil {
			h.log.Error("usecase error", slog.String("client", clientName), slog.String("error", err.Error()))
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// Shutdown blocks until every in-flight webhook (including its retries) has
// finished, or ctx is done — whichever comes first.
func (h *WebhookHandler) Shutdown(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		h.log.Warn("shutdown timed out waiting for in-flight webhook processing")
	}
}
