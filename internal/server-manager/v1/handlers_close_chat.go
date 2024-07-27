package managerv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	"github.com/zestagio/chat-service/internal/middlewares"
	closechat "github.com/zestagio/chat-service/internal/usecases/manager/resolve-problem"
)

func (h Handlers) PostCloseChat(eCtx echo.Context, params PostCloseChatParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	var req ChatId
	if err := eCtx.Bind(&req); err != nil {
		return fmt.Errorf("bind request: %w", err)
	}

	if err := h.resolveProblemUseCase.Handle(ctx, closechat.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
		ChatID:    req.ChatId,
	}); err != nil {
		if errors.Is(err, closechat.ErrInvalidRequest) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid request", err)
		}

		return fmt.Errorf("handle `resolve problem` use case: %v", err)
	}

	return eCtx.JSON(http.StatusOK, &CloseChatResponse{})
}
