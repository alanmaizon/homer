package connectors

import (
	"context"

	"github.com/alanmaizon/homer/backend/internal/domain"
)

type NoopConnector struct{}

func NewNoopConnector() *NoopConnector {
	return &NoopConnector{}
}

func (n *NoopConnector) Name() string {
	return "none"
}

func (n *NoopConnector) ImportDocument(_ context.Context, _ ImportRequest) (domain.Document, error) {
	return domain.Document{}, ErrNotImplemented
}

func (n *NoopConnector) ExportContent(_ context.Context, _ ExportRequest) error {
	return ErrNotImplemented
}
