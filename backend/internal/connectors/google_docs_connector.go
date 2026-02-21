package connectors

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/alanmaizon/homer/backend/internal/domain"
)

type GoogleDocsConnector struct {
	clientID string
}

func NewGoogleDocsConnectorFromEnv() (*GoogleDocsConnector, error) {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if clientID == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID is required")
	}

	return &GoogleDocsConnector{clientID: clientID}, nil
}

func (g *GoogleDocsConnector) Name() string {
	return "google_docs"
}

func (g *GoogleDocsConnector) ImportDocument(_ context.Context, req ImportRequest) (domain.Document, error) {
	_ = req
	// TODO: implement OAuth2 token handling and Google Docs API read flow.
	return domain.Document{}, ErrNotImplemented
}

func (g *GoogleDocsConnector) ExportContent(_ context.Context, req ExportRequest) error {
	_ = req
	// TODO: implement Google Docs API write/update flow.
	return ErrNotImplemented
}
