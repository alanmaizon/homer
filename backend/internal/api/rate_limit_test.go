package api

import (
	"testing"
	"time"
)

func TestLoadConnectorRateLimitPerMinuteFromEnvDefault(t *testing.T) {
	t.Setenv("CONNECTOR_RATE_LIMIT_PER_MINUTE", "")

	limit := loadConnectorRateLimitPerMinuteFromEnv()
	if limit != defaultConnectorRateLimitPerMinute {
		t.Fatalf("expected default limit %d, got %d", defaultConnectorRateLimitPerMinute, limit)
	}
}

func TestLoadConnectorRateLimitPerMinuteFromEnvInvalidFallsBack(t *testing.T) {
	t.Setenv("CONNECTOR_RATE_LIMIT_PER_MINUTE", "invalid")

	limit := loadConnectorRateLimitPerMinuteFromEnv()
	if limit != defaultConnectorRateLimitPerMinute {
		t.Fatalf("expected default limit %d, got %d", defaultConnectorRateLimitPerMinute, limit)
	}
}

func TestLoadConnectorRateLimitPerMinuteFromEnvDisable(t *testing.T) {
	t.Setenv("CONNECTOR_RATE_LIMIT_PER_MINUTE", "0")

	limit := loadConnectorRateLimitPerMinuteFromEnv()
	if limit != 0 {
		t.Fatalf("expected disabled limit 0, got %d", limit)
	}
}

func TestFixedWindowLimiterAllow(t *testing.T) {
	current := time.Date(2026, 2, 22, 12, 0, 30, 0, time.UTC)
	limiter := newFixedWindowLimiter(2, func() time.Time { return current })

	if !limiter.Allow() {
		t.Fatalf("expected first request to be allowed")
	}
	if !limiter.Allow() {
		t.Fatalf("expected second request to be allowed")
	}
	if limiter.Allow() {
		t.Fatalf("expected third request to be blocked")
	}

	current = current.Add(45 * time.Second)
	if !limiter.Allow() {
		t.Fatalf("expected new time window request to be allowed")
	}
}
