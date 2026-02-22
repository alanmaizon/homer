package llm

import (
	"context"
	"log"
	"time"

	"github.com/alanmaizon/homer/backend/internal/metrics"
	"github.com/alanmaizon/homer/backend/internal/middleware"
)

func observeProviderOperation(ctx context.Context, provider string, operation string, call func() (string, error)) (string, error) {
	started := time.Now()
	requestID := middleware.GetRequestIDFromContext(ctx)

	log.Printf(
		"request_id=%s component=provider provider=%s operation=%s event=start",
		requestID,
		provider,
		operation,
	)

	result, err := call()

	status := "success"
	errorCategory := "none"
	if err != nil {
		status = "error"
		errorCategory = providerErrorCategory(err)
	}

	duration := time.Since(started)
	metrics.RecordProviderCall(provider, operation, status, errorCategory, duration)
	log.Printf(
		"request_id=%s component=provider provider=%s operation=%s status=%s error_category=%s duration_ms=%d",
		requestID,
		provider,
		operation,
		status,
		errorCategory,
		duration.Milliseconds(),
	)

	return result, err
}
