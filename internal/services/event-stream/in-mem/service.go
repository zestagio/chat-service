package inmemeventstream

import (
	"context"
	"fmt"
	"sync"

	"github.com/imkira/go-observer"
	"go.uber.org/zap"

	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	"github.com/zestagio/chat-service/internal/types"
)

const serviceName = "event-stream"

type Service struct {
	wg        sync.WaitGroup
	mu        sync.RWMutex
	subs      map[types.UserID]observer.Property
	subsCount map[types.UserID]int
	logger    *zap.Logger
}

func New() *Service {
	return &Service{
		wg:        sync.WaitGroup{},
		mu:        sync.RWMutex{},
		subs:      map[types.UserID]observer.Property{},
		subsCount: map[types.UserID]int{},
		logger:    zap.L().Named(serviceName),
	}
}

func (s *Service) Subscribe(ctx context.Context, userID types.UserID) (<-chan eventstream.Event, error) {
	s.mu.Lock()
	p, ok := s.subs[userID]
	if !ok {
		p = observer.NewProperty(nil)
		s.subs[userID] = p
	}

	s.subsCount[userID]++
	s.mu.Unlock()

	stream := p.Observe()
	events := make(chan eventstream.Event)

	s.wg.Add(1)
	go func() {
		defer func() {
			s.mu.Lock()
			s.subsCount[userID]--
			s.mu.Unlock()

			close(events)
			s.wg.Done()
		}()

		for {
			select {
			case <-ctx.Done():
				return

			case <-stream.Changes():
				select {
				case <-ctx.Done():
					return
				case events <- stream.Next().(eventstream.Event):
				}
			}
		}
	}()
	return events, nil
}

func (s *Service) Publish(_ context.Context, userID types.UserID, event eventstream.Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %v", err)
	}

	if v := s.getSubsCount(userID); v == 0 {
		s.logger.With(zap.Stringer("user_id", userID)).Debug("no subscribers")
		return nil
	}

	s.mu.RLock()
	p := s.subs[userID]
	s.mu.RUnlock()

	p.Update(event)
	return nil
}

func (s *Service) Close() error {
	s.wg.Wait()
	return nil
}

func (s *Service) getSubsCount(uid types.UserID) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.subsCount[uid]
}
