package middlewares

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	internalerrors "github.com/zestagio/chat-service/internal/errors"
)

func NewRequestLogger(lg *zap.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().Method == http.MethodOptions
		},
		LogValuesFunc: func(eCtx echo.Context, v middleware.RequestLoggerValues) error {
			lg := lg.With(
				zap.Duration("latency", v.Latency),
				zap.String("remote_ip", v.RemoteIP),
				zap.String("host", v.Host),
				zap.String("method", v.Method),
				zap.String("path", v.URIPath),
				zap.String("request_id", v.RequestID),
				zap.String("user_agent", v.UserAgent),
			)

			uid, _ := userID(eCtx)
			lg = lg.With(zap.Stringer("user_id", uid))

			status := v.Status

			if err := v.Error; err != nil {
				lg = lg.With(zap.Error(err))
				status = internalerrors.GetServerErrorCode(err)
			}

			lg = lg.With(zap.Int("status", status))

			switch {
			case status >= 1000:
				lg.Error("business logic error")
			case status >= 500:
				lg.Error("server error")
			case status >= 400:
				lg.Error("client error")
			default:
				lg.Info("success")
			}

			return nil
		},
		LogLatency:   true,
		LogRemoteIP:  true,
		LogHost:      true,
		LogMethod:    true,
		LogURIPath:   true,
		LogRequestID: true,
		LogUserAgent: true,
		LogStatus:    true,
		LogError:     true,
	})
}
