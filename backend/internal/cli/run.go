package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type apiClient struct {
	baseURL          string
	authToken        string
	connectorKey     string
	connectorSession string
	httpClient       *http.Client
}

type apiError struct {
	Status    int
	Code      string
	Message   string
	RequestID string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("api error status=%d code=%s message=%s", e.Status, e.Code, e.Message)
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	root := flag.NewFlagSet("homer", flag.ContinueOnError)
	root.SetOutput(stderr)

	defaultBaseURL := envOrDefault("HOMER_BASE_URL", "http://localhost:8080")
	defaultAuthToken := strings.TrimSpace(os.Getenv("HOMER_AUTH_TOKEN"))
	defaultConnectorKey := strings.TrimSpace(os.Getenv("HOMER_CONNECTOR_KEY"))
	defaultConnectorSession := strings.TrimSpace(os.Getenv("HOMER_CONNECTOR_SESSION"))

	baseURL := root.String("base-url", defaultBaseURL, "Homer API base URL")
	authToken := root.String("auth-token", defaultAuthToken, "Bearer token for Authorization header")
	connectorKey := root.String("connector-key", defaultConnectorKey, "Connector API key for X-Connector-Key header")
	connectorSession := root.String("connector-session", defaultConnectorSession, "Connector session key for X-Connector-Session header")
	timeout := root.Duration("timeout", 15*time.Second, "HTTP timeout, e.g. 15s")

	if err := root.Parse(args); err != nil {
		writeCLIError(stdout, "invalid_arguments", err.Error(), 0)
		return 2
	}

	remaining := root.Args()
	if len(remaining) == 0 {
		writeCLIError(stdout, "missing_command", usageText(), 0)
		return 2
	}

	client := &apiClient{
		baseURL:          strings.TrimRight(strings.TrimSpace(*baseURL), "/"),
		authToken:        strings.TrimSpace(*authToken),
		connectorKey:     strings.TrimSpace(*connectorKey),
		connectorSession: strings.TrimSpace(*connectorSession),
		httpClient: &http.Client{
			Timeout: *timeout,
		},
	}

	command := remaining[0]
	commandArgs := remaining[1:]

	switch command {
	case "health":
		return runRequest(context.Background(), client, stdout, http.MethodGet, "/api/health", nil)
	case "capabilities":
		return runRequest(context.Background(), client, stdout, http.MethodGet, "/api/capabilities", nil)
	case "summarize":
		return runSummarize(context.Background(), client, stdout, stderr, commandArgs)
	case "rewrite":
		return runRewrite(context.Background(), client, stdout, stderr, commandArgs)
	case "connector-import":
		return runConnectorImport(context.Background(), client, stdout, stderr, commandArgs)
	case "connector-export":
		return runConnectorExport(context.Background(), client, stdout, stderr, commandArgs)
	default:
		writeCLIError(stdout, "unknown_command", fmt.Sprintf("unknown command %q\n%s", command, usageText()), 0)
		return 2
	}
}

func runSummarize(ctx context.Context, client *apiClient, stdout io.Writer, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("summarize", flag.ContinueOnError)
	fs.SetOutput(stderr)

	documentID := fs.String("id", "doc-1", "Document ID")
	title := fs.String("title", "Document", "Document title")
	content := fs.String("content", "", "Document content")
	style := fs.String("style", "paragraph", "Summary style")
	instructions := fs.String("instructions", "", "Additional instructions")
	enableCritic := fs.Bool("critic", false, "Enable critic pass")

	if err := fs.Parse(args); err != nil {
		writeCLIError(stdout, "invalid_arguments", err.Error(), 0)
		return 2
	}

	if strings.TrimSpace(*content) == "" {
		writeCLIError(stdout, "missing_content", "summarize requires -content", 0)
		return 2
	}

	payload := map[string]any{
		"task": "summarize",
		"documents": []map[string]string{
			{
				"id":      strings.TrimSpace(*documentID),
				"title":   strings.TrimSpace(*title),
				"content": strings.TrimSpace(*content),
			},
		},
		"style":        strings.TrimSpace(*style),
		"instructions": strings.TrimSpace(*instructions),
		"enableCritic": *enableCritic,
	}

	return runRequest(ctx, client, stdout, http.MethodPost, "/api/task", payload)
}

