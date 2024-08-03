package resolveproblem_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	problemresolvedjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/problem-resolved"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
	resolveproblem "github.com/zestagio/chat-service/internal/usecases/manager/resolve-problem"
	resolveproblemmocks "github.com/zestagio/chat-service/internal/usecases/manager/resolve-problem/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl        *gomock.Controller
	msgRepo     *resolveproblemmocks.MockmessagesRepository
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
	s.msgRepo = resolveproblemmocks.NewMockmessagesRepository(s.ctrl)
	s.outBoxSvc = resolveproblemmocks.NewMockoutboxService(s.ctrl)
	s.problemRepo = resolveproblemmocks.NewMockproblemsRepository(s.ctrl)
	s.txtor = resolveproblemmocks.NewMocktransactor(s.ctrl)

	var err error
	s.uCase, err = resolveproblem.New(resolveproblem.NewOptions(s.msgRepo, s.outBoxSvc, s.problemRepo, s.txtor))
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
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestGetAssignedProblemID_UnexpectedError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()

	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).
		Return(types.ProblemIDNil, errors.New("unexpected"))

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestGetAssignedProblemID_ProblemNotFoundError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()

	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).
		Return(types.ProblemIDNil, problemsrepo.ErrAssignedProblemNotFound)

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().ErrorIs(err, resolveproblem.ErrAssignedProblemNotFound)
}

func (s *UseCaseSuite) TestResolveProblemError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})

	problemID := types.NewProblemID()
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.problemRepo.EXPECT().ResolveProblem(gomock.Any(), reqID, problemID).Return(errors.New("unexpected"))

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestCreateServiceMessageForClientError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})

	problemID := types.NewProblemID()
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.problemRepo.EXPECT().ResolveProblem(gomock.Any(), reqID, problemID).Return(nil)

	s.msgRepo.EXPECT().CreateServiceMessageForClient(gomock.Any(), reqID, problemID, chatID, gomock.Any()).
		Return(types.MessageIDNil, errors.New("unexpected"))

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestPutJobError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})

	problemID := types.NewProblemID()
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.problemRepo.EXPECT().ResolveProblem(gomock.Any(), reqID, problemID).Return(nil)

	s.msgRepo.EXPECT().CreateServiceMessageForClient(gomock.Any(), reqID, problemID, chatID, gomock.Any()).
		Return(types.NewMessageID(), nil)

	s.outBoxSvc.EXPECT().Put(gomock.Any(), problemresolvedjob.Name, gomock.Any(), gomock.Any()).
		Return(types.JobIDNil, errors.New("unexpected"))

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
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

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			_ = f(ctx)
			return sql.ErrTxDone
		})

	problemID := types.NewProblemID()
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.problemRepo.EXPECT().ResolveProblem(gomock.Any(), reqID, problemID).Return(nil)

	s.msgRepo.EXPECT().CreateServiceMessageForClient(gomock.Any(), reqID, problemID, chatID, gomock.Any()).
		Return(types.NewMessageID(), nil)

	s.outBoxSvc.EXPECT().Put(gomock.Any(), problemresolvedjob.Name, gomock.Any(), gomock.Any()).
		Return(types.NewJobID(), nil)

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestSuccessStory() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})

	problemID := types.NewProblemID()
	s.problemRepo.EXPECT().GetAssignedProblemID(gomock.Any(), managerID, chatID).Return(problemID, nil)
	s.problemRepo.EXPECT().ResolveProblem(gomock.Any(), reqID, problemID).Return(nil)

	s.msgRepo.EXPECT().CreateServiceMessageForClient(gomock.Any(), reqID, problemID, chatID, gomock.Any()).
		Return(types.NewMessageID(), nil)

	s.outBoxSvc.EXPECT().Put(gomock.Any(), problemresolvedjob.Name, gomock.Any(), gomock.Any()).
		Return(types.NewJobID(), nil)

	req := resolveproblem.Request{
		ID:        reqID,
		ManagerID: managerID,
		ChatID:    chatID,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
}
