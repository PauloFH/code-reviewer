package http

import (
	"context"
	"net/http"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type contextKey string

const rawWebhookKey contextKey = "raw_webhook"
const userClaimsKey contextKey = "user_claims"

type rawWebhookCarrier struct {
	headers map[string]string
	body    []byte
}

// UserClaims holds the authenticated user's identity extracted from the JWT.
type UserClaims struct {
	ID       int64
	Username string
	Role     domain.Role
}

func withRawWebhook(r *http.Request, raw rawWebhookCarrier) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), rawWebhookKey, raw))
}

func rawWebhookFromCtx(r *http.Request) rawWebhookCarrier {
	v, _ := r.Context().Value(rawWebhookKey).(rawWebhookCarrier)
	return v
}

func withUserClaims(r *http.Request, claims UserClaims) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userClaimsKey, claims))
}

func userClaimsFromCtx(r *http.Request) (UserClaims, bool) {
	v, ok := r.Context().Value(userClaimsKey).(UserClaims)
	return v, ok
}
