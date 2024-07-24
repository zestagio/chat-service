package sendmanagermessagejob_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	msgproducer "github.com/zestagio/chat-service/internal/services/msg-producer"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	sendmanagermessagejob "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-manager-message"
	sendmanagermessagejobmocks "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-manager-message/mocks"
	"github.com/zestagio/chat-service/internal/types"
)

func TestJob_Handle(t *testing.T) {
	// Arrange.
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventStream := sendmanagermessagejobmocks.NewMockeventStream(ctrl)
	msgProducer := sendmanagermessagejobmocks.NewMockmessageProducer(ctrl)
	chatsRepo := sendmanagermessagejobmocks.NewMockchatRepository(ctrl)
	msgRepo := sendmanagermessagejobmocks.NewMockmessageRepository(ctrl)
	job, err := sendmanagermessagejob.New(sendmanagermessagejob.NewOptions(eventStream, msgProducer, chatsRepo, msgRepo))
	require.NoError(t, err)

	clientID := types.NewUserID()
	managerID := types.NewUserID()
	msgID := types.NewMessageID()
	chatID := types.NewChatID()
	const body = "Hello!"

	msg := messagesrepo.Message{
		ID:                  msgID,
		ChatID:              chatID,
		AuthorID:            managerID,
		ManagerID:           types.UserIDNil,
		Body:                body,
		CreatedAt:           time.Now(),
		IsVisibleForClient:  true,
		IsVisibleForManager: true,
		IsBlocked:           false,
		IsService:           false,
		InitialRequestID:    types.NewRequestID(),
	}
	msgRepo.EXPECT().GetMessageByID(gomock.Any(), msgID).Return(&msg, nil)

	chat := chatsrepo.Chat{
		ID:       chatID,
		ClientID: clientID,
	}
	chatsRepo.EXPECT().GetChatByID(gomock.Any(), msg.ChatID).Return(&chat, nil)

	msgProducer.EXPECT().ProduceMessage(gomock.Any(), msgproducer.Message{
		ID:         msgID,
		ChatID:     chatID,
		Body:       body,
		FromClient: false,
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

	eventStream.EXPECT().Publish(gomock.Any(), managerID,
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
