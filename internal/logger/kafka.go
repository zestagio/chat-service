package logger

import (
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

var _ kafka.Logger = (*KafkaAdapted)(nil)

type KafkaAdapted struct {
	forErrors bool
	z         *zap.Logger
}

func NewKafkaAdapted() *KafkaAdapted {
	return &KafkaAdapted{
		z: zap.L(),
	}
}

func (k *KafkaAdapted) ForErrors() *KafkaAdapted {
	k.forErrors = true
	return k
}

func (k *KafkaAdapted) WithServiceName(n string) *KafkaAdapted {
	k.z = k.z.Named(n)
	return k
}

func (k *KafkaAdapted) Printf(s string, args ...any) {
	if k.forErrors {
		k.z.Sugar().Errorf(s, args...)
	} else {
		k.z.Sugar().Debugf(s, args...)
	}
}
