package managerv1_test

import (
	"errors"
	"fmt"
	"net/http"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
	"github.com/zestagio/chat-service/internal/types"
	getchats "github.com/zestagio/chat-service/internal/usecases/manager/get-chats"
)

func (s *HandlersSuite) TestGetChats_Usecase_InvalidRequest() {
	// Arrange.
	reqID := types.NewRequestID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/getChats", "")

	s.getChatsUseCase.EXPECT().Handle(eCtx.Request().Context(), getchats.Request{
		ID:        reqID,
		ManagerID: s.managerID,
	}).Return(getchats.Response{}, getchats.ErrInvalidRequest)

	// Action.
	err := s.handlers.PostGetChats(eCtx, managerv1.PostGetChatsParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Equal(http.StatusBadRequest, internalerrors.GetServerErrorCode(err))
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestGetChats_Usecase_UnknownError() {
	// Arrange.
	reqID := types.NewRequestID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/getChats", "")
	s.getChatsUseCase.EXPECT().Handle(eCtx.Request().Context(), getchats.Request{
		ID:        reqID,
		ManagerID: s.managerID,
	}).Return(getchats.Response{}, errors.New("something went wrong"))

	// Action.
	err := s.handlers.PostGetChats(eCtx, managerv1.PostGetChatsParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestGetChats_Usecase_Success() {
	// Arrange.
	reqID := types.NewRequestID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/getChats", "")

	chats := []getchats.Chat{
		{
			ID:       types.NewChatID(),
			ClientID: types.NewUserID(),
		},
		{
			ID:       types.NewChatID(),
			ClientID: types.NewUserID(),
		},
	}
	s.getChatsUseCase.EXPECT().Handle(eCtx.Request().Context(), getchats.Request{
		ID:        reqID,
		ManagerID: s.managerID,
	}).Return(getchats.Response{
		Chats: chats,
	}, nil)

	// Action.
	err := s.handlers.PostGetChats(eCtx, managerv1.PostGetChatsParams{XRequestID: reqID})

	// Assert.
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.Code)
	s.JSONEq(fmt.Sprintf(`
		{
			"data":
			{
				"chats":
				[
					{
						"chatId": %q,
						"clientId": %q
					},
					{
						"chatId": %q,
						"clientId": %q
					}
				]
			}
		}`, chats[0].ID, chats[0].ClientID, chats[1].ID, chats[1].ClientID), resp.Body.String())
}
