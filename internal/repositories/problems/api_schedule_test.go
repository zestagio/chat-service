//go:build integration

package problemsrepo_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
)

const (
	msgBody = "whatever"
)

type ProblemsRepoScheduleAPISuite struct {
	testingh.DBSuite
	repo *problemsrepo.Repo
}

func TestProblemsRepoScheduleAPISuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ProblemsRepoScheduleAPISuite{DBSuite: testingh.NewDBSuite("TestProblemsRepoScheduleAPISuite")})
}

func (s *ProblemsRepoScheduleAPISuite) SetupSuite() {
	s.DBSuite.SetupSuite()

	var err error

	s.repo, err = problemsrepo.New(problemsrepo.NewOptions(s.Database))
	s.Require().NoError(err)
}

func (s *ProblemsRepoScheduleAPISuite) SetupSubTest() {
	s.DBSuite.SetupTest()

	s.Database.Message(s.Ctx).Delete().ExecX(s.Ctx)
	s.Database.Problem(s.Ctx).Delete().ExecX(s.Ctx)
	s.Database.Chat(s.Ctx).Delete().ExecX(s.Ctx)
}

func (s *ProblemsRepoScheduleAPISuite) Test_GetAvailableProblems() {
	s.Run("two problems with visible messages", func() {
		s.createMessage(true)
		s.createMessage(true)

		{
			problems, err := s.repo.GetAvailableProblems(s.Ctx)

			s.Require().NoError(err)
			s.Require().Len(problems, 2)
		}
	})

	s.Run("no messages visible for manager", func() {
		s.createMessage(false)

		{
			problems, err := s.repo.GetAvailableProblems(s.Ctx)

			s.Require().NoError(err)
			s.Require().Empty(problems)
		}
	})

	s.Run("get first problem when second with manager", func() {
		s.createMessage(true)

		clientID := types.NewUserID()
		problemID, chatID := s.createProblemAndChatWithManager(clientID)
		_ = s.Database.Message(s.Ctx).Create().
			SetID(types.NewMessageID()).
			SetChatID(chatID).
			SetAuthorID(clientID).
			SetProblemID(problemID).
			SetBody(msgBody).
			SetIsBlocked(false).
			SetIsVisibleForClient(true).
			SetIsVisibleForManager(true).
			SetInitialRequestID(types.NewRequestID()).
			SaveX(s.Ctx)

		{
			problems, err := s.repo.GetAvailableProblems(s.Ctx)

			s.Require().NoError(err)
			s.Require().Len(problems, 1)
		}
	})
}

func (s *ProblemsRepoScheduleAPISuite) Test_SetManagerForProblem() {
	s.Run("set manager for problem", func() {
		authorID := types.NewUserID()
		managerID := types.NewUserID()

		problemID, chatID := s.createProblemAndChat(authorID)
		_ = s.Database.Message(s.Ctx).Create().
			SetID(types.NewMessageID()).
			SetChatID(chatID).
			SetAuthorID(authorID).
			SetProblemID(problemID).
			SetBody(msgBody).
			SetIsBlocked(false).
			SetIsVisibleForClient(true).
			SetIsVisibleForManager(true).
			SetIsService(false).
			SetInitialRequestID(types.NewRequestID()).
			SaveX(s.Ctx)

		err := s.repo.SetManagerForProblem(s.Ctx, problemID, managerID)

		{
			s.Require().NoError(err)
			problem, err := s.Database.Problem(s.Ctx).Get(s.Ctx, problemID)

			s.Require().NoError(err)
			s.Equal(managerID, problem.ManagerID)
		}
	})

	s.Run("no found available problems", func() {
		s.Database.Problem(s.Ctx).Delete().ExecX(s.Ctx)

		managerID := types.NewUserID()
		authorID := types.NewUserID()

		problemID, chatID := s.createProblemAndChatWithManager(managerID)

		_ = s.Database.Message(s.Ctx).Create().
			SetID(types.NewMessageID()).
			SetChatID(chatID).
			SetAuthorID(authorID).
			SetProblemID(problemID).
			SetBody(msgBody).
			SetIsBlocked(false).
			SetIsVisibleForClient(true).
			SetIsVisibleForManager(true).
			SetIsService(false).
			SetInitialRequestID(types.NewRequestID()).
			SaveX(s.Ctx)

		err := s.repo.SetManagerForProblem(s.Ctx, problemID, managerID)

		s.Require().ErrorIs(err, problemsrepo.ErrProblemNotFound)
	})
}

