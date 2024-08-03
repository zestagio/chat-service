//go:build integration

package chatsrepo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
)

type ChatsRepoSuite struct {
	testingh.DBSuite
	repo *chatsrepo.Repo
}

func TestChatsRepoSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ChatsRepoSuite{DBSuite: testingh.NewDBSuite("TestChatsRepoSuite")})
}

func (s *ChatsRepoSuite) SetupSuite() {
	s.DBSuite.SetupSuite()

	var err error

	s.repo, err = chatsrepo.New(chatsrepo.NewOptions(s.Database))
	s.Require().NoError(err)
}

func (s *ChatsRepoSuite) Test_CreateIfNotExists() {
	s.Run("chat does not exist, should be created", func() {
		clientID := types.NewUserID()

		chatID, err := s.repo.CreateIfNotExists(s.Ctx, clientID)
		s.Require().NoError(err)
		s.NotEmpty(chatID)
	})

	s.Run("chat already exists", func() {
		clientID := types.NewUserID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		chatID, err := s.repo.CreateIfNotExists(s.Ctx, clientID)
		s.Require().NoError(err)
		s.Require().NotEmpty(chatID)
		s.Equal(chat.ID, chatID)
	})
}

func (s *ChatsRepoSuite) Test_GetChatClient() {
	expectedClientID := types.NewUserID()

	// Create chat.
	chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(expectedClientID).Save(s.Ctx)
	s.Require().NoError(err)

	// Finally get manager ID by chat ID.
	clientID, err := s.repo.GetChatClient(s.Ctx, chat.ID)
	s.Require().NoError(err)
	s.Equal(expectedClientID, clientID)
}

func (s *ChatsRepoSuite) TestRepo_GetChatManager() {
	s.Run("chat has manager", func() {
		clientID := types.NewUserID()
		expectedManagerID := types.NewUserID()

		chatID := s.createChatAndAssignedProblem(clientID, expectedManagerID)

		// Finally get manager ID by chat ID.
		managerID, err := s.repo.GetChatManager(s.Ctx, chatID)
		s.Require().NoError(err)
		s.Equal(expectedManagerID, managerID)
	})

	s.Run("chat has no problem and no manager at all", func() {
		clientID := types.NewUserID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		// Finally get manager ID by chat ID.
		managerID, err := s.repo.GetChatManager(s.Ctx, chat.ID)
		s.Require().ErrorIs(err, chatsrepo.ErrChatWithoutManager)
		s.Require().Empty(managerID)
	})

	s.Run("chat has problem with no manager", func() {
		clientID := types.NewUserID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		// Assign unresolved problem without manager to chat.
		_, err = s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).Save(s.Ctx)
		s.Require().NoError(err)

		// Finally get manager ID by chat ID.
		managerID, err := s.repo.GetChatManager(s.Ctx, chat.ID)
		s.Require().ErrorIs(err, chatsrepo.ErrChatWithoutManager)
		s.Require().Empty(managerID)
	})

	s.Run("chat has resolved problem", func() {
		clientID := types.NewUserID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		// Assign resolved problem with manager to chat.
		_, err = s.Database.Problem(s.Ctx).Create().
			SetChatID(chat.ID).
			SetResolvedAt(time.Now()).
			SetManagerID(types.NewUserID()).
			Save(s.Ctx)
		s.Require().NoError(err)

		// Finally get manager ID by chat ID.
		managerID, err := s.repo.GetChatManager(s.Ctx, chat.ID)
		s.Require().ErrorIs(err, chatsrepo.ErrChatWithoutManager)
		s.Require().Empty(managerID)
	})
}

func (s *ChatsRepoSuite) TestRepo_GetChatsWithOpenProblems() {
	s.Run("chats with open problems exists", func() {
		managerID := types.NewUserID()

		// 1.
		clientID1 := types.NewUserID()
		chatID1 := s.createChatAndAssignedProblem(clientID1, managerID)
		time.Sleep(10 * time.Millisecond)

		// 2.
		clientID2 := types.NewUserID()
		chatID2 := s.createChatAndAssignedProblem(clientID2, managerID)

		// 3.
		clientID3 := types.NewUserID()
		_ = s.createChatAndAssignedProblem(clientID3, types.NewUserID()) // Other manager.

		// Finally get chats with open problems.
		chats, err := s.repo.GetChatsWithOpenProblems(s.Ctx, managerID)
		s.Require().NoError(err)
		s.Require().Len(chats, 2)

		s.Equal(chatID1, chats[0].ID)
		s.Equal(clientID1, chats[0].ClientID)

		s.Equal(chatID2, chats[1].ID)
		s.Equal(clientID2, chats[1].ClientID)
	})

	s.Run("chat has only closed problem assigned to manager", func() {
		clientID := types.NewUserID()
		managerID := types.NewUserID()

		// Create chat.
		chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
		s.Require().NoError(err)

		// Assign resolved problem to chat.
		_, err = s.Database.Problem(s.Ctx).Create().
			SetChatID(chat.ID).
			SetManagerID(managerID).
			SetResolvedAt(time.Now()).
			Save(s.Ctx)
		s.Require().NoError(err)

		// Finally get chats with open problems.
		chats, err := s.repo.GetChatsWithOpenProblems(s.Ctx, managerID)
		s.Require().NoError(err)
		s.Empty(chats)
	})

	s.Run("no chats with problem assigned to manager", func() {
		managerID := types.NewUserID()

		// Finally get chats with open problems.
		chats, err := s.repo.GetChatsWithOpenProblems(s.Ctx, managerID)
		s.Require().NoError(err)
		s.Empty(chats)
	})
}

func (s *ChatsRepoSuite) createChatAndAssignedProblem(clientID, managerID types.UserID) types.ChatID {
	s.T().Helper()

	// Create chat.
	chat, err := s.Database.Chat(s.Ctx).Create().SetClientID(clientID).Save(s.Ctx)
	s.Require().NoError(err)

	// Assign open problem to chat.
	_, err = s.Database.Problem(s.Ctx).Create().SetChatID(chat.ID).SetManagerID(managerID).Save(s.Ctx)
	s.Require().NoError(err)

	return chat.ID
}
