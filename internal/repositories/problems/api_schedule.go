package problemsrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/zestagio/chat-service/internal/store"
	storemessage "github.com/zestagio/chat-service/internal/store/message"
	storeproblem "github.com/zestagio/chat-service/internal/store/problem"
	"github.com/zestagio/chat-service/internal/types"
)

var (
	ErrReqIDNotFount   = errors.New("request id not found")
	ErrProblemNotFound = errors.New("problem not found")
)

func (r *Repo) GetAvailableProblems(ctx context.Context) ([]Problem, error) {
	prbls, err := r.db.Problem(ctx).Query().
		Unique(false).
		Where(storeproblem.ManagerIDIsNil()).
		Where(storeproblem.HasMessagesWith(
			storemessage.IsVisibleForManager(true),
		)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query available problems: %w", err)
	}

	result := make([]Problem, 0, len(prbls))
	for _, p := range prbls {
		result = append(result, adaptStoreProblem(p))
	}
	return result, nil
}

func (r *Repo) SetManagerForProblem(ctx context.Context, problemID types.ProblemID, managerID types.UserID) error {
	_, err := r.db.Problem(ctx).
		UpdateOneID(problemID).
		Where(storeproblem.ManagerIDIsNil()).
		SetManagerID(managerID).
		Save(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return fmt.Errorf("set manager for problem: %w", ErrProblemNotFound)
		}
		return fmt.Errorf("set manager for problem: %w", err)
	}

	return nil
}

func (r *Repo) GetProblemRequestID(ctx context.Context, problemID types.ProblemID) (types.RequestID, error) {
	problem, err := r.db.Problem(ctx).Query().
		Unique(false).
		WithMessages(func(query *store.MessageQuery) {
			query.Where(storemessage.IsVisibleForManager(true)).
				Order(store.Asc(storemessage.FieldCreatedAt)).
				Limit(1)
		}).
		Where(storeproblem.ID(problemID)).
		First(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return types.RequestIDNil, fmt.Errorf("get problem request id: %w", ErrReqIDNotFount)
		}
		return types.RequestIDNil, fmt.Errorf("get problem request id: %w", err)
	}

	if len(problem.Edges.Messages) > 0 {
		m := problem.Edges.Messages[0]
		return m.InitialRequestID, nil
	}

	return types.RequestIDNil, fmt.Errorf("get problem request id: %w", ErrReqIDNotFount)
}
