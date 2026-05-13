package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingPreservesFlusher(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := logging(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("wrapped ResponseWriter does not implement http.Flusher")
		}
		w.WriteHeader(http.StatusNoContent)
		flusher.Flush()
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
