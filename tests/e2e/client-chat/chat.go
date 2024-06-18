//go:build e2e

package clientchat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/golang-jwt/jwt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	keycloakclient "github.com/zestagio/chat-service/internal/clients/keycloak"
	"github.com/zestagio/chat-service/internal/types"
	apiclientv1 "github.com/zestagio/chat-service/tests/e2e/api/client/v1"
	clientchat "github.com/zestagio/chat-service/tests/e2e/client-chat"
)

const pageSize = 10

var (
	errNoResponseBody   = errors.New("no response body")
	errNoDataInResponse = errors.New("no data field in response")
)

type Message struct {
	ID         types.MessageID
	AuthorID   types.UserID
	Body       string
	IsService  bool
	IsBlocked  bool
	IsReceived bool
	CreatedAt  time.Time
}

//go:generate options-gen -out-filename=chat_options.gen.go -from-struct=Options
type Options struct {
	id    types.UserID                     `option:"mandatory" validate:"required"`
	token string                           `option:"mandatory" validate:"required"`
	api   *apiclientv1.ClientWithResponses `option:"mandatory" validate:"required"`
}

type Chat struct {
	Options

	cursor string

	msgMu        *sync.RWMutex
	messagesByID map[types.MessageID]*Message
	messages     []*Message
}

func New(opts Options) (*Chat, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	return &Chat{
		Options:      opts,
		cursor:       "",
		msgMu:        new(sync.RWMutex),
		messagesByID: make(map[types.MessageID]*Message),
		messages:     nil,
	}, nil
}

func (c *Chat) ClientID() types.UserID {
	return c.id
}

func (c *Chat) AccessToken() string {
	return c.token
}

func (c *Chat) LastMessage() (Message, bool) {
	c.msgMu.RLock()
	defer c.msgMu.RUnlock()

	if len(c.messages) == 0 {
		return Message{}, false
	}
	return *c.messages[len(c.messages)-1], true
}

func (c *Chat) Messages() []Message {
	c.msgMu.RLock()
	defer c.msgMu.RUnlock()

	result := make([]Message, 0, len(c.messages))
	for _, m := range c.messages {
		result = append(result, *m)
	}
	return result
}

func (c *Chat) MessagesCount() int {
	c.msgMu.RLock()
	defer c.msgMu.RUnlock()

	return len(c.messages)
}

// Refresh emulates the browser page reloading.
func (c *Chat) Refresh(ctx context.Context) error {
	c.msgMu.Lock()
	{
		c.messages = nil
		c.messagesByID = make(map[types.MessageID]*Message)
		c.cursor = ""
	}
	c.msgMu.Unlock()

	return c.GetHistory(ctx)
}

func (c *Chat) GetHistory(ctx context.Context) error {
	resp, err := c.api.PostGetHistoryWithResponse(ctx,
		&apiclientv1.PostGetHistoryParams{XRequestID: types.NewRequestID()},
		apiclientv1.PostGetHistoryJSONRequestBody{
			Cursor:   pointer.Ptr(c.cursor),
			PageSize: pointer.Ptr(pageSize),
		},
	)
	if err != nil {
		return fmt.Errorf("post request: %v", err)
	}
	if resp.JSON200 == nil {
		return errNoResponseBody
	}
	if err := resp.JSON200.Error; err != nil {
		return fmt.Errorf("%v: %v", err.Code, err.Message)
	}

	data := resp.JSON200.Data
	if data == nil {
		return errNoDataInResponse
	}

	for _, m := range data.Messages {
		msg := &Message{
			ID:         m.Id,
			Body:       m.Body,
			IsService:  m.IsService,
			IsBlocked:  m.IsBlocked,
			IsReceived: m.IsReceived,
			CreatedAt:  m.CreatedAt,
		}
		if uid := m.AuthorId; uid != nil {
			msg.AuthorID = *uid
		}
		c.addMessageToStart(msg)
	}

	c.cursor = data.Next
	return nil
}

func WithRequestID(id types.RequestID) SendMessageOption {
	return func(opts *sendMessageOpts) {
		opts.reqID = id
	}
}

type SendMessageOption func(opts *sendMessageOpts)

type sendMessageOpts struct {
	reqID types.RequestID
}

func (c *Chat) SendMessage(ctx context.Context, body string, opts ...SendMessageOption) error {
	options := sendMessageOpts{
		reqID: types.NewRequestID(),
	}
	for _, o := range opts {
		o(&options)
	}

	resp, err := c.api.PostSendMessageWithResponse(ctx,
		&apiclientv1.PostSendMessageParams{XRequestID: options.reqID},
		apiclientv1.PostSendMessageJSONRequestBody{MessageBody: body},
	)
	if err != nil {
		return fmt.Errorf("post request: %v", err)
	}
	if resp.JSON200 == nil {
		return errNoResponseBody
	}
	if err := resp.JSON200.Error; err != nil {
		return fmt.Errorf("%v: %v", err.Code, err.Message)
	}

	data := resp.JSON200.Data
	if data == nil {
		return errNoDataInResponse
	}

	msg := &Message{
		ID:         data.Id,
		AuthorID:   types.UserIDNil,
		Body:       body,
		IsService:  false,
		IsBlocked:  false,
		IsReceived: false,
		CreatedAt:  data.CreatedAt,
	}
	if uid := data.AuthorId; uid != nil {
		msg.AuthorID = *uid
	}
	c.addMessageToEnd(msg)

	time.Sleep(10 * time.Millisecond)
	return nil
}

func (c *Chat) addMessageToStart(msg *Message) {
	c.msgMu.Lock()
	defer c.msgMu.Unlock()

	if _, ok := c.messagesByID[msg.ID]; !ok {
		c.messages = append([]*Message{msg}, c.messages...)
		c.messagesByID[msg.ID] = msg
	}
}

func (c *Chat) addMessageToEnd(msg *Message) {
	c.msgMu.Lock()
	defer c.msgMu.Unlock()

	if _, ok := c.messagesByID[msg.ID]; !ok {
		c.messages = append(c.messages, msg)
		c.messagesByID[msg.ID] = msg
	}
}
