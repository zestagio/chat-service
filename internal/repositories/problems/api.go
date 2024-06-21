package problemsrepo

import (
	"context"
	"fmt"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/problem"
	"github.com/zestagio/chat-service/internal/types"
)

func (r *Repo) CreateIfNotExists(ctx context.Context, chatID types.ChatID) (types.ProblemID, error) {
	problemID, err := r.db.Problem(ctx).Query().Where(problem.ChatIDEQ(chatID), problem.ResolvedAtIsNil()).FirstID(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return r.createProblem(ctx, chatID)
		}
		return types.ProblemIDNil, fmt.Errorf("find existing problem: %v", err)
	}

	return problemID, nil
}

func (r *Repo) createProblem(ctx context.Context, chatID types.ChatID) (types.ProblemID, error) {
	newProblem, err := r.db.Problem(ctx).Create().SetChatID(chatID).Save(ctx)
	if err != nil {
		return types.ProblemIDNil, fmt.Errorf("create new problem: %v", err)
	}
	return newProblem.ID, nil
}
