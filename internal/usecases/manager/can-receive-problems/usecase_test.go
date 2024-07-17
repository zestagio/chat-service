package canreceiveproblems_test

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
	canreceiveproblems "github.com/zestagio/chat-service/internal/usecases/manager/can-receive-problems"
	canreceiveproblemsmocks "github.com/zestagio/chat-service/internal/usecases/manager/can-receive-problems/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl      *gomock.Controller
	mLoadMock *canreceiveproblemsmocks.MockmanagerLoadService
	mPoolMock *canreceiveproblemsmocks.MockmanagerPool
	uCase     canreceiveproblems.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mLoadMock = canreceiveproblemsmocks.NewMockmanagerLoadService(s.ctrl)
	s.mPoolMock = canreceiveproblemsmocks.NewMockmanagerPool(s.ctrl)

	var err error
	s.uCase, err = canreceiveproblems.New(canreceiveproblems.NewOptions(s.mLoadMock, s.mPoolMock))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *UseCaseSuite) TestRequestValidationError() {
	// Arrange.
	req := canreceiveproblems.Request{}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestContains_Error() {
	// Arrange.
	managerID := types.NewUserID()

	s.mPoolMock.EXPECT().Contains(gomock.Any(), managerID).Return(false, errors.New("unexpected"))

	req := canreceiveproblems.Request{
		ID:        types.NewRequestID(),
		ManagerID: managerID,
	}

	// Action.
	result, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.False(result.Result)
}

func (s *UseCaseSuite) TestContains_True() {
	// Arrange.
	managerID := types.NewUserID()

	s.mPoolMock.EXPECT().Contains(gomock.Any(), managerID).Return(true, nil)

	req := canreceiveproblems.Request{
		ID:        types.NewRequestID(),
		ManagerID: managerID,
	}

	// Action.
	result, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.False(result.Result)
}

func (s *UseCaseSuite) TestCanManagerTakeProblem_Error() {
	// Arrange.
	managerID := types.NewUserID()

	s.mPoolMock.EXPECT().Contains(gomock.Any(), managerID).Return(false, nil)
	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).Return(false, errors.New("unexpected"))

	req := canreceiveproblems.Request{
		ID:        types.NewRequestID(),
		ManagerID: managerID,
	}

	// Action.
	result, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.False(result.Result)
}

func (s *UseCaseSuite) TestCanManagerTakeProblem_True() {
	// Arrange.
	managerID := types.NewUserID()

	s.mPoolMock.EXPECT().Contains(gomock.Any(), managerID).Return(false, nil)
	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).Return(true, nil)

	req := canreceiveproblems.Request{
		ID:        types.NewRequestID(),
		ManagerID: managerID,
	}

	// Action.
	result, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.True(result.Result)
}

func (s *UseCaseSuite) TestCanManagerTakeProblem_False() {
	// Arrange.
	managerID := types.NewUserID()

	s.mPoolMock.EXPECT().Contains(gomock.Any(), managerID).Return(false, nil)
	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).Return(false, nil)

	req := canreceiveproblems.Request{
		ID:        types.NewRequestID(),
		ManagerID: managerID,
	}

	// Action.
	result, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.False(result.Result)
}
