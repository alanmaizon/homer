# Homer

Homer is a platform-agnostic, multi-agent text service written in Go.

CI status:
- GitHub Actions workflow: `.github/workflows/go-ci.yml` (runs `go vet` and `go test` on push/PR)

## What it does
- `summarize`: summarizes one or more documents
- `rewrite`: rewrites text in a selected mode
- Uses explicit agent orchestration: **Planner -> Executor -> Critic (optional)**
- Exposes HTTP API endpoints:
  - `GET /api/health`
  - `GET /api/capabilities`
  - `POST /api/connectors/import`
  - `POST /api/connectors/export`
  - `POST /api/task`

## Architecture
```text
Client (web/CLI/Docs/etc.)
        |
        v
POST /api/task
        |
        v
Planner -> Executor -> (Optional) Critic
        |
        v
Provider (mock, OpenAI, or Gemini)
```

## Connector integration
Connector implementations live in `backend/internal/connectors`:
- `Connector` interface defines import/export operations
- `NoopConnector` keeps core API independent from integrations
- `GoogleDocsConnector` performs Google Docs import/export through Google Docs API

Current status:
- Core `/api/task` flow does not require any connector and works independently
- Connector routes use structured errors when connector credentials are missing or invalid

## Request shape
`POST /api/task`

```json
{
  "task": "summarize",
  "documents": [{ "id": "d1", "title": "Doc", "content": "Text" }],
  "text": "",
  "mode": "professional",
  "instructions": "Focus on action items",
  "style": "paragraph",
  "enableCritic": false
}
```

Notes:
- `task` must be `summarize` or `rewrite`
- `documents` required for `summarize`
- `text` required for `rewrite`

## Error response
Validation and runtime errors return:

```json
{
  "error": {
    "code": "missing_text",
    "message": "text is required for rewrite",
    "requestId": "4e11fe43-e81c-40e8-b5cf-f9d4f0a65fe6"
  }
}
```

## API spec
- OpenAPI: `backend/openapi.yaml`

## Examples
Capabilities:

```bash
curl -sS http://localhost:8080/api/capabilities
```

Summarize:

```bash
curl -sS -X POST http://localhost:8080/api/task \
  -H "Content-Type: application/json" \
  -d '{
    "task":"summarize",
    "documents":[{"id":"d1","title":"Doc","content":"Launch is planned for Q1."}],
    "style":"paragraph",
    "instructions":"Focus on milestones"
  }'
```

Rewrite:

```bash
curl -sS -X POST http://localhost:8080/api/task \
  -H "Content-Type: application/json" \
  -d '{
    "task":"rewrite",
    "documents":[],
    "text":"We will utilize the system to optimize efficiency.",
    "mode":"simplify",
    "instructions":"Keep it short"
  }'
```

Connector import (requires `CONNECTOR_PROVIDER=google_docs` and credentials):

```bash
curl -sS -X POST http://localhost:8080/api/connectors/import \
  -H "Content-Type: application/json" \
  -H "X-Connector-Key: ${CONNECTOR_API_KEY}" \
  -d '{
    "documentId":"doc-123"
  }'
```

## Environment
Copy `.env.example` values into your shell/session:
- `PORT` (default `8080`)
- `LLM_PROVIDER` (`mock`, `openai`, or `gemini`)
- `LLM_TIMEOUT_MS` (outbound LLM call timeout in ms; default `15000`)
- `LLM_MAX_RETRIES` (bounded retry count per outbound LLM call; default `2`, max `5`)
- `OPENAI_API_KEY` (required when provider is `openai`)
- `OPENAI_MODEL` (default `gpt-4o-mini`)
- `GEMINI_API_KEY` or `GOOGLE_API_KEY` (required when provider is `gemini`)
- `GEMINI_MODEL` (default `gemini-2.5-flash`)
- `CONNECTOR_PROVIDER` (`none` or `google_docs`; default `none`)
- `CONNECTOR_API_KEY` (optional; when set, required for connector import/export routes)
- `CONNECTOR_RATE_LIMIT_PER_MINUTE` (connector route request cap per minute; default `60`, set `0` to disable)
- `GOOGLE_DOCS_ACCESS_TOKEN` (recommended for local dev connector calls)
- `GOOGLE_APPLICATION_CREDENTIALS` (alternative service account credentials file path)

Example Gemini setup:

```bash
export LLM_PROVIDER=gemini
export GEMINI_API_KEY=your_api_key
export GEMINI_MODEL=gemini-2.5-flash
```

## Run
```bash
cd backend
go run ./cmd/server
```

## Docker
Build:

```bash
docker build -f backend/Dockerfile -t homer-backend:local .
```

Run with compose:

```bash
docker compose up --build
```

## Test
```bash
cd backend
go test ./...
```

## Repo layout
```text
backend/
  cmd/server/main.go
  internal/api/
  internal/agents/
  internal/connectors/
  internal/domain/
  internal/llm/
  internal/middleware/
```
