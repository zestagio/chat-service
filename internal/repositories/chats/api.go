package chatsrepo

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/chat"
	"github.com/zestagio/chat-service/internal/store/problem"
	"github.com/zestagio/chat-service/internal/types"
)

var ErrChatWithoutManager = errors.New("chat without manager")

func (r *Repo) CreateIfNotExists(ctx context.Context, userID types.UserID) (types.ChatID, error) {
	chatID, err := r.db.Chat(ctx).Create().
		SetClientID(userID).
		OnConflictColumns(chat.FieldClientID).Ignore().
		// More performant way:
		//	OnConflict(
		//		sql.ConflictColumns(chat.FieldClientID),
		//		sql.ResolveWith(func(set *sql.UpdateSet) {
		//			set.SetIgnore(chat.FieldClientID)
		//		}),
		//	).
		ID(ctx)
	if err != nil {
		return types.ChatIDNil, fmt.Errorf("create new chat: %v", err)
	}

	return chatID, nil
}

func (r *Repo) GetChatClient(ctx context.Context, chatID types.ChatID) (types.UserID, error) {
	c, err := r.db.Chat(ctx).Query().
		Unique(false).
		Select(chat.FieldClientID).
		Where(chat.ID(chatID)).
		Only(ctx)
	if err != nil {
		return types.UserIDNil, fmt.Errorf("query chat: %v", err)
	}
	return c.ClientID, nil
}

func (r *Repo) GetChatManager(ctx context.Context, chatID types.ChatID) (types.UserID, error) {
	var managersIDs []types.UserID

	if err := r.db.Problem(ctx).Query().
		Unique(false).
		Where(
			problem.ChatID(chatID),
			problem.ManagerIDNotNil(),
			problem.ResolvedAtIsNil(),
		).
		Limit(1).
		Select(problem.FieldManagerID).
		Scan(ctx, &managersIDs); err != nil {
		return types.UserIDNil, fmt.Errorf("query current chat problem manager: %v", err)
	}
	if len(managersIDs) == 0 {
		return types.UserIDNil, ErrChatWithoutManager
	}

	return managersIDs[0], nil
}

type Chat struct {
	ID       types.ChatID
	ClientID types.UserID
}

func (r *Repo) GetChatsWithOpenProblems(ctx context.Context, managerID types.UserID) ([]Chat, error) {
	chatsWithOpenProblems, err := r.db.Chat(ctx).Query().
		Unique(false).
		Select(chat.FieldID, chat.FieldClientID).
		Where(func(s *sql.Selector) {
			problemT := sql.Table(problem.Table)
			s.Join(problemT).On(s.C(chat.FieldID), problemT.C(problem.FieldChatID))
			s.Where(sql.And(
				sql.EQ(problemT.C(problem.FieldManagerID), managerID),
				sql.IsNull(problemT.C(problem.FieldResolvedAt))),
			)
		}).
		Order(store.Asc(problem.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query problems with chats: %v", err)
	}

	result := make([]Chat, 0, len(chatsWithOpenProblems))
	for _, c := range chatsWithOpenProblems {
		result = append(result, Chat{
			ID:       c.ID,
			ClientID: c.ClientID,
		})
	}
	return result, nil
}
