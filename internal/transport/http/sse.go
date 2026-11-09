package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// EventBroadcaster fans out ReviewEvents to all active SSE subscribers.
type EventBroadcaster struct {
	mu          sync.RWMutex
	subscribers []chan domain.ReviewEvent
}

func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{}
}

// Publish sends e to every subscriber without blocking.
// Slow or disconnected subscribers are skipped silently.
func (b *EventBroadcaster) Publish(e domain.ReviewEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- e:
		default:
		}
	}
}

func (b *EventBroadcaster) subscribe() chan domain.ReviewEvent {
	ch := make(chan domain.ReviewEvent, 16)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, ch)
	b.mu.Unlock()
	return ch
}

func (b *EventBroadcaster) unsubscribe(ch chan domain.ReviewEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.subscribers[:0]
	for _, s := range b.subscribers {
		if s != ch {
			out = append(out, s)
		}
	}
	b.subscribers = out
}

func (b *EventBroadcaster) ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := b.subscribe()
	defer b.unsubscribe(ch)

	fmt.Fprintf(w, ": ping\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-ch:
			data, err := json.Marshal(e)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
