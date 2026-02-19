package llm

import (
	"context"
	"os"

	"github.com/alanmaizon/homer/backend/internal/domain"
)

type LLMProvider interface {
	Name() string
	Summarize(ctx context.Context, docs []domain.Document, style string, instructions string) (string, error)
	Rewrite(ctx context.Context, text string, mode string, instructions string) (string, error)
}

var currentProvider LLMProvider = NewProviderFromEnv()

func NewProviderFromEnv() LLMProvider {
	if os.Getenv("LLM_PROVIDER") == "openai" {
		if provider, err := NewOpenAIProviderFromEnv(); err == nil {
			return provider
		}
	}
	return NewMockProvider()
}

func CurrentProvider() LLMProvider {
	return currentProvider
}

func SetProvider(provider LLMProvider) {
	if provider != nil {
		currentProvider = provider
	}
}
