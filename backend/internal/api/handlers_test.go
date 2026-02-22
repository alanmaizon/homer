package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alanmaizon/homer/backend/internal/llm"
	"github.com/alanmaizon/homer/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

type errorEnvelope struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	} `json:"error"`
}

type taskEnvelope struct {
	Result   string `json:"result"`
	Metadata struct {
		Provider  string `json:"provider"`
		RequestID string `json:"requestId"`
	} `json:"metadata"`
}

type capabilitiesEnvelope struct {
	Runtime struct {
		RequestedProvider  string `json:"requestedProvider"`
		ActiveProvider     string `json:"activeProvider"`
		ProviderFallback   bool   `json:"providerFallback"`
		RequestedConnector string `json:"requestedConnector"`
		ActiveConnector    string `json:"activeConnector"`
		ConnectorFallback  bool   `json:"connectorFallback"`
	} `json:"runtime"`
	Features struct {
		Critic          bool `json:"critic"`
		ConnectorImport bool `json:"connectorImport"`
		ConnectorExport bool `json:"connectorExport"`
	} `json:"features"`
}

func testRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID())
	RegisterRoutes(router)
	return router
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}
	if strings.TrimSpace(res.Body.String()) != "{\"ok\":true}" {
		t.Fatalf("unexpected body: %s", res.Body.String())
	}
}

func TestCapabilitiesDefault(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	t.Setenv("CONNECTOR_PROVIDER", "none")
	llm.SetProvider(llm.NewMockProvider())

	req := httptest.NewRequest(http.MethodGet, "/api/capabilities", nil)
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}

	var payload capabilitiesEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Runtime.RequestedProvider != "mock" || payload.Runtime.ActiveProvider != "mock" {
		t.Fatalf("unexpected provider runtime payload: %+v", payload.Runtime)
	}
	if payload.Runtime.ProviderFallback {
		t.Fatalf("expected provider fallback=false")
	}
	if payload.Runtime.RequestedConnector != "none" || payload.Runtime.ActiveConnector != "none" {
		t.Fatalf("unexpected connector runtime payload: %+v", payload.Runtime)
	}
	if payload.Runtime.ConnectorFallback {
		t.Fatalf("expected connector fallback=false")
	}
	if !payload.Features.Critic {
		t.Fatalf("expected critic feature enabled")
	}
	if payload.Features.ConnectorImport || payload.Features.ConnectorExport {
		t.Fatalf("expected connector import/export features disabled")
	}
}

func TestCapabilitiesFallbackVisibility(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("CONNECTOR_PROVIDER", "google_docs")
	t.Setenv("GOOGLE_DOCS_ACCESS_TOKEN", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	llm.SetProvider(llm.NewMockProvider())

	req := httptest.NewRequest(http.MethodGet, "/api/capabilities", nil)
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}

	var payload capabilitiesEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Runtime.RequestedProvider != "openai" || payload.Runtime.ActiveProvider != "mock" {
		t.Fatalf("unexpected provider runtime payload: %+v", payload.Runtime)
	}
	if !payload.Runtime.ProviderFallback {
		t.Fatalf("expected provider fallback=true")
	}
	if payload.Runtime.RequestedConnector != "google_docs" || payload.Runtime.ActiveConnector != "none" {
		t.Fatalf("unexpected connector runtime payload: %+v", payload.Runtime)
	}
	if !payload.Runtime.ConnectorFallback {
		t.Fatalf("expected connector fallback=true")
	}
}

func TestCapabilitiesConnectorEnabled(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	t.Setenv("CONNECTOR_PROVIDER", "google_docs")
	t.Setenv("GOOGLE_DOCS_ACCESS_TOKEN", "test-token")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	llm.SetProvider(llm.NewMockProvider())

	req := httptest.NewRequest(http.MethodGet, "/api/capabilities", nil)
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}

	var payload capabilitiesEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Runtime.ActiveConnector != "google_docs" {
		t.Fatalf("expected active connector google_docs, got %q", payload.Runtime.ActiveConnector)
	}
	if !payload.Features.ConnectorImport || !payload.Features.ConnectorExport {
		t.Fatalf("expected connector features enabled: %+v", payload.Features)
	}
}

