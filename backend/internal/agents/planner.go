package agents

import (
	"fmt"

	"github.com/alanmaizon/homer/backend/internal/domain"
)

func Plan(req domain.TaskRequest) ([]domain.PlanStep, error) {
	steps := make([]domain.PlanStep, 0, 2)

	switch req.Task {
	case domain.TaskSummarize:
		steps = append(steps, domain.PlanStep{ID: "step-1", Role: domain.RoleExecutor, Action: string(domain.TaskSummarize)})
	case domain.TaskRewrite:
		steps = append(steps, domain.PlanStep{ID: "step-1", Role: domain.RoleExecutor, Action: string(domain.TaskRewrite)})
	default:
		return nil, fmt.Errorf("unsupported task: %s", req.Task)
	}

	if req.EnableCritic {
		steps = append(steps, domain.PlanStep{ID: "step-2", Role: domain.RoleCritic, Action: "review_result"})
	}

	return steps, nil
}
