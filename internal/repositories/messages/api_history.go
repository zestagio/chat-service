package messagesrepo

import (
	"context"
	"errors"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/chat"
	"github.com/zestagio/chat-service/internal/store/message"
	"github.com/zestagio/chat-service/internal/store/predicate"
	"github.com/zestagio/chat-service/internal/types"
)

const (
	emptyPageSize = 0
	minPageSize   = 10
	maxPageSize   = 100
)

var (
	ErrInvalidPageSize = errors.New("invalid page size")
	ErrInvalidCursor   = errors.New("invalid cursor")
)

type Cursor struct {
	LastCreatedAt time.Time
	PageSize      int
}

// GetClientChatMessages returns Nth page of messages in the chat for client side.
func (r *Repo) GetClientChatMessages(
	ctx context.Context,
	clientID types.UserID,
	pageSize int,
	cursor *Cursor,
) ([]Message, *Cursor, error) {
	var limit int
	var lastCreatedAt time.Time

	if cursor != nil {
		if err := r.validateCursor(cursor); err != nil {
			return nil, nil, err
		}
		limit = cursor.PageSize
		lastCreatedAt = cursor.LastCreatedAt
	} else {
		if err := r.validatePageSize(pageSize); err != nil {
			return nil, nil, err
		}
		limit = pageSize
	}

	res, err := r.buildQuery(ctx, clientID, pageSize, lastCreatedAt).All(ctx)
	if err != nil {
		return nil, nil, err
	}

	messages := make([]Message, 0, len(res))
	for _, msg := range res {
		messages = append(messages, adaptStoreMessage(msg))
	}

	if len(messages) == 0 {
		return messages, nil, nil
	}

	var newCursor *Cursor
	exists, _ := r.buildQuery(ctx, clientID, 1, res[len(res)-1].CreatedAt).Exist(ctx)
	if exists {
		newCursor = &Cursor{
			PageSize:      limit,
			LastCreatedAt: res[len(res)-1].CreatedAt,
		}
	}

	return messages, newCursor, nil
}

func (r *Repo) validatePageSize(pageSize int) error {
	if pageSize == emptyPageSize {
		return nil
	}
	if pageSize < minPageSize || pageSize > maxPageSize {
		return ErrInvalidPageSize
	}
	return nil
}

func (r *Repo) validateCursor(cursor *Cursor) error {
	if cursor == nil {
		return ErrInvalidCursor
	}
	if cursor.LastCreatedAt.IsZero() {
		return ErrInvalidCursor
	}
	if err := r.validatePageSize(cursor.PageSize); err != nil {
		return ErrInvalidCursor
	}
	return nil
}

func (r *Repo) buildQuery(ctx context.Context, clientID types.UserID, pageSize int, lastCreatedAt time.Time) *store.MessageQuery {
	predicates := make([]predicate.Message, 0, 3)
	predicates = append(predicates, message.HasChatWith(chat.ClientID(clientID)))
	predicates = append(predicates, message.IsVisibleForClient(true))

	if !lastCreatedAt.IsZero() {
		predicates = append(predicates, message.CreatedAtLT(lastCreatedAt))
	}

	return r.db.Message(ctx).
		Query().
		Where(predicates...).
		Order(message.ByCreatedAt(sql.OrderDesc())).
		Limit(pageSize)
}
