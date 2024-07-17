package afcverdictsprocessor

import (
	"context"
	"io"

	"github.com/segmentio/kafka-go"

	"github.com/zestagio/chat-service/internal/logger"
)

const dlqWriterName = "dlq-writer"

//go:generate mockgen -source=$GOFILE -destination=mocks/dlq_writer_mock.gen.go -package=afcverdictsprocessormocks

type KafkaDLQWriter interface {
	io.Closer
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

func NewKafkaDLQWriter(brokers []string, topic string) KafkaDLQWriter {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		BatchSize:    1,
		Balancer:     &kafka.CRC32Balancer{},
		Async:        false,
		RequiredAcks: kafka.RequireOne,
		Logger:       logger.NewKafkaAdapted().WithServiceName(dlqWriterName),
		ErrorLogger:  logger.NewKafkaAdapted().WithServiceName(dlqWriterName).ForErrors(),
	}
}
