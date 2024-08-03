package managerv1_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/zestagio/chat-service/internal/middlewares"
	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
	managerv1mocks "github.com/zestagio/chat-service/internal/server-manager/v1/mocks"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
)

type HandlersSuite struct {
	testingh.ContextSuite

	ctrl                      *gomock.Controller
	canReceiveProblemsUseCase *managerv1mocks.MockcanReceiveProblemsUseCase
	freeHandsSignalUseCase    *managerv1mocks.MockfreeHandsSignalUseCase
	getChatsUseCase           *managerv1mocks.MockgetChatsUseCase
	getChatHistoryUseCase     *managerv1mocks.MockgetChatHistoryUseCase
	resolveProblemUseCase     *managerv1mocks.MockresolveProblemUseCase
	sendMessageUseCase        *managerv1mocks.MocksendMessageUseCase
	handlers                  managerv1.Handlers

	managerID types.UserID
}

func TestHandlersSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(HandlersSuite))
}

func (s *HandlersSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.canReceiveProblemsUseCase = managerv1mocks.NewMockcanReceiveProblemsUseCase(s.ctrl)
	s.freeHandsSignalUseCase = managerv1mocks.NewMockfreeHandsSignalUseCase(s.ctrl)
	s.getChatsUseCase = managerv1mocks.NewMockgetChatsUseCase(s.ctrl)
	s.getChatHistoryUseCase = managerv1mocks.NewMockgetChatHistoryUseCase(s.ctrl)
	s.resolveProblemUseCase = managerv1mocks.NewMockresolveProblemUseCase(s.ctrl)
	s.sendMessageUseCase = managerv1mocks.NewMocksendMessageUseCase(s.ctrl)
	{
		var err error
		s.handlers, err = managerv1.NewHandlers(managerv1.NewOptions(
			s.canReceiveProblemsUseCase,
			s.freeHandsSignalUseCase,
			s.getChatsUseCase,
			s.getChatHistoryUseCase,
			s.resolveProblemUseCase,
			s.sendMessageUseCase,
		))
		s.Require().NoError(err)
	}
	s.managerID = types.NewUserID()

	s.ContextSuite.SetupTest()
}

func (s *HandlersSuite) TearDownTest() {
	s.ctrl.Finish()

	s.ContextSuite.TearDownTest()
}

func (s *HandlersSuite) newEchoCtx(
	requestID types.RequestID,
	path string,
	body string,
) (*httptest.ResponseRecorder, echo.Context) {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderXRequestID, requestID.String())

	resp := httptest.NewRecorder()

	ctx := echo.New().NewContext(req, resp)
	middlewares.SetToken(ctx, s.managerID)

	return resp, ctx
}
