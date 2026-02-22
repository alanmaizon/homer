package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alanmaizon/homer/backend/internal/connectors"
	"github.com/alanmaizon/homer/backend/internal/domain"
)

type stubConnector struct {
	name      string
	importDoc domain.Document
	importErr error
	exportErr error
}

func (s *stubConnector) Name() string {
	if s.name == "" {
		return "none"
	}
	return s.name
}

func (s *stubConnector) ImportDocument(_ context.Context, _ connectors.ImportRequest) (domain.Document, error) {
	if s.importErr != nil {
		return domain.Document{}, s.importErr
	}
	if s.importDoc.ID == "" {
		return domain.Document{
			ID:      "doc-1",
			Title:   "Doc 1",
			Content: "Content",
		}, nil
	}
	return s.importDoc, nil
}

func (s *stubConnector) ExportContent(_ context.Context, _ connectors.ExportRequest) error {
	return s.exportErr
}

func setConnectorFactoryForTest(t *testing.T, connector connectors.Connector) {
	t.Helper()

	previous := newConnectorFromEnv
	newConnectorFromEnv = func() connectors.Connector {
		return connector
	}
	t.Cleanup(func() {
		newConnectorFromEnv = previous
	})
}

func TestConnectorImportErrorStatusMappingsIntegration(t *testing.T) {
	testCases := []struct {
		name       string
		importErr  error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "forbidden",
			importErr:  connectors.ErrForbidden,
			wantStatus: http.StatusForbidden,
			wantCode:   "connector_forbidden",
		},
		{
			name:       "not_found",
			importErr:  connectors.ErrDocumentNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "connector_document_not_found",
		},
		{
			name:       "upstream_unauthorized",
			importErr:  connectors.ErrUnauthorized,
			wantStatus: http.StatusBadGateway,
			wantCode:   "connector_upstream_unauthorized",
		},
		{
			name:       "service_unavailable",
			importErr:  connectors.ErrUnavailable,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "connector_service_unavailable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setConnectorFactoryForTest(t, &stubConnector{
				name:      "google_docs",
				importErr: tc.importErr,
			})

			req := httptest.NewRequest(http.MethodPost, "/api/connectors/import", strings.NewReader(`{"documentId":"doc-1"}`))
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
		})
	}
}

func TestConnectorExportErrorStatusMappingsIntegration(t *testing.T) {
	testCases := []struct {
		name       string
		exportErr  error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "forbidden",
			exportErr:  connectors.ErrForbidden,
			wantStatus: http.StatusForbidden,
			wantCode:   "connector_forbidden",
		},
		{
			name:       "not_found",
			exportErr:  connectors.ErrDocumentNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "connector_document_not_found",
		},
		{
			name:       "upstream_unauthorized",
			exportErr:  connectors.ErrUnauthorized,
			wantStatus: http.StatusBadGateway,
			wantCode:   "connector_upstream_unauthorized",
		},
		{
			name:       "service_unavailable",
			exportErr:  connectors.ErrUnavailable,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "connector_service_unavailable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setConnectorFactoryForTest(t, &stubConnector{
				name:      "google_docs",
				exportErr: tc.exportErr,
			})

			req := httptest.NewRequest(http.MethodPost, "/api/connectors/export", strings.NewReader(`{"documentId":"doc-1","content":"updated"}`))
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
		})
	}
}

func TestConnectorAuthAndRateLimitInteractionIntegration(t *testing.T) {
	t.Setenv("CONNECTOR_API_KEY", "secret")
	t.Setenv("CONNECTOR_RATE_LIMIT_PER_MINUTE", "1")
	setConnectorFactoryForTest(t, &stubConnector{name: "none"})
	router := testRouter()

	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/api/connectors/import", strings.NewReader(`{"documentId":"doc-1"}`))
	unauthorizedReq.Header.Set("Content-Type", "application/json")
	unauthorizedRes := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedRes, unauthorizedReq)
	if unauthorizedRes.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status 401, got %d", unauthorizedRes.Code)
	}

	firstAuthorizedReq := httptest.NewRequest(http.MethodPost, "/api/connectors/import", strings.NewReader(`{"documentId":"doc-1"}`))
	firstAuthorizedReq.Header.Set("Content-Type", "application/json")
	firstAuthorizedReq.Header.Set("X-Connector-Key", "secret")
	firstAuthorizedRes := httptest.NewRecorder()
	router.ServeHTTP(firstAuthorizedRes, firstAuthorizedReq)
	if firstAuthorizedRes.Code != http.StatusBadRequest {
		t.Fatalf("expected first authorized status 400, got %d", firstAuthorizedRes.Code)
	}

	secondAuthorizedReq := httptest.NewRequest(http.MethodPost, "/api/connectors/import", strings.NewReader(`{"documentId":"doc-1"}`))
	secondAuthorizedReq.Header.Set("Content-Type", "application/json")
	secondAuthorizedReq.Header.Set("X-Connector-Key", "secret")
	secondAuthorizedRes := httptest.NewRecorder()
	router.ServeHTTP(secondAuthorizedRes, secondAuthorizedReq)
	if secondAuthorizedRes.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second authorized status 429, got %d", secondAuthorizedRes.Code)
	}
}

func TestCapabilitiesReflectConnectorFeatureFlagsIntegration(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "mock")
	t.Setenv("CONNECTOR_PROVIDER", "google_docs")
	setConnectorFactoryForTest(t, &stubConnector{name: "google_docs"})

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
