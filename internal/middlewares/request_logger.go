package middlewares

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func NewRequestLogger(lg *zap.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
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
		LogValuesFunc: func(eCtx echo.Context, v middleware.RequestLoggerValues) error {
			level := zap.InfoLevel
			if eCtx.Response().Status >= http.StatusBadRequest {
				level = zap.ErrorLevel
			}

			lg.Log(
				level,
				"request",
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
	})
}
