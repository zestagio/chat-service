//go:build integration

package problemsrepo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
)

type ProblemsRepoScheduleAPISuite struct {
	testingh.DBSuite
	repo *problemsrepo.Repo
}

func TestProblemsRepoScheduleAPISuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ProblemsRepoScheduleAPISuite{DBSuite: testingh.NewDBSuite("ProblemsRepoScheduleAPISuite")})
}

func (s *ProblemsRepoScheduleAPISuite) SetupSuite() {
	s.DBSuite.SetupSuite()

	var err error

	s.repo, err = problemsrepo.New(problemsrepo.NewOptions(s.Database))
	s.Require().NoError(err)
}

func (s *ProblemsRepoScheduleAPISuite) Test_GetProblemsWithoutManager() {
	s.Run("invalid limit", func() {
		for _, l := range []int{-1, 0} {
			problems, err := s.repo.GetProblemsWithoutManager(s.Ctx, l)
			s.Require().Error(err)
			s.Empty(problems)
		}
	})

	s.Run("no open problems without manager", func() {
		clientID := types.NewUserID()
		managerID := types.NewUserID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		// Assign open problem with manager to chat.
		_, err = s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).SetManagerID(managerID).Save(s.Ctx)
		s.Require().NoError(err)

		problems, err := s.repo.GetProblemsWithoutManager(s.Ctx, 3)
		s.Require().NoError(err)
		s.Empty(problems)
	})

	s.Run("no open problems with messages visible for manager", func() {
		clientID := types.NewUserID()
		managerID := types.NewUserID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		// Problem without manager.
		_, err = s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).Save(s.Ctx)
		s.Require().NoError(err)

		// Problem without manager-visible messages.
		p, err := s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).SetManagerID(managerID).Save(s.Ctx)
		s.Require().NoError(err)

		for i := 0; i < 3; i++ {
			_, err = s.Database.Message(s.Ctx).Create().
				SetID(types.NewMessageID()).
				SetChatID(chat.ID).
				SetAuthorID(clientID).
				SetProblemID(p.ID).
				SetBody("SMS code is 4321").
				SetIsVisibleForClient(true).
				SetIsVisibleForManager(false).
				SetIsBlocked(true).
				SetIsService(false).
				SetInitialRequestID(types.NewRequestID()).
				Save(s.Ctx)
			s.Require().NoError(err)
		}

		problems, err := s.repo.GetProblemsWithoutManager(s.Ctx, 3)
		s.Require().NoError(err)
		s.Empty(problems)
	})

	s.Run("open problems without manager exist", func() {
		const (
			problemsCount = 10
		)

		clientID := types.NewUserID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		for i := 0; i < problemsCount*2; i++ {
			// Assign open problem without manager to chat.
			p, err := s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).Save(s.Ctx)
			s.Require().NoError(err)

			_, err = s.Database.Message(s.Ctx).Create().
				SetID(types.NewMessageID()).
				SetChatID(chat.ID).
				SetAuthorID(clientID).
				SetProblemID(p.ID).
				SetBody("Hello!").
				SetIsVisibleForClient(true).
				SetIsVisibleForManager(true).
				SetIsBlocked(false).
				SetIsService(false).
				SetInitialRequestID(types.NewRequestID()).
				Save(s.Ctx)
			s.Require().NoError(err)
		}

		problems, err := s.repo.GetProblemsWithoutManager(s.Ctx, problemsCount)
		s.Require().NoError(err)

		s.Len(problems, problemsCount)
		for _, p := range problems {
			s.Equal(chat.ID, p.ChatID)
		}
	})
}

func (s *ProblemsRepoScheduleAPISuite) Test_SetManagerForProblem() {
	chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(types.NewUserID()).Save(s.Ctx)
	s.Require().NoError(err)

	p, err := s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).Save(s.Ctx)
	s.Require().NoError(err)

	managerID := types.NewUserID()
	err = s.repo.SetManagerForProblem(s.Ctx, p.ID, managerID)
	s.Require().NoError(err)

	p, err = s.Database.Problem(s.Ctx).Get(s.Ctx, p.ID)
	s.Require().NoError(err)
	s.Equal(managerID, p.ManagerID)
}

func (s *ProblemsRepoScheduleAPISuite) Test_GetProblemInitialRequestID() {
	s.Run("no problem messages", func() {
		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(types.NewUserID()).Save(s.Ctx)
		s.Require().NoError(err)

		// Assign open problem with manager to chat.
		p, err := s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).SetManagerID(types.NewUserID()).Save(s.Ctx)
		s.Require().NoError(err)

		reqID, err := s.repo.GetProblemInitialRequestID(s.Ctx, p.ID)
		s.Require().Error(err)
		s.Empty(reqID)
	})

	s.Run("no manager-visible problem messages", func() {
		clientID := types.NewUserID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		// Assign open problem with manager to chat.
		p, err := s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).SetManagerID(types.NewUserID()).Save(s.Ctx)
		s.Require().NoError(err)

		_, err = s.Database.Message(s.Ctx).Create().
			SetID(types.NewMessageID()).
			SetChatID(chat.ID).
			SetAuthorID(clientID).
			SetProblemID(p.ID).
			SetBody("SMS code is 5566").
			SetIsVisibleForClient(true).
			SetIsVisibleForManager(false).
			SetIsBlocked(true).
			SetIsService(false).
			SetInitialRequestID(types.NewRequestID()).
			Save(s.Ctx)
		s.Require().NoError(err)

		reqID, err := s.repo.GetProblemInitialRequestID(s.Ctx, p.ID)
		s.Require().Error(err)
		s.Empty(reqID)
	})

	s.Run("manager-visible problem messages exist", func() {
		clientID := types.NewUserID()
		expReqID := types.NewRequestID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		// Assign open problem with manager to chat.
		p, err := s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).SetManagerID(types.NewUserID()).Save(s.Ctx)
		s.Require().NoError(err)

		_, err = s.Database.Message(s.Ctx).Create().
			SetID(types.NewMessageID()).
			SetChatID(chat.ID).
			SetAuthorID(clientID).
			SetProblemID(p.ID).
			SetBody("SMS code is 5566").
			SetIsVisibleForClient(true).
			SetIsVisibleForManager(false).
			SetIsBlocked(true).
			SetIsService(false).
			SetInitialRequestID(types.NewRequestID()).
			Save(s.Ctx)
		s.Require().NoError(err)

		time.Sleep(10 * time.Millisecond)
		_, err = s.Database.Message(s.Ctx).Create().
			SetID(types.NewMessageID()).
			SetChatID(chat.ID).
			SetAuthorID(clientID).
			SetProblemID(p.ID).
			SetBody("I need help").
			SetIsVisibleForClient(true).
			SetIsVisibleForManager(true).
			SetIsBlocked(true).
			SetIsService(false).
			SetInitialRequestID(expReqID).
			Save(s.Ctx)
		s.Require().NoError(err)

		time.Sleep(10 * time.Millisecond)
		_, err = s.Database.Message(s.Ctx).Create().
			SetID(types.NewMessageID()).
			SetChatID(chat.ID).
			SetAuthorID(clientID).
			SetProblemID(p.ID).
			SetBody("please").
			SetIsVisibleForClient(true).
			SetIsVisibleForManager(true).
			SetIsBlocked(true).
			SetIsService(false).
			SetInitialRequestID(types.NewRequestID()).
			Save(s.Ctx)
		s.Require().NoError(err)

		reqID, err := s.repo.GetProblemInitialRequestID(s.Ctx, p.ID)
		s.Require().NoError(err)
		s.Equal(expReqID, reqID)
	})
}
