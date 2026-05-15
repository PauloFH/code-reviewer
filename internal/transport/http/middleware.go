package http

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// webhookAuth reads the full request body (so it can be reused by the handler),
// validates the webhook secret via the platform client, and injects both the
// body and the normalised headers into the request context.
func webhookAuth(registry clientRegistry, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientName := clientNameFromPath(r.URL.Path)

			client, ok := registry.get(clientName)
			if !ok {
				log.Warn("webhook received for unknown client", slog.String("client", clientName))
				http.Error(w, "unknown client", http.StatusNotFound)
				return
			}

			body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10 MB limit
			if err != nil {
				http.Error(w, "reading body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			raw := buildRawWebhook(r, body)

			if !client.ValidateWebhook(domain.RawWebhook{Headers: raw.headers, Body: raw.body}) {
				log.Warn("webhook secret validation failed", slog.String("client", clientName))
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Store for the handler
			r = withRawWebhook(r, raw)
			next.ServeHTTP(w, r)
		})
	}
}

// logging wraps every request with structured access logs.
func logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			log.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// buildRawWebhook normalises all header keys to lowercase for uniform access.
func buildRawWebhook(r *http.Request, body []byte) rawWebhookCarrier {
	headers := make(map[string]string, len(r.Header))
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[strings.ToLower(k)] = v[0]
		}
	}
	return rawWebhookCarrier{headers: headers, body: body}
}

// clientNameFromPath extracts the client name from /webhook/{clientName}.
func clientNameFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// requireRole returns a middleware that checks the authenticated user's role.
func requireRole(allowed ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := userClaimsFromCtx(r)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			for _, role := range allowed {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}
