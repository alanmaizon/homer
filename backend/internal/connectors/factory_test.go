package connectors

import "testing"

func TestNewConnectorFromEnvDefaultsToNoop(t *testing.T) {
	t.Setenv("CONNECTOR_PROVIDER", "")
	t.Setenv("GOOGLE_CLIENT_ID", "")

	connector := NewConnectorFromEnv()
	if connector.Name() != "none" {
		t.Fatalf("expected none connector, got %q", connector.Name())
	}
}

func TestNewConnectorFromEnvGoogleDocs(t *testing.T) {
	t.Setenv("CONNECTOR_PROVIDER", "google_docs")
	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")

	connector := NewConnectorFromEnv()
	if connector.Name() != "google_docs" {
		t.Fatalf("expected google_docs connector, got %q", connector.Name())
	}
}

func TestNewConnectorFromEnvGoogleDocsMissingClientIDFallsBackToNoop(t *testing.T) {
	t.Setenv("CONNECTOR_PROVIDER", "google_docs")
	t.Setenv("GOOGLE_CLIENT_ID", "")

	connector := NewConnectorFromEnv()
	if connector.Name() != "none" {
		t.Fatalf("expected fallback none connector, got %q", connector.Name())
	}
}
