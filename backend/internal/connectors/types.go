package connectors

import (
	"context"
	"errors"

	"github.com/alanmaizon/homer/backend/internal/domain"
)

var ErrNotImplemented = errors.New("connector operation not implemented")
var ErrUnavailable = errors.New("connector unavailable")
var ErrUnauthorized = errors.New("connector unauthorized")
var ErrForbidden = errors.New("connector forbidden")
var ErrDocumentNotFound = errors.New("connector document not found")

type ImportRequest struct {
	DocumentID string
}

type ExportRequest struct {
	DocumentID string
	Content    string
}

// Connector abstracts document source/destination integrations.
type Connector interface {
	Name() string
	ImportDocument(ctx context.Context, req ImportRequest) (domain.Document, error)
	ExportContent(ctx context.Context, req ExportRequest) error
}
