package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()

		log.Printf(
			"request_id=%s method=%s path=%s status=%d duration_ms=%d",
			GetRequestID(c),
			c.Request.Method,
			c.FullPath(),
			c.Writer.Status(),
			time.Since(started).Milliseconds(),
		)
	}
}
