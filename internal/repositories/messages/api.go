package messagesrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/message"
	"github.com/zestagio/chat-service/internal/types"
)

var ErrMsgNotFound = errors.New("message not found")

func (r *Repo) GetMessageByID(ctx context.Context, msgID types.MessageID) (*Message, error) {
	m, err := r.db.Message(ctx).Get(ctx, msgID)
	if err != nil {
		return nil, fmt.Errorf("query message by id: %v", err)
	}

	mm := adaptStoreMessage(m)
	return &mm, nil
}

func (r *Repo) GetMessageByRequestID(ctx context.Context, reqID types.RequestID) (*Message, error) {
	m, err := r.db.Message(ctx).Query().
		Unique(false).
		Where(
			message.InitialRequestID(reqID),
			message.IsService(false),
		).
		Only(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, fmt.Errorf("request id %v: %w", reqID, ErrMsgNotFound)
		}
		return nil, fmt.Errorf("query message by request id %v: %v", reqID, err)
	}

	mm := adaptStoreMessage(m)
	return &mm, nil
}

func (r *Repo) GetServiceMessageByRequestID(ctx context.Context, reqID types.RequestID) (*Message, error) {
	m, err := r.db.Message(ctx).Query().
		Unique(false).
		Where(
			message.InitialRequestID(reqID),
			message.IsService(true),
		).
		Only(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return nil, fmt.Errorf("request id %v: %w", reqID, ErrMsgNotFound)
		}
		return nil, fmt.Errorf("query service message by request id %v: %v", reqID, err)
	}

	mm := adaptStoreMessage(m)
	return &mm, nil
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
	m, err := r.db.Message(ctx).Create().
		SetChatID(chatID).
		SetProblemID(problemID).
		SetAuthorID(authorID).
		SetIsVisibleForClient(true).
		SetIsVisibleForManager(false).
		SetBody(msgBody).
		SetInitialRequestID(reqID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create msg: %v", err)
	}

	mm := adaptStoreMessage(m)
	return &mm, nil
}

func (r *Repo) CreateFullVisible(
	ctx context.Context,
	reqID types.RequestID,
	problemID types.ProblemID,
	chatID types.ChatID,
	authorID types.UserID,
	msgBody string,
) (*Message, error) {
	msg, err := r.db.Message(ctx).Create().
		SetChatID(chatID).
		SetProblemID(problemID).
		SetAuthorID(authorID).
		SetIsVisibleForClient(true).
		SetIsVisibleForManager(true).
		SetBody(msgBody).
		SetInitialRequestID(reqID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create msg: %v", err)
	}

	mm := adaptStoreMessage(msg)
	return &mm, nil
}

func (r *Repo) CreateServiceMessageForClient(
	ctx context.Context,
	reqID types.RequestID,
	problemID types.ProblemID,
	chatID types.ChatID,
	msgBody string,
) (types.MessageID, error) {
	return r.createServiceMessage(ctx, reqID, problemID, chatID, msgBody, true, false)
}

func (r *Repo) createServiceMessage(
	ctx context.Context,
	reqID types.RequestID,
	problemID types.ProblemID,
	chatID types.ChatID,
	msgBody string,
	isVisibleForClient bool,
	isVisibleForManager bool,
) (types.MessageID, error) {
	msg, err := r.db.Message(ctx).Create().
		SetChatID(chatID).
		SetProblemID(problemID).
		SetIsVisibleForClient(isVisibleForClient).
		SetIsVisibleForManager(isVisibleForManager).
		SetBody(msgBody).
		SetInitialRequestID(reqID).
		SetIsService(true).
		Save(ctx)
	if err != nil {
		return types.MessageIDNil, fmt.Errorf("create msg: %v", err)
	}

	return msg.ID, nil
}
