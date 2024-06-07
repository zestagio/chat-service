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

	e.PUT("/log/level", s.LogLevel)

	e.GET("/debug/pprof/", s.PprofIndex)
	e.GET("/debug/pprof/allocs", s.PprofAllocs)
	e.GET("/debug/pprof/block", s.PprofBlock)
	e.GET("/debug/pprof/cmdline", s.PprofCmdline)
	e.GET("/debug/pprof/goroutine", s.PprofGoroutine)
	e.GET("/debug/pprof/heap", s.PprofHeap)
	e.GET("/debug/pprof/mutex", s.PprofMutex)
	e.GET("/debug/pprof/profile", s.PprofProfile)
	e.GET("/debug/pprof/threadcreate", s.PprofThreadcreate)
	e.GET("/debug/pprof/trace", s.PprofTrace)
	index.addPage("/debug/pprof/", "Go std profiler")
	index.addPage("/debug/pprof/profile?seconds=30", "Take half-min profile")

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

func (s *Server) LogLevel(eCtx echo.Context) error {
	logger.LogLevel.ServeHTTP(eCtx.Response().Writer, eCtx.Request())

	return nil
}

func (s *Server) PprofIndex(eCtx echo.Context) error {
	pprof.Index(eCtx.Response().Writer, eCtx.Request())
	return nil
}

func (s *Server) PprofAllocs(eCtx echo.Context) error {
	pprof.Handler("allocs").ServeHTTP(eCtx.Response().Writer, eCtx.Request())
	return nil
}

func (s *Server) PprofBlock(eCtx echo.Context) error {
	pprof.Handler("block").ServeHTTP(eCtx.Response().Writer, eCtx.Request())
	return nil
}

func (s *Server) PprofCmdline(eCtx echo.Context) error {
	pprof.Cmdline(eCtx.Response().Writer, eCtx.Request())
	return nil
}

func (s *Server) PprofGoroutine(eCtx echo.Context) error {
	pprof.Handler("goroutine").ServeHTTP(eCtx.Response().Writer, eCtx.Request())
	return nil
}

func (s *Server) PprofHeap(eCtx echo.Context) error {
	pprof.Handler("heap").ServeHTTP(eCtx.Response().Writer, eCtx.Request())
	return nil
}

func (s *Server) PprofMutex(eCtx echo.Context) error {
	pprof.Handler("mutex").ServeHTTP(eCtx.Response().Writer, eCtx.Request())
	return nil
}

func (s *Server) PprofProfile(eCtx echo.Context) error {
	pprof.Profile(eCtx.Response().Writer, eCtx.Request())
	return nil
}

func (s *Server) PprofThreadcreate(eCtx echo.Context) error {
	pprof.Handler("threadcreate").ServeHTTP(eCtx.Response().Writer, eCtx.Request())
	return nil
}

func (s *Server) PprofTrace(eCtx echo.Context) error {
	pprof.Trace(eCtx.Response().Writer, eCtx.Request())
	return nil
}
