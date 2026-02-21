package api

import (
	"net/http"
	"strings"

	"github.com/alanmaizon/homer/backend/internal/agents"
	"github.com/alanmaizon/homer/backend/internal/domain"
	"github.com/alanmaizon/homer/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	router.POST("/api/task", func(c *gin.Context) {
		var req domain.TaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_payload", "invalid task payload")
			return
		}

		if validationErr := validateTaskRequest(req); validationErr != nil {
			writeError(c, http.StatusBadRequest, validationErr.Code, validationErr.Message)
			return
		}

		response, err := agents.ExecuteTask(c.Request.Context(), req)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		c.JSON(http.StatusOK, response)
	})
}

func validateTaskRequest(req domain.TaskRequest) *domain.APIError {
	task := strings.TrimSpace(string(req.Task))
	if task == "" {
		return &domain.APIError{Code: "missing_task", Message: "task is required"}
	}

	switch req.Task {
	case domain.TaskSummarize:
		if len(req.Documents) == 0 {
			return &domain.APIError{
				Code:    "missing_documents",
				Message: "documents are required for summarize",
			}
		}
	case domain.TaskRewrite:
		if strings.TrimSpace(req.Text) == "" {
			return &domain.APIError{
				Code:    "missing_text",
				Message: "text is required for rewrite",
			}
		}
	default:
		return &domain.APIError{
			Code:    "unsupported_task",
			Message: "unsupported task",
		}
	}

	return nil
}

func writeError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, domain.APIErrorResponse{
		Error: domain.APIError{
			Code:      code,
			Message:   message,
			RequestID: middleware.GetRequestID(c),
		},
	})
}
