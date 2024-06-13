package middlewares

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func NewRecover(lg *zap.Logger) echo.MiddlewareFunc {
	return middleware.RecoverWithConfig(middleware.RecoverConfig{
		Skipper:           middleware.DefaultSkipper,
		StackSize:         4 << 10,
		DisableStackAll:   false,
		DisablePrintStack: false,
		LogLevel:          0,
		LogErrorFunc: func(_ echo.Context, err error, stack []byte) error {
			lg.Error("recover",
				zap.String("error", err.Error()),
				zap.String("stack", string(stack)),
			)
			return nil
		},
		DisableErrorHandler: false,
	})
}
