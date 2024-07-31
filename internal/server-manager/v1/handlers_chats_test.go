package managerv1_test

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
	"github.com/zestagio/chat-service/internal/types"
	getchathistory "github.com/zestagio/chat-service/internal/usecases/manager/get-chat-history"
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

func (s *HandlersSuite) TestGetChatHistory_BindRequestError() {
	// Arrange.
	reqID := types.NewRequestID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/getChatHistory", `{"page_size":`)

	// Action.
	err := s.handlers.PostGetChatHistory(eCtx, managerv1.PostGetChatHistoryParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Equal(http.StatusBadRequest, internalerrors.GetServerErrorCode(err))
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestGetChatHistory_UseCase_InvalidRequest() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/getChatHistory", fmt.Sprintf(`{"pageSize":9,"chatId":"%s"}`, chatID.String()))
	s.getChatHistoryUseCase.EXPECT().Handle(eCtx.Request().Context(), getchathistory.Request{
		ID:        reqID,
		ManagerID: s.managerID,
		ChatID:    chatID,
		PageSize:  9,
		Cursor:    "",
	}).Return(getchathistory.Response{}, getchathistory.ErrInvalidRequest)

	// Action.
	err := s.handlers.PostGetChatHistory(eCtx, managerv1.PostGetChatHistoryParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Equal(http.StatusBadRequest, internalerrors.GetServerErrorCode(err))
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestGetChatHistory_UseCase_InvalidCursor() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/getChatHistory", fmt.Sprintf(`{"cursor":"abracadabra","chatId":"%s"}`, chatID.String()))
	s.getChatHistoryUseCase.EXPECT().Handle(eCtx.Request().Context(), getchathistory.Request{
		ID:        reqID,
		ManagerID: s.managerID,
		ChatID:    chatID,
		PageSize:  0,
		Cursor:    "abracadabra",
	}).Return(getchathistory.Response{}, getchathistory.ErrInvalidCursor)

	// Action.
	err := s.handlers.PostGetChatHistory(eCtx, managerv1.PostGetChatHistoryParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Equal(http.StatusBadRequest, internalerrors.GetServerErrorCode(err))
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestGetChatHistory_UseCase_UnknownError() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/getChatHistory", fmt.Sprintf(`{"pageSize":10,"chatId":"%s"}`, chatID.String()))
	s.getChatHistoryUseCase.EXPECT().Handle(eCtx.Request().Context(), getchathistory.Request{
		ID:        reqID,
		ManagerID: s.managerID,
		ChatID:    chatID,
		PageSize:  10,
	}).Return(getchathistory.Response{}, errors.New("something went wrong"))

	// Action.
	err := s.handlers.PostGetChatHistory(eCtx, managerv1.PostGetChatHistoryParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestGetChatHistory_UseCase_Success() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/getChatHistory", fmt.Sprintf(`{"pageSize":10,"chatId":"%s"}`, chatID.String()))

	msgs := []getchathistory.Message{
		{
			ID:        types.NewMessageID(),
			AuthorID:  types.NewUserID(),
			Body:      "hello!",
			CreatedAt: time.Unix(1, 1).UTC(),
		},
		{
			ID:        types.NewMessageID(),
			AuthorID:  types.NewUserID(),
			Body:      "some message",
			CreatedAt: time.Unix(2, 2).UTC(),
		},
	}

	s.getChatHistoryUseCase.EXPECT().Handle(eCtx.Request().Context(), getchathistory.Request{
		ID:        reqID,
		ManagerID: s.managerID,
		ChatID:    chatID,
		PageSize:  10,
	}).Return(getchathistory.Response{
		Messages:   msgs,
		NextCursor: "",
	}, nil)

	// Action.
	err := s.handlers.PostGetChatHistory(eCtx, managerv1.PostGetChatHistoryParams{XRequestID: reqID})

	// Assert.
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.Code)
	s.JSONEq(fmt.Sprintf(`
		{
			"data":
			{
				"messages":
				[
					{
						"authorId": %q,
						"body": "hello!",
						"createdAt": "1970-01-01T00:00:01.000000001Z",
						"id": %q
					},
					{
						"authorId": %q,
						"body": "some message",
						"createdAt": "1970-01-01T00:00:02.000000002Z",
						"id": %q
					}
				],
				"next": ""
			}
		}`, msgs[0].AuthorID, msgs[0].ID, msgs[1].AuthorID, msgs[1].ID), resp.Body.String())
}
