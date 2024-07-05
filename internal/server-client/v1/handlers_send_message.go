package clientv1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	"github.com/zestagio/chat-service/internal/middlewares"
	sendmessage "github.com/zestagio/chat-service/internal/usecases/client/send-message"
	"github.com/zestagio/chat-service/pkg/pointer"
)

func (h Handlers) PostSendMessage(eCtx echo.Context, params PostSendMessageParams) error {
	ctx := eCtx.Request().Context()
	clientID := middlewares.MustUserID(eCtx)

	var req SendMessageRequest
	if err := eCtx.Bind(&req); err != nil {
		return fmt.Errorf("bind request: %w", err)
	}

	resp, err := h.sendMessage.Handle(ctx, sendmessage.Request{
		ID:          params.XRequestID,
		ClientID:    clientID,
		MessageBody: req.MessageBody,
	})
	if err != nil {
		if errors.Is(err, sendmessage.ErrInvalidRequest) {
			return internalerrors.NewServerError(http.StatusBadRequest, "invalid request", err)
		}

		if errors.Is(err, sendmessage.ErrChatNotCreated) {
			return internalerrors.NewServerError(ErrorCodeCreateChatError, "create chat error", err)
		}

		if errors.Is(err, sendmessage.ErrProblemNotCreated) {
			return internalerrors.NewServerError(ErrorCodeCreateProblemError, "create problem error", err)
		}

		return fmt.Errorf("handle `send message` use case: %v", err)
	}

	return eCtx.JSON(http.StatusOK, SendMessageResponse{Data: &MessageHeader{
		AuthorId:  pointer.PtrWithZeroAsNil(resp.AuthorID),
		CreatedAt: resp.CreatedAt,
		Id:        resp.MessageID,
	}})
}
