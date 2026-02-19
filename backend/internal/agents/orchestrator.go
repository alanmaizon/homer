package agents

import (
	"context"
	"time"

	"github.com/alanmaizon/homer/backend/internal/domain"
	"github.com/alanmaizon/homer/backend/internal/llm"
)

func ExecuteTask(ctx context.Context, req domain.TaskRequest) (domain.TaskResponse, error) {
	started := time.Now()

	plan, err := Plan(req)
	if err != nil {
		return domain.TaskResponse{}, err
	}

	result := ""
	for _, step := range plan {
		switch step.Role {
		case domain.RoleExecutor:
			result, err = ExecuteStep(ctx, step, req)
			if err != nil {
				return domain.TaskResponse{}, err
			}
		case domain.RoleCritic:
			result = Critique(result)
		}
	}

	return domain.TaskResponse{
		Result: result,
		Plan:   plan,
		Metadata: domain.Metadata{
			Provider:        llm.CurrentProvider().Name(),
			ExecutionTimeMs: time.Since(started).Milliseconds(),
		},
	}, nil
}
