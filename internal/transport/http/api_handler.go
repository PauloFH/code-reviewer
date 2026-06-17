package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
	"github.com/paulofh/code-reviewer/internal/usecase"
)

// APIHandler serves all /api/v1/ routes.
type APIHandler struct {
	eventRepo  domain.ReviewEventRepository
	reviewRepo domain.ReviewRepository
	clientSvc  *usecase.ClientManagementService
	uc         mrUsecase
}

func NewAPIHandler(
	eventRepo domain.ReviewEventRepository,
	reviewRepo domain.ReviewRepository,
	clientSvc *usecase.ClientManagementService,
	uc mrUsecase,
) *APIHandler {
	return &APIHandler{eventRepo: eventRepo, reviewRepo: reviewRepo, clientSvc: clientSvc, uc: uc}
}

// ── Events ────────────────────────────────────────────────────────────────────

func (h *APIHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	f := domain.ReviewEventFilter{
		ClientName: r.URL.Query().Get("client"),
		Search:     r.URL.Query().Get("search"),
		Limit:      intQuery(r, "limit", 50),
		Offset:     intQuery(r, "offset", 0),
	}
	if s := r.URL.Query().Get("since"); s != "" {
		f.Since, _ = time.Parse(time.RFC3339, s)
	}
	if s := r.URL.Query().Get("until"); s != "" {
		f.Until, _ = time.Parse(time.RFC3339, s)
	}

	events, err := h.eventRepo.ListEvents(f)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	total, err := h.eventRepo.CountEvents(f)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": events,
		"total": total,
	})
}

func (h *APIHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	clientName := r.URL.Query().Get("client")
	sinceStr := r.URL.Query().Get("since")
	untilStr := r.URL.Query().Get("until")
	stats, err := h.eventRepo.GetStats(clientName, sinceStr, untilStr)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if stats != nil && stats.ByClient == nil {
		stats.ByClient = []domain.ClientStat{}
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *APIHandler) GetDailyCosts(w http.ResponseWriter, r *http.Request) {
	clientName := r.URL.Query().Get("client")
	days := intQuery(r, "days", 30)
	points, err := h.eventRepo.GetDailyCosts(clientName, days)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if points == nil {
		points = []domain.DailyCostPoint{}
	}
	writeJSON(w, http.StatusOK, points)
}

// ── Clients ───────────────────────────────────────────────────────────────────

func (h *APIHandler) ListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clientSvc.ListClients()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, redactClients(clients))
}

func (h *APIHandler) GetClient(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	client, err := h.clientSvc.GetClient(name)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if client == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, redactClient(*client))
}

func (h *APIHandler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var c domain.Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	created, err := h.clientSvc.CreateClient(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, redactClient(*created))
}

func (h *APIHandler) UpdateClient(w http.ResponseWriter, r *http.Request) {
	oldName := r.PathValue("name")
	var c domain.Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	updated, err := h.clientSvc.UpdateClient(oldName, c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, redactClient(*updated))
}

func (h *APIHandler) DeleteClient(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := h.clientSvc.DeleteClient(name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func redactClient(c domain.Client) domain.Client {
	c.Token = "***"
	c.WebhookSecret = "***"
	return c
}

func redactClients(clients []domain.Client) []domain.Client {
	out := make([]domain.Client, len(clients))
	for i, c := range clients {
		out[i] = redactClient(c)
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func intQuery(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

// ── Re-analyze ───────────────────────────────────────────────────────────────

type reanalyzeRequest struct {
	ClientName string `json:"client_name"`
	ProjectRef string `json:"project_ref"`
	MRIID      int64  `json:"mr_iid"`
}

func (h *APIHandler) Reanalyze(w http.ResponseWriter, r *http.Request) {
	var req reanalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ClientName == "" || req.ProjectRef == "" || req.MRIID == 0 {
		http.Error(w, "client_name, project_ref, and mr_iid are required", http.StatusBadRequest)
		return
	}

	if err := h.reviewRepo.ClearReview(req.ClientName, req.ProjectRef, req.MRIID); err != nil {
		http.Error(w, "failed to clear review record", http.StatusInternalServerError)
		return
	}

	event := &domain.WebhookEvent{
		ClientName:  req.ClientName,
		ProjectRef:  req.ProjectRef,
		MRIID:       req.MRIID,
		Action:      "reanalyze",
		IsReanalyze: true,
	}

	go func() { _ = h.uc.Execute(event) }()

	w.WriteHeader(http.StatusAccepted)
}
