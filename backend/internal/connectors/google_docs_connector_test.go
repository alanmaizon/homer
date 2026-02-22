package connectors

import (
	"context"
	"errors"
	"testing"

	"github.com/alanmaizon/homer/backend/internal/domain"
	"google.golang.org/api/docs/v1"
)

type fakeGoogleDocsClient struct {
	document     *docs.Document
	getErr       error
	batchErr     error
	lastExportID string
	lastBatchReq *docs.BatchUpdateDocumentRequest
}

func (f *fakeGoogleDocsClient) GetDocument(_ context.Context, _ string) (*docs.Document, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.document, nil
}

func (f *fakeGoogleDocsClient) BatchUpdate(_ context.Context, documentID string, req *docs.BatchUpdateDocumentRequest) (*docs.BatchUpdateDocumentResponse, error) {
	if f.batchErr != nil {
		return nil, f.batchErr
	}
	f.lastExportID = documentID
	f.lastBatchReq = req
	return &docs.BatchUpdateDocumentResponse{}, nil
}

func TestGoogleDocsImportDocument(t *testing.T) {
	client := &fakeGoogleDocsClient{
		document: &docs.Document{
			Title: "Roadmap",
			Body: &docs.Body{
				Content: []*docs.StructuralElement{
					{
						Paragraph: &docs.Paragraph{
							Elements: []*docs.ParagraphElement{
								{TextRun: &docs.TextRun{Content: "Hello "}},
								{TextRun: &docs.TextRun{Content: "world\n"}},
							},
						},
					},
				},
			},
		},
	}

	connector := &GoogleDocsConnector{
		newClient: func(_ context.Context) (googleDocsClient, error) {
			return client, nil
		},
	}

	document, err := connector.ImportDocument(context.Background(), ImportRequest{DocumentID: "doc-1"})
	if err != nil {
		t.Fatalf("import returned error: %v", err)
	}

	expected := domain.Document{
		ID:      "doc-1",
		Title:   "Roadmap",
		Content: "Hello world",
	}
	if document != expected {
		t.Fatalf("unexpected document: %+v", document)
	}
}

func TestGoogleDocsImportUnavailable(t *testing.T) {
	connector := &GoogleDocsConnector{
		newClient: func(_ context.Context) (googleDocsClient, error) {
			return nil, ErrUnavailable
		},
	}

	_, err := connector.ImportDocument(context.Background(), ImportRequest{DocumentID: "doc-1"})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

func TestGoogleDocsExportContent(t *testing.T) {
	client := &fakeGoogleDocsClient{
		document: &docs.Document{
			Body: &docs.Body{
				Content: []*docs.StructuralElement{
					{EndIndex: 1},
					{EndIndex: 21},
				},
			},
		},
	}

	connector := &GoogleDocsConnector{
		newClient: func(_ context.Context) (googleDocsClient, error) {
			return client, nil
		},
	}

	err := connector.ExportContent(context.Background(), ExportRequest{
		DocumentID: "doc-42",
		Content:    "Updated content",
	})
	if err != nil {
		t.Fatalf("export returned error: %v", err)
	}

	if client.lastBatchReq == nil {
		t.Fatalf("expected batch request to be captured")
	}
	if client.lastExportID != "doc-42" {
		t.Fatalf("expected export id doc-42, got %q", client.lastExportID)
	}
	if len(client.lastBatchReq.Requests) != 2 {
		t.Fatalf("expected 2 batch requests, got %d", len(client.lastBatchReq.Requests))
	}

	deleteReq := client.lastBatchReq.Requests[0].DeleteContentRange
	if deleteReq == nil || deleteReq.Range == nil {
		t.Fatalf("expected first request to be delete content range")
	}
	if deleteReq.Range.StartIndex != 1 || deleteReq.Range.EndIndex != 20 {
		t.Fatalf("unexpected delete range: %+v", deleteReq.Range)
	}

	insertReq := client.lastBatchReq.Requests[1].InsertText
	if insertReq == nil || insertReq.Location == nil {
		t.Fatalf("expected second request to be insert text")
	}
	if insertReq.Location.Index != 1 || insertReq.Text != "Updated content" {
		t.Fatalf("unexpected insert request: %+v", insertReq)
	}
}

func TestGoogleDocsExportContentEmptyDoc(t *testing.T) {
	client := &fakeGoogleDocsClient{
		document: &docs.Document{
			Body: &docs.Body{},
		},
	}

	connector := &GoogleDocsConnector{
		newClient: func(_ context.Context) (googleDocsClient, error) {
			return client, nil
		},
	}

	err := connector.ExportContent(context.Background(), ExportRequest{
		DocumentID: "doc-100",
		Content:    "New content",
	})
	if err != nil {
		t.Fatalf("export returned error: %v", err)
	}

	if len(client.lastBatchReq.Requests) != 1 {
		t.Fatalf("expected 1 request for empty doc, got %d", len(client.lastBatchReq.Requests))
	}
	if client.lastBatchReq.Requests[0].InsertText == nil {
		t.Fatalf("expected insert request for empty doc")
	}
}
