package connectors

import "testing"

func TestNewConnectorFromEnvDefaultsToNoop(t *testing.T) {
	t.Setenv("CONNECTOR_PROVIDER", "")
	t.Setenv("GOOGLE_DOCS_ACCESS_TOKEN", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	connector := NewConnectorFromEnv()
	if connector.Name() != "none" {
		t.Fatalf("expected none connector, got %q", connector.Name())
	}
}

func TestNewConnectorFromEnvGoogleDocs(t *testing.T) {
	t.Setenv("CONNECTOR_PROVIDER", "google_docs")
	t.Setenv("GOOGLE_DOCS_ACCESS_TOKEN", "test-token")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	connector := NewConnectorFromEnv()
	if connector.Name() != "google_docs" {
		t.Fatalf("expected google_docs connector, got %q", connector.Name())
	}
}

func TestNewConnectorFromEnvGoogleDocsMissingCredentialsFallsBackToNoop(t *testing.T) {
	t.Setenv("CONNECTOR_PROVIDER", "google_docs")
	t.Setenv("GOOGLE_DOCS_ACCESS_TOKEN", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	connector := NewConnectorFromEnv()
	if connector.Name() != "none" {
		t.Fatalf("expected fallback none connector, got %q", connector.Name())
	}
}
