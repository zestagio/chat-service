package managerload

import (
	"context"
	"fmt"

	"github.com/zestagio/chat-service/internal/types"
)

func (s *Service) CanManagerTakeProblem(ctx context.Context, managerID types.UserID) (bool, error) {
	count, err := s.problemsRepo.GetManagerOpenProblemsCount(ctx, managerID)
	if err != nil {
		return false, fmt.Errorf("get manager open problem err: %v", err)
	}

	return count < s.maxProblemsAtTime, nil
}
