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
  - `GET /metrics`
  - `GET /api/capabilities`
  - `GET /api/connectors/google_docs/auth/start`
  - `GET /api/connectors/google_docs/auth/callback`
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
- Google Docs connector supports OAuth authorization-code flow with in-memory session token storage

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

Connector routes may additionally return:
- `403 connector_forbidden`
- `404 connector_document_not_found`
- `429 connector_rate_limited`
- `502 connector_upstream_unauthorized`
- `503 connector_service_unavailable`

OAuth callback routes may additionally return:
- `400 oauth_access_denied`
- `400 invalid_oauth_state`
- `502 oauth_exchange_failed`

## API spec
- OpenAPI: `backend/openapi.yaml`

## Observability
- Prometheus-compatible metrics endpoint: `GET /metrics`
- Provider metrics:
  - `homer_provider_requests_total`
  - `homer_provider_request_duration_seconds`
- Connector metrics:
  - `homer_connector_requests_total`
  - `homer_connector_request_duration_seconds`
- Provider and connector operation logs include `request_id` for request correlation.

## Examples
Capabilities:

```bash
curl -sS http://localhost:8080/api/capabilities
```

Metrics:

```bash
curl -sS http://localhost:8080/metrics
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
  -H "X-Connector-Session: ${CONNECTOR_SESSION_KEY}" \
  -d '{
    "documentId":"doc-123"
  }'
```

Google Docs OAuth start:

```bash
curl -sS http://localhost:8080/api/connectors/google_docs/auth/start
```

Google Docs OAuth callback (state and code from provider redirect):

```bash
curl -sS "http://localhost:8080/api/connectors/google_docs/auth/callback?state=${STATE}&code=${CODE}"
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
- `GOOGLE_OAUTH_CLIENT_ID` (required for OAuth authorization-code flow)
- `GOOGLE_OAUTH_CLIENT_SECRET` (required for OAuth authorization-code flow)
- `GOOGLE_OAUTH_REDIRECT_URL` (required for OAuth authorization-code flow callback)
- `GOOGLE_OAUTH_SCOPES` (optional comma/space separated scopes; defaults to Google Docs scope)
- `GOOGLE_OAUTH_STATE_TTL` (optional OAuth state lifetime, default `10m`)

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

## CLI demo
Run the platform-agnostic CLI against a local or remote Homer backend:

```bash
cd backend
go run ./cmd/cli --base-url http://localhost:8080 health
```

Common commands:

```bash
# Runtime info
go run ./cmd/cli --base-url http://localhost:8080 capabilities

# Summarize
go run ./cmd/cli --base-url http://localhost:8080 summarize \
  -id doc-1 \
  -title "Meeting Notes" \
  -content "Launch is planned for Q1 and hiring starts next week." \
  -style bullet

# Rewrite
go run ./cmd/cli --base-url http://localhost:8080 rewrite \
  -text "We will utilize the platform to optimize efficiency." \
  -mode simplify

# Connector import/export (when connector routes are enabled)
go run ./cmd/cli --base-url http://localhost:8080 \
  --connector-key "$CONNECTOR_API_KEY" \
  --connector-session "$CONNECTOR_SESSION_KEY" \
  connector-import -document-id doc-123

go run ./cmd/cli --base-url http://localhost:8080 \
  --connector-key "$CONNECTOR_API_KEY" \
  --connector-session "$CONNECTOR_SESSION_KEY" \
  connector-export -document-id doc-123 -content "Updated content"
```

CLI environment variables:
- `HOMER_BASE_URL` (default API base URL for CLI)
- `HOMER_AUTH_TOKEN` (Bearer token for `Authorization`)
- `HOMER_CONNECTOR_KEY` (default `X-Connector-Key`)
- `HOMER_CONNECTOR_SESSION` (default `X-Connector-Session`)

## Docker
Build:

```bash
docker build -f backend/Dockerfile -t homer-backend:local .
```

Run with compose:

```bash
docker compose up --build
```

## Cloud Run deploy
Deployment script:
- `scripts/gcp/deploy_cloud_run.sh`

Runtime env template:
- `deploy/cloudrun.env.template`

IAM and API prerequisites:
1. Enable APIs: Cloud Run, Artifact Registry, IAM.
2. Deploy identity roles:
   - `roles/run.admin`
   - `roles/artifactregistry.writer`
   - `roles/iam.serviceAccountUser` (for the runtime service account)
3. Runtime service account should have access to any secrets/APIs required by your selected provider mode.

One-command deploy:

```bash
GCP_PROJECT_ID=my-project \
GCP_REGION=us-central1 \
CLOUD_RUN_SERVICE=homer-backend \
IMAGE_REPO=homer \
CLOUD_RUN_ENV_FILE=deploy/cloudrun.env.template \
./scripts/gcp/deploy_cloud_run.sh
```

Post-deploy smoke checks:

```bash
SERVICE_URL="$(gcloud run services describe homer-backend --project my-project --region us-central1 --format='value(status.url)')"
curl -fsS "${SERVICE_URL}/api/health"
curl -fsS "${SERVICE_URL}/api/capabilities"
```

Runtime config matrix:

| Mode | Required runtime env vars | Notes |
| --- | --- | --- |
| Mock only | `LLM_PROVIDER=mock`, `CONNECTOR_PROVIDER=none` | Fastest smoke-test mode |
| OpenAI no connector | `LLM_PROVIDER=openai`, `OPENAI_API_KEY`, `CONNECTOR_PROVIDER=none` | Set optional `OPENAI_MODEL` |
| Gemini no connector | `LLM_PROVIDER=gemini`, `GEMINI_API_KEY` (or `GOOGLE_API_KEY`), `CONNECTOR_PROVIDER=none` | Set optional `GEMINI_MODEL` |
| Google Docs via env token | `CONNECTOR_PROVIDER=google_docs`, `GOOGLE_DOCS_ACCESS_TOKEN` | Good for quick non-user OAuth testing |
| Google Docs via OAuth | `CONNECTOR_PROVIDER=google_docs`, `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, `GOOGLE_OAUTH_REDIRECT_URL` | Use `/api/connectors/google_docs/auth/start` and callback flow |

## Test
```bash
cd backend
go test ./...
```

## Repo layout
```text
backend/
  cmd/cli/main.go
  cmd/server/main.go
  internal/api/
  internal/agents/
  internal/cli/
  internal/connectors/
  internal/domain/
  internal/llm/
  internal/middleware/
deploy/
  cloudrun.env.template
scripts/gcp/
  deploy_cloud_run.sh
```