func runRewrite(ctx context.Context, client *apiClient, stdout io.Writer, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("rewrite", flag.ContinueOnError)
	fs.SetOutput(stderr)

	text := fs.String("text", "", "Input text")
	mode := fs.String("mode", "simplify", "Rewrite mode")
	instructions := fs.String("instructions", "", "Additional instructions")
	enableCritic := fs.Bool("critic", false, "Enable critic pass")

	if err := fs.Parse(args); err != nil {
		writeCLIError(stdout, "invalid_arguments", err.Error(), 0)
		return 2
	}

	if strings.TrimSpace(*text) == "" {
		writeCLIError(stdout, "missing_text", "rewrite requires -text", 0)
		return 2
	}

	payload := map[string]any{
		"task":         "rewrite",
		"text":         strings.TrimSpace(*text),
		"mode":         strings.TrimSpace(*mode),
		"instructions": strings.TrimSpace(*instructions),
		"documents":    []any{},
		"enableCritic": *enableCritic,
	}

	return runRequest(ctx, client, stdout, http.MethodPost, "/api/task", payload)
}

func runConnectorImport(ctx context.Context, client *apiClient, stdout io.Writer, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("connector-import", flag.ContinueOnError)
	fs.SetOutput(stderr)

	documentID := fs.String("document-id", "", "Connector document ID")
	if err := fs.Parse(args); err != nil {
		writeCLIError(stdout, "invalid_arguments", err.Error(), 0)
		return 2
	}
	if strings.TrimSpace(*documentID) == "" {
		writeCLIError(stdout, "missing_document_id", "connector-import requires -document-id", 0)
		return 2
	}

	payload := map[string]string{
		"documentId": strings.TrimSpace(*documentID),
	}
	return runRequest(ctx, client, stdout, http.MethodPost, "/api/connectors/import", payload)
}

func runConnectorExport(ctx context.Context, client *apiClient, stdout io.Writer, stderr io.Writer, args []string) int {
	fs := flag.NewFlagSet("connector-export", flag.ContinueOnError)
	fs.SetOutput(stderr)

	documentID := fs.String("document-id", "", "Connector document ID")
	content := fs.String("content", "", "Content to export")
	if err := fs.Parse(args); err != nil {
		writeCLIError(stdout, "invalid_arguments", err.Error(), 0)
		return 2
	}
	if strings.TrimSpace(*documentID) == "" {
		writeCLIError(stdout, "missing_document_id", "connector-export requires -document-id", 0)
		return 2
	}
	if strings.TrimSpace(*content) == "" {
		writeCLIError(stdout, "missing_content", "connector-export requires -content", 0)
		return 2
	}

	payload := map[string]string{
		"documentId": strings.TrimSpace(*documentID),
		"content":    strings.TrimSpace(*content),
	}
	return runRequest(ctx, client, stdout, http.MethodPost, "/api/connectors/export", payload)
}

func runRequest(ctx context.Context, client *apiClient, stdout io.Writer, method string, path string, payload any) int {
	responseBody, err := client.request(ctx, method, path, payload)
	if err != nil {
		var apiErr *apiError
		if errors.As(err, &apiErr) {
			writeCLIError(stdout, apiErr.Code, apiErr.Message, apiErr.Status)
			return 1
		}
		writeCLIError(stdout, "request_failed", err.Error(), 0)
		return 1
	}

	if err := writeStructuredJSON(stdout, responseBody); err != nil {
		writeCLIError(stdout, "invalid_response", err.Error(), 0)
		return 1
	}
	return 0
}

func (c *apiClient) request(ctx context.Context, method string, path string, payload any) ([]byte, error) {
	requestURL, err := c.resolveURL(path)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, err
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	if c.connectorKey != "" {
		req.Header.Set("X-Connector-Key", c.connectorKey)
	}
	if c.connectorSession != "" {
		req.Header.Set("X-Connector-Session", c.connectorSession)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 {
		apiErr := &apiError{
			Status:  res.StatusCode,
			Code:    "http_error",
			Message: strings.TrimSpace(string(responseBody)),
		}

		var envelope struct {
			Error struct {
				Code      string `json:"code"`
				Message   string `json:"message"`
				RequestID string `json:"requestId"`
			} `json:"error"`
		}

		if err := json.Unmarshal(responseBody, &envelope); err == nil && envelope.Error.Code != "" {
			apiErr.Code = envelope.Error.Code
			apiErr.Message = envelope.Error.Message
			apiErr.RequestID = envelope.Error.RequestID
		}

		return nil, apiErr
	}

	return responseBody, nil
}

func (c *apiClient) resolveURL(path string) (string, error) {
	base := strings.TrimSpace(c.baseURL)
	if base == "" {
		return "", errors.New("base URL is required")
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	pathURL, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return baseURL.ResolveReference(pathURL).String(), nil
}

func writeStructuredJSON(output io.Writer, body []byte) error {
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func writeCLIError(output io.Writer, code string, message string, status int) {
	payload := map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	if status > 0 {
		payload["error"].(map[string]any)["status"] = status
	}

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

func usageText() string {
	return strings.Join([]string{
		"usage: homer [global flags] <command> [command flags]",
		"commands: health, capabilities, summarize, rewrite, connector-import, connector-export",
		"global flags: -base-url -auth-token -connector-key -connector-session -timeout",
	}, "\n")
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
