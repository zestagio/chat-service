package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	chatsrepo "github.com/zestagio/chat-service/internal/repositories/chats"
	messagesrepo "github.com/zestagio/chat-service/internal/repositories/messages"
	problemsrepo "github.com/zestagio/chat-service/internal/repositories/problems"
	serverclient "github.com/zestagio/chat-service/internal/server-client"
	clienterrhandler "github.com/zestagio/chat-service/internal/server-client/errhandler"
	clientv1 "github.com/zestagio/chat-service/internal/server-client/v1"
	"github.com/zestagio/chat-service/internal/store"
	gethistory "github.com/zestagio/chat-service/internal/usecases/client/get-history"
	sendmessage "github.com/zestagio/chat-service/internal/usecases/client/send-message"
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

	db *store.Database,
	chatsRepo *chatsrepo.Repo,
	msgRepo *messagesrepo.Repo,
	problemsRepo *problemsrepo.Repo,
) (*serverclient.Server, error) {
	getHistoryUseCase, err := gethistory.New(gethistory.NewOptions(msgRepo))
	if err != nil {
		return nil, fmt.Errorf("create gethistory usecase: %v", err)
	}

	sendMessageUseCase, err := sendmessage.New(sendmessage.NewOptions(
		chatsRepo,
		msgRepo,
		problemsRepo,
		db,
	))
	if err != nil {
		return nil, fmt.Errorf("create sendmessage usecase: %v", err)
	}

	v1Handlers, err := clientv1.NewHandlers(clientv1.NewOptions(getHistoryUseCase, sendMessageUseCase))
	if err != nil {
		return nil, fmt.Errorf("create v1 handlers: %v", err)
	}

	lg := zap.L().Named(nameServerClient)

	httpErrorHandler, err := clienterrhandler.New(clienterrhandler.NewOptions(
		lg,
		productionMode,
		clienterrhandler.ResponseBuilder,
	))
	if err != nil {
		return nil, fmt.Errorf("create http error handler: %v", err)
	}

	srv, err := serverclient.New(serverclient.NewOptions(
		lg,
		addr,
		allowOrigins,
		keycloak,
		requiredResource,
		requiredRole,
		v1Swagger,
		v1Handlers,
		httpErrorHandler.Handle,
	))
	if err != nil {
		return nil, fmt.Errorf("build server: %v", err)
	}

	return srv, nil
}
