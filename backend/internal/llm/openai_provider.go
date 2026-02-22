package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alanmaizon/homer/backend/internal/domain"
)

const openAIURL = "https://api.openai.com/v1/chat/completions"

type OpenAIProvider struct {
	apiKey     string
	model      string
	client     *http.Client
	timeout    time.Duration
	maxRetries int
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

	policy := loadRuntimePolicyFromEnv()

	return &OpenAIProvider{
		apiKey:     apiKey,
		model:      model,
		client:     http.DefaultClient,
		timeout:    policy.timeout,
		maxRetries: policy.maxRetries,
	}, nil
}

func (o *OpenAIProvider) Name() string {
	return "openai"
}

func (o *OpenAIProvider) Summarize(ctx context.Context, docs []domain.Document, style string, instructions string) (string, error) {
	return observeProviderOperation(ctx, o.Name(), "summarize", func() (string, error) {
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
	})
}

func (o *OpenAIProvider) Rewrite(ctx context.Context, text string, mode string, instructions string) (string, error) {
	return observeProviderOperation(ctx, o.Name(), "rewrite", func() (string, error) {
		prompt := fmt.Sprintf("Rewrite this text in %s mode.\nInstructions: %s\n\n%s", mode, instructions, text)
		return o.call(ctx, prompt)
	})
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

	totalAttempts := o.maxRetries + 1
	for attempt := 0; attempt < totalAttempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, o.timeout)

		request, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, openAIURL, bytes.NewReader(body))
		if err != nil {
			cancel()
			return "", err
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+o.apiKey)

		response, err := o.client.Do(request)
		if err != nil {
			cancel()
			if shouldRetryError(err) && attempt < totalAttempts-1 {
				if waitErr := waitForBackoff(ctx, attempt); waitErr != nil {
					return "", waitErr
				}
				continue
			}
			return "", err
		}

		if response.StatusCode >= http.StatusBadRequest {
			responseBytes, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
			_ = response.Body.Close()
			cancel()

			httpErr := &providerHTTPError{
				provider:   "openai",
				statusCode: response.StatusCode,
				message:    strings.TrimSpace(string(responseBytes)),
			}
			if shouldRetryHTTPStatus(response.StatusCode) && attempt < totalAttempts-1 {
				if waitErr := waitForBackoff(ctx, attempt); waitErr != nil {
					return "", waitErr
				}
				continue
			}
			return "", httpErr
		}

		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		decodeErr := json.NewDecoder(response.Body).Decode(&parsed)
		_ = response.Body.Close()
		cancel()
		if decodeErr != nil {
			if shouldRetryError(decodeErr) && attempt < totalAttempts-1 {
				if waitErr := waitForBackoff(ctx, attempt); waitErr != nil {
					return "", waitErr
				}
				continue
			}
			return "", decodeErr
		}
		if len(parsed.Choices) == 0 {
			return "", errors.New("openai returned no choices")
		}
		return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
	}

	return "", errors.New("openai request failed after retries")
}
