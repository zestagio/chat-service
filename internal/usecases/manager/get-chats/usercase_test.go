package getchats_test

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
	getchats "github.com/zestagio/chat-service/internal/usecases/manager/get-chats"
	getchatsmocks "github.com/zestagio/chat-service/internal/usecases/manager/get-chats/mocks"
)

type UseCaseSuite struct {
	testingh.ContextSuite

	ctrl          *gomock.Controller
	chatsRepoMock *getchatsmocks.MockchatsRepository
	uCase         getchats.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.chatsRepoMock = getchatsmocks.NewMockchatsRepository(s.ctrl)

	var err error
	s.uCase, err = getchats.New(getchats.NewOptions(s.chatsRepoMock))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *UseCaseSuite) TestRequestValidationError() {
	// Arrange.
	req := getchats.Request{}

	// Action.
	_, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
}

func (s *UseCaseSuite) TestGetChatsWithOpenProblemsError() {
	// Arrange.
	managerID := types.NewUserID()

	s.chatsRepoMock.EXPECT().GetChatsWithOpenProblems(gomock.Any(), managerID).
		Return(nil, errors.New("unexpected"))

	req := getchats.Request{
		ID:        types.NewRequestID(),
		ManagerID: managerID,
	}

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Empty(resp.Chats)
}

func (s *UseCaseSuite) TestSuccessStory() {
	// Arrange.
	managerID := types.NewUserID()

	repoResp := []chatsrepo.Chat{
		{ID: types.NewChatID(), ClientID: types.NewUserID()},
		{ID: types.NewChatID(), ClientID: types.NewUserID()},
		{ID: types.NewChatID(), ClientID: types.NewUserID()},
	}
	s.chatsRepoMock.EXPECT().GetChatsWithOpenProblems(gomock.Any(), managerID).Return(repoResp, nil)

	req := getchats.Request{
		ID:        types.NewRequestID(),
		ManagerID: managerID,
	}

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Equal(getchats.Response{
		Chats: []getchats.Chat{
			{ID: repoResp[0].ID, ClientID: repoResp[0].ClientID},
			{ID: repoResp[1].ID, ClientID: repoResp[1].ClientID},
			{ID: repoResp[2].ID, ClientID: repoResp[2].ClientID},
		},
	}, resp)
}
