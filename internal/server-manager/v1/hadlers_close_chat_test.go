package managerv1_test

import (
	"fmt"
	"net/http"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
	"github.com/zestagio/chat-service/internal/types"
	resolveproblem "github.com/zestagio/chat-service/internal/usecases/manager/resolve-problem"
)

func (s *HandlersSuite) TestCloseChat_BindRequestError() {
	// Arrange.
	reqID := types.NewRequestID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/closeChat", `{"chatId":"`)

	// Action.
	err := s.handlers.PostCloseChat(eCtx, managerv1.PostCloseChatParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Equal(http.StatusBadRequest, internalerrors.GetServerErrorCode(err))
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestCloseChat_UseCase_InvalidRequestError() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()

	resp, eCtx := s.newEchoCtx(reqID, "/v1/closeChat", fmt.Sprintf(`{"chatId": %q}`, chatID))
	s.resolveProblemUseCase.EXPECT().Handle(eCtx.Request().Context(), resolveproblem.Request{
		ID:        reqID,
		ManagerID: s.managerID,
		ChatID:    chatID,
	}).Return(resolveproblem.ErrInvalidRequest)

	// Action.
	err := s.handlers.PostCloseChat(eCtx, managerv1.PostCloseChatParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Equal(http.StatusBadRequest, internalerrors.GetServerErrorCode(err))
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestCloseChat_UseCase_Success() {
	// Arrange.
	reqID := types.NewRequestID()
	chatID := types.NewChatID()

	resp, eCtx := s.newEchoCtx(reqID, "/v1/closeChat", fmt.Sprintf(`{"chatId": %q}`, chatID))
	s.resolveProblemUseCase.EXPECT().Handle(eCtx.Request().Context(), resolveproblem.Request{
		ID:        reqID,
		ManagerID: s.managerID,
		ChatID:    chatID,
	}).Return(nil)

	// Action.
	err := s.handlers.PostCloseChat(eCtx, managerv1.PostCloseChatParams{XRequestID: reqID})

	// Assert.
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.Code)
	s.JSONEq(`
{
   "data": null
}`, resp.Body.String())
}
