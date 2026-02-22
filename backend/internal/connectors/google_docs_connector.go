package connectors

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alanmaizon/homer/backend/internal/domain"
	"golang.org/x/oauth2"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type GoogleDocsConnector struct {
	newClient func(ctx context.Context) (googleDocsClient, error)
}

type googleDocsClient interface {
	GetDocument(ctx context.Context, documentID string) (*docs.Document, error)
	BatchUpdate(ctx context.Context, documentID string, req *docs.BatchUpdateDocumentRequest) (*docs.BatchUpdateDocumentResponse, error)
}

type googleDocsAPIClient struct {
	service *docs.Service
}

func (c *googleDocsAPIClient) GetDocument(ctx context.Context, documentID string) (*docs.Document, error) {
	return c.service.Documents.Get(documentID).Context(ctx).Do()
}

func (c *googleDocsAPIClient) BatchUpdate(ctx context.Context, documentID string, req *docs.BatchUpdateDocumentRequest) (*docs.BatchUpdateDocumentResponse, error) {
	return c.service.Documents.BatchUpdate(documentID, req).Context(ctx).Do()
}

func NewGoogleDocsConnectorFromEnv() (*GoogleDocsConnector, error) {
	if !hasGoogleDocsCredentials() {
		return nil, errors.New("GOOGLE_DOCS_ACCESS_TOKEN or GOOGLE_APPLICATION_CREDENTIALS is required")
	}

	return &GoogleDocsConnector{
		newClient: newGoogleDocsClientFromEnv,
	}, nil
}

func (g *GoogleDocsConnector) Name() string {
	return "google_docs"
}

func (g *GoogleDocsConnector) ImportDocument(ctx context.Context, req ImportRequest) (domain.Document, error) {
	client, err := g.newClient(ctx)
	if err != nil {
		return domain.Document{}, err
	}

	document, err := client.GetDocument(ctx, req.DocumentID)
	if err != nil {
		return domain.Document{}, mapGoogleDocsError(err)
	}

	title := strings.TrimSpace(document.Title)
	if title == "" {
		title = req.DocumentID
	}

	return domain.Document{
		ID:      req.DocumentID,
		Title:   title,
		Content: extractDocumentText(document),
	}, nil
}

func (g *GoogleDocsConnector) ExportContent(ctx context.Context, req ExportRequest) error {
	client, err := g.newClient(ctx)
	if err != nil {
		return err
	}

	document, err := client.GetDocument(ctx, req.DocumentID)
	if err != nil {
		return mapGoogleDocsError(err)
	}

	startIndex, endIndex := editableDocumentRange(document)
	requests := make([]*docs.Request, 0, 2)

	if endIndex > startIndex {
		requests = append(requests, &docs.Request{
			DeleteContentRange: &docs.DeleteContentRangeRequest{
				Range: &docs.Range{
					StartIndex: startIndex,
					EndIndex:   endIndex,
				},
			},
		})
	}

	requests = append(requests, &docs.Request{
		InsertText: &docs.InsertTextRequest{
			Location: &docs.Location{
				Index: startIndex,
			},
			Text: req.Content,
		},
	})

	_, err = client.BatchUpdate(ctx, req.DocumentID, &docs.BatchUpdateDocumentRequest{
		Requests: requests,
	})
	return mapGoogleDocsError(err)
}

func newGoogleDocsClientFromEnv(ctx context.Context) (googleDocsClient, error) {
	token := strings.TrimSpace(os.Getenv("GOOGLE_DOCS_ACCESS_TOKEN"))
	credentialsFile := strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	if token == "" && credentialsFile == "" {
		return nil, fmt.Errorf("%w: set GOOGLE_DOCS_ACCESS_TOKEN or GOOGLE_APPLICATION_CREDENTIALS", ErrUnavailable)
	}

	opts := []option.ClientOption{option.WithScopes(docs.DocumentsScope)}

	switch {
	case token != "":
		opts = append(opts, option.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
		})))
	case credentialsFile != "":
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}

	service, err := docs.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	return &googleDocsAPIClient{service: service}, nil
}

func hasGoogleDocsCredentials() bool {
	return strings.TrimSpace(os.Getenv("GOOGLE_DOCS_ACCESS_TOKEN")) != "" ||
		strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")) != ""
}

func extractDocumentText(document *docs.Document) string {
	if document == nil || document.Body == nil {
		return ""
	}

	var builder strings.Builder
	appendStructuralElements(&builder, document.Body.Content)
	return strings.TrimSpace(builder.String())
}

func appendStructuralElements(builder *strings.Builder, elements []*docs.StructuralElement) {
	for _, element := range elements {
		if element == nil {
			continue
		}

		if paragraph := element.Paragraph; paragraph != nil {
			for _, paragraphElement := range paragraph.Elements {
				if paragraphElement == nil || paragraphElement.TextRun == nil {
					continue
				}
				builder.WriteString(paragraphElement.TextRun.Content)
			}
		}

		if table := element.Table; table != nil {
			for _, row := range table.TableRows {
				if row == nil {
					continue
				}
				for _, cell := range row.TableCells {
					if cell == nil {
						continue
					}
					appendStructuralElements(builder, cell.Content)
				}
			}
		}

		if toc := element.TableOfContents; toc != nil {
			appendStructuralElements(builder, toc.Content)
		}
	}
}

func editableDocumentRange(document *docs.Document) (startIndex int64, endIndex int64) {
	startIndex = 1
	endIndex = 1

	if document == nil || document.Body == nil {
		return startIndex, endIndex
	}

	for _, element := range document.Body.Content {
		if element == nil {
			continue
		}
		if element.EndIndex > endIndex {
			endIndex = element.EndIndex
		}
	}

	// Keep the trailing newline sentinel in the doc.
	if endIndex > 1 {
		endIndex--
	}
	if endIndex < startIndex {
		endIndex = startIndex
	}

	return startIndex, endIndex
}

func mapGoogleDocsError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case 401:
			return fmt.Errorf("%w: %v", ErrUnauthorized, err)
		case 403:
			return fmt.Errorf("%w: %v", ErrForbidden, err)
		case 404:
			return fmt.Errorf("%w: %v", ErrDocumentNotFound, err)
		case 429:
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		default:
			if apiErr.Code >= 500 {
				return fmt.Errorf("%w: %v", ErrUnavailable, err)
			}
		}
	}

	return err
}
