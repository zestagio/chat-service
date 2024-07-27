//go:build integration

package problemsrepo_test

import (
	"testing"
	"time"

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
		managerID := types.NewUserID()
		chatID := types.NewChatID()

		// Action.
		_, err := s.repo.Resolve(s.Ctx, managerID, chatID)

		// Assert.
		s.ErrorIs(err, problemsrepo.ErrProblemNotFound)
	})

	s.Run("problem assigned to other manager", func() {
		// Arrange.
		managerID := types.NewUserID()
		otherManagerID := types.NewUserID()

		chatID, _ := s.createChatWithProblemAssignedTo(managerID)
		s.createChatWithProblemAssignedTo(otherManagerID)

		// Action.
		_, err := s.repo.Resolve(s.Ctx, otherManagerID, chatID)

		// Assert.
		s.ErrorIs(err, problemsrepo.ErrProblemNotFound)
	})

	s.Run("problem assigned to other chat", func() {
		// Arrange.
		managerID := types.NewUserID()

		s.createChatWithProblemAssignedTo(managerID)

		// Action.
		_, err := s.repo.Resolve(s.Ctx, managerID, types.NewChatID())

		// Assert.
		s.ErrorIs(err, problemsrepo.ErrProblemNotFound)
	})

	s.Run("problem already resolved", func() {
		// Arrange.
		managerID := types.NewUserID()

		chatID, problemID := s.createChatWithProblemAssignedTo(managerID)

		_, err := s.Database.Problem(s.Ctx).UpdateOneID(problemID).SetResolvedAt(time.Now()).Save(s.Ctx)
		s.Require().NoError(err)

		// Action.
		_, err = s.repo.Resolve(s.Ctx, managerID, chatID)

		// Assert.
		s.ErrorIs(err, problemsrepo.ErrProblemNotFound)
	})

	s.Run("problem resolved successfully", func() {
		// Arrange.
		managerID := types.NewUserID()

		chatID, problemID := s.createChatWithProblemAssignedTo(managerID)

		// Action.
		result, err := s.repo.Resolve(s.Ctx, managerID, chatID)

		// Assert.
		s.Require().NoError(err)

		dbProblem := s.Database.Problem(s.Ctx).GetX(s.Ctx, problemID)

		s.NotEmpty(problem.ResolvedAt)
		s.Equal(problemID, result)
		s.Equal(managerID, dbProblem.ManagerID)
		s.Equal(chatID, dbProblem.ChatID)
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
