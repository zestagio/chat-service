package afcverdictsprocessor

import (
	"github.com/golang-jwt/jwt"

	"github.com/zestagio/chat-service/internal/types"
	"github.com/zestagio/chat-service/internal/validator"
)

type msgStatus string

const (
	msgStatusOK         msgStatus = "ok"
	msgStatusSuspicious msgStatus = "suspicious"
)

var _ jwt.Claims = verdict{}

type verdict struct {
	ChatID    types.ChatID    `json:"chatId" validate:"required"`
	MessageID types.MessageID `json:"messageId" validate:"required"`
	Status    msgStatus       `json:"status" validate:"required"`
}

func (v verdict) Valid() error {
	return validator.Validator.Struct(v)
}
