package managerv1

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/zestagio/chat-service/internal/middlewares"
	getchathistory "github.com/zestagio/chat-service/internal/usecases/manager/get-chat-history"
	getchats "github.com/zestagio/chat-service/internal/usecases/manager/get-chats"
	"github.com/zestagio/chat-service/pkg/pointer"
)

func (h Handlers) PostGetChats(eCtx echo.Context, params PostGetChatsParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	resp, err := h.getChats.Handle(ctx, getchats.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
	})
	if err != nil {
		return fmt.Errorf("handle `get chats` use case: %v", err)
	}

	result := make([]Chat, 0, len(resp.Chats))
	for _, c := range resp.Chats {
		result = append(result, Chat{
			ChatId:   c.ID,
			ClientId: c.ClientID,
		})
	}
	return eCtx.JSON(http.StatusOK, GetChatsResponse{Data: &ChatList{
		Chats: result,
	}})
}

func (h Handlers) PostGetChatHistory(eCtx echo.Context, params PostGetChatHistoryParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	var req GetChatHistoryRequest
	if err := eCtx.Bind(&req); err != nil {
		return fmt.Errorf("bind request: %w", err)
	}

	resp, err := h.getChatHistory.Handle(ctx, getchathistory.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
		ChatID:    req.ChatId,
		Cursor:    pointer.Indirect(req.Cursor),
		PageSize:  pointer.Indirect(req.PageSize),
	})
	if err != nil {
		return fmt.Errorf("handle `get chat history` use case: %v", err)
	}

	page := make([]Message, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		page = append(page, Message{
			Id:        m.ID,
			AuthorId:  m.AuthorID,
			Body:      m.Body,
			CreatedAt: m.CreatedAt,
		})
	}
	return eCtx.JSON(http.StatusOK, GetChatHistoryResponse{Data: &MessagesPage{
		Messages: page,
		Next:     resp.NextCursor,
	}})
}
