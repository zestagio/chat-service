package serverclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	oapimdlwr "github.com/oapi-codegen/echo-middleware"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	"github.com/zestagio/chat-service/internal/middlewares"
	clientv1 "github.com/zestagio/chat-service/internal/server-client/v1"
)

const (
	readHeaderTimeout = time.Second
	shutdownTimeout   = 3 * time.Second
)

//go:generate options-gen -out-filename=server_options.gen.go -from-struct=Options
type Options struct {
	logger         *zap.Logger              `option:"mandatory" validate:"required"`
	keycloakClient *keycloakclient.Client   `option:"mandatory" validate:"required"`
	addr           string                   `option:"mandatory" validate:"required,hostname_port"`
	resource       string                   `option:"mandatory" validate:"required"`
	role           string                   `option:"mandatory" validate:"required"`
	allowOrigins   []string                 `option:"mandatory" validate:"min=1"`
	v1Swagger      *openapi3.T              `option:"mandatory" validate:"required"`
	v1Handlers     clientv1.ServerInterface `option:"mandatory" validate:"required"`
}

type Server struct {
	lg  *zap.Logger
	srv *http.Server
}

func New(opts Options) (*Server, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	e := echo.New()
	e.Use(middlewares.NewRequestLogger(opts.logger))
	e.Use(middlewares.NewRecover(opts.logger))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{AllowOrigins: opts.allowOrigins, AllowMethods: []string{echo.POST}}))
	e.Use(middlewares.NewKeycloakTokenAuth(opts.keycloakClient, opts.resource, opts.role))
	// 3000 unicode chars. Each 4 bytes max. 3000 * 4 = 12 kilobytes. +1 kilobyte reserved for JSON extra chars.
	e.Use(middleware.BodyLimit("13K"))

	s := &Server{
		lg: opts.logger,
		srv: &http.Server{
			Addr:              opts.addr,
			Handler:           e,
			ReadHeaderTimeout: readHeaderTimeout,
		},
	}

	v1 := e.Group("v1", oapimdlwr.OapiRequestValidatorWithOptions(opts.v1Swagger, &oapimdlwr.Options{
		Options: openapi3filter.Options{
			ExcludeRequestBody:  false,
			ExcludeResponseBody: true,
			AuthenticationFunc:  openapi3filter.NoopAuthenticationFunc,
		},
	}))
	clientv1.RegisterHandlers(v1, opts.v1Handlers)

	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return s.srv.Shutdown(ctx) //nolint:contextcheck // graceful shutdown with new context
	})

	eg.Go(func() error {
		s.lg.Info("listen and serve", zap.String("addr", s.srv.Addr))

		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve: %v", err)
		}
		return nil
	})

	return eg.Wait()
}
