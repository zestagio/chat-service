package freehandssignal_test

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
	freehandssignal "github.com/zestagio/chat-service/internal/usecases/manager/free-hands-signal"
	freehandssignalmocks "github.com/zestagio/chat-service/internal/usecases/manager/free-hands-signal/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl      *gomock.Controller
	mLoadMock *freehandssignalmocks.MockmanagerLoadService
	mPool     *freehandssignalmocks.MockmanagerPool
	uCase     freehandssignal.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mLoadMock = freehandssignalmocks.NewMockmanagerLoadService(s.ctrl)
	s.mPool = freehandssignalmocks.NewMockmanagerPool(s.ctrl)

	var err error
	s.uCase, err = freehandssignal.New(freehandssignal.NewOptions(s.mLoadMock, s.mPool))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *UseCaseSuite) TestRequestValidationError() {
	// Arrange.
	req := freehandssignal.Request{}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestManagerCannotTakeProblem() {
	s.Run("unknown error", func() {
		// Arrange.
		managerID := types.NewUserID()

		s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).Return(false, errors.New("unexpected"))

		req := freehandssignal.Request{
			ID:        types.NewRequestID(),
			ManagerID: managerID,
		}

		// Action.
		_, err := s.uCase.Handle(s.Ctx, req)

		// Assert.
		s.Require().Error(err)
	})

	s.Run("manager overloaded", func() {
		// Arrange.
		managerID := types.NewUserID()

		s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).Return(false, nil)

		req := freehandssignal.Request{
			ID:        types.NewRequestID(),
			ManagerID: managerID,
		}

		// Action.
		_, err := s.uCase.Handle(s.Ctx, req)

		// Assert.
		s.Require().ErrorIs(err, freehandssignal.ErrManagerOverloaded)
	})
}

func (s *UseCaseSuite) TestPutInThePoolUnexpectedError() {
	managerID := types.NewUserID()

	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).Return(true, nil)
	s.mPool.EXPECT().Put(gomock.Any(), managerID).Return(errors.New("unexpected"))

	req := freehandssignal.Request{
		ID:        types.NewRequestID(),
		ManagerID: managerID,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestSuccessStory() {
	managerID := types.NewUserID()

	s.mLoadMock.EXPECT().CanManagerTakeProblem(gomock.Any(), managerID).Return(true, nil)
	s.mPool.EXPECT().Put(gomock.Any(), managerID).Return(nil)

	req := freehandssignal.Request{
		ID:        types.NewRequestID(),
		ManagerID: managerID,
	}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
}
