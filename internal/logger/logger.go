package logger

import (
	"errors"
	"fmt"
	stdlog "log"
	"os"
	"syscall"

	"github.com/TheZeroSlave/zapsentry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/zestagio/chat-service/internal/buildinfo"
)

var Level zap.AtomicLevel

//go:generate options-gen -out-filename=logger_options.gen.go -from-struct=Options
type Options struct {
	level          string `option:"mandatory" validate:"required,oneof=debug info warn error"`
	sentryDsn      string `validate:"omitempty,http_url"`
	env            string `validate:"omitempty,oneof=dev stage prod"`
	productionMode bool
}

func MustInit(opts Options) {
	if err := Init(opts); err != nil {
		panic(err)
	}
}

func Init(opts Options) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("validate options: %v", err)
	}

	var err error
	Level, err = zap.ParseAtomicLevel(opts.level)
	if err != nil {
		return fmt.Errorf("parse level: %v", err)
	}

	cfg := zap.NewProductionEncoderConfig()
	cfg.NameKey = "component"
	cfg.TimeKey = "T"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if opts.productionMode {
		cfg.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewJSONEncoder(cfg)
	} else {
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(cfg)
	}

	cores := []zapcore.Core{
		zapcore.NewCore(encoder, os.Stdout, Level),
	}
	l := zap.New(zapcore.NewTee(cores...))

	if opts.sentryDsn != "" {
		sentryClient, err := NewSentryClient(opts.sentryDsn, opts.env, buildinfo.BuildInfo.GoVersion)
		if err != nil {
			return fmt.Errorf("new sentry client: %v", err)
		}

		cfg := zapsentry.Configuration{
			Level:             zapcore.WarnLevel,
			EnableBreadcrumbs: true,
			BreadcrumbLevel:   zapcore.WarnLevel,
			Tags:              map[string]string{"component": "system"},
		}

		core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(sentryClient))
		if err != nil {
			return fmt.Errorf("zapsentry new core: %v", err)
		}

		l = zapsentry.AttachCoreToLogger(core, l)
	}

	zap.ReplaceGlobals(l)

	return nil
}

func Sync() {
	if err := zap.L().Sync(); err != nil && !errors.Is(err, syscall.ENOTTY) {
		stdlog.Printf("cannot sync logger: %v", err)
	}
}
