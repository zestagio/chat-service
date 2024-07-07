package inmemeventstream

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	"github.com/zestagio/chat-service/internal/types"
)

const serviceName = "event-stream"

type Service struct {
	wg     sync.WaitGroup
	mu     sync.RWMutex
	subs   map[types.UserID][]chan eventstream.Event
	lg     *zap.Logger
	closed bool
}

func New() *Service {
	return &Service{
		subs: make(map[types.UserID][]chan eventstream.Event),
		lg:   zap.L().Named(serviceName),
	}
}

func (s *Service) Subscribe(ctx context.Context, userID types.UserID) (<-chan eventstream.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(chan eventstream.Event)
	in := make(chan eventstream.Event, 1024)

	s.wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		for {
			select {
			case ev := <-in:
				out <- ev
			case <-ctx.Done():
				close(out)
				s.lg.Info("client is offline", zap.String("id", userID.String()))
				return
			}
		}
	}(&s.wg)

	s.subs[userID] = append(s.subs[userID], in)

	s.lg.Info("client subscribed to events", zap.String("id", userID.String()))

	return out, nil
}

func (s *Service) Publish(_ context.Context, userID types.UserID, event eventstream.Event) error {
	if s.closed {
		return nil
	}

	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate event for user %v: %v", userID, event)
	}

	s.mu.RLock()
	channels, ok := s.subs[userID]
	s.mu.RUnlock()
	if !ok {
		s.lg.Warn("no online clients for publish")
		return nil
	}

	for _, ch := range channels {
		ch <- event
	}

	return nil
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
	}

	s.wg.Wait()

	s.lg.Info("event stream is close")
	return nil
}
