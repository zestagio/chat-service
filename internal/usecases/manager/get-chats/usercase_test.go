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

	ctrl      *gomock.Controller
	chatsRepo *getchatsmocks.MockchatsRepository
	uCase     getchats.UseCase
}

func TestUseCaseSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UseCaseSuite))
}

func (s *UseCaseSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.chatsRepo = getchatsmocks.NewMockchatsRepository(s.ctrl)

	var err error
	s.uCase, err = getchats.New(getchats.NewOptions(s.chatsRepo))
	s.Require().NoError(err)

	s.ContextSuite.SetupTest()
}

func (s *UseCaseSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *UseCaseSuite) TestRequestValidationError() {
	// Arrange.
	req := getchats.Request{}

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().ErrorIs(err, getchats.ErrInvalidRequest)
	s.Empty(resp.Chats)
}

func (s *UseCaseSuite) TestGetChatsError() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	errExpected := errors.New("any error")

	s.chatsRepo.EXPECT().GetManagerChatsWithProblems(gomock.Any(), managerID).
		Return(nil, errExpected)

	req := getchats.Request{
		ID:        reqID,
		ManagerID: managerID,
	}

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().Error(err)
	s.Empty(resp.Chats)
}

func (s *UseCaseSuite) TestGetChatsSuccess() {
	// Arrange.
	reqID := types.NewRequestID()
	managerID := types.NewUserID()
	chat := chatsrepo.Chat{
		ID:       types.NewChatID(),
		ClientID: types.NewUserID(),
	}

	s.chatsRepo.EXPECT().GetManagerChatsWithProblems(gomock.Any(), managerID).
		Return([]chatsrepo.Chat{chat}, nil)

	req := getchats.Request{
		ID:        reqID,
		ManagerID: managerID,
	}

	// Action.
	resp, err := s.uCase.Handle(s.Ctx, req)

	// Assert.
	s.Require().NoError(err)
	s.Require().Len(resp.Chats, 1)
}
