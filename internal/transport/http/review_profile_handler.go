package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ReviewProfileHandler serves /api/v1/review-profiles routes.
type ReviewProfileHandler struct {
	repo domain.ReviewProfileRepository
}

func NewReviewProfileHandler(repo domain.ReviewProfileRepository) *ReviewProfileHandler {
	return &ReviewProfileHandler{repo: repo}
}

func (h *ReviewProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.repo.ListProfiles()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if profiles == nil {
		profiles = []domain.ReviewProfile{}
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (h *ReviewProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	p, err := h.repo.GetProfile(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if p == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ReviewProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var p domain.ReviewProfile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if p.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	created, err := h.repo.CreateProfile(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *ReviewProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var p domain.ReviewProfile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	updated, err := h.repo.UpdateProfile(id, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *ReviewProfileHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.DeleteProfile(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReviewProfileHandler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	assignments, err := h.repo.ListProfileAssignments()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if assignments == nil {
		assignments = []domain.ReviewProfileAssignment{}
	}
	writeJSON(w, http.StatusOK, assignments)
}

func (h *ReviewProfileHandler) SetAssignment(w http.ResponseWriter, r *http.Request) {
	var a domain.ReviewProfileAssignment
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if a.Scope == "" || a.ProfileID == 0 {
		http.Error(w, "scope and profile_id are required", http.StatusBadRequest)
		return
	}
	if a.Scope != "global" && a.Scope != "client" && a.Scope != "project" {
		http.Error(w, "scope must be global, client, or project", http.StatusBadRequest)
		return
	}
	if err := h.repo.SetProfileAssignment(a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReviewProfileHandler) RemoveAssignment(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	clientName := r.URL.Query().Get("client_name")
	projectRef := r.URL.Query().Get("project_ref")
	if scope == "" {
		http.Error(w, "scope query param is required", http.StatusBadRequest)
		return
	}
	if err := h.repo.RemoveProfileAssignment(scope, clientName, projectRef); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
