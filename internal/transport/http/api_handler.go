package http

import (
	"net/http"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
	"github.com/paulofh/code-reviewer/internal/usecase"
)

type apiHandler struct {
	events  domain.ReviewEventRepository
	clients *usecase.ClientManagementService
	jobs    *jobs
}

// eventFilter reads the dashboard filters; since/until are RFC 3339.
func eventFilter(w http.ResponseWriter, r *http.Request) (domain.ReviewEventFilter, bool) {
	q := r.URL.Query()
	f := domain.ReviewEventFilter{
		ClientName: q.Get("client"),
		Status:     domain.ReviewStatus(q.Get("status")),
		Search:     q.Get("search"),
		Limit:      intQuery(r, "limit", 0),
		Offset:     intQuery(r, "offset", 0),
	}
	for key, dst := range map[string]*time.Time{"since": &f.Since, "until": &f.Until} {
		if v := q.Get(key); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				http.Error(w, key+" must be an RFC 3339 timestamp", http.StatusBadRequest)
				return f, false
			}
			*dst = t
		}
	}
	return f, true
}

func (h *apiHandler) listEvents(w http.ResponseWriter, r *http.Request) {
	f, ok := eventFilter(w, r)
	if !ok {
		return
	}
	events, err := h.events.ListEvents(f)
	if err != nil {
		writeError(w, err)
		return
	}
	total, err := h.events.CountEvents(f)
	if err != nil {
		writeError(w, err)
		return
	}
	if events == nil {
		events = []domain.ReviewEvent{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events, "total": total})
}

func (h *apiHandler) stats(w http.ResponseWriter, r *http.Request) {
	f, ok := eventFilter(w, r)
	if !ok {
		return
	}
	stats, err := h.events.GetStats(f)
	if err != nil {
		writeError(w, err)
		return
	}
	if stats.ByClient == nil {
		stats.ByClient = []domain.ClientStat{}
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *apiHandler) dailyCosts(w http.ResponseWriter, r *http.Request) {
	points, err := h.events.GetDailyCosts(r.URL.Query().Get("client"), intQuery(r, "days", 30))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, points)
}

func (h *apiHandler) listClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clients.ListClients()
	if err != nil {
		writeError(w, err)
		return
	}
	for i := range clients {
		clients[i] = redactClient(clients[i])
	}
	writeJSON(w, http.StatusOK, clients)
}

func (h *apiHandler) getClient(w http.ResponseWriter, r *http.Request) {
	client, err := h.clients.GetClient(r.PathValue("name"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, redactClient(*client))
}

func (h *apiHandler) createClient(w http.ResponseWriter, r *http.Request) {
	var c domain.Client
	if !decodeJSON(w, r, &c) {
		return
	}
	created, err := h.clients.CreateClient(c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, redactClient(*created))
}

func (h *apiHandler) updateClient(w http.ResponseWriter, r *http.Request) {
	oldName := r.PathValue("name")
	var c domain.Client
	if !decodeJSON(w, r, &c) {
		return
	}
	// Reads never return the real token/secret, so an empty or redacted value
	// means "keep the stored one"; otherwise editing any other field would wipe them.
	if keepSecret(c.Token) || keepSecret(c.WebhookSecret) {
		existing, err := h.clients.GetClient(oldName)
		if err != nil {
			writeError(w, err)
			return
		}
		if keepSecret(c.Token) {
			c.Token = existing.Token
		}
		if keepSecret(c.WebhookSecret) {
			c.WebhookSecret = existing.WebhookSecret
		}
	}
	updated, err := h.clients.UpdateClient(oldName, c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, redactClient(*updated))
}

func (h *apiHandler) deleteClient(w http.ResponseWriter, r *http.Request) {
	if err := h.clients.DeleteClient(r.PathValue("name")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

const redactedSecret = "***"

func redactClient(c domain.Client) domain.Client {
	c.Token = redactedSecret
	c.WebhookSecret = redactedSecret
	return c
}

func keepSecret(s string) bool { return s == "" || s == redactedSecret }

// reanalyze re-runs the review of an MR from the dashboard.
func (h *apiHandler) reanalyze(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientName string `json:"client_name"`
		ProjectRef string `json:"project_ref"`
		MRIID      int64  `json:"mr_iid"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ClientName == "" || req.ProjectRef == "" || req.MRIID == 0 {
		writeError(w, domain.Invalid("client_name, project_ref, and mr_iid are required"))
		return
	}
	h.jobs.run(&domain.WebhookEvent{
		ClientName:  req.ClientName,
		ProjectRef:  req.ProjectRef,
		MRIID:       req.MRIID,
		Action:      "reanalyze",
		IsReanalyze: true,
	})
	w.WriteHeader(http.StatusAccepted)
}
