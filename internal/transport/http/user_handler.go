package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type UserHandler struct {
	repo domain.UserRepository
}

func NewUserHandler(repo domain.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

type createUserRequest struct {
	Username string      `json:"username"`
	Password string      `json:"password"`
	Role     domain.Role `json:"role"`
}

type updateUserRequest struct {
	Username string      `json:"username"`
	Password string      `json:"password"`
	Role     domain.Role `json:"role"`
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.ListUsers()
	if err != nil {
		http.Error(w, "failed to list users", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	if req.Role != domain.RoleSuper && req.Role != domain.RoleGestor && req.Role != domain.RolePadrao {
		http.Error(w, "role must be super, gestor, or padrao", http.StatusBadRequest)
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	user, err := h.repo.CreateUser(domain.User{
		Username: req.Username,
		Password: hash,
		Role:     req.Role,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.repo.GetByID(id)
	if err != nil || user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Role != "" && req.Role != domain.RoleSuper && req.Role != domain.RoleGestor && req.Role != domain.RolePadrao {
		http.Error(w, "role must be super, gestor, or padrao", http.StatusBadRequest)
		return
	}

	var passwordHash string
	if req.Password != "" {
		hash, err := HashPassword(req.Password)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		passwordHash = hash
	}

	user, err := h.repo.UpdateUser(id, domain.User{
		Username: req.Username,
		Password: passwordHash,
		Role:     req.Role,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	existing, err := h.repo.GetByID(id)
	if err != nil || existing == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	if existing.Role == domain.RoleSuper {
		count, _ := h.repo.CountUsers()
		superCount := 0
		users, _ := h.repo.ListUsers()
		for _, u := range users {
			if u.Role == domain.RoleSuper {
				superCount++
			}
		}
		_ = count
		if superCount <= 1 {
			http.Error(w, "cannot delete the last super user", http.StatusForbidden)
			return
		}
	}

	if err := h.repo.DeleteUser(id); err != nil {
		http.Error(w, "failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
