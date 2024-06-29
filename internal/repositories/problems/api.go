package problemsrepo

import (
	"context"
	"fmt"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/chat"
	"github.com/zestagio/chat-service/internal/store/problem"
	"github.com/zestagio/chat-service/internal/types"
)

func (r *Repo) CreateIfNotExists(ctx context.Context, chatID types.ChatID) (types.ProblemID, error) {
	pID, err := r.db.Problem(ctx).Query().
		Unique(false).
		Where(
			problem.HasChatWith(chat.ID(chatID)),
			problem.ResolvedAtIsNil(),
		).
		FirstID(ctx)
	if nil == err {
		return pID, nil
	}
	if !store.IsNotFound(err) {
		return types.ProblemIDNil, fmt.Errorf("select existent problem: %v", err)
	}

	p, err := r.db.Problem(ctx).Create().
		SetChatID(chatID).
		Save(ctx)
	if err != nil {
		return types.ProblemIDNil, fmt.Errorf("create new problem: %v", err)
	}

	return p.ID, nil
}

func (r *Repo) GetManagerOpenProblemsCount(ctx context.Context, managerID types.UserID) (int, error) {
	return r.db.Problem(ctx).Query().
		Unique(false).
		Where(
			problem.ManagerID(managerID),
			problem.ResolvedAtIsNil(),
		).
		Count(ctx)
}
