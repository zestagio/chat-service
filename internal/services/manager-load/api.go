package managerload

import (
	"context"
	"fmt"

	"github.com/zestagio/chat-service/internal/types"
)

func (s *Service) CanManagerTakeProblem(ctx context.Context, managerID types.UserID) (bool, error) {
	pCount, err := s.problemsRepo.GetManagerOpenProblemsCount(ctx, managerID)
	if err != nil {
		return false, fmt.Errorf("get manager open problems count: %v", err)
	}

	return pCount < s.maxProblemsAtTime, nil
}
