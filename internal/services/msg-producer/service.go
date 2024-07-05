package msgproducer

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/segmentio/kafka-go"
)

type KafkaWriter interface {
	io.Closer
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

//go:generate options-gen -out-filename=service_options.gen.go -from-struct=Options
type Options struct {
	wr           KafkaWriter `option:"mandatory" validate:"required"`
	encryptKey   string      `validate:"omitempty,hexadecimal"`
	nonceFactory func(size int) ([]byte, error)
}

type Service struct {
	wr           KafkaWriter
	cipher       cipher.AEAD
	nonceFactory func(size int) ([]byte, error)
}

func New(opts Options) (*Service, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	if opts.nonceFactory == nil {
		opts.nonceFactory = defaultNonceFactory
	}

	var aeadCipher cipher.AEAD
	if opts.encryptKey != "" {
		key, err := hex.DecodeString(opts.encryptKey)
		if err != nil {
			return nil, fmt.Errorf("msg producer decode encrypt key error: %v", err)
		}

		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("msg producer new cipher error: %v", err)
		}

		aeadCipher, err = cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("msg producer new gcm error: %v", err)
		}
	}

	return &Service{
		wr:           opts.wr,
		cipher:       aeadCipher,
		nonceFactory: opts.nonceFactory,
	}, nil
}

func defaultNonceFactory(size int) (nonce []byte, err error) {
	nonce = make([]byte, size)
	_, err = rand.Read(nonce)
	return nonce, err
}
