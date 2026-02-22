package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunHealthSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/health" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var stdout strings.Builder
	var stderr strings.Builder

	exitCode := Run([]string{"-base-url", server.URL, "health"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d stdout=%s stderr=%s", exitCode, stdout.String(), stderr.String())
	}

	if !strings.Contains(stdout.String(), `"ok": true`) {
		t.Fatalf("expected health response in output, got %s", stdout.String())
	}
}

func TestRunRewriteMissingText(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	exitCode := Run([]string{"rewrite"}, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}

	if !strings.Contains(stdout.String(), `"code": "missing_text"`) {
		t.Fatalf("expected missing_text error, got %s", stdout.String())
	}
}

func TestRunConnectorImportAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/connectors/import" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"connector_unauthorized","message":"connector API key is invalid","requestId":"req-1"}}`))
	}))
	defer server.Close()

	var stdout strings.Builder
	var stderr strings.Builder

	exitCode := Run([]string{
		"-base-url", server.URL,
		"connector-import",
		"-document-id", "doc-1",
	}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), `"code": "connector_unauthorized"`) {
		t.Fatalf("expected connector_unauthorized error, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"status": 401`) {
		t.Fatalf("expected status 401 in error output, got %s", stdout.String())
	}
}

func TestRunConnectorImportSendsHeaders(t *testing.T) {
	var seenAuthorization string
	var seenConnectorKey string
	var seenConnectorSession string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuthorization = r.Header.Get("Authorization")
		seenConnectorKey = r.Header.Get("X-Connector-Key")
		seenConnectorSession = r.Header.Get("X-Connector-Session")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"connector":"google_docs","document":{"id":"doc-1","title":"Doc","content":"Text"}}`))
	}))
	defer server.Close()

	var stdout strings.Builder
	var stderr strings.Builder

	exitCode := Run([]string{
		"-base-url", server.URL,
		"-auth-token", "token-1",
		"-connector-key", "key-1",
		"-connector-session", "session-1",
		"connector-import",
		"-document-id", "doc-1",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d stdout=%s stderr=%s", exitCode, stdout.String(), stderr.String())
	}

	if seenAuthorization != "Bearer token-1" {
		t.Fatalf("expected authorization header, got %q", seenAuthorization)
	}
	if seenConnectorKey != "key-1" {
		t.Fatalf("expected connector key header, got %q", seenConnectorKey)
	}
	if seenConnectorSession != "session-1" {
		t.Fatalf("expected connector session header, got %q", seenConnectorSession)
	}

	var output map[string]any
	if err := json.Unmarshal([]byte(stdout.String()), &output); err != nil {
		t.Fatalf("expected json output, got error: %v", err)
	}
}
