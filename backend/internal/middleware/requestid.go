package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDKey = "requestId"

type requestIDContextToken struct{}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(requestIDKey, requestID)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDContextToken{}, requestID))
		c.Writer.Header().Set("X-Request-Id", requestID)
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	if value, ok := c.Get(requestIDKey); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}

func GetRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value := ctx.Value(requestIDContextToken{}); value != nil {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
