package agents

import (
	"context"
	"errors"

	"github.com/alanmaizon/homer/backend/internal/domain"
	"github.com/alanmaizon/homer/backend/internal/llm"
)

func ExecuteStep(ctx context.Context, step domain.PlanStep, req domain.TaskRequest) (string, error) {
	provider := llm.CurrentProvider()
	switch step.Action {
	case string(domain.TaskSummarize):
		return provider.Summarize(ctx, req.Documents, req.Style, req.Instructions)
	case string(domain.TaskRewrite):
		return provider.Rewrite(ctx, req.Text, req.Mode, req.Instructions)
	default:
		return "", errors.New("unsupported executor action")
	}
}
