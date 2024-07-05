package managerv1_test

import (
	"errors"
	"net/http"

	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
	"github.com/zestagio/chat-service/internal/types"
	freehands "github.com/zestagio/chat-service/internal/usecases/manager/free-hands"
)

func (s *HandlersSuite) TestFreeHands_UseCase_Business_Error() {
	// Arrange.
	reqID := types.NewRequestID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/freeHands", "")

	s.freeHandsUseCase.EXPECT().Handle(eCtx.Request().Context(), freehands.Request{
		ID:        reqID,
		ManagerID: s.managerID,
	}).Return(freehands.ErrManagerOverloaded)

	// Action.
	err := s.handlers.PostFreeHands(eCtx, managerv1.PostFreeHandsParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestFreeHands_UseCase_Error() {
	// Arrange.
	reqID := types.NewRequestID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/freeHands", "")

	s.freeHandsUseCase.EXPECT().Handle(eCtx.Request().Context(), freehands.Request{
		ID:        reqID,
		ManagerID: s.managerID,
	}).Return(errors.New("unexpected"))

	// Action.
	err := s.handlers.PostFreeHands(eCtx, managerv1.PostFreeHandsParams{XRequestID: reqID})

	// Assert.
	s.Require().Error(err)
	s.Empty(resp.Body)
}

func (s *HandlersSuite) TestFreeHands_UseCase_Success() {
	// Arrange.
	reqID := types.NewRequestID()
	resp, eCtx := s.newEchoCtx(reqID, "/v1/freeHands", "")

	s.freeHandsUseCase.EXPECT().Handle(eCtx.Request().Context(), freehands.Request{
		ID:        reqID,
		ManagerID: s.managerID,
	}).Return(nil)

	// Action.
	err := s.handlers.PostFreeHands(eCtx, managerv1.PostFreeHandsParams{XRequestID: reqID})

	// Assert.
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.Code)
	s.JSONEq(`
{
   "data": null
}`, resp.Body.String())
}
