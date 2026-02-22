package llm

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestLoadRuntimePolicyFromEnvDefaults(t *testing.T) {
	t.Setenv("LLM_TIMEOUT_MS", "")
	t.Setenv("LLM_MAX_RETRIES", "")

	policy := loadRuntimePolicyFromEnv()
	if policy.timeout != defaultLLMTimeout {
		t.Fatalf("expected default timeout %s, got %s", defaultLLMTimeout, policy.timeout)
	}
	if policy.maxRetries != defaultLLMMaxRetries {
		t.Fatalf("expected default maxRetries %d, got %d", defaultLLMMaxRetries, policy.maxRetries)
	}
}

func TestLoadRuntimePolicyFromEnvOverridesAndBounds(t *testing.T) {
	t.Setenv("LLM_TIMEOUT_MS", "8000")
	t.Setenv("LLM_MAX_RETRIES", "99")

	policy := loadRuntimePolicyFromEnv()
	if policy.timeout != 8*time.Second {
		t.Fatalf("expected timeout 8s, got %s", policy.timeout)
	}
	if policy.maxRetries != maxLLMMaxRetries {
		t.Fatalf("expected maxRetries capped at %d, got %d", maxLLMMaxRetries, policy.maxRetries)
	}
}

func TestShouldRetryHTTPStatus(t *testing.T) {
	if !shouldRetryHTTPStatus(429) {
		t.Fatalf("expected 429 to be retryable")
	}
	if !shouldRetryHTTPStatus(503) {
		t.Fatalf("expected 503 to be retryable")
	}
	if shouldRetryHTTPStatus(400) {
		t.Fatalf("expected 400 to not be retryable")
	}
}

func TestShouldRetryError(t *testing.T) {
	if !shouldRetryError(context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded to be retryable")
	}

	httpErr := &providerHTTPError{provider: "openai", statusCode: 500, message: "upstream failed"}
	if !shouldRetryError(httpErr) {
		t.Fatalf("expected 500 provider error to be retryable")
	}

	badRequestErr := &providerHTTPError{provider: "openai", statusCode: 400, message: "bad request"}
	if shouldRetryError(badRequestErr) {
		t.Fatalf("expected 400 provider error to not be retryable")
	}

	netErr := &net.DNSError{IsTimeout: true}
	if !shouldRetryError(netErr) {
		t.Fatalf("expected timeout network error to be retryable")
	}

	if shouldRetryError(errors.New("invalid input")) {
		t.Fatalf("expected invalid input error to not be retryable")
	}
}
