package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/alanmaizon/homer/backend/internal/agents"
	"github.com/alanmaizon/homer/backend/internal/connectors"
	"github.com/alanmaizon/homer/backend/internal/domain"
	"github.com/alanmaizon/homer/backend/internal/llm"
	"github.com/alanmaizon/homer/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	router.GET("/api/capabilities", func(c *gin.Context) {
		requestedProvider := envOrDefault("LLM_PROVIDER", "mock")
		requestedConnector := envOrDefault("CONNECTOR_PROVIDER", "none")

		activeProvider := llm.CurrentProvider().Name()
		activeConnector := connectors.NewConnectorFromEnv().Name()

		c.JSON(http.StatusOK, domain.CapabilitiesResponse{
			Runtime: domain.RuntimeCapabilities{
				RequestedProvider:  requestedProvider,
				ActiveProvider:     activeProvider,
				ProviderFallback:   requestedProvider != activeProvider,
				RequestedConnector: requestedConnector,
				ActiveConnector:    activeConnector,
				ConnectorFallback:  requestedConnector != activeConnector,
			},
			Features: domain.FeatureFlags{
				Critic:          true,
				ConnectorImport: false,
				ConnectorExport: false,
			},
		})
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
		response.Metadata.RequestID = middleware.GetRequestID(c)

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

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
