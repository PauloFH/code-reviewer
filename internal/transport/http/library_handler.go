package http

import (
	"net/http"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type validatable interface{ Validate() error }

// libraryHandler serves the CRUD of a domain.Library. Records are validated
// before writing; prepare (optional) completes a record before it is written
// (id is 0 on create) and present (optional) shapes responses.
type libraryHandler[T validatable] struct {
	lib     domain.Library[T]
	prepare func(id int64, v *T) error
	present func(T) any
}

func (h libraryHandler[T]) show(v T) any {
	if h.present == nil {
		return v
	}
	return h.present(v)
}

func (h libraryHandler[T]) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.lib.List()
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]any, len(items))
	for i, v := range items {
		out[i] = h.show(v)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h libraryHandler[T]) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	v, err := h.lib.Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.show(*v))
}

func (h libraryHandler[T]) create(w http.ResponseWriter, r *http.Request) {
	h.write(w, r, 0)
}

func (h libraryHandler[T]) update(w http.ResponseWriter, r *http.Request) {
	if id, ok := pathID(w, r); ok {
		h.write(w, r, id)
	}
}

func (h libraryHandler[T]) write(w http.ResponseWriter, r *http.Request, id int64) {
	var v T
	if !decodeJSON(w, r, &v) {
		return
	}
	if err := v.Validate(); err != nil {
		writeError(w, err)
		return
	}
	if h.prepare != nil {
		if err := h.prepare(id, &v); err != nil {
			writeError(w, err)
			return
		}
	}
	var saved *T
	var err error
	status := http.StatusOK
	if id == 0 {
		saved, err = h.lib.Create(v)
		status = http.StatusCreated
	} else {
		saved, err = h.lib.Update(id, v)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, status, h.show(*saved))
}

func (h libraryHandler[T]) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.lib.Delete(id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// assignmentHandler serves a library's scope assignments.
type assignmentHandler struct{ a domain.Assignments }

func (h assignmentHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.a.ListAssignments()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h assignmentHandler) set(w http.ResponseWriter, r *http.Request) {
	var a domain.Assignment
	if !decodeJSON(w, r, &a) {
		return
	}
	if err := h.a.SetAssignment(a); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h assignmentHandler) remove(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	k := domain.ScopeKey{Scope: domain.Scope(q.Get("scope")), ClientName: q.Get("client_name"), ProjectRef: q.Get("project_ref")}
	if err := h.a.RemoveAssignment(k); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
