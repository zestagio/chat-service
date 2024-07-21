package messagesrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/chat"
	"github.com/zestagio/chat-service/internal/store/message"
	"github.com/zestagio/chat-service/internal/store/problem"
	"github.com/zestagio/chat-service/internal/types"
)

var (
	ErrInvalidPageSize = errors.New("invalid page size")
	ErrInvalidCursor   = errors.New("invalid cursor")
)

const (
	minPageSize = 10
	maxPageSize = 100
)

type Cursor struct {
	LastCreatedAt time.Time
	PageSize      int
}

func (c Cursor) Validate() error {
	if c.LastCreatedAt.IsZero() {
		return errors.New("LastCreatedAt field must be specified")
	}
	return validatePageSize(c.PageSize)
}

func validatePageSize(ps int) error {
	if ps := ps; ps < minPageSize || ps > maxPageSize {
		return fmt.Errorf("PageSize field must be in [%d, %d]", minPageSize, maxPageSize)
	}
	return nil
}

// GetClientChatMessages returns Nth page of messages in the chat for client side.
func (r *Repo) GetClientChatMessages(
	ctx context.Context,
	clientID types.UserID,
	pageSize int,
	cursor *Cursor,
) ([]Message, *Cursor, error) {
	query := r.db.Message(ctx).Query().
		Unique(false).
		Where(message.IsVisibleForClient(true)).
		Where(message.HasChatWith(chat.ClientID(clientID)))

	return r.getChatMessages(ctx, query, pageSize, cursor)
}

// GetProblemMessages returns Nth page of messages in the chat for manager side (specific problem).
func (r *Repo) GetProblemMessages(
	ctx context.Context,
	problemID types.ProblemID,
	pageSize int,
	cursor *Cursor,
) ([]Message, *Cursor, error) {
	query := r.db.Message(ctx).Query().
		Unique(false).
		Where(message.IsVisibleForManager(true)).
		Where(message.HasProblemWith(
			problem.ID(problemID),
			problem.ResolvedAtIsNil(),
		))

	return r.getChatMessages(ctx, query, pageSize, cursor)
}

// getChatMessages returns messages either by clientID or by problemID.
func (r *Repo) getChatMessages(
	ctx context.Context,
	query *store.MessageQuery,
	pageSize int,
	cursor *Cursor,
) ([]Message, *Cursor, error) {
	lastCreatedAt := time.Now().AddDate(100, 0, 0)
	if cursor != nil {
		if err := cursor.Validate(); err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrInvalidCursor, err)
		}
		pageSize, lastCreatedAt = cursor.PageSize, cursor.LastCreatedAt
	} else {
		if err := validatePageSize(pageSize); err != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrInvalidPageSize, err)
		}
	}

	msgs, err := query.
		Where(message.CreatedAtLT(lastCreatedAt)).
		Order(store.Desc(message.FieldCreatedAt)).
		Limit(pageSize + 1).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("select messages: %v", err)
	}

	result := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, adaptStoreMessage(m))
	}

	if len(result) <= pageSize {
		return result, nil, nil
	}

	result = result[:len(result)-1]
	return result, &Cursor{
		LastCreatedAt: result[len(result)-1].CreatedAt,
		PageSize:      pageSize,
	}, nil
}
