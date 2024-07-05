package inmemmanagerpool

import (
	"context"
	"sync"

	managerpool "github.com/zestagio/chat-service/internal/services/manager-pool"
	"github.com/zestagio/chat-service/internal/types"
)

const (
	serviceName = "manager-pool"
	managersMax = 1000
)

type Service struct {
	sync.RWMutex
	q []types.UserID
}

func New() *Service {
	return &Service{
		q: make([]types.UserID, 0, managersMax),
	}
}

func (s *Service) Close() error {
	s.Lock()
	defer s.Unlock()

	s.q = s.q[0:]
	return nil
}

func (s *Service) Get(ctx context.Context) (types.UserID, error) {
	select {
	case <-ctx.Done():
		return types.UserIDNil, ctx.Err()
	default:
	}

	if s.Size() == 0 {
		return types.UserIDNil, managerpool.ErrNoAvailableManagers
	}

	var firstID types.UserID
	s.Lock()
	firstID, s.q = s.q[0], s.q[1:]
	s.Unlock()

	return firstID, nil
}

func (s *Service) Put(ctx context.Context, managerID types.UserID) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	contains, err := s.Contains(ctx, managerID)
	if err != nil {
		return err
	}

	s.Lock()
	if !contains {
		s.q = append(s.q, managerID)
	}
	s.Unlock()

	return nil
}

func (s *Service) Contains(ctx context.Context, managerID types.UserID) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	s.RLock()
	defer s.RUnlock()
	for _, el := range s.q {
		if el.Matches(managerID) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) Size() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.q)
}
