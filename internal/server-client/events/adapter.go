package clientevents

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
	case *eventstream.NewMessageEvent:
		event.EventId = v.EventID
		event.RequestId = v.RequestID

		err = event.FromNewMessageEvent(NewMessageEvent{
			AuthorId:  v.AuthorID.AsPointer(),
			Body:      v.MessageBody,
			CreatedAt: v.CreatedAt,
			IsService: v.IsService,
			MessageId: v.MessageID,
		})

	case *eventstream.MessageSentEvent:
		event.EventId = v.EventID
		event.RequestId = v.RequestID

		err = event.FromMessageSentEvent(MessageSentEvent{
			MessageId: v.MessageID,
		})

	case *eventstream.MessageBlockedEvent:
		event.EventId = v.EventID
		event.RequestId = v.RequestID

		err = event.FromMessageBlockedEvent(MessageBlockedEvent{
			MessageId: v.MessageID,
		})

	default:
		return nil, fmt.Errorf("unknown client event: %v (%T)", v, v)
	}
	if err != nil {
		return nil, err
	}
	if event.EventId.IsZero() || event.RequestId.IsZero() {
		panic(fmt.Sprintf("incomplete event: %#v", event))
	}

	return event, nil
}
