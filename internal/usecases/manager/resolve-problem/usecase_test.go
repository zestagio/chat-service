package resolveproblem_test

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
	closechatjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/close-chat"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
	resolveproblem "github.com/zestagio/chat-service/internal/usecases/manager/resolve-problem"
	resolveproblemmocks "github.com/zestagio/chat-service/internal/usecases/manager/resolve-problem/mocks"
)

const msgBody = "Your question has been marked as resolved.\nThank you for being with us!"

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl        *gomock.Controller
	msgRepo     *resolveproblemmocks.MockmessageRepository
	problemRepo *resolveproblemmocks.MockproblemsRepository
	txtor       *resolveproblemmocks.Mocktransactor
	outBoxSvc   *resolveproblemmocks.MockoutboxService
	uCase       resolveproblem.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.msgRepo = resolveproblemmocks.NewMockmessageRepository(s.ctrl)
	s.problemRepo = resolveproblemmocks.NewMockproblemsRepository(s.ctrl)
	s.outBoxSvc = resolveproblemmocks.NewMockoutboxService(s.ctrl)
	s.txtor = resolveproblemmocks.NewMocktransactor(s.ctrl)

	var err error
	s.uCase, err = resolveproblem.New(resolveproblem.NewOptions(s.msgRepo, s.problemRepo, s.outBoxSvc, s.txtor))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *UseCaseSuite) TestRequestValidationError() {
	// Arrange.
	req := resolveproblem.Request{}

	// Action.
	err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.ErrorIs(err, resolveproblem.ErrInvalidRequest)
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
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).
		Return(types.ProblemIDNil, errors.New("unexpected"))

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
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
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).
		Return(types.ProblemIDNil, problemsrepo.ErrProblemNotFound)

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestResolve_UnexpectedError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.problemRepo.EXPECT().Resolve(gomock.Any(), problemID).Return(errors.New("unexpected"))

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestCreateClientService_UnexpectedError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.problemRepo.EXPECT().Resolve(gomock.Any(), problemID).Return(nil)
	s.msgRepo.EXPECT().CreateClientService(gomock.Any(), problemID, chatID, msgBody).Return(nil, errors.New("unexpected"))

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestPubJobError_UnexpectedError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()
	messageID := types.NewMessageID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.problemRepo.EXPECT().Resolve(gomock.Any(), problemID).Return(nil)
	s.msgRepo.EXPECT().CreateClientService(gomock.Any(), problemID, chatID, msgBody).
		Return(&messagesrepo.Message{
			ID:                 messageID,
			ChatID:             chatID,
			ProblemID:          problemID,
			Body:               msgBody,
			CreatedAt:          time.Now(),
			IsVisibleForClient: true,
			IsService:          true,
		}, nil)
	s.outBoxSvc.EXPECT().Put(gomock.Any(), closechatjob.Name, gomock.Any(), gomock.Any()).
		Return(types.JobIDNil, errors.New("unexpected"))

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestTransactionError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()
	messageID := types.NewMessageID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			_ = f(ctx)
			return sql.ErrTxDone
		})
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.problemRepo.EXPECT().Resolve(gomock.Any(), problemID).Return(nil)
	s.msgRepo.EXPECT().CreateClientService(gomock.Any(), problemID, chatID, msgBody).
		Return(&messagesrepo.Message{
			ID:                 messageID,
			ChatID:             chatID,
			ProblemID:          problemID,
			Body:               msgBody,
			CreatedAt:          time.Now(),
			IsVisibleForClient: true,
			IsService:          true,
		}, nil)
	s.outBoxSvc.EXPECT().Put(gomock.Any(), closechatjob.Name, gomock.Any(), gomock.Any()).
		Return(types.JobIDNil, nil)

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestProblemResolvedSuccessfully() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()
	managerID := types.NewUserID()
	problemID := types.NewProblemID()
	messageID := types.NewMessageID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.problemRepo.EXPECT().Resolve(gomock.Any(), problemID).Return(nil)
	s.msgRepo.EXPECT().CreateClientService(gomock.Any(), problemID, chatID, msgBody).
		Return(&messagesrepo.Message{
			ID:                 messageID,
			ChatID:             chatID,
			ProblemID:          problemID,
			Body:               msgBody,
			CreatedAt:          time.Now(),
			IsVisibleForClient: true,
			IsService:          true,
		}, nil)
	s.outBoxSvc.EXPECT().Put(gomock.Any(), closechatjob.Name, gomock.Any(), gomock.Any()).
		Return(types.JobIDNil, nil)

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action
	err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
}
