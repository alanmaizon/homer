package agents

import (
	"testing"

	"github.com/alanmaizon/homer/backend/internal/domain"
)

func TestPlanWithoutCritic(t *testing.T) {
	steps, err := Plan(domain.TaskRequest{Task: domain.TaskSummarize})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Role != domain.RoleExecutor {
		t.Fatalf("expected executor role, got %s", steps[0].Role)
	}
}

func TestPlanWithCritic(t *testing.T) {
	steps, err := Plan(domain.TaskRequest{Task: domain.TaskRewrite, EnableCritic: true})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[1].Role != domain.RoleCritic {
		t.Fatalf("expected critic role, got %s", steps[1].Role)
	}
}
