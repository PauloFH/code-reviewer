package usecase

import (
	"log/slog"

	"github.com/paulofh/code-reviewer/internal/domain"
)

// ClientFactory builds a live PlatformClient from a client record, validating it.
type ClientFactory interface {
	Build(c domain.Client) (domain.PlatformClient, error)
}

// ClientManagementService keeps the persisted clients and the live registry in
// step: a client is built (and so validated) before it is stored.
type ClientManagementService struct {
	clientRepo domain.ClientRepository
	registry   domain.ClientRegistry
	factory    ClientFactory
	log        *slog.Logger
}

func NewClientManagementService(
	clientRepo domain.ClientRepository,
	registry domain.ClientRegistry,
	factory ClientFactory,
	log *slog.Logger,
) *ClientManagementService {
	return &ClientManagementService{clientRepo: clientRepo, registry: registry, factory: factory, log: log}
}

func (s *ClientManagementService) ListClients() ([]domain.Client, error) {
	return s.clientRepo.ListClients()
}

func (s *ClientManagementService) GetClient(name string) (*domain.Client, error) {
	return s.clientRepo.GetClient(name)
}

func (s *ClientManagementService) CreateClient(c domain.Client) (*domain.Client, error) {
	live, err := s.factory.Build(c)
	if err != nil {
		return nil, err
	}
	created, err := s.clientRepo.CreateClient(c)
	if err != nil {
		return nil, err
	}
	s.registry.Register(live)
	s.log.Info("client registered", slog.String("name", created.Name), slog.String("platform", string(created.Platform)))
	return created, nil
}

func (s *ClientManagementService) UpdateClient(oldName string, c domain.Client) (*domain.Client, error) {
	live, err := s.factory.Build(c)
	if err != nil {
		return nil, err
	}
	updated, err := s.clientRepo.UpdateClient(oldName, c)
	if err != nil {
		return nil, err
	}
	s.registry.Replace(oldName, live)
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
