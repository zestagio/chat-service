package freehands_test

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
	freehands "github.com/zestagio/chat-service/internal/usecases/manager/free-hands"
	freehandsmocks "github.com/zestagio/chat-service/internal/usecases/manager/free-hands/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	managerID types.UserID
	req       freehands.Request

	ctrl      *gomock.Controller
	mLoadMock *freehandsmocks.MockmanagerLoadService
	mPoolMock *freehandsmocks.MockmanagerPool
	uCase     freehands.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())

	s.mLoadMock = freehandsmocks.NewMockmanagerLoadService(s.ctrl)
	s.mPoolMock = freehandsmocks.NewMockmanagerPool(s.ctrl)

	s.managerID = types.NewUserID()

	s.req = freehands.Request{
		ID:        types.NewRequestID(),
		ManagerID: s.managerID,
	}

	var err error
	s.uCase, err = freehands.New(freehands.NewOptions(s.mLoadMock, s.mPoolMock))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *UseCaseSuite) TestRequestValidationError() {
	// Arrange.
	req := freehands.Request{}

	// Action.
	err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestSuccessPutInPool() {
	// Arrange.
	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), s.managerID).
		Return(true, nil)

	s.mPoolMock.EXPECT().Put(gomock.Any(), s.managerID).
		Return(nil)

	// Action.
	err := s.uCase.Handle(s.Ctx, s.req)

	// Assert.
	s.Require().NoError(err)
}

func (s *UseCaseSuite) TestCanTakeProblem_ErrManagerOverload() {
	// Arrange.
	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), s.managerID).
		Return(false, nil)

	// Action.
	err := s.uCase.Handle(s.Ctx, s.req)

	// Assert.
	s.Require().Error(err)
	s.Require().ErrorIs(err, freehands.ErrManagerOverloaded)
}

func (s *UseCaseSuite) TestCanTakeProblemError() {
	// Arrange.
	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), s.managerID).
		Return(false, errors.New("unexpected"))

	// Action.
	err := s.uCase.Handle(s.Ctx, s.req)

	// Assert.
	s.Require().Error(err)
	s.Require().NotErrorIs(err, freehands.ErrManagerOverloaded)
}

func (s *UseCaseSuite) TestManagerPoolError() {
	// Arrange.
	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), s.managerID).
		Return(true, nil)

	s.mPoolMock.EXPECT().Put(gomock.Any(), s.managerID).
		Return(errors.New("unexpected"))

	// Action.
	err := s.uCase.Handle(s.Ctx, s.req)

	// Assert.
	s.Require().Error(err)
	s.Require().NotErrorIs(err, freehands.ErrManagerOverloaded)
}
