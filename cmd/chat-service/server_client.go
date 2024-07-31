package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	"github.com/zestagio/chat-service/internal/server"
	serverclient "github.com/zestagio/chat-service/internal/server-client"
	clienterrhandler "github.com/zestagio/chat-service/internal/server-client/errhandler"
	clientevents "github.com/zestagio/chat-service/internal/server-client/events"
	clientv1 "github.com/zestagio/chat-service/internal/server-client/v1"
	"github.com/zestagio/chat-service/internal/server/errhandler"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	"github.com/zestagio/chat-service/internal/services/outbox"
	"github.com/zestagio/chat-service/internal/store"
	gethistory "github.com/zestagio/chat-service/internal/usecases/client/get-history"
	sendmessage "github.com/zestagio/chat-service/internal/usecases/client/send-message"
	websocketstream "github.com/zestagio/chat-service/internal/websocket-stream"
)

const nameServerClient = "server-client"

//nolint:revive // ignore argument-limit rule to keep server client init in single place
func initServerClient(
	productionMode bool,
	addr string,
	allowOrigins []string,
	v1Swagger *openapi3.T,

	keycloak *keycloakclient.Client,
	requiredResource string,
	requiredRole string,
	secWsProtocol string,

	eventStream eventstream.EventStream,
	outBox *outbox.Service,

	db *store.Database,
	chatsRepo *chatsrepo.Repo,
	msgRepo *messagesrepo.Repo,
	problemsRepo *problemsrepo.Repo,
) (*server.Server, error) {
	getHistoryUseCase, err := gethistory.New(gethistory.NewOptions(msgRepo))
	if err != nil {
		return nil, fmt.Errorf("create gethistory usecase: %v", err)
	}

	sendMessageUseCase, err := sendmessage.New(sendmessage.NewOptions(
		chatsRepo,
		msgRepo,
		outBox,
		problemsRepo,
		db,
	))
	if err != nil {
		return nil, fmt.Errorf("create sendmessage usecase: %v", err)
	}

	v1Handlers, err := clientv1.NewHandlers(clientv1.NewOptions(
		getHistoryUseCase,
		sendMessageUseCase,
	))
	if err != nil {
		return nil, fmt.Errorf("create v1 handlers: %v", err)
	}

	shutdownCh := make(chan struct{})
	shutdownFn := func() {
		close(shutdownCh)
	}

	lg := zap.L().Named(nameServerClient)

	wsHandler, err := websocketstream.NewHTTPHandler(websocketstream.NewOptions(
		lg,
		eventStream,
		clientevents.Adapter{},
		websocketstream.JSONEventWriter{},
		websocketstream.NewUpgrader(allowOrigins, secWsProtocol),
		shutdownCh,
	))
	if err != nil {
		return nil, fmt.Errorf("create ws handler: %v", err)
	}

	httpErrorHandler, err := errhandler.New(errhandler.NewOptions(lg, productionMode, clienterrhandler.ResponseBuilder))
	if err != nil {
		return nil, fmt.Errorf("create http error handler: %v", err)
	}

	srv, err := server.New(server.NewOptions(
		lg,
		addr,
		allowOrigins,
		keycloak,
		requiredResource,
		requiredRole,
		secWsProtocol,
		serverclient.NewHandlersRegistrar(v1Swagger, v1Handlers, wsHandler.Serve, httpErrorHandler.Handle),
		shutdownFn,
	))
	if err != nil {
		return nil, fmt.Errorf("build server: %v", err)
	}

	return srv, nil
}
