package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alanmaizon/homer/backend/internal/domain"
	"google.golang.org/genai"
)

type GeminiProvider struct {
	model      string
	client     *genai.Client
	timeout    time.Duration
	maxRetries int
}

func NewGeminiProviderFromEnv() (*GeminiProvider, error) {
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
	}
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY (or GOOGLE_API_KEY) is required")
	}

	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if model == "" {
		model = "gemini-2.5-flash"
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gemini client: %w", err)
	}

	policy := loadRuntimePolicyFromEnv()

	return &GeminiProvider{
		model:      model,
		client:     client,
		timeout:    policy.timeout,
		maxRetries: policy.maxRetries,
	}, nil
}

func (g *GeminiProvider) Name() string {
	return "gemini"
}

func (g *GeminiProvider) Summarize(ctx context.Context, docs []domain.Document, style string, instructions string) (string, error) {
	var builder strings.Builder
	builder.WriteString("Summarize the provided documents for the end user.\n")
	if style != "" {
		builder.WriteString("Style: " + style + "\n")
	}
	if instructions != "" {
		builder.WriteString("Instructions: " + instructions + "\n")
	}
	for _, doc := range docs {
		builder.WriteString("\n# " + doc.Title + "\n")
		builder.WriteString(doc.Content + "\n")
	}
	return g.call(ctx, builder.String())
}

func (g *GeminiProvider) Rewrite(ctx context.Context, text string, mode string, instructions string) (string, error) {
	prompt := fmt.Sprintf("Rewrite this text in %s mode.\nInstructions: %s\n\n%s", mode, instructions, text)
	return g.call(ctx, prompt)
}

func (g *GeminiProvider) call(ctx context.Context, prompt string) (string, error) {
	totalAttempts := g.maxRetries + 1
	for attempt := 0; attempt < totalAttempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, g.timeout)
		response, err := g.client.Models.GenerateContent(
			attemptCtx,
			g.model,
			genai.Text(prompt),
			nil,
		)
		cancel()
		if err != nil {
			if shouldRetryError(err) && attempt < totalAttempts-1 {
				if waitErr := waitForBackoff(ctx, attempt); waitErr != nil {
					return "", waitErr
				}
				continue
			}
			return "", err
		}

		text := strings.TrimSpace(response.Text())
		if text == "" {
			return "", errors.New("gemini returned no text")
		}
		return text, nil
	}

	return "", errors.New("gemini request failed after retries")
}
