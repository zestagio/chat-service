package problemsrepo

import (
	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/types"
)

type Problem struct {
	ID        types.ProblemID
	ChatID    types.ChatID
	ManagerID types.UserID
}

func adaptStoreProblem(p *store.Problem) Problem {
	return Problem{
		ID:        p.ID,
		ChatID:    p.ChatID,
		ManagerID: p.ManagerID,
	}
}
