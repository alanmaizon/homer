package api

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alanmaizon/homer/backend/internal/agents"
	"github.com/alanmaizon/homer/backend/internal/connectors"
	"github.com/alanmaizon/homer/backend/internal/domain"
	"github.com/alanmaizon/homer/backend/internal/llm"
	"github.com/alanmaizon/homer/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	connectorRateLimiter := newConnectorRateLimiterFromEnv()

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
				ConnectorImport: activeConnector != "none",
				ConnectorExport: activeConnector != "none",
			},
		})
	})

	router.GET("/api/connectors/google_docs/auth/start", func(c *gin.Context) {
		manager, err := connectors.NewGoogleDocsOAuthManagerFromEnv(connectors.OAuthStore())
		if err != nil {
			writeError(c, http.StatusServiceUnavailable, "connector_service_unavailable", "google docs oauth is not configured")
			return
		}

		result, err := manager.StartAuth()
		if err != nil {
			writeError(c, http.StatusServiceUnavailable, "connector_service_unavailable", "failed to initialize google docs oauth")
			return
		}

		c.JSON(http.StatusOK, domain.ConnectorAuthStartResponse{
			Connector:      "google_docs",
			SessionKey:     result.SessionKey,
			AuthURL:        result.AuthURL,
			StateExpiresAt: result.StateExpiresAt.UTC().Format(time.RFC3339),
		})
	})

	router.GET("/api/connectors/google_docs/auth/callback", func(c *gin.Context) {
		manager, err := connectors.NewGoogleDocsOAuthManagerFromEnv(connectors.OAuthStore())
		if err != nil {
			writeError(c, http.StatusServiceUnavailable, "connector_service_unavailable", "google docs oauth is not configured")
			return
		}

		if oauthErr := strings.TrimSpace(c.Query("error")); oauthErr != "" {
			message := strings.TrimSpace(c.Query("error_description"))
			if message == "" {
				message = oauthErr
			}
			writeError(c, http.StatusBadRequest, "oauth_access_denied", message)
			return
		}

		state := strings.TrimSpace(c.Query("state"))
		if state == "" {
			writeError(c, http.StatusBadRequest, "missing_oauth_state", "state is required")
			return
		}
		code := strings.TrimSpace(c.Query("code"))
		if code == "" {
			writeError(c, http.StatusBadRequest, "missing_oauth_code", "code is required")
			return
		}

		result, err := manager.CompleteAuth(c.Request.Context(), state, code)
		if err != nil {
			if errors.Is(err, connectors.ErrOAuthStateInvalid) {
				writeError(c, http.StatusBadRequest, "invalid_oauth_state", "oauth state is invalid or expired")
				return
			}
			if errors.Is(err, connectors.ErrOAuthExchangeFailed) {
				writeError(c, http.StatusBadGateway, "oauth_exchange_failed", "oauth code exchange failed")
				return
			}
			if errors.Is(err, connectors.ErrOAuthUnavailable) {
				writeError(c, http.StatusServiceUnavailable, "connector_service_unavailable", "google docs oauth is not configured")
				return
			}
			writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		response := domain.ConnectorAuthCallbackResponse{
			Connector:     "google_docs",
			SessionKey:    result.SessionKey,
			Authenticated: true,
		}
		if result.ExpiresAt != nil {
			response.ExpiresAt = result.ExpiresAt.UTC().Format(time.RFC3339)
		}

		c.JSON(http.StatusOK, response)
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

	router.POST("/api/connectors/import", func(c *gin.Context) {
		if !authorizeConnectorRequest(c) {
			return
		}
		if !enforceConnectorRateLimit(c, connectorRateLimiter) {
			return
		}

		var req domain.ConnectorImportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_payload", "invalid connector import payload")
			return
		}
		if strings.TrimSpace(req.DocumentID) == "" {
			writeError(c, http.StatusBadRequest, "missing_document_id", "documentId is required")
			return
		}

		connector := connectors.NewConnectorFromEnv()
		if connector.Name() == "none" {
			writeError(c, http.StatusBadRequest, "connector_unavailable", "no connector is configured")
			return
		}

		document, err := connector.ImportDocument(c.Request.Context(), connectors.ImportRequest{
			DocumentID: req.DocumentID,
			SessionKey: connectorSessionKeyFromRequest(c),
		})
		if err != nil {
			if errors.Is(err, connectors.ErrUnauthorized) {
				writeError(c, http.StatusBadGateway, "connector_upstream_unauthorized", "connector upstream credentials are invalid")
				return
			}
			if errors.Is(err, connectors.ErrForbidden) {
				writeError(c, http.StatusForbidden, "connector_forbidden", "connector access is forbidden for this document")
				return
			}
			if errors.Is(err, connectors.ErrDocumentNotFound) {
				writeError(c, http.StatusNotFound, "connector_document_not_found", "connector document was not found")
				return
			}
			if errors.Is(err, connectors.ErrUnavailable) {
				writeError(c, http.StatusServiceUnavailable, "connector_service_unavailable", "connector service is unavailable")
				return
			}
			if errors.Is(err, connectors.ErrNotImplemented) {
				writeError(c, http.StatusNotImplemented, "connector_not_implemented", "connector import is not implemented yet")
				return
			}
			writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		c.JSON(http.StatusOK, domain.ConnectorImportResponse{
			Connector: connector.Name(),
			Document:  document,
		})
	})

	router.POST("/api/connectors/export", func(c *gin.Context) {
		if !authorizeConnectorRequest(c) {
			return
		}
		if !enforceConnectorRateLimit(c, connectorRateLimiter) {
			return
		}

		var req domain.ConnectorExportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_payload", "invalid connector export payload")
			return
		}
		if strings.TrimSpace(req.DocumentID) == "" {
			writeError(c, http.StatusBadRequest, "missing_document_id", "documentId is required")
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			writeError(c, http.StatusBadRequest, "missing_content", "content is required")
			return
		}

		connector := connectors.NewConnectorFromEnv()
		if connector.Name() == "none" {
			writeError(c, http.StatusBadRequest, "connector_unavailable", "no connector is configured")
			return
		}

		err := connector.ExportContent(c.Request.Context(), connectors.ExportRequest{
			DocumentID: req.DocumentID,
			Content:    req.Content,
			SessionKey: connectorSessionKeyFromRequest(c),
		})
		if err != nil {
			if errors.Is(err, connectors.ErrUnauthorized) {
				writeError(c, http.StatusBadGateway, "connector_upstream_unauthorized", "connector upstream credentials are invalid")
				return
			}
			if errors.Is(err, connectors.ErrForbidden) {
				writeError(c, http.StatusForbidden, "connector_forbidden", "connector access is forbidden for this document")
				return
			}
			if errors.Is(err, connectors.ErrDocumentNotFound) {
				writeError(c, http.StatusNotFound, "connector_document_not_found", "connector document was not found")
				return
			}
			if errors.Is(err, connectors.ErrUnavailable) {
				writeError(c, http.StatusServiceUnavailable, "connector_service_unavailable", "connector service is unavailable")
				return
			}
			if errors.Is(err, connectors.ErrNotImplemented) {
				writeError(c, http.StatusNotImplemented, "connector_not_implemented", "connector export is not implemented yet")
				return
			}
			writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		c.JSON(http.StatusOK, domain.ConnectorExportResponse{
			Connector: connector.Name(),
			Exported:  true,
		})
	})
}

func connectorSessionKeyFromRequest(c *gin.Context) string {
	return strings.TrimSpace(c.GetHeader("X-Connector-Session"))
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

func authorizeConnectorRequest(c *gin.Context) bool {
	requiredKey := strings.TrimSpace(os.Getenv("CONNECTOR_API_KEY"))
	if requiredKey == "" {
		return true
	}

	providedKey := strings.TrimSpace(c.GetHeader("X-Connector-Key"))
	if providedKey == "" {
		authorization := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
			providedKey = strings.TrimSpace(authorization[7:])
		}
	}

	if subtle.ConstantTimeCompare([]byte(requiredKey), []byte(providedKey)) == 1 {
		return true
	}

	writeError(c, http.StatusUnauthorized, "connector_unauthorized", "connector API key is invalid")
	return false
}
