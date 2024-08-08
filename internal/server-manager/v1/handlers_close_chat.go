package managerv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	"github.com/zestagio/chat-service/internal/middlewares"
	resolveproblem "github.com/zestagio/chat-service/internal/usecases/manager/resolve-problem"
)

func (h Handlers) PostCloseChat(eCtx echo.Context, params PostCloseChatParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	var req CloseChatRequest
	if err := eCtx.Bind(&req); err != nil {
		return fmt.Errorf("bind request: %w", err)
	}

	if _, err := h.resolveProblem.Handle(ctx, resolveproblem.Request{
		ID:        params.XRequestID,
		ManagerID: managerID,
		ChatID:    req.ChatId,
	}); err != nil {
		if errors.Is(err, resolveproblem.ErrAssignedProblemNotFound) {
			return internalerrors.NewServerError(int(ErrorCodeAssignedProblemNotFound),
				"assigned to manager problem was not found", err)
		}

		return fmt.Errorf("handle resolve problem: %v", err)
	}

	var empty map[string]any
	return eCtx.JSON(http.StatusOK, CloseChatResponse{Data: &empty})
}