func (s *ProblemsRepoScheduleAPISuite) Test_GetProblemReqID() {
	s.Run("request id from first message in problem", func() {
		authorID := types.NewUserID()
		requestID := types.NewRequestID()

		problemID, chatID := s.createProblemAndChat(authorID)
		firstMsg := s.Database.Message(s.Ctx).Create().
			SetID(types.NewMessageID()).
			SetChatID(chatID).
			SetAuthorID(authorID).
			SetProblemID(problemID).
			SetBody(msgBody).
			SetIsBlocked(false).
			SetIsVisibleForClient(true).
			SetIsVisibleForManager(true).
			SetIsService(false).
			SetInitialRequestID(requestID).
			SaveX(s.Ctx)

		_ = s.Database.Message(s.Ctx).Create().
			SetID(types.NewMessageID()).
			SetChatID(chatID).
			SetAuthorID(authorID).
			SetProblemID(problemID).
			SetBody(msgBody).
			SetIsBlocked(false).
			SetIsVisibleForClient(true).
			SetIsVisibleForManager(true).
			SetIsService(false).
			SetInitialRequestID(types.NewRequestID()).
			SaveX(s.Ctx)

		expReqID, err := s.repo.GetProblemRequestID(s.Ctx, problemID)

		{
			s.Require().NoError(err)
			s.Require().Equal(expReqID, firstMsg.InitialRequestID)
		}
	})

	s.Run("no found request id when no available problems", func() {
		authorID := types.NewUserID()

		problemID, _ := s.createProblemAndChatWithManager(authorID)

		_, err := s.repo.GetProblemRequestID(s.Ctx, problemID)

		{
			s.Require().ErrorIs(err, problemsrepo.ErrReqIDNotFount)
		}
	})

	s.Run("no found request id when no visible message", func() {
		authorID := types.NewUserID()
		requestID := types.NewRequestID()

		problemID, chatID := s.createProblemAndChat(authorID)
		firstMsg := s.Database.Message(s.Ctx).Create().
			SetID(types.NewMessageID()).
			SetChatID(chatID).
			SetAuthorID(authorID).
			SetProblemID(problemID).
			SetBody(msgBody).
			SetIsBlocked(false).
			SetIsVisibleForClient(true).
			SetIsVisibleForManager(false).
			SetIsService(false).
			SetInitialRequestID(requestID).
			SaveX(s.Ctx)

		expReqID, err := s.repo.GetProblemRequestID(s.Ctx, problemID)

		{
			s.Require().ErrorIs(err, problemsrepo.ErrReqIDNotFount)
			s.NotEqual(expReqID, firstMsg.InitialRequestID)
		}
	})

	s.Run("no found request id when no messages", func() {
		authorID := types.NewUserID()
		problemID, _ := s.createProblemAndChat(authorID)

		expReqID, err := s.repo.GetProblemRequestID(s.Ctx, problemID)

		{
			s.Require().ErrorIs(err, problemsrepo.ErrReqIDNotFount)
			s.Equal(expReqID, types.RequestIDNil)
		}
	})
}

func (s *ProblemsRepoScheduleAPISuite) createMessage(isVisibleForManager bool) {
	s.T().Helper()

	authorID := types.NewUserID()
	problemID, chatID := s.createProblemAndChat(authorID)
	msgID := types.NewMessageID()

	_, err := s.Database.Message(s.Ctx).Create().
		SetID(msgID).
		SetChatID(chatID).
		SetAuthorID(authorID).
		SetProblemID(problemID).
		SetBody(msgBody).
		SetIsBlocked(false).
		SetIsVisibleForClient(true).
		SetIsVisibleForManager(isVisibleForManager).
		SetIsService(false).
		SetInitialRequestID(types.NewRequestID()).
		Save(s.Ctx)
	s.Require().NoError(err)
}

func (s *ProblemsRepoScheduleAPISuite) createProblemAndChat(clientID types.UserID) (types.ProblemID, types.ChatID) {
	s.T().Helper()

	chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
	s.Require().NoError(err)

	problem, err := s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).Save(s.Ctx)
	s.Require().NoError(err)

	return problem.ID, chat.ID
}

func (s *ProblemsRepoScheduleAPISuite) createProblemAndChatWithManager(clientID types.UserID) (types.ProblemID, types.ChatID) {
	s.T().Helper()

	managerID := types.NewUserID()

	chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
	s.Require().NoError(err)

	problem, err := s.Database.Problem(s.Ctx).Create().
		SetChatID(chat.ID).
		SetManagerID(managerID).
		Save(s.Ctx)
	s.Require().NoError(err)

	return problem.ID, chat.ID
}
