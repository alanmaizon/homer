package agents

import (
	"context"
	"strings"
	"testing"

	"github.com/alanmaizon/homer/backend/internal/domain"
	"github.com/alanmaizon/homer/backend/internal/llm"
)

func TestExecuteTaskSummarize(t *testing.T) {
	llm.SetProvider(llm.NewMockProvider())

	response, err := ExecuteTask(context.Background(), domain.TaskRequest{
		Task: domain.TaskSummarize,
		Documents: []domain.Document{
			{ID: "1", Title: "Doc", Content: "Hello world"},
		},
		Style: "bullet",
	})
	if err != nil {
		t.Fatalf("ExecuteTask returned error: %v", err)
	}
	if !strings.Contains(response.Result, "[mock summary:bullet]") {
		t.Fatalf("unexpected result: %s", response.Result)
	}
	if response.Metadata.Provider != "mock" {
		t.Fatalf("expected mock provider, got %s", response.Metadata.Provider)
	}
}

func TestExecuteTaskCritic(t *testing.T) {
	llm.SetProvider(llm.NewMockProvider())

	response, err := ExecuteTask(context.Background(), domain.TaskRequest{
		Task:         domain.TaskRewrite,
		Text:         "Rewrite this.",
		Mode:         "professional",
		EnableCritic: true,
	})
	if err != nil {
		t.Fatalf("ExecuteTask returned error: %v", err)
	}
	if !strings.Contains(response.Result, "[critic reviewed]") {
		t.Fatalf("critic output missing: %s", response.Result)
	}
}