func TestConnectorImportConnectorUnavailable(t *testing.T) {
	t.Setenv("CONNECTOR_PROVIDER", "none")
	req := httptest.NewRequest(http.MethodPost, "/api/connectors/import", strings.NewReader(`{"documentId":"doc-1"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}

	var payload errorEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error.Code != "connector_unavailable" {
		t.Fatalf("expected connector_unavailable, got %q", payload.Error.Code)
	}
}

func TestConnectorImportUnauthorizedWhenAPIKeyConfigured(t *testing.T) {
	t.Setenv("CONNECTOR_API_KEY", "secret")
	t.Setenv("CONNECTOR_PROVIDER", "none")

	req := httptest.NewRequest(http.MethodPost, "/api/connectors/import", strings.NewReader(`{"documentId":"doc-1"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", res.Code)
	}

	var payload errorEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error.Code != "connector_unauthorized" {
		t.Fatalf("expected connector_unauthorized, got %q", payload.Error.Code)
	}
}

func TestConnectorImportAuthorizedWithAPIKey(t *testing.T) {
	t.Setenv("CONNECTOR_API_KEY", "secret")
	t.Setenv("CONNECTOR_PROVIDER", "none")

	req := httptest.NewRequest(http.MethodPost, "/api/connectors/import", strings.NewReader(`{"documentId":"doc-1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Connector-Key", "secret")
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}

	var payload errorEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error.Code != "connector_unavailable" {
		t.Fatalf("expected connector_unavailable, got %q", payload.Error.Code)
	}
}

func TestConnectorImportCredentialsUnavailable(t *testing.T) {
	t.Setenv("CONNECTOR_PROVIDER", "google_docs")
	t.Setenv("GOOGLE_DOCS_ACCESS_TOKEN", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	req := httptest.NewRequest(http.MethodPost, "/api/connectors/import", strings.NewReader(`{"documentId":"doc-1"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}

	var payload errorEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error.Code != "connector_unavailable" {
		t.Fatalf("expected connector_unavailable, got %q", payload.Error.Code)
	}
}

func TestConnectorExportValidation(t *testing.T) {
	t.Setenv("CONNECTOR_PROVIDER", "google_docs")
	t.Setenv("GOOGLE_DOCS_ACCESS_TOKEN", "test-token")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	req := httptest.NewRequest(http.MethodPost, "/api/connectors/export", strings.NewReader(`{"documentId":"doc-1","content":" "}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}

	var payload errorEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error.Code != "missing_content" {
		t.Fatalf("expected missing_content, got %q", payload.Error.Code)
	}
}

func TestConnectorExportCredentialsUnavailable(t *testing.T) {
	t.Setenv("CONNECTOR_PROVIDER", "google_docs")
	t.Setenv("GOOGLE_DOCS_ACCESS_TOKEN", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	req := httptest.NewRequest(http.MethodPost, "/api/connectors/export", strings.NewReader(`{"documentId":"doc-1","content":"Updated content"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", res.Code)
	}

	var payload errorEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error.Code != "connector_unavailable" {
		t.Fatalf("expected connector_unavailable, got %q", payload.Error.Code)
	}
}

func TestConnectorExportUnauthorizedWhenAPIKeyConfigured(t *testing.T) {
	t.Setenv("CONNECTOR_API_KEY", "secret")
	t.Setenv("CONNECTOR_PROVIDER", "none")

	req := httptest.NewRequest(http.MethodPost, "/api/connectors/export", strings.NewReader(`{"documentId":"doc-1","content":"Updated content"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", res.Code)
	}

	var payload errorEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Error.Code != "connector_unauthorized" {
		t.Fatalf("expected connector_unauthorized, got %q", payload.Error.Code)
	}
}

func TestTaskValidationErrors(t *testing.T) {
	testCases := []struct {
		name       string
		body       string
		wantCode   string
		wantStatus int
	}{
		{
			name:       "invalid_json",
			body:       "{\"task\":",
			wantCode:   "invalid_payload",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing_task",
			body:       "{}",
			wantCode:   "missing_task",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unsupported_task",
			body:       "{\"task\":\"translate\"}",
			wantCode:   "unsupported_task",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing_documents",
			body:       "{\"task\":\"summarize\",\"documents\":[]}",
			wantCode:   "missing_documents",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing_text",
			body:       "{\"task\":\"rewrite\",\"text\":\"   \"}",
			wantCode:   "missing_text",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/task", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()

			testRouter().ServeHTTP(res, req)

			if res.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, res.Code)
			}

			var payload errorEnvelope
			if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if payload.Error.Code != tc.wantCode {
				t.Fatalf("expected code %q, got %q", tc.wantCode, payload.Error.Code)
			}
			if payload.Error.Message == "" {
				t.Fatalf("expected error message to be populated")
			}
			if payload.Error.RequestID == "" {
				t.Fatalf("expected requestId to be populated")
			}
			if gotHeader := res.Header().Get("X-Request-Id"); gotHeader == "" {
				t.Fatalf("expected X-Request-Id header to be set")
			}
		})
	}
}

func TestTaskSuccess(t *testing.T) {
	llm.SetProvider(llm.NewMockProvider())

	body := `{
		"task":"summarize",
		"documents":[{"id":"d1","title":"Doc","content":"Hello world"}],
		"style":"bullet"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/task", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}

	var payload taskEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.Contains(payload.Result, "[mock summary:bullet]") {
		t.Fatalf("unexpected result: %s", payload.Result)
	}
	if payload.Metadata.Provider != "mock" {
		t.Fatalf("expected provider mock, got %q", payload.Metadata.Provider)
	}
	if payload.Metadata.RequestID == "" {
		t.Fatalf("expected requestId in metadata")
	}
}
