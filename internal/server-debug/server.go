package serverdebug

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/zestagio/chat-service/internal/buildinfo"
	"github.com/zestagio/chat-service/internal/logger"
	clientv1 "github.com/zestagio/chat-service/internal/server-client/v1"
)

const (
	readHeaderTimeout = time.Second
	shutdownTimeout   = 3 * time.Second
)

//go:generate options-gen -out-filename=server_options.gen.go -from-struct=Options
type Options struct {
	addr string `option:"mandatory" validate:"required,hostname_port"`
}

type Server struct {
	lg  *zap.Logger
	srv *http.Server
}

func New(opts Options) (*Server, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	lg := zap.L().Named("server-debug")

	e := echo.New()
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency:   true,
		LogRemoteIP:  true,
		LogHost:      true,
		LogMethod:    true,
		LogURIPath:   true,
		LogRequestID: true,
		LogUserAgent: true,
		LogStatus:    true,
		HandleError:  true,
		Skipper: func(eCtx echo.Context) bool {
			return eCtx.Request().Method == http.MethodOptions
		},
		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
			zap.L().Info("request",
				zap.Duration("latency", v.Latency),
				zap.String("remote_ip", v.RemoteIP),
				zap.String("host", v.Host),
				zap.String("method", v.Method),
				zap.String("path", v.URIPath),
				zap.String("request_id", v.RequestID),
				zap.String("user_agent", v.UserAgent),
				zap.Int("status", v.Status),
			)
			return nil
		},
	}))
	e.Use(middleware.Recover())

	s := &Server{
		lg: lg,
		srv: &http.Server{
			Addr:              opts.addr,
			Handler:           e,
			ReadHeaderTimeout: readHeaderTimeout,
		},
	}
	index := newIndexPage()

	e.GET("/version", s.Version)
	index.addPage("/version", "Get build information")

	e.PUT("/log/level", echo.WrapHandler(logger.Level))

	{
		pprofMux := http.NewServeMux()
		pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
		pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		e.GET("/debug/pprof/*", echo.WrapHandler(pprofMux))
		index.addPage("/debug/pprof/", "Go std profiler")
		index.addPage("/debug/pprof/profile?seconds=30", "Take half-min profile")
	}

	e.GET("/debug/error", s.DebugError)
	index.addPage("/debug/error", "Debug Sentry error event")

	e.GET("/schema/client", s.SchemaClient)
	index.addPage("/schema/client", "Get client OpenAPI specification")

	e.GET("/", index.handler)
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(
		func() error {
			<-ctx.Done()

			ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()

			return s.srv.Shutdown(ctx) //nolint:contextcheck // graceful shutdown with new context
		},
	)

	eg.Go(
		func() error {
			s.lg.Info("listen and serve", zap.String("addr", s.srv.Addr))

			if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("listen and serve: %v", err)
			}
			return nil
		},
	)

	return eg.Wait()
}

func (s *Server) Version(eCtx echo.Context) error {
	return eCtx.JSON(http.StatusOK, buildinfo.BuildInfo)
}

func (s *Server) DebugError(eCtx echo.Context) error {
	s.lg.Error("look for me in the sentry")

	return eCtx.String(http.StatusOK, "event sent")
}

func (s *Server) SchemaClient(eCtx echo.Context) error {
	swagger, _ := clientv1.GetSwagger()

	return eCtx.JSON(http.StatusOK, swagger)
}
