package jobs

import (
	"encoding/json"
	"fmt"

	"github.com/zestagio/chat-service/internal/types"
	"github.com/zestagio/chat-service/internal/validator"
)

type Payload struct {
	MessageID types.MessageID `json:"messageId" validate:"required"`
}

func (p Payload) Validate() error {
	return validator.Validator.Struct(p)
}

func MarshalPayload(messageID types.MessageID) (string, error) {
	p := Payload{
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

func UnmarshalPayload(data string) (p Payload, err error) {
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		return Payload{}, fmt.Errorf("unmarshal: %v", err)
	}

	return p, nil
}
