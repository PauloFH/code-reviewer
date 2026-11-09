package usecase

import (
	"errors"
	"testing"

	"github.com/paulofh/code-reviewer/internal/domain"
)

type fakeClientRepo struct {
	clients map[string]domain.Client
	err     error
}

func (r *fakeClientRepo) ListClients() ([]domain.Client, error) { return nil, nil }
func (r *fakeClientRepo) GetClient(name string) (*domain.Client, error) {
	c, ok := r.clients[name]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &c, nil
}
func (r *fakeClientRepo) CreateClient(c domain.Client) (*domain.Client, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.clients[c.Name] = c
	return &c, nil
}
func (r *fakeClientRepo) UpdateClient(oldName string, c domain.Client) (*domain.Client, error) {
	if r.err != nil {
		return nil, r.err
	}
	delete(r.clients, oldName)
	r.clients[c.Name] = c
	return &c, nil
}
func (r *fakeClientRepo) DeleteClient(name string) error {
	if r.err != nil {
		return r.err
	}
	delete(r.clients, name)
	return nil
}
func (r *fakeClientRepo) SeedClients([]domain.Client) error { return nil }
func (r *fakeClientRepo) CountClients() (int, error)        { return len(r.clients), nil }

type factoryFunc func(domain.Client) (domain.PlatformClient, error)

func (f factoryFunc) Build(c domain.Client) (domain.PlatformClient, error) { return f(c) }

// buildUnlessNoToken rejects a client without token, like the real factory's validation.
var buildUnlessNoToken = factoryFunc(func(c domain.Client) (domain.PlatformClient, error) {
	if c.Token == "" {
		return nil, domain.Invalid("token is required")
	}
	return &mockClient{name: c.Name}, nil
})

func newService() (*ClientManagementService, *fakeClientRepo, fakeRegistry) {
	repo := &fakeClientRepo{clients: map[string]domain.Client{"old": {Name: "old", Token: "t"}}}
	reg := fakeRegistry{"old": &mockClient{name: "old"}}
	return NewClientManagementService(repo, reg, buildUnlessNoToken, discardLogger()), repo, reg
}

func TestClientManagement_InvalidClientIsNeitherStoredNorRegistered(t *testing.T) {
	ops := map[string]func(*ClientManagementService) error{
		"create": func(s *ClientManagementService) error {
			_, err := s.CreateClient(domain.Client{Name: "new"})
			return err
		},
		"update": func(s *ClientManagementService) error {
			_, err := s.UpdateClient("old", domain.Client{Name: "new"})
			return err
		},
	}
	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			svc, repo, reg := newService()
			var invalid *domain.InvalidError
			if err := op(svc); !errors.As(err, &invalid) {
				t.Fatalf("err = %v, want InvalidError", err)
			}
			if _, ok := repo.clients["new"]; ok {
				t.Error("invalid client was written to the repository")
			}
			if _, ok := reg.Get("old"); !ok || len(reg) != 1 {
				t.Errorf("registry changed: %v", reg)
			}
		})
	}
}

func TestClientManagement_CreateRegisters(t *testing.T) {
	svc, repo, reg := newService()
	if _, err := svc.CreateClient(domain.Client{Name: "new", Token: "t"}); err != nil {
		t.Fatalf("CreateClient: %v", err)
	}
	if _, ok := repo.clients["new"]; !ok {
		t.Error("client not stored")
	}
	if _, ok := reg.Get("new"); !ok {
		t.Error("client not registered")
	}
}

func TestClientManagement_RenameReplacesRegistryEntry(t *testing.T) {
	svc, _, reg := newService()
	if _, err := svc.UpdateClient("old", domain.Client{Name: "new", Token: "t"}); err != nil {
		t.Fatalf("UpdateClient: %v", err)
	}
	if _, ok := reg.Get("old"); ok {
		t.Error("old name still registered")
	}
	if _, ok := reg.Get("new"); !ok {
		t.Error("new name not registered")
	}
}

func TestClientManagement_Delete(t *testing.T) {
	svc, repo, reg := newService()
	repo.err = errors.New("db down")
	if err := svc.DeleteClient("old"); err == nil {
		t.Fatal("expected repository error")
	}
	if _, ok := reg.Get("old"); !ok {
		t.Fatal("client unregistered although the delete failed")
	}

	repo.err = nil
	if err := svc.DeleteClient("old"); err != nil {
		t.Fatalf("DeleteClient: %v", err)
	}
	if _, ok := reg.Get("old"); ok {
		t.Error("deleted client still registered")
	}
}
