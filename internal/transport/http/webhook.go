package http

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// webhookHandler authenticates and parses a platform webhook, then runs the
// review in the background: the platform gets 200 as soon as the event is
// accepted, and failures after the usecase's own retries surface as an MR
// comment instead of an HTTP error.
type webhookHandler struct {
	registry domain.ClientRegistry
	jobs     *jobs
	log      *slog.Logger
}

const maxWebhookBody = 10 << 20

func (h *webhookHandler) handle(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("client")
	client, ok := h.registry.Get(name)
	if !ok {
		h.log.Warn("webhook received for unknown client", slog.String("client", name))
		http.Error(w, "unknown client", http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody))
	if err != nil {
		http.Error(w, "reading body", http.StatusBadRequest)
		return
	}
	raw := domain.NewRawWebhook(r.Header, body)
	if !client.ValidateWebhook(raw) {
		h.log.Warn("webhook secret validation failed", slog.String("client", name))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	event, err := client.ParseWebhook(raw)
	if err != nil {
		// 200 so the platform does not retry event types we ignore.
		h.log.Info("webhook ignored", slog.String("client", name), slog.String("reason", err.Error()))
		w.WriteHeader(http.StatusOK)
		return
	}
	h.jobs.run(event)
	w.WriteHeader(http.StatusOK)
}
