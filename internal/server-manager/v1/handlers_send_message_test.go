package managerv1_test

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
	"github.com/zestagio/chat-service/internal/types"
	sendmessage "github.com/zestagio/chat-service/internal/usecases/manager/send-message"
)

func (s *HandlersSuite) TestSendMessage_BindRequestError() {
	// Arrange.
	reqID := types.NewRequestID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/sendMessage", `{"messageBody": "Hel`)

	// Action.
	err := s.handlers.PostSendMessage(eCtx, managerv1.PostSendMessageParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Equal(http.StatusBadRequest, internalerrors.GetServerErrorCode(err))
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestSendMessage_Usecase_InvalidRequest() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()
	msgBody := strings.Repeat("!", 3001)

	resp, eCtx := s.newEchoCtx(reqID, "/v1/sendMessage", fmt.Sprintf(`{"messageBody": "%s", "chatId": %q}`, msgBody, chatID))
	s.sendMessageUseCase.EXPECT().Handle(eCtx.Request().Context(), sendmessage.Request{
		ID:          reqID,
		ManagerID:   s.managerID,
		ChatID:      chatID,
		MessageBody: msgBody,
	}).Return(sendmessage.Response{}, sendmessage.ErrInvalidRequest)

	// Action.
	err := s.handlers.PostSendMessage(eCtx, managerv1.PostSendMessageParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Equal(http.StatusBadRequest, internalerrors.GetServerErrorCode(err))
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestSendMessage_Usecase_Success() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()

	resp, eCtx := s.newEchoCtx(reqID, "/v1/sendMessage", fmt.Sprintf(`{"messageBody": "Hello!", "chatId": %q}`, chatID))
	msgID := types.NewMessageID()
	s.sendMessageUseCase.EXPECT().Handle(eCtx.Request().Context(), sendmessage.Request{
		ID:          reqID,
		ManagerID:   s.managerID,
		ChatID:      chatID,
		MessageBody: "Hello!",
	}).Return(sendmessage.Response{
		MessageID: msgID,
		CreatedAt: time.Unix(1, 1).UTC(),
	}, nil)

	// Action.
	err := s.handlers.PostSendMessage(eCtx, managerv1.PostSendMessageParams{XRequestID: reqID})

	// Assert.
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.Code)
	s.JSONEq(fmt.Sprintf(`
{
    "data":
    {
        "authorId": "%s",
        "createdAt": "1970-01-01T00:00:01.000000001Z",
        "id": "%s"
    }
}`, s.managerID, msgID), resp.Body.String())
}
