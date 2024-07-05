package sendclientmessagejob

import (
	"encoding/json"
	"fmt"

	"github.com/zestagio/chat-service/internal/types"
	"github.com/zestagio/chat-service/internal/validator"
)

type payload struct {
	MessageID types.MessageID `json:"messageId" validate:"required"`
}

func (p payload) Validate() error {
	return validator.Validator.Struct(p)
}

func MarshalPayload(messageID types.MessageID) (string, error) {
	p := payload{
		MessageID: messageID,
	}
	if err := p.Validate(); err != nil {
		return "", fmt.Errorf("validate: %v", err)
	}

	data, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal: %v", err)
	}
	return string(data), nil
}

func unmarshalPayload(data string) (p payload, err error) {
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		return payload{}, fmt.Errorf("unmarshal: %v", err)
	}

	return p, nil
}
