package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/alanmaizon/homer/backend/internal/domain"
)

const openAIURL = "https://api.openai.com/v1/chat/completions"

type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewOpenAIProviderFromEnv() (*OpenAIProvider, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY is required")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIProvider{apiKey: apiKey, model: model, client: http.DefaultClient}, nil
}

func (o *OpenAIProvider) Name() string {
	return "openai"
}

func (o *OpenAIProvider) Summarize(ctx context.Context, docs []domain.Document, style string, instructions string) (string, error) {
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
	return o.call(ctx, builder.String())
}

func (o *OpenAIProvider) Rewrite(ctx context.Context, text string, mode string, instructions string) (string, error) {
	prompt := fmt.Sprintf("Rewrite this text in %s mode.\nInstructions: %s\n\n%s", mode, instructions, text)
	return o.call(ctx, prompt)
}

func (o *OpenAIProvider) call(ctx context.Context, prompt string) (string, error) {
	payload := map[string]any{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+o.apiKey)

	response, err := o.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("openai request failed with status %d", response.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(response.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("openai returned no choices")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}
