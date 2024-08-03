package problemsrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/problem"
	"github.com/zestagio/chat-service/internal/types"
)

var ErrAssignedProblemNotFound = errors.New("assigned problem not found")

func (r *Repo) GetProblemByResolveRequestID(ctx context.Context, reqID types.RequestID) (*Problem, error) {
	p, err := r.db.Problem(ctx).Query().
		Unique(false).
		Where(problem.ResolveRequestID(reqID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("query problem by request id %v: %v", reqID, err)
	}

	pp := adaptStoreProblem(p)
	return &pp, nil
}

func (r *Repo) CreateIfNotExists(ctx context.Context, chatID types.ChatID) (types.ProblemID, error) {
	pID, err := r.db.Problem(ctx).Query().
		Unique(false).
		Where(
			problem.ChatID(chatID),
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

func (r *Repo) GetAssignedProblemID(
	ctx context.Context,
	managerID types.UserID,
	chatID types.ChatID,
) (types.ProblemID, error) {
	pID, err := r.db.Problem(ctx).Query().
		Unique(false).
		Where(
			problem.ManagerID(managerID),
			problem.ChatID(chatID),
			problem.ResolvedAtIsNil(),
		).
		FirstID(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return types.ProblemIDNil, ErrAssignedProblemNotFound
		}
		return types.ProblemIDNil, fmt.Errorf("query assigned problem: %v", err)
	}

	return pID, nil
}

func (r *Repo) ResolveProblem(ctx context.Context, requestID types.RequestID, problemID types.ProblemID) error {
	return r.db.Problem(ctx).UpdateOneID(problemID).
		SetResolveRequestID(requestID).
		SetResolvedAt(time.Now()).
		Exec(ctx)
}
