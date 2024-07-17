package clientevents

import (
	"errors"
	"fmt"

	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	websocketstream "github.com/zestagio/chat-service/internal/websocket-stream"
	"github.com/zestagio/chat-service/pkg/pointer"
)

var _ websocketstream.EventAdapter = Adapter{}

type Adapter struct{}

func (Adapter) Adapt(ev eventstream.Event) (any, error) {
	if err := ev.Validate(); err != nil {
		return nil, fmt.Errorf("validate while adapt: %v", err)
	}

	switch e := ev.(type) {
	case *eventstream.NewMessageEvent:
		return &NewMessageEvent{
			EventID:   e.EventID,
			MessageID: e.MessageID,
			RequestID: e.RequestID,
			CreatedAt: e.CreatedAt,
			IsService: e.IsService,
			Body:      e.MessageBody,
			EventType: EventTypeNewMessageEvent,
			AuthorID:  pointer.PtrWithZeroAsNil(e.AuthorID),
		}, nil
	case *eventstream.MessageSentEvent:
		return &MessageSentEvent{
			EventID:   e.EventID,
			MessageID: e.MessageID,
			RequestID: e.RequestID,
			EventType: EventTypeMessageSentEvent,
		}, nil
	case *eventstream.MessageBlockedEvent:
		return &MessageBlockedEvent{
			EventID:   e.EventID,
			MessageID: e.MessageID,
			RequestID: e.RequestID,
			EventType: EventTypeMessageBlockedEvent,
		}, nil
	}
	return nil, errors.New("invalid type")
}
