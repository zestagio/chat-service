package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	"github.com/zestagio/chat-service/internal/server"
	servermanager "github.com/zestagio/chat-service/internal/server-manager"
	managererrhandler "github.com/zestagio/chat-service/internal/server-manager/errhandler"
	managerevents "github.com/zestagio/chat-service/internal/server-manager/events"
	managerv1 "github.com/zestagio/chat-service/internal/server-manager/v1"
	"github.com/zestagio/chat-service/internal/server/errhandler"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	managerload "github.com/zestagio/chat-service/internal/services/manager-load"
	managerpool "github.com/zestagio/chat-service/internal/services/manager-pool"
	canreceiveproblems "github.com/zestagio/chat-service/internal/usecases/manager/can-receive-problems"
	freehandssignal "github.com/zestagio/chat-service/internal/usecases/manager/free-hands-signal"
	websocketstream "github.com/zestagio/chat-service/internal/websocket-stream"
)

const nameServerManager = "server-manager"

//nolint:revive // ignore argument-limit rule to keep server manager init in single place
func initServerManager(
	productionMode bool,
	addr string,
	allowOrigins []string,
	v1Swagger *openapi3.T,

	keycloak *keycloakclient.Client,
	requiredResource string,
	requiredRole string,
	secWsProtocol string,

	eventStream eventstream.EventStream,
	mLoadSvc *managerload.Service,
	mPool managerpool.Pool,
) (*server.Server, error) {
	canReceiveProblemsUseCase, err := canreceiveproblems.New(canreceiveproblems.NewOptions(mLoadSvc, mPool))
	if err != nil {
		return nil, fmt.Errorf("create canreceiveproblems usecase: %v", err)
	}

	freeHandsSignalUseCase, err := freehandssignal.New(freehandssignal.NewOptions(mLoadSvc, mPool))
	if err != nil {
		return nil, fmt.Errorf("create freehandssignal usecase: %v", err)
	}

	v1Handlers, err := managerv1.NewHandlers(managerv1.NewOptions(
		canReceiveProblemsUseCase,
		freeHandsSignalUseCase,
	))
	if err != nil {
		return nil, fmt.Errorf("create v1 handlers: %v", err)
	}

	shutdownCh := make(chan struct{})
	shutdownFn := func() {
		close(shutdownCh)
	}

	lg := zap.L().Named(nameServerManager)

	wsHandler, err := websocketstream.NewHTTPHandler(websocketstream.NewOptions(
		lg,
		eventStream,
		managerevents.Adapter{},
		websocketstream.JSONEventWriter{},
		websocketstream.NewUpgrader(allowOrigins, secWsProtocol),
		shutdownCh,
	))
	if err != nil {
		return nil, fmt.Errorf("create ws handler: %v", err)
	}

	httpErrorHandler, err := errhandler.New(errhandler.NewOptions(lg, productionMode, managererrhandler.ResponseBuilder))
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
		servermanager.NewHandlersRegistrar(v1Swagger, v1Handlers, wsHandler.Serve, httpErrorHandler.Handle),
		shutdownFn,
	))
	if err != nil {
		return nil, fmt.Errorf("build server: %v", err)
	}

	return srv, nil
}
