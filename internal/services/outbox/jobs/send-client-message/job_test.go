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
	"github.com/zestagio/chat-service/internal/services/outbox/jobs"
	sendclientmessagejob "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-client-message"
	sendclientmessagejobmocks "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-client-message/mocks"
	"github.com/zestagio/chat-service/internal/types"
)

func TestJob_Handle(t *testing.T) {
	// Arrange.
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	msgProducer := sendclientmessagejobmocks.NewMockmessageProducer(ctrl)
	msgRepo := sendclientmessagejobmocks.NewMockmessageRepository(ctrl)
	eventStream := sendclientmessagejobmocks.NewMockeventStream(ctrl)
	job, err := sendclientmessagejob.New(sendclientmessagejob.NewOptions(msgProducer, msgRepo, eventStream))
	require.NoError(t, err)

	clientID := types.NewUserID()
	msgID := types.NewMessageID()
	chatID := types.NewChatID()
	reqID := types.NewRequestID()
	createdAt := time.Now()
	const body = "Hello!"

	msg := messagesrepo.Message{
		ID:                  msgID,
		ChatID:              chatID,
		AuthorID:            clientID,
		InitialRequestID:    reqID,
		Body:                body,
		CreatedAt:           createdAt,
		IsVisibleForClient:  true,
		IsVisibleForManager: false,
		IsBlocked:           false,
		IsService:           false,
	}
	msgRepo.EXPECT().GetMessageByID(gomock.Any(), msgID).Return(&msg, nil)

	msgProducer.EXPECT().ProduceMessage(gomock.Any(), msgproducer.Message{
		ID:         msgID,
		ChatID:     chatID,
		Body:       body,
		FromClient: true,
	}).Return(nil)

	eventStream.EXPECT().Publish(gomock.Any(), clientID, newMessageEventMatcher(eventstream.NewNewMessageEvent(
		types.NewEventID(),
		reqID,
		chatID,
		msgID,
		clientID,
		createdAt,
		body,
		false,
	),
	)).Return(nil)

	// Action & assert.
	payload, err := jobs.MarshalPayload(msgID)
	require.NoError(t, err)

	err = job.Handle(ctx, payload)
	require.NoError(t, err)
}

type eqNewMessageEventParamsMatcher struct {
	arg *eventstream.NewMessageEvent
}

func newMessageEventMatcher(ev *eventstream.NewMessageEvent) gomock.Matcher {
	return &eqNewMessageEventParamsMatcher{arg: ev}
}

func (e *eqNewMessageEventParamsMatcher) Matches(x any) bool {
	ev, ok := x.(*eventstream.NewMessageEvent)
	if !ok {
		return false
	}

	switch {
	case e.arg.RequestID.String() != ev.RequestID.String():
		return false
	case !e.arg.AuthorID.Matches(ev.AuthorID):
		return false
	case !e.arg.ChatID.Matches(ev.ChatID):
		return false
	case !e.arg.MessageID.Matches(ev.MessageID):
		return false
	case e.arg.MessageBody != ev.MessageBody:
		return false
	case e.arg.IsService != ev.IsService:
		return false
	case e.arg.CreatedAt.String() != ev.CreatedAt.String():
		return false
	}

	return true
}

func (e *eqNewMessageEventParamsMatcher) String() string {
	return fmt.Sprintf("%v", e.arg)
}
