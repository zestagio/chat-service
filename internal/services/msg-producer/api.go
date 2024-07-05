package msgproducer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"

	"github.com/zestagio/chat-service/internal/types"
)

type Message struct {
	ID         types.MessageID
	ChatID     types.ChatID
	Body       string
	FromClient bool
}

func (s *Service) ProduceMessage(ctx context.Context, msg Message) error {
	val, err := s.getMessageValue(msg)
	if err != nil {
		return fmt.Errorf("compute msg value: %v", err)
	}

	return s.wr.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.ChatID.String()),
		Value: val,
	})
}

func (s *Service) Close() error {
	return s.wr.Close()
}

func (s *Service) getMessageValue(msg Message) ([]byte, error) {
	data, err := json.Marshal(struct {
		ID         string `json:"id"`
		ChatID     string `json:"chatId"`
		Body       string `json:"body"`
		FromClient bool   `json:"fromClient"`
	}{
		ID:         msg.ID.String(),
		ChatID:     msg.ChatID.String(),
		Body:       msg.Body,
		FromClient: msg.FromClient,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal msg: %v", err)
	}

	if s.cipher == nil {
		return data, nil
	}
	return s.encryptData(data)
}

func (s *Service) encryptData(msg []byte) ([]byte, error) {
	nonce, err := s.nonceFactory(s.cipher.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("build nonce: %v", err)
	}
	return s.cipher.Seal(nonce, nonce, msg, nil), nil
}
