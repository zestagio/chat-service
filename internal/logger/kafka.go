package logger

import (
	"fmt"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const (
	infoType  = "info"
	errorType = "error"
)

var _ kafka.Logger = (*KafkaAdapted)(nil)

type KafkaAdapted struct {
	lg      *zap.Logger
	logType string
}

func NewKafkaAdapted() *KafkaAdapted {
	return &KafkaAdapted{
		lg:      zap.L(),
		logType: infoType,
	}
}

func (k *KafkaAdapted) Printf(format string, v ...any) {
	result := fmt.Sprintf(format, v...)
	switch k.logType {
	case infoType:
		k.lg.Info(result)
	case errorType:
		k.lg.Error(result)
	}
}

func (k *KafkaAdapted) WithServiceName(name string) *KafkaAdapted {
	k.lg.Named(name)
	return k
}

func (k *KafkaAdapted) ForErrors() *KafkaAdapted {
	k.logType = errorType
	return k
}
