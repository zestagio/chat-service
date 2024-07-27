package closechatjob_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
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
	managerLoad := closechatjobmocks.NewMockmanagerLoadService(ctrl)
	job, err := closechatjob.New(closechatjob.NewOptions(eventStream, chatsRepo, problemRepo, managerLoad))
	require.NoError(t, err)

	reqID := types.NewRequestID()
	clientID := types.NewUserID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()

	problem := problemsrepo.Problem{
		ID:        problemID,
		ChatID:    chatID,
		ManagerID: managerID,
	}
	problemRepo.EXPECT().GetProblemByID(gomock.Any(), problemID).Return(&problem, nil)
	problemRepo.EXPECT().GetProblemRequestID(ctx, problemID).Return(reqID, nil)

	chat := chatsrepo.Chat{
		ID:       chatID,
		ClientID: clientID,
	}
	chatsRepo.EXPECT().GetChatByID(gomock.Any(), chatID).Return(&chat, nil)

	managerLoad.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).Return(true, nil)

	eventStream.EXPECT().Publish(gomock.Any(), clientID, newMessageEventMatcher{
		NewMessageEvent: &eventstream.NewMessageEvent{
			EventID:     types.EventIDNil, // No possibility to check.
			RequestID:   reqID,
			ChatID:      chatID,
			MessageID:   types.MessageIDNil, // No possibility to check.
			AuthorID:    types.UserIDNil,    // No possibility to check.
			CreatedAt:   time.Now(),         // No possibility to check.
			MessageBody: closechatjob.CloseMsgBody,
			IsService:   true,
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
	err = job.Handle(ctx, simpleid.MustMarshal(problemID))
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
