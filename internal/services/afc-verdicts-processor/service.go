package afcverdictsprocessor

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang-jwt/jwt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	clientmessageblockedjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/client-message-blocked"
	clientmessagesentjob "github.com/zestagio/chat-service/internal/services/outbox/jobs/client-message-sent"
	"github.com/zestagio/chat-service/internal/services/outbox/jobs/payload/simpleid"
	"github.com/zestagio/chat-service/internal/types"
)

const (
	serviceName       = "afc-verdicts-processor"
	dlqSubServiceName = "afc-verdicts-processor.dlq"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/service_mocks.gen.go -package=afcverdictsprocessormocks

type messagesRepository interface {
	BlockMessage(ctx context.Context, msgID types.MessageID) error
	MarkAsVisibleForManager(ctx context.Context, msgID types.MessageID) error
}

type outboxService interface {
	Put(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error)
}

type transactor interface {
	RunInTx(ctx context.Context, f func(context.Context) error) error
}

//go:generate options-gen -out-filename=service_options.gen.go -from-struct=Options
type Options struct {
	backoffInitialInterval time.Duration `default:"100ms" validate:"min=50ms,max=1s"`
	backoffMaxElapsedTime  time.Duration `default:"5s" validate:"min=500ms,max=1m"`

	brokers          []string `option:"mandatory" validate:"min=1"`
	consumers        int      `option:"mandatory" validate:"min=1,max=16"`
	consumerGroup    string   `option:"mandatory" validate:"required"`
	verdictsTopic    string   `option:"mandatory" validate:"required"`
	verdictsSignKey  string
	processBatchSize int `default:"1" validate:"min=1"`

	readerFactory KafkaReaderFactory `option:"mandatory" validate:"required"`
	dlqWriter     KafkaDLQWriter     `option:"mandatory" validate:"required"`

	txtor   transactor         `option:"mandatory" validate:"required"`
	msgRepo messagesRepository `option:"mandatory" validate:"required"`
	outBox  outboxService      `option:"mandatory" validate:"required"`
}

type Service struct {
	Options
	verdictsSignKey *rsa.PublicKey
	dlq             chan erroredMessage
	logger          *zap.Logger
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	var verdictsSignPubKey *rsa.PublicKey
	if key := opts.verdictsSignKey; key != "" {
		var err error
		verdictsSignPubKey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(key))
		if err != nil {
			return nil, fmt.Errorf("parse verdicts signing key: %v", err)
		}
	}

	lg := zap.L().Named(serviceName)

	if verdictsSignPubKey == nil {
		lg.Info("verdicts signature validation disabled")
	} else {
		lg.Info("verdicts signature validation enabled")
	}

	return &Service{
		Options:         opts,
		verdictsSignKey: verdictsSignPubKey,
		dlq:             make(chan erroredMessage),
		logger:          lg,
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return s.startDLQProducer(ctx)
	})

	for i := 0; i < s.consumers; i++ {
		i := i + 1
		eg.Go(func() error {
			return s.startConsumer(ctx, i)
		})
	}

	return eg.Wait()
}

func (s *Service) startConsumer(ctx context.Context, consumerIdx int) (errReturned error) {
	logger := s.logger.With(zap.Int("consumer", consumerIdx))

	consumer := s.readerFactory(s.brokers, s.consumerGroup, s.verdictsTopic)
	defer multierr.AppendInvoke(&errReturned, multierr.Close(consumer))

	b := backoff.WithContext(s.newProcessMessageBackOff(), ctx)

	var messagesProcessed int
	for {
		msg, err := consumer.FetchMessage(ctx)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if errors.Is(err, context.Canceled) {
			return nil
		}
		if err != nil {
			logger.Error("fetch message error", zap.Error(err))
			return err
		}

		logger := logger.With(zap.Int64("offset", msg.Offset))

		if err := backoff.Retry(func() error {
			err := s.processMessage(ctx, msg, logger)
			if nil == err {
				return nil
			}

			if !isRetriable(err) {
				logger.Error("process message unretriable error", zap.Error(err), zap.Any("message", msg))
				return backoff.Permanent(err)
			}

			// Retriable error.
			fields := []zap.Field{zap.Error(err)}
			if v, ok := extractVerdict(err); ok {
				fields = append(fields,
					zap.Stringer("chat_id", v.ChatID),
					zap.Stringer("message_id", v.MessageID),
				)
			}
			logger.Error("process message error", fields...)
			return err
		}, b); err != nil {
			go func() {
				select {
				case <-ctx.Done():
				case s.dlq <- erroredMessage{msg: msg, lastErr: err}:
				}
			}()
		}

		messagesProcessed++

		if messagesProcessed == s.processBatchSize {
			if err := consumer.CommitMessages(ctx, msg); err != nil {
				logger.Error("commit message error", zap.Error(err))
				return err
			}
			messagesProcessed = 0
		}
	}
}

func (s *Service) newProcessMessageBackOff() *backoff.ExponentialBackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     s.backoffInitialInterval,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          1.5,
		MaxInterval:         backoff.DefaultMaxInterval,
		MaxElapsedTime:      s.backoffMaxElapsedTime,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	return b
}

func (s *Service) processMessage(ctx context.Context, msg kafka.Message, logger *zap.Logger) error {
	var v verdict

	if s.verdictsSignKey != nil {
		if _, err := jwt.ParseWithClaims(string(msg.Value), &v, func(_ *jwt.Token) (any, error) {
			return s.verdictsSignKey, nil
		}); err != nil {
			return fmt.Errorf("validate msg signature: %v", err)
		}
	} else {
		if err := json.Unmarshal(msg.Value, &v); err != nil {
			return fmt.Errorf("unmarshal verdict: %v", err)
		}

		if err := v.Valid(); err != nil {
			return fmt.Errorf("invalid verdict: %v", err)
		}
	}

	logger.Info("process verdict",
		zap.Stringer("chat_id", v.ChatID),
		zap.Stringer("message_id", v.MessageID),
		zap.String("message_status", string(v.Status)),
	)

	var err error
	switch status := v.Status; status {
	case msgStatusOK:
		err = s.processValidMessage(ctx, v.MessageID)

	case msgStatusSuspicious:
		err = s.processSuspiciousMessage(ctx, v.MessageID)

	default:
		return fmt.Errorf("unknown verdict: %q", status)
	}

	if err != nil {
		return attachVerdict(newRetriableError(err), v)
	}
	return nil
}

func (s *Service) processValidMessage(ctx context.Context, msgID types.MessageID) error {
	return s.txtor.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.msgRepo.MarkAsVisibleForManager(ctx, msgID); err != nil {
			return fmt.Errorf("mark message %q as visible for manager: %v", msgID.String(), err)
		}

		_, err := s.outBox.Put(ctx, clientmessagesentjob.Name, simpleid.MustMarshal(msgID), time.Now())
		return err
	})
}

func (s *Service) processSuspiciousMessage(ctx context.Context, msgID types.MessageID) error {
	return s.txtor.RunInTx(ctx, func(ctx context.Context) error {
		if err := s.msgRepo.BlockMessage(ctx, msgID); err != nil {
			return fmt.Errorf("block message: %v", err)
		}

		_, err := s.outBox.Put(ctx, clientmessageblockedjob.Name, simpleid.MustMarshal(msgID), time.Now())
		return err
	})
}
