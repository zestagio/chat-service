package managerload_test

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	managerload "github.com/zestagio/chat-service/internal/services/manager-load"
	managerloadmocks "github.com/zestagio/chat-service/internal/services/manager-load/mocks"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
)

type ServiceSuite struct {
	testingh.ContextSuite

	ctrl *gomock.Controller

	problemsRepo *managerloadmocks.MockproblemsRepository
	managerLoad  *managerload.Service
}

func TestServiceSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.problemsRepo = managerloadmocks.NewMockproblemsRepository(s.ctrl)

	var err error
	s.managerLoad, err = managerload.New(managerload.NewOptions(2, s.problemsRepo))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *ServiceSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *ServiceSuite) TestCanManagerTakeProblem_ManagerFree() {
	// Arrange.
	managerID := types.NewUserID()

	s.problemsRepo.EXPECT().GetManagerOpenProblemsCount(gomock.Any(), managerID).
		Return(0, nil)

	// Action.
	can, err := s.managerLoad.CanManagerTakeProblem(s.Ctx, managerID)

	// Assert.
	s.Require().NoError(err)
	s.Require().True(can)
}

func (s *ServiceSuite) TestCanManagerTakeProblem_OneProblem() {
	// Arrange.
	managerID := types.NewUserID()

	s.problemsRepo.EXPECT().GetManagerOpenProblemsCount(gomock.Any(), managerID).
		Return(1, nil)

	// Action.
	can, err := s.managerLoad.CanManagerTakeProblem(s.Ctx, managerID)

	// Assert.
	s.Require().NoError(err)
	s.Require().True(can)
}

func (s *ServiceSuite) TestCanManagerTakeProblem_TwoProblems_Busy() {
	// Arrange.
	managerID := types.NewUserID()

	s.problemsRepo.EXPECT().GetManagerOpenProblemsCount(gomock.Any(), managerID).
		Return(2, nil)

	// Action.
	can, err := s.managerLoad.CanManagerTakeProblem(s.Ctx, managerID)

	// Assert.
	s.Require().NoError(err)
	s.Require().False(can)
}

func (s *ServiceSuite) TestCanManagerTakeProblem_Error() {
	// Arrange.
	managerID := types.NewUserID()
	repoErr := errors.New("unknown")

	s.problemsRepo.EXPECT().GetManagerOpenProblemsCount(gomock.Any(), managerID).
		Return(0, repoErr)

	// Action.
	can, err := s.managerLoad.CanManagerTakeProblem(s.Ctx, managerID)

	// Assert.
	s.Require().Error(err)
	s.Require().False(can)
}
