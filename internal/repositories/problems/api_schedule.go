package problemsrepo

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/message"
	"github.com/zestagio/chat-service/internal/store/problem"
	"github.com/zestagio/chat-service/internal/types"
)

func (r *Repo) GetProblemsWithoutManager(ctx context.Context, lim int) ([]Problem, error) {
	if lim <= 0 {
		return nil, errors.New("invalid limit")
	}

	problems, err := r.db.Problem(ctx).Query().
		Unique(false).
		Modify(func(s *sql.Selector) {
			t1 := sql.Table(message.Table)

			s.Select(s.C(problem.FieldID), s.C(problem.FieldChatID), s.C(problem.FieldManagerID))
			s.Join(t1.As(message.Table)).On(s.C(problem.FieldID), t1.C(message.FieldProblemID))
			s.Where(sql.And(
				sql.IsNull(s.C(problem.FieldManagerID)),
				sql.IsTrue(t1.C(message.FieldIsVisibleForManager)),
			))
			s.GroupBy(s.C(problem.FieldID), s.C(problem.FieldChatID))
			s.Having(sql.GT(sql.Count(t1.C(message.FieldID)), sql.Raw("0")))
			s.OrderBy(sql.Asc(s.C(problem.FieldCreatedAt)))
			s.Limit(lim)
		}).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("select problems: %v", err)
	}

	result := make([]Problem, 0, len(problems))
	for _, p := range problems {
		result = append(result, adaptStoreProblem(p))
	}
	return result, nil
}

func (r *Repo) SetManagerForProblem(
	ctx context.Context,
	problemID types.ProblemID,
	managerID types.UserID,
) error {
	_, err := r.db.Problem(ctx).Update().
		Where(problem.ID(problemID)).
		SetManagerID(managerID).
		Save(ctx)
	return err
}

// GetProblemInitialRequestID returns InitialRequestID of first manager-visible problem message.
func (r *Repo) GetProblemInitialRequestID(ctx context.Context, pID types.ProblemID) (types.RequestID, error) {
	msg, err := r.db.Message(ctx).Query().
		Unique(false).
		Select(message.FieldInitialRequestID).
		Where(
			message.ProblemID(pID),
			message.IsVisibleForManager(true),
		).
		Order(store.Asc(message.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return types.RequestIDNil, errors.New("no suitable problem messages")
		}
	}
	return msg.InitialRequestID, nil
}
