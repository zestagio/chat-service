//go:build integration

package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/testingh"
	"github.com/zestagio/chat-service/internal/types"
)

type StoreSuite struct {
	testingh.DBSuite
}

func TestStoreSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &StoreSuite{DBSuite: testingh.NewDBSuite("TestStoreSuite")})
}

func (s *StoreSuite) TestRollbackAfterPanic() {
	var err error
	var wasRollback bool

	s.PanicsWithValue("unexpected error", func() {
		err = s.Database.RunInTx(s.Ctx, func(ctx context.Context) error {
			tx := store.TxFromContext(ctx)
			s.Require().NotNil(tx)
			tx.OnRollback(func(next store.Rollbacker) store.Rollbacker {
				return store.RollbackFunc(func(ctx context.Context, tx *store.Tx) error {
					wasRollback = true
					return next.Rollback(ctx, tx)
				})
			})

			chat, err := s.Database.Chat(ctx).Create().
				SetClientID(types.NewUserID()).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create chat: %v", err)
			}

			for i := 0; i < 3; i++ {
				_, err = s.Database.Problem(ctx).Create().
					SetChatID(chat.ID).
					SetManagerID(types.NewUserID()).
					Save(ctx)
				if err != nil {
					return fmt.Errorf("create problem #%d: %v", i, err)
				}
			}

			panic("unexpected error")
		})
	})
	s.Require().NoError(err)
	s.True(wasRollback)
	s.Equal(0, s.Database.Chat(s.Ctx).Query().CountX(s.Ctx))
	s.Equal(0, s.Database.Problem(s.Ctx).Query().CountX(s.Ctx))
}

func (s *StoreSuite) TestRollbackAfterError() {
	var wasRollback bool

	err := s.Database.RunInTx(s.Ctx, func(ctx context.Context) error {
		tx := store.TxFromContext(ctx)
		s.Require().NotNil(tx)
		tx.OnRollback(func(next store.Rollbacker) store.Rollbacker {
			return store.RollbackFunc(func(ctx context.Context, tx *store.Tx) error {
				wasRollback = true
				return next.Rollback(ctx, tx)
			})
		})

		_, err := s.Database.Chat(ctx).Create().
			SetClientID(types.NewUserID()).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("create chat: %v", err)
		}

		_, err = s.Database.Problem(ctx).Create().
			// No required chat id.
			SetManagerID(types.NewUserID()).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("create problem: %w", err)
		}

		return nil
	})
	s.Require().Error(err)
	s.Require().True(store.IsValidationError(err))
	s.True(wasRollback)
	s.Equal(0, s.Database.Chat(s.Ctx).Query().CountX(s.Ctx))
	s.Equal(0, s.Database.Problem(s.Ctx).Query().CountX(s.Ctx))
}

func (s *StoreSuite) TestSuccessfulCommit() {
	var wasCommit bool

	err := s.Database.RunInTx(s.Ctx, func(ctx context.Context) error {
		tx := store.TxFromContext(ctx)
		s.Require().NotNil(tx)
		tx.OnCommit(func(next store.Committer) store.Committer {
			return store.CommitFunc(func(ctx context.Context, tx *store.Tx) error {
				wasCommit = true
				return next.Commit(ctx, tx)
			})
		})

		chat, err := s.Database.Chat(ctx).Create().
			SetClientID(types.NewUserID()).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("create chat: %v", err)
		}

		for i := 0; i < 3; i++ {
			_, err = s.Database.Problem(ctx).Create().
				SetChatID(chat.ID).
				SetManagerID(types.NewUserID()).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("create problem #%d: %v", i, err)
			}
		}

		return nil
	})
	s.Require().NoError(err)
	s.True(wasCommit)
	s.Equal(1, s.Database.Chat(s.Ctx).Query().CountX(s.Ctx))
	s.Equal(3, s.Database.Problem(s.Ctx).Query().CountX(s.Ctx))
}

func (s *StoreSuite) TestNoNestedTransactions() {
	tx, err := s.Store.Tx(s.Ctx)
	s.Require().NoError(err)

	ctx := store.NewTxContext(s.Ctx, tx)

	err = s.Database.RunInTx(ctx, func(ctx context.Context) error {
		tx2 := store.TxFromContext(ctx)
		s.True(tx == tx2, "we should reuse existing transaction") // The same pointers.
		return nil
	})
	s.Require().NoError(err)
}
