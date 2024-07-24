package managerv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	"github.com/zestagio/chat-service/internal/middlewares"
	sendmessage "github.com/zestagio/chat-service/internal/usecases/manager/send-message"
)

func (h Handlers) PostSendMessage(eCtx echo.Context, params PostSendMessageParams) error {
	ctx := eCtx.Request().Context()
	managerID := middlewares.MustUserID(eCtx)

	var req SendMessageRequest
	if err := eCtx.Bind(&req); err != nil {
		return fmt.Errorf("bind request: %w", err)
	}

	resp, err := h.sendMessageUseCase.Handle(ctx, sendmessage.Request{
		ID:          params.XRequestID,
		ManagerID:   managerID,
		ChatID:      req.ChatId,
		MessageBody: req.MessageBody,
	})
	if err != nil {
		if errors.Is(err, sendmessage.ErrInvalidRequest) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid request", err)
		}

		return fmt.Errorf("handle `send message` use case: %v", err)
	}

	return eCtx.JSON(http.StatusOK, SendMessageResponse{Data: &MessageWithoutBody{
		AuthorId:  managerID,
		CreatedAt: resp.CreatedAt,
		Id:        resp.MessageID,
	}})
}
