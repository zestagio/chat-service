package sendmessage_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	sendmanagermessagejob "github.com/zestagio/chat-service/internal/services/outbox/jobs/send-manager-message"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
	sendmessage "github.com/zestagio/chat-service/internal/usecases/manager/send-message"
	sendmessagemocks "github.com/zestagio/chat-service/internal/usecases/manager/send-message/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl        *gomock.Controller
	msgRepo     *sendmessagemocks.MockmessagesRepository
	problemRepo *sendmessagemocks.MockproblemsRepository
	txtor       *sendmessagemocks.Mocktransactor
	outBoxSvc   *sendmessagemocks.MockoutboxService
	uCase       sendmessage.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.msgRepo = sendmessagemocks.NewMockmessagesRepository(s.ctrl)
	s.outBoxSvc = sendmessagemocks.NewMockoutboxService(s.ctrl)
	s.problemRepo = sendmessagemocks.NewMockproblemsRepository(s.ctrl)
	s.txtor = sendmessagemocks.NewMocktransactor(s.ctrl)

	var err error
	s.uCase, err = sendmessage.New(sendmessage.NewOptions(s.msgRepo, s.problemRepo, s.outBoxSvc, s.txtor))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *UseCaseSuite) TestRequestValidationError() {
	// Arrange.
	req := sendmessage.Request{}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.ErrorIs(err, sendmessage.ErrInvalidRequest)
}

func (s *UseCaseSuite) TestGetAssignedProblemID_UnexpectedError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.msgRepo.EXPECT().GetMessageByRequestID(gomock.Any(), reqID).Return(nil, errors.New("unexpected"))

	req := sendmessage.Request{
		ID:          reqID,
		ManagerID:   managerID,
		ChatID:      chatID,
		MessageBody: "Hello!",
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestGetMessageByRequestID_MsgFound() {
	// Arrange.
	reqID := types.NewRequestID()
	clientID := types.NewUserID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	const msgBody = "Hello!"
	createdAt := time.Now()
	messageID := types.NewMessageID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.msgRepo.EXPECT().GetMessageByRequestID(gomock.Any(), reqID).
		Return(&messagesrepo.Message{
			ID:                  messageID,
			ChatID:              types.NewChatID(),
			AuthorID:            clientID,
			Body:                msgBody,
			CreatedAt:           createdAt,
			IsVisibleForClient:  true,
			IsVisibleForManager: true,
			IsBlocked:           false,
			IsService:           false,
		}, nil)

	req := sendmessage.Request{
		ID:          reqID,
		ManagerID:   managerID,
		ChatID:      chatID,
		MessageBody: msgBody,
	}

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Require().Equal(messageID, resp.MessageID)
	s.Require().True(createdAt.Equal(resp.CreatedAt))
}

func (s *UseCaseSuite) TestGetAssignedProblemID_ProblemNotFound() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.msgRepo.EXPECT().GetMessageByRequestID(gomock.Any(), reqID).Return(nil, messagesrepo.ErrMsgNotFound)
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).
		Return(types.ProblemIDNil, problemsrepo.ErrProblemNotFound)

	req := sendmessage.Request{
		ID:          reqID,
		ManagerID:   managerID,
		ChatID:      chatID,
		MessageBody: "Hello!",
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestCreateMessageError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()
	const msgBody = "Hello!"

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.msgRepo.EXPECT().GetMessageByRequestID(gomock.Any(), reqID).Return(nil, messagesrepo.ErrMsgNotFound)
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).
		Return(problemID, nil)
	s.msgRepo.EXPECT().
		CreateFullVisible(gomock.Any(), reqID, problemID, chatID, managerID, msgBody).
		Return(nil, errors.New("unexpected"))

	req := sendmessage.Request{
		ID:          reqID,
		ManagerID:   managerID,
		ChatID:      chatID,
		MessageBody: msgBody,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestPubJobError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()
	const msgBody = "Hello!"

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.msgRepo.EXPECT().GetMessageByRequestID(gomock.Any(), reqID).Return(nil, messagesrepo.ErrMsgNotFound)
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).
		Return(problemID, nil)
	s.msgRepo.EXPECT().CreateFullVisible(gomock.Any(), reqID, problemID, chatID, managerID, msgBody).
		Return(&messagesrepo.Message{ID: types.NewMessageID()}, nil)
	s.outBoxSvc.EXPECT().Put(gomock.Any(), sendmanagermessagejob.Name, gomock.Any(), gomock.Any()).
		Return(types.JobIDNil, errors.New("unexpected"))

	req := sendmessage.Request{
		ID:          reqID,
		ManagerID:   managerID,
		ChatID:      chatID,
		MessageBody: msgBody,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestTransactionError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()
	const msgBody = "Hello!"

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			_ = f(ctx)
			return sql.ErrTxDone
		})
	s.msgRepo.EXPECT().GetMessageByRequestID(gomock.Any(), reqID).Return(nil, messagesrepo.ErrMsgNotFound)
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).
		Return(problemID, nil)
	s.msgRepo.EXPECT().
		CreateFullVisible(gomock.Any(), reqID, problemID, chatID, managerID, msgBody).
		Return(&messagesrepo.Message{ID: types.NewMessageID()}, nil)
	s.outBoxSvc.EXPECT().Put(gomock.Any(), sendmanagermessagejob.Name, gomock.Any(), gomock.Any()).
		Return(types.NewJobID(), nil)

	req := sendmessage.Request{
		ID:          reqID,
		ManagerID:   managerID,
		ChatID:      chatID,
		MessageBody: msgBody,
	}

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Empty(resp.MessageID)
	s.Empty(resp.CreatedAt)
}

func (s *UseCaseSuite) TestNewMsgCreatedSuccessfully() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()
	const msgBody = "Hello!"
	createdAt := time.Now()
	messageID := types.NewMessageID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.msgRepo.EXPECT().GetMessageByRequestID(gomock.Any(), reqID).Return(nil, messagesrepo.ErrMsgNotFound)
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.msgRepo.EXPECT().CreateFullVisible(gomock.Any(), reqID, problemID, chatID, managerID, msgBody).
		Return(&messagesrepo.Message{
			ID:                  messageID,
			ChatID:              chatID,
			AuthorID:            managerID,
			Body:                msgBody,
			CreatedAt:           createdAt,
			IsVisibleForClient:  true,
			IsVisibleForManager: true,
			IsBlocked:           false,
			IsService:           false,
		}, nil)
	s.outBoxSvc.EXPECT().Put(gomock.Any(), sendmanagermessagejob.Name, gomock.Any(), gomock.Any()).
		Return(types.NewJobID(), nil)

	req := sendmessage.Request{
		ID:          reqID,
		ManagerID:   managerID,
		ChatID:      chatID,
		MessageBody: msgBody,
	}

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Require().Equal(messageID, resp.MessageID)
	s.Require().True(createdAt.Equal(resp.CreatedAt))
}
