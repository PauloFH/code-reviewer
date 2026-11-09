package http

import (
	"net/http"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type userHandler struct {
	repo domain.UserRepository
}

func (h *userHandler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.ListUsers()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *userHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	user, err := h.repo.GetByID(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *userHandler) create(w http.ResponseWriter, r *http.Request) { h.write(w, r, 0) }

func (h *userHandler) update(w http.ResponseWriter, r *http.Request) {
	if id, ok := pathID(w, r); ok {
		h.write(w, r, id)
	}
}

func (h *userHandler) write(w http.ResponseWriter, r *http.Request, id int64) {
	var req struct {
		Username string      `json:"username"`
		Password string      `json:"password"`
		Role     domain.Role `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	u := domain.User{Username: req.Username, Password: req.Password, Role: req.Role}
	if err := u.Validate(id == 0); err != nil {
		writeError(w, err)
		return
	}
	if u.Password != "" {
		hash, err := HashPassword(u.Password)
		if err != nil {
			writeError(w, err)
			return
		}
		u.Password = hash
	}
	var saved *domain.User
	var err error
	status := http.StatusOK
	if id == 0 {
		saved, err = h.repo.CreateUser(u)
		status = http.StatusCreated
	} else {
		saved, err = h.repo.UpdateUser(id, u)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, status, saved)
}

func (h *userHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.repo.DeleteUser(id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
