package api

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const defaultConnectorRateLimitPerMinute = 60

type connectorRequestLimiter interface {
	Allow() bool
}

type fixedWindowLimiter struct {
	mu          sync.Mutex
	limit       int
	windowStart time.Time
	count       int
	now         func() time.Time
}

func newFixedWindowLimiter(limit int, now func() time.Time) *fixedWindowLimiter {
	if now == nil {
		now = time.Now
	}

	return &fixedWindowLimiter{
		limit: limit,
		now:   now,
	}
}

func (l *fixedWindowLimiter) Allow() bool {
	currentWindow := l.now().UTC().Truncate(time.Minute)

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.windowStart.IsZero() || !l.windowStart.Equal(currentWindow) {
		l.windowStart = currentWindow
		l.count = 0
	}

	if l.count >= l.limit {
		return false
	}

	l.count++
	return true
}

func loadConnectorRateLimitPerMinuteFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("CONNECTOR_RATE_LIMIT_PER_MINUTE"))
	if raw == "" {
		return defaultConnectorRateLimitPerMinute
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return defaultConnectorRateLimitPerMinute
	}

	return parsed
}

func newConnectorRateLimiterFromEnv() connectorRequestLimiter {
	limit := loadConnectorRateLimitPerMinuteFromEnv()
	if limit == 0 {
		return nil
	}

	return newFixedWindowLimiter(limit, time.Now)
}

func enforceConnectorRateLimit(c *gin.Context, limiter connectorRequestLimiter) bool {
	if limiter == nil {
		return true
	}

	if limiter.Allow() {
		return true
	}

	writeError(c, http.StatusTooManyRequests, "connector_rate_limited", "connector rate limit exceeded")
	return false
}
