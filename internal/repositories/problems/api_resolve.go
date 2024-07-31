package problemsrepo

import (
	"context"
	"time"

	"github.com/zestagio/chat-service/internal/types"
)

func (r *Repo) Resolve(ctx context.Context, problemID types.ProblemID) error {
	return r.db.Problem(ctx).UpdateOneID(problemID).
		SetResolvedAt(time.Now()).
		Exec(ctx)
}
