package http

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVitalsHandler(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode int
		wantLog  string
	}{
		{"valid", `{"path":"/dashboard","lcp":1200,"cls":0.02,"inp":80}`, http.StatusNoContent, "lcp_ms=1200"},
		{"partial metrics", `{"path":"/events","ttfb":35}`, http.StatusNoContent, "ttfb_ms=35"},
		{"negative value", `{"path":"/dashboard","lcp":-1}`, http.StatusBadRequest, ""},
		{"absurd value", `{"path":"/dashboard","inp":1e9}`, http.StatusBadRequest, ""},
		{"relative path", `{"path":"dashboard"}`, http.StatusBadRequest, ""},
		{"not json", `lcp=1`, http.StatusBadRequest, ""},
		{"oversized body", `{"path":"/` + strings.Repeat("a", 4<<10) + `"}`, http.StatusBadRequest, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			h := vitalsHandler(slog.New(slog.NewTextHandler(&logs, nil)))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/vitals", strings.NewReader(tc.body)))

			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, tc.wantCode, rec.Body.String())
			}
			if tc.wantLog == "" {
				if logs.Len() != 0 {
					t.Fatalf("rejected report was logged: %s", logs.String())
				}
				return
			}
			if !strings.Contains(logs.String(), "web vital") || !strings.Contains(logs.String(), tc.wantLog) {
				t.Fatalf("log = %q, want it to contain %q", logs.String(), tc.wantLog)
			}
		})
	}
}
