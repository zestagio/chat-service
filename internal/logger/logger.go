package logger

import (
	"errors"
	"fmt"
	stdlog "log"
	"os"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var LogLevel = zap.NewAtomicLevel()

//go:generate options-gen -out-filename=logger_options.gen.go -from-struct=Options
type Options struct {
	level          string `option:"mandatory" validate:"required,oneof=debug info warn error"`
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

	level, err := zapcore.ParseLevel(opts.level)
	if err != nil {
		return fmt.Errorf("invalid logger level: %v", err)
	}
	LogLevel.SetLevel(level)

	encoder := zapcore.NewConsoleEncoder
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "component",
		TimeKey:        "T",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	if opts.productionMode {
		encoder = zapcore.NewJSONEncoder
		encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	cores := []zapcore.Core{
		zapcore.NewCore(encoder(encoderCfg), os.Stdout, LogLevel),
	}
	l := zap.New(zapcore.NewTee(cores...))
	zap.ReplaceGlobals(l)

	return nil
}

func Sync() {
	if err := zap.L().Sync(); err != nil && !errors.Is(err, syscall.ENOTTY) {
		stdlog.Printf("cannot sync logger: %v", err)
	}
}
