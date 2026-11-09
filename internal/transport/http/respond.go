package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// writeJSON encodes v; a nil slice is sent as [] so clients never see null lists.
func writeJSON(w http.ResponseWriter, status int, v any) {
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice && rv.IsNil() {
		v = []struct{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("writing json response", slog.String("error", err.Error()))
	}
}

// writeError maps domain errors to statuses; anything unexpected is logged and
// answered without its text, which may carry SQL or crypto details.
func writeError(w http.ResponseWriter, err error) {
	var invalid *domain.InvalidError
	switch {
	case errors.As(err, &invalid):
		http.Error(w, invalid.Msg, http.StatusBadRequest)
	case errors.Is(err, domain.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, domain.ErrLastSuper):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, domain.ErrConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		slog.Error("request failed", slog.String("error", err.Error()))
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

const maxRequestBody = 1 << 20

// decodeJSON reads the request body into v, answering 400 itself on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody)).Decode(v); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return false
	}
	return true
}

// pathID parses the {id} path segment, answering 400 itself on failure.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func intQuery(r *http.Request, key string, def int) int {
	if n, err := strconv.Atoi(r.URL.Query().Get(key)); err == nil {
		return n
	}
	return def
}
