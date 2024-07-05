package managerload_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	managerload "github.com/zestagio/chat-service/internal/services/manager-load"
	managerloadmocks "github.com/zestagio/chat-service/internal/services/manager-load/mocks"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
)

const maxProblemAtSameTime = 5

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
	s.managerLoad, err = managerload.New(managerload.NewOptions(maxProblemAtSameTime, s.problemsRepo))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *ServiceSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *ServiceSuite) TestCanManagerTakeProblem() {
	for _, tt := range []struct {
		int
		bool
	}{
		{maxProblemAtSameTime - 1, true},
		{maxProblemAtSameTime, false},
		{maxProblemAtSameTime + 1, false},
	} {
		s.Run("", func() {
			manager := types.NewUserID()
			s.problemsRepo.EXPECT().GetManagerOpenProblemsCount(gomock.Any(), manager).Return(tt.int, nil)

			result, err := s.managerLoad.CanManagerTakeProblem(s.Ctx, manager)
			s.Require().NoError(err)
			s.Equal(tt.bool, result)
		})
	}
}

func (s *ServiceSuite) TestCanManagerTakeProblem_Error() {
	s.problemsRepo.EXPECT().GetManagerOpenProblemsCount(gomock.Any(), gomock.Any()).Return(0, context.Canceled)
	result, err := s.managerLoad.CanManagerTakeProblem(s.Ctx, types.NewUserID())
	s.Require().Error(err)
	s.False(result)
}
