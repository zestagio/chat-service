package sendclientmessagejob_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	msgproducer "github.com/zestagio/chat-service/internal/services/msg-producer"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	sendclientmessagejob "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-client-message"
	sendclientmessagejobmocks "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-client-message/mocks"
	"github.com/zestagio/chat-service/internal/types"
)

func TestJob_Handle(t *testing.T) {
	// Arrange.
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventStream := sendclientmessagejobmocks.NewMockeventStream(ctrl)
	msgProducer := sendclientmessagejobmocks.NewMockmessageProducer(ctrl)
	msgRepo := sendclientmessagejobmocks.NewMockmessageRepository(ctrl)
	job, err := sendclientmessagejob.New(sendclientmessagejob.NewOptions(eventStream, msgProducer, msgRepo))
	require.NoError(t, err)

	clientID := types.NewUserID()
	msgID := types.NewMessageID()
	chatID := types.NewChatID()
	const body = "Hello!"

	msg := messagesrepo.Message{
		ID:                  msgID,
		ChatID:              chatID,
		AuthorID:            clientID,
		Body:                body,
		CreatedAt:           time.Now(),
		IsVisibleForClient:  true,
		IsVisibleForManager: false,
		IsBlocked:           false,
		IsService:           false,
		InitialRequestID:    types.NewRequestID(),
	}
	msgRepo.EXPECT().GetMessageByID(gomock.Any(), msgID).Return(&msg, nil)

	msgProducer.EXPECT().ProduceMessage(gomock.Any(), msgproducer.Message{
		ID:         msgID,
		ChatID:     chatID,
		Body:       body,
		FromClient: true,
	}).Return(nil)

	eventStream.EXPECT().Publish(gomock.Any(), clientID,
		newMessageEventMatcher{
			NewMessageEvent: &eventstream.NewMessageEvent{
				EventID:     types.EventIDNil, // No possibility to check.
				RequestID:   msg.InitialRequestID,
				ChatID:      msg.ChatID,
				MessageID:   msg.ID,
				AuthorID:    msg.AuthorID,
				CreatedAt:   msg.CreatedAt,
				MessageBody: msg.Body,
				IsService:   false,
			},
		})

	// Action & assert.
	err = job.Handle(ctx, simpleid.MustMarshal(msgID))
	require.NoError(t, err)
}

var _ gomock.Matcher = newMessageEventMatcher{}

type newMessageEventMatcher struct {
	*eventstream.NewMessageEvent
}

func (m newMessageEventMatcher) Matches(x any) bool {
	envelope, ok := x.(eventstream.Event)
	if !ok {
		return false
	}

	ev, ok := envelope.(*eventstream.NewMessageEvent)
	if !ok {
		return false
	}

	return !ev.EventID.IsZero() &&
		ev.RequestID == m.RequestID &&
		ev.ChatID == m.ChatID &&
		ev.MessageID == m.MessageID &&
		ev.AuthorID == m.AuthorID &&
		ev.CreatedAt.Equal(m.CreatedAt) &&
		ev.MessageBody == m.MessageBody &&
		ev.IsService == m.IsService
}

func (m newMessageEventMatcher) String() string {
	return fmt.Sprintf("%v", m.NewMessageEvent)
}
