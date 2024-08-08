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

func (s *HandlersSuite) TestGetChats_Usecase_Error() {
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

	s.getChatsUseCase.EXPECT().Handle(eCtx.Request().Context(), getchats.Request{
		ID:        reqID,
		ManagerID: s.managerID,
	}).Return(getchats.Response{
		Chats: []getchats.Chat{
			{
				ID:       types.MustParse[types.ChatID]("20b99498-a9d3-11ed-92b0-461e464ebed8"),
				ClientID: types.MustParse[types.UserID]("4faa9042-a9d3-11ed-8bfe-461e464ebed8"),
			},
			{
				ID:       types.MustParse[types.ChatID]("214db664-a9d3-11ed-9a8f-461e464ebed8"),
				ClientID: types.MustParse[types.UserID]("42fb6208-a9d3-11ed-b40b-461e464ebed8"),
			},
			{
				ID:       types.MustParse[types.ChatID]("2b50973a-a9d3-11ed-818b-461e464ebed8"),
				ClientID: types.MustParse[types.UserID]("463fde4e-a9d3-11ed-886f-461e464ebed8"),
			},
		},
	}, nil)

	// Action.
	err := s.handlers.PostGetChats(eCtx, managerv1.PostGetChatsParams{XRequestID: reqID})

	// Assert.
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.Code)
	s.JSONEq(`
{
    "data":
    {
        "chats":
        [
            {
                "chatId": "20b99498-a9d3-11ed-92b0-461e464ebed8",
                "clientId": "4faa9042-a9d3-11ed-8bfe-461e464ebed8"
            },
            {
                "chatId": "214db664-a9d3-11ed-9a8f-461e464ebed8",
                "clientId": "42fb6208-a9d3-11ed-b40b-461e464ebed8"
            },
            {
                "chatId": "2b50973a-a9d3-11ed-818b-461e464ebed8",
                "clientId": "463fde4e-a9d3-11ed-886f-461e464ebed8"
            }
        ]
    }
}`, resp.Body.String())
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

func (s *HandlersSuite) TestGetChatHistory_Usecase_UnknownError() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/getChatHistory", fmt.Sprintf(`{"pageSize":10,"chatId":%q}`, chatID))
	s.getChatHistoryUseCase.EXPECT().Handle(eCtx.Request().Context(), getchathistory.Request{
		ID:        reqID,
		ManagerID: s.managerID,
		ChatID:    chatID,
		PageSize:  10,
		Cursor:    "",
	}).Return(getchathistory.Response{}, errors.New("something went wrong"))

	// Action.
	err := s.handlers.PostGetChatHistory(eCtx, managerv1.PostGetChatHistoryParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestGetChatHistory_Usecase_Success() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/getChatHistory", fmt.Sprintf(`{"pageSize":10,"chatId":%q}`, chatID))
	s.getChatHistoryUseCase.EXPECT().Handle(eCtx.Request().Context(), getchathistory.Request{
		ID:        reqID,
		ManagerID: s.managerID,
		ChatID:    chatID,
		PageSize:  10,
		Cursor:    "",
	}).Return(getchathistory.Response{
		Messages: []getchathistory.Message{
			{
				ID:        types.MustParse[types.MessageID]("027c483c-ac2f-11ed-8ac8-461e464ebed8"),
				AuthorID:  types.MustParse[types.UserID]("086eafc8-ac2f-11ed-a746-461e464ebed8"),
				Body:      "Hello!",
				CreatedAt: time.Unix(1, 1).UTC(),
			},
			{
				ID:        types.MustParse[types.MessageID]("05061024-ac2f-11ed-b21c-461e464ebed8"),
				AuthorID:  s.managerID,
				Body:      "How can I help you?",
				CreatedAt: time.Unix(2, 2).UTC(),
			},
		},
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
                "authorId": "086eafc8-ac2f-11ed-a746-461e464ebed8",
                "body": "Hello!",
                "createdAt": "1970-01-01T00:00:01.000000001Z",
                "id": "027c483c-ac2f-11ed-8ac8-461e464ebed8"
            },
            {
                "authorId": %q,
                "body": "How can I help you?",
                "createdAt": "1970-01-01T00:00:02.000000002Z",
                "id": "05061024-ac2f-11ed-b21c-461e464ebed8"
            }
        ],
        "next": ""
    }
}`, s.managerID), resp.Body.String())
}
