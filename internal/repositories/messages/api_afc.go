package messagesrepo

import (
	"context"
	"time"

	"github.com/zestagio/chat-service/internal/types"
)

func (r *Repo) MarkAsVisibleForManager(ctx context.Context, msgID types.MessageID) error {
	return r.db.Message(ctx).UpdateOneID(msgID).
		SetCheckedAt(time.Now()).
		SetIsVisibleForManager(true).
		Exec(ctx)
}

func (r *Repo) BlockMessage(ctx context.Context, msgID types.MessageID) error {
	return r.db.Message(ctx).UpdateOneID(msgID).
		SetCheckedAt(time.Now()).
		SetIsBlocked(true).
		Exec(ctx)
}
