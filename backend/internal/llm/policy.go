package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLLMTimeout    = 15 * time.Second
	defaultLLMMaxRetries = 2
	maxLLMMaxRetries     = 5
	retryBaseDelay       = 200 * time.Millisecond
)

type runtimePolicy struct {
	timeout    time.Duration
	maxRetries int
}

type providerHTTPError struct {
	provider   string
	statusCode int
	message    string
}

func (e *providerHTTPError) Error() string {
	if strings.TrimSpace(e.message) == "" {
		return fmt.Sprintf("%s request failed with status %d", e.provider, e.statusCode)
	}
	return fmt.Sprintf("%s request failed with status %d: %s", e.provider, e.statusCode, e.message)
}

func loadRuntimePolicyFromEnv() runtimePolicy {
	timeout := defaultLLMTimeout
	maxRetries := defaultLLMMaxRetries

	if raw := strings.TrimSpace(os.Getenv("LLM_TIMEOUT_MS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			timeout = time.Duration(parsed) * time.Millisecond
		}
	}

	if raw := strings.TrimSpace(os.Getenv("LLM_MAX_RETRIES")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			maxRetries = parsed
		}
	}

	if maxRetries > maxLLMMaxRetries {
		maxRetries = maxLLMMaxRetries
	}

	return runtimePolicy{
		timeout:    timeout,
		maxRetries: maxRetries,
	}
}

func shouldRetryHTTPStatus(statusCode int) bool {
	return statusCode == 429 || statusCode >= 500
}

func shouldRetryError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var httpErr *providerHTTPError
	if errors.As(err, &httpErr) {
		return shouldRetryHTTPStatus(httpErr.statusCode)
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
		if netErr.Temporary() {
			return true
		}
	}

	message := strings.ToLower(err.Error())
	retryableTokens := []string{
		"timeout",
		"temporarily unavailable",
		"connection reset",
		"connection refused",
		"broken pipe",
		"eof",
		"429",
		"500",
		"502",
		"503",
		"504",
	}
	for _, token := range retryableTokens {
		if strings.Contains(message, token) {
			return true
		}
	}

	return false
}

func waitForBackoff(ctx context.Context, attempt int) error {
	delay := retryBaseDelay << attempt
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
