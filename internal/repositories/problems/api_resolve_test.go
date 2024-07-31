//go:build integration

package problemsrepo_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	"github.com/zestagio/chat-service/internal/store/problem"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
)

type ProblemsRepoResolveAPISuite struct {
	testingh.DBSuite
	repo *problemsrepo.Repo
}

func TestProblemsResolveAPISuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ProblemsRepoResolveAPISuite{DBSuite: testingh.NewDBSuite("ProblemsRepoResolveAPISuite")})
}

func (s *ProblemsRepoResolveAPISuite) SetupSuite() {
	s.DBSuite.SetupSuite()

	var err error

	s.repo, err = problemsrepo.New(problemsrepo.NewOptions(s.Database))
	s.Require().NoError(err)
}

func (s *ProblemsRepoResolveAPISuite) Test_Resolve() {
	s.Run("problem not exist", func() {
		// Arrange.
		problemID := types.NewProblemID()

		// Action.
		err := s.repo.Resolve(s.Ctx, problemID)

		// Assert.
		s.Require().Error(err)
	})

	s.Run("problem resolved successfully", func() {
		// Arrange.
		managerID := types.NewUserID()

		_, problemID := s.createChatWithProblemAssignedTo(managerID)

		// Action.
		err := s.repo.Resolve(s.Ctx, problemID)

		// Assert.
		s.Require().NoError(err)
		s.NotEmpty(problem.ResolvedAt)
	})
}

func (s *ProblemsRepoResolveAPISuite) createChatWithProblemAssignedTo(managerID types.UserID) (types.ChatID, types.ProblemID) {
	s.T().Helper()

	chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(types.NewUserID()).Save(s.Ctx)
	s.Require().NoError(err)

	p, err := s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).SetManagerID(managerID).Save(s.Ctx)
	s.Require().NoError(err)

	return chat.ID, p.ID
}
