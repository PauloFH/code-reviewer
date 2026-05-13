package http

import (
	"log/slog"
	"net/http"

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
type WebhookHandler struct {
	usecase  mrUsecase
	registry clientRegistry
	log      *slog.Logger
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

	if err := h.usecase.Execute(event); err != nil {
		h.log.Error("usecase error", slog.String("client", clientName), slog.String("error", err.Error()))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
