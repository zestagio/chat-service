package managerv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	"github.com/zestagio/chat-service/internal/middlewares"
	getchathistory "github.com/zestagio/chat-service/internal/usecases/manager/get-chat-history"
	getchats "github.com/zestagio/chat-service/internal/usecases/manager/get-chats"
	"github.com/zestagio/chat-service/pkg/pointer"
)

func (h Handlers) PostGetChats(eCtx echo.Context, params PostGetChatsParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	resp, err := h.getChatsUseCase.Handle(ctx, getchats.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
	})
	if err != nil {
		if errors.Is(err, getchats.ErrInvalidRequest) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid request", err)
		}

		return fmt.Errorf("handle `get chats` use case: %w", err)
	}

	result := make([]Chat, 0)
	for _, c := range resp.Chats {
		result = append(result, Chat{
			ChatId:   c.ID,
			ClientId: c.ClientID,
		})
	}

	return eCtx.JSON(http.StatusOK, GetChatsResponse{Data: &ChatList{
		result,
	}})
}

func (h Handlers) PostGetChatHistory(eCtx echo.Context, params PostGetChatHistoryParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	var req GetChatHistoryRequest
	if err := eCtx.Bind(&req); err != nil {
		return fmt.Errorf("bind request: %w", err)
	}

	resp, err := h.getChatHistoryUseCase.Handle(ctx, getchathistory.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
		ChatID:    req.ChatId,
		PageSize:  pointer.Indirect(req.PageSize),
		Cursor:    pointer.Indirect(req.Cursor),
	})
	if err != nil {
		if errors.Is(err, getchathistory.ErrInvalidRequest) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid request", err)
		}

		if errors.Is(err, getchathistory.ErrInvalidCursor) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid cursor", err)
		}

		return fmt.Errorf("handle `get chat history`: %v", err)
	}

	page := make([]Message, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		mm := Message{
			AuthorId:  m.AuthorID,
			Body:      m.Body,
			CreatedAt: m.CreatedAt,
			Id:        m.ID,
		}
		page = append(page, mm)
	}

	return eCtx.JSON(http.StatusOK, &GetChatHistoryResponse{Data: &MessagesPage{
		Messages: page,
		Next:     resp.NextCursor,
	}})
}
