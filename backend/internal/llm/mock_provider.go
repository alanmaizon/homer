package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/alanmaizon/homer/backend/internal/domain"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) Summarize(ctx context.Context, docs []domain.Document, style string, instructions string) (string, error) {
	return observeProviderOperation(ctx, m.Name(), "summarize", func() (string, error) {
		parts := make([]string, 0, len(docs))
		for _, doc := range docs {
			parts = append(parts, strings.TrimSpace(doc.Content))
		}
		body := strings.TrimSpace(strings.Join(parts, " "))
		if body == "" {
			body = "No document content provided."
		}
		if instructions != "" {
			return fmt.Sprintf("[mock summary:%s] %s (instructions: %s)", style, body, instructions), nil
		}
		return fmt.Sprintf("[mock summary:%s] %s", style, body), nil
	})
}

func (m *MockProvider) Rewrite(ctx context.Context, text string, mode string, instructions string) (string, error) {
	return observeProviderOperation(ctx, m.Name(), "rewrite", func() (string, error) {
		rewritten := strings.TrimSpace(text)
		if rewritten == "" {
			rewritten = "No text provided."
		}
		if instructions != "" {
			return fmt.Sprintf("[mock rewrite:%s] %s (instructions: %s)", mode, rewritten, instructions), nil
		}
		return fmt.Sprintf("[mock rewrite:%s] %s", mode, rewritten), nil
	})
}
