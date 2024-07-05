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
	s.Require().ErrorIs(err, canreceiveproblems.ErrInvalidRequest)
}

func (s *UseCaseSuite) TestManagerPoolError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()

	s.mPoolMock.EXPECT().Contains(gomock.Any(), managerID).
		Return(false, errors.New("unexpected"))

	req := canreceiveproblems.Request{
		ID:        reqID,
		ManagerID: managerID,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Require().ErrorIs(err, canreceiveproblems.ErrManagerPoolContains)
}

func (s *UseCaseSuite) TestManagerInPool() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()

	s.mPoolMock.EXPECT().Contains(gomock.Any(), managerID).
		Return(true, nil)

	req := canreceiveproblems.Request{
		ID:        reqID,
		ManagerID: managerID,
	}

	// Action.
	response, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Require().False(response.Available)
	s.Require().True(response.InPool)
}

func (s *UseCaseSuite) TestManagerCanTakeProblem_Error() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()

	s.mPoolMock.EXPECT().Contains(gomock.Any(), managerID).
		Return(false, nil)

	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).
		Return(false, errors.New("unexpected"))

	req := canreceiveproblems.Request{
		ID:        reqID,
		ManagerID: managerID,
	}

	// Action.
	response, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Require().ErrorIs(err, canreceiveproblems.ErrManagerLoadService)
	s.Require().False(response.Available)
	s.Require().False(response.InPool)
}

func (s *UseCaseSuite) TestManagerCanTakeProblem_True() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()

	s.mPoolMock.EXPECT().Contains(gomock.Any(), managerID).
		Return(false, nil)

	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).
		Return(true, nil)

	req := canreceiveproblems.Request{
		ID:        reqID,
		ManagerID: managerID,
	}

	// Action.
	response, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Require().True(response.Available)
	s.Require().False(response.InPool)
}

func (s *UseCaseSuite) TestManagerCanTakeProblem_False() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()

	s.mPoolMock.EXPECT().Contains(gomock.Any(), managerID).
		Return(false, nil)

	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).
		Return(false, nil)

	req := canreceiveproblems.Request{
		ID:        reqID,
		ManagerID: managerID,
	}

	// Action.
	response, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Require().False(response.Available)
	s.Require().False(response.InPool)
}
