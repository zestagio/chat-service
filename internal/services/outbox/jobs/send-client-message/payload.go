package sendclientmessagejob

import (
	"encoding/json"
	"fmt"

	"github.com/zestagio/chat-service/internal/types"
)

type msgPayload struct {
	MessageID types.MessageID `json:"messageId"`
}

func MarshalPayload(messageID types.MessageID) (string, error) {
	if err := messageID.Validate(); err != nil {
		return "", fmt.Errorf("invalid msg id: %v", err)
	}
	payload := msgPayload{MessageID: messageID}

	data, err := json.Marshal(&payload)
	if err != nil {
		return "", fmt.Errorf("json marshal err: %v", err)
	}

	return string(data), nil
}

func unmarshalPayload(payload string) (types.MessageID, error) {
	var pl msgPayload

	if err := json.Unmarshal([]byte(payload), &pl); err != nil {
		return types.MessageIDNil, fmt.Errorf("json unmarshal err: %v", err)
	}

	return pl.MessageID, nil
}
