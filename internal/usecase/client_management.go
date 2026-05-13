package usecase

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ClientFactory builds a live PlatformClient from a domain.Client record.
// Implemented in infrastructure/platform and injected here to keep the usecase
// free of platform-specific imports.
type ClientFactory interface {
	BuildFromDomain(c domain.Client, timeout time.Duration) (domain.PlatformClient, error)
}

// ClientManagementService handles CRUD for platform clients.
// Every mutating operation persists to the DB and hot-reloads the registry.
type ClientManagementService struct {
	clientRepo domain.ClientRepository
	registry   domain.ClientRegistry
	factory    ClientFactory
	timeout    time.Duration
	log        *slog.Logger
}

func NewClientManagementService(
	clientRepo domain.ClientRepository,
	registry domain.ClientRegistry,
	factory ClientFactory,
	timeout time.Duration,
	log *slog.Logger,
) *ClientManagementService {
	return &ClientManagementService{
		clientRepo: clientRepo,
		registry:   registry,
		factory:    factory,
		timeout:    timeout,
		log:        log,
	}
}

func (s *ClientManagementService) ListClients() ([]domain.Client, error) {
	return s.clientRepo.ListClients()
}

func (s *ClientManagementService) GetClient(name string) (*domain.Client, error) {
	return s.clientRepo.GetClient(name)
}

func (s *ClientManagementService) CreateClient(c domain.Client) (*domain.Client, error) {
	created, err := s.clientRepo.CreateClient(c)
	if err != nil {
		return nil, err
	}

	platformClient, err := s.factory.BuildFromDomain(*created, s.timeout)
	if err != nil {
		// Roll back DB entry so state stays consistent.
		_ = s.clientRepo.DeleteClient(created.Name)
		return nil, fmt.Errorf("building platform client: %w", err)
	}

	s.registry.Register(platformClient)
	s.log.Info("client registered", slog.String("name", created.Name), slog.String("platform", string(created.Platform)))
	return created, nil
}

func (s *ClientManagementService) UpdateClient(oldName string, c domain.Client) (*domain.Client, error) {
	updated, err := s.clientRepo.UpdateClient(oldName, c)
	if err != nil {
		return nil, err
	}

	platformClient, err := s.factory.BuildFromDomain(*updated, s.timeout)
	if err != nil {
		return nil, fmt.Errorf("rebuilding platform client: %w", err)
	}

	// Replace handles rename atomically (removes old key, sets new key).
	if replacer, ok := s.registry.(interface {
		Replace(oldName string, client domain.PlatformClient)
	}); ok {
		replacer.Replace(oldName, platformClient)
	} else {
		s.registry.Unregister(oldName)
		s.registry.Register(platformClient)
	}

	s.log.Info("client updated", slog.String("name", updated.Name))
	return updated, nil
}

func (s *ClientManagementService) DeleteClient(name string) error {
	if err := s.clientRepo.DeleteClient(name); err != nil {
		return err
	}
	s.registry.Unregister(name)
	s.log.Info("client deleted", slog.String("name", name))
	return nil
}
