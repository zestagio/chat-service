package managerv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	"github.com/zestagio/chat-service/internal/middlewares"
	getchats "github.com/zestagio/chat-service/internal/usecases/manager/get-chats"
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
