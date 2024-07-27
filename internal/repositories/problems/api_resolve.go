package problemsrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/zestagio/chat-service/internal/store/problem"
	"github.com/zestagio/chat-service/internal/types"
)

func (r *Repo) Resolve(ctx context.Context, managerID types.UserID, chatID types.ChatID) (types.ProblemID, error) {
	c, err := r.db.Problem(ctx).Update().
		Where(
			problem.ManagerID(managerID),
			problem.ChatID(chatID),
			problem.ResolvedAtIsNil(),
		).
		SetResolvedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return types.ProblemIDNil, fmt.Errorf("resolve: %v", err)
	}

	if c == 0 {
		return types.ProblemIDNil, ErrProblemNotFound
	}

	return r.db.Problem(ctx).Query().
		Unique(false).
		Where(
			problem.ManagerID(managerID),
			problem.ChatID(chatID),
			problem.ResolvedAtNotNil(),
		).
		FirstID(ctx)
}
