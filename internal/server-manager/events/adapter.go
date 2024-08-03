package managerevents

import (
	"fmt"

	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	websocketstream "github.com/zestagio/chat-service/internal/websocket-stream"
)

var _ websocketstream.EventAdapter = Adapter{}

type Adapter struct{}

func (Adapter) Adapt(sEvent eventstream.Event) (any, error) {
	var event Event
	var err error

	switch v := sEvent.(type) {
	case *eventstream.NewChatEvent:
		event.EventId = v.EventID
		event.RequestId = v.RequestID

		err = event.FromNewChatEvent(NewChatEvent{
			ChatId:              v.ChatID,
			ClientId:            v.ClientID,
			CanTakeMoreProblems: v.CanTakeMoreProblems,
		})

	case *eventstream.NewMessageEvent:
		event.EventId = v.EventID
		event.RequestId = v.RequestID

		err = event.FromNewMessageEvent(NewMessageEvent{
			AuthorId:  v.AuthorID,
			Body:      v.MessageBody,
			ChatId:    v.ChatID,
			CreatedAt: v.CreatedAt,
			MessageId: v.MessageID,
		})

	case *eventstream.ChatClosedEvent:
		event.EventId = v.EventID
		event.RequestId = v.RequestID

		err = event.FromChatClosedEvent(ChatClosedEvent{
			CanTakeMoreProblems: v.CanTakeMoreProblems,
			ChatId:              v.ChatID,
		})

	default:
		return nil, fmt.Errorf("unknown manager event: %v (%T)", v, v)
	}
	if err != nil {
		return nil, err
	}
	if event.EventId.IsZero() || event.RequestId.IsZero() {
		panic(fmt.Sprintf("incomplete event: %#v", event))
	}

	return event, nil
}
