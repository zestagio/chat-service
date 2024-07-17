package inmemmanagerpool

import (
	"context"
	"sync"

	"go.uber.org/zap"

	managerpool "github.com/zestagio/chat-service/internal/services/manager-pool"
	"github.com/zestagio/chat-service/internal/types"
)

const (
	serviceName = "manager-pool"
	managersMax = 1000
)

type Service struct {
	managers []types.UserID
	mu       sync.RWMutex
	lg       *zap.Logger
}

func New() *Service {
	return &Service{
		managers: make([]types.UserID, 0, managersMax),
		mu:       sync.RWMutex{},
		lg:       zap.L().Named(serviceName),
	}
}

func (s *Service) Close() error {
	return nil
}

func (s *Service) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.managers)
}

func (s *Service) Get(_ context.Context) (types.UserID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.managers) == 0 {
		return types.UserIDNil, managerpool.ErrNoAvailableManagers
	}

	first := s.managers[0]
	s.managers = s.managers[1:]

	s.lg.Info("manager removed", zap.Stringer("manager_id", first))
	return first, nil
}

func (s *Service) Put(ctx context.Context, managerID types.UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.contains(ctx, managerID) {
		return nil
	}

	s.managers = append(s.managers, managerID)
	s.lg.Info("manager stored", zap.Stringer("manager_id", managerID))

	return nil
}

func (s *Service) Contains(ctx context.Context, managerID types.UserID) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.contains(ctx, managerID), nil
}

func (s *Service) contains(_ context.Context, managerID types.UserID) bool {
	for _, mID := range s.managers { // Small O(N).
		if mID == managerID {
			return true
		}
	}
	return false
}
