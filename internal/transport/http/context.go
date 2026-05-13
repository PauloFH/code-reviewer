package http

import (
	"context"
	"net/http"
)

type contextKey string

const rawWebhookKey contextKey = "raw_webhook"

type rawWebhookCarrier struct {
	headers map[string]string
	body    []byte
}

func withRawWebhook(r *http.Request, raw rawWebhookCarrier) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), rawWebhookKey, raw))
}

func rawWebhookFromCtx(r *http.Request) rawWebhookCarrier {
	v, _ := r.Context().Value(rawWebhookKey).(rawWebhookCarrier)
	return v
}
