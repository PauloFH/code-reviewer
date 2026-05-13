package platform

import (
	"fmt"
	"sync"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// Registry is the concrete implementation of domain.ClientRegistry.
// It is safe for concurrent reads after all clients are registered at startup.
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

// Replace atomically swaps old → new, handling renames safely.
func (r *Registry) Replace(oldName string, client domain.PlatformClient) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if oldName != client.ClientName() {
		delete(r.clients, oldName)
	}
	r.clients[client.ClientName()] = client
}

func (r *Registry) Get(name string) (domain.PlatformClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[name]
	return c, ok
}

func (r *Registry) All() []domain.PlatformClient {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.PlatformClient, 0, len(r.clients))
	for _, c := range r.clients {
		out = append(out, c)
	}
	return out
}

// MustGet returns the client or panics — useful during startup wiring.
func (r *Registry) MustGet(name string) domain.PlatformClient {
	c, ok := r.Get(name)
	if !ok {
		panic(fmt.Sprintf("platform client %q not registered", name))
	}
	return c
}
