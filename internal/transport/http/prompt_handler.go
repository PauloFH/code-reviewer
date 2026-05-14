package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// PromptHandler serves /api/v1/prompts routes.
type PromptHandler struct {
	repo domain.PromptRepository
}

func NewPromptHandler(repo domain.PromptRepository) *PromptHandler {
	return &PromptHandler{repo: repo}
}

func (h *PromptHandler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	prompts, err := h.repo.ListPrompts()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if prompts == nil {
		prompts = []domain.Prompt{}
	}
	writeJSON(w, http.StatusOK, prompts)
}

func (h *PromptHandler) GetPrompt(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	p, err := h.repo.GetPrompt(id)
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

func (h *PromptHandler) CreatePrompt(w http.ResponseWriter, r *http.Request) {
	var p domain.Prompt
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if p.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	created, err := h.repo.CreatePrompt(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *PromptHandler) UpdatePrompt(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var p domain.Prompt
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	updated, err := h.repo.UpdatePrompt(id, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *PromptHandler) DeletePrompt(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.DeletePrompt(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PromptHandler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	assignments, err := h.repo.ListAssignments()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if assignments == nil {
		assignments = []domain.PromptAssignment{}
	}
	writeJSON(w, http.StatusOK, assignments)
}

func (h *PromptHandler) SetAssignment(w http.ResponseWriter, r *http.Request) {
	var a domain.PromptAssignment
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if a.Scope == "" || a.PromptID == 0 {
		http.Error(w, "scope and prompt_id are required", http.StatusBadRequest)
		return
	}
	if a.Scope != "global" && a.Scope != "client" && a.Scope != "project" {
		http.Error(w, "scope must be global, client, or project", http.StatusBadRequest)
		return
	}
	if err := h.repo.SetAssignment(a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PromptHandler) RemoveAssignment(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	clientName := r.URL.Query().Get("client_name")
	projectRef := r.URL.Query().Get("project_ref")
	if scope == "" {
		http.Error(w, "scope query param is required", http.StatusBadRequest)
		return
	}
	if err := h.repo.RemoveAssignment(scope, clientName, projectRef); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
