package resolveproblem_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	closechatjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/close-chat"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
	resolveproblem "github.com/zestagio/chat-service/internal/usecases/manager/resolve-problem"
	resolveproblemmocks "github.com/zestagio/chat-service/internal/usecases/manager/resolve-problem/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl        *gomock.Controller
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
	s.problemRepo = resolveproblemmocks.NewMockproblemsRepository(s.ctrl)
	s.outBoxSvc = resolveproblemmocks.NewMockoutboxService(s.ctrl)
	s.txtor = resolveproblemmocks.NewMocktransactor(s.ctrl)

	var err error
	s.uCase, err = resolveproblem.New(resolveproblem.NewOptions(s.problemRepo, s.outBoxSvc, s.txtor))
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

func (s *UseCaseSuite) TestResolve_UnexpectedError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.problemRepo.EXPECT().Resolve(gomock.Any(), managerID, chatID).Return(types.ProblemIDNil, errors.New("unexpected"))

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

func (s *UseCaseSuite) TestResolve_ProblemNotFound() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.problemRepo.EXPECT().Resolve(gomock.Any(), managerID, chatID).Return(types.ProblemIDNil, problemsrepo.ErrProblemNotFound)

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

func (s *UseCaseSuite) TestPubJobError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chatID := types.NewChatID()
	problemID := types.NewProblemID()

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.problemRepo.EXPECT().Resolve(gomock.Any(), managerID, chatID).
		Return(problemID, nil)
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

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			_ = f(ctx)
			return sql.ErrTxDone
		})
	s.problemRepo.EXPECT().Resolve(gomock.Any(), managerID, chatID).
		Return(problemID, nil)
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

	s.txtor.EXPECT().RunInTx(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, f func(ctx context.Context) error) error {
			return f(ctx)
		})
	s.problemRepo.EXPECT().Resolve(gomock.Any(), managerID, chatID).
		Return(problemID, nil)
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
