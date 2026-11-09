package platform

import (
	"sync"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// Registry is the concurrency-safe domain.ClientRegistry; clients are added,
// replaced and removed at runtime by the client management API.
type Registry struct {
	mu      sync.RWMutex
	clients map[string]domain.PlatformClient
}

func NewRegistry() *Registry {
	return &Registry{clients: make(map[string]domain.PlatformClient)}
}

func (r *Registry) Register(client domain.PlatformClient) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[client.ClientName()] = client
}

func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, name)
}

func (r *Registry) Replace(oldName string, client domain.PlatformClient) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, oldName)
	r.clients[client.ClientName()] = client
}

func (r *Registry) Get(name string) (domain.PlatformClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[name]
	return c, ok
}
