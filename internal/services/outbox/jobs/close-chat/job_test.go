package closechatjob_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	closechatjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/close-chat"
	closechatjobmocks "github.com/zestagio/chat-service/internal/services/outbox/jobs/close-chat/mocks"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

func TestJob_Handle(t *testing.T) {
	// Arrange.
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventStream := closechatjobmocks.NewMockeventStream(ctrl)
	chatsRepo := closechatjobmocks.NewMockchatsRepository(ctrl)
	problemRepo := closechatjobmocks.NewMockproblemRepository(ctrl)
	msgRepo := closechatjobmocks.NewMockmessageRepository(ctrl)
	managerLoad := closechatjobmocks.NewMockmanagerLoadService(ctrl)
	job, err := closechatjob.New(closechatjob.NewOptions(eventStream, chatsRepo, problemRepo, msgRepo, managerLoad))
	require.NoError(t, err)

	reqID := types.NewRequestID()
	clientID := types.NewUserID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()
	messageID := types.NewMessageID()

	msg := messagesrepo.Message{
		ID:                 messageID,
		ChatID:             chatID,
		ProblemID:          problemID,
		Body:               "Your question has been marked as resolved.\nThank you for being with us!",
		CreatedAt:          time.Now(),
		IsVisibleForClient: true,
		IsService:          true,
	}
	msgRepo.EXPECT().GetMessageByID(gomock.Any(), messageID).
		Return(&msg, nil)
	chatsRepo.EXPECT().GetChatByID(gomock.Any(), chatID).
		Return(&chatsrepo.Chat{
			ID:       chatID,
			ClientID: clientID,
		}, nil)
	problemRepo.EXPECT().GetProblemByID(gomock.Any(), problemID).Return(&problemsrepo.Problem{
		ID:        problemID,
		ChatID:    chatID,
		ManagerID: managerID,
	}, nil)
	problemRepo.EXPECT().GetProblemRequestID(gomock.Any(), problemID).Return(reqID, nil)
	managerLoad.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).Return(true, nil)

	eventStream.EXPECT().Publish(gomock.Any(), clientID, newMessageEventMatcher{
		NewMessageEvent: &eventstream.NewMessageEvent{
			EventID:     types.EventIDNil, // No possibility to check.
			RequestID:   reqID,
			ChatID:      chatID,
			MessageID:   msg.ID,
			AuthorID:    msg.AuthorID,
			CreatedAt:   msg.CreatedAt,
			MessageBody: msg.Body,
			IsService:   msg.IsService,
		},
	})

	eventStream.EXPECT().Publish(gomock.Any(), managerID, chatClosedEventMatcher{
		ChatClosedEvent: &eventstream.ChatClosedEvent{
			EventID:            types.EventIDNil, // No possibility to check.
			RequestID:          reqID,
			ChatID:             chatID,
			CanTakeMoreProblem: true,
		},
	})

	// Action & assert.
	err = job.Handle(ctx, simpleid.MustMarshal(messageID))
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
		!ev.MessageID.IsZero() &&
		!ev.CreatedAt.IsZero() &&
		ev.MessageBody == m.MessageBody &&
		ev.IsService == m.IsService
}

func (m newMessageEventMatcher) String() string {
	return fmt.Sprintf("%v", m.NewMessageEvent)
}

var _ gomock.Matcher = chatClosedEventMatcher{}

type chatClosedEventMatcher struct {
	*eventstream.ChatClosedEvent
}

func (m chatClosedEventMatcher) Matches(x any) bool {
	envelope, ok := x.(eventstream.Event)
	if !ok {
		return false
	}

	ev, ok := envelope.(*eventstream.ChatClosedEvent)
	if !ok {
		return false
	}

	return !ev.EventID.IsZero() &&
		ev.RequestID == m.RequestID &&
		ev.ChatID == m.ChatID &&
		ev.CanTakeMoreProblem == m.CanTakeMoreProblem
}

func (m chatClosedEventMatcher) String() string {
	return fmt.Sprintf("%v", m.ChatClosedEvent)
}
