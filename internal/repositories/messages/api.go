package messagesrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/message"
	"github.com/zestagio/chat-service/internal/types"
	"github.com/zestagio/chat-service/pkg/pointer"
)

var ErrMsgNotFound = errors.New("message not found")

func (r *Repo) GetMessageByRequestID(ctx context.Context, reqID types.RequestID) (*Message, error) {
	msg, err := r.db.Message(ctx).Query().Where(message.InitialRequestIDEQ(reqID)).First(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, ErrMsgNotFound
		}
		return nil, fmt.Errorf("get chat message: %v", err)
	}

	return pointer.Ptr(adaptStoreMessage(msg)), nil
}

// CreateClientVisible creates a message that is visible only to the client.
func (r *Repo) CreateClientVisible(
	ctx context.Context,
	reqID types.RequestID,
	problemID types.ProblemID,
	chatID types.ChatID,
	authorID types.UserID,
	msgBody string,
) (*Message, error) {
	msg, err := r.db.Message(ctx).Create().
		SetInitialRequestID(reqID).
		SetProblemID(problemID).
		SetChatID(chatID).
		SetAuthorID(authorID).
		SetBody(msgBody).
		SetIsVisibleForManager(false).
		SetIsVisibleForClient(true).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create chat message: %v", err)
	}

	return pointer.Ptr(adaptStoreMessage(msg)), nil
}
