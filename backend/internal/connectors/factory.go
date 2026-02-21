package connectors

import "os"

func NewConnectorFromEnv() Connector {
	if os.Getenv("CONNECTOR_PROVIDER") == "google_docs" {
		if connector, err := NewGoogleDocsConnectorFromEnv(); err == nil {
			return connector
		}
	}

	return NewNoopConnector()
}
