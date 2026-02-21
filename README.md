# Homer

Homer is a platform-agnostic, multi-agent text service written in Go.

## What it does
- `summarize`: summarizes one or more documents
- `rewrite`: rewrites text in a selected mode
- Uses explicit agent orchestration: **Planner -> Executor -> Critic (optional)**
- Exposes HTTP API endpoints:
  - `GET /api/health`
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

## Environment
Copy `.env.example` values into your shell/session:
- `PORT` (default `8080`)
- `LLM_PROVIDER` (`mock`, `openai`, or `gemini`)
- `OPENAI_API_KEY` (required when provider is `openai`)
- `OPENAI_MODEL` (default `gpt-4o-mini`)
- `GEMINI_API_KEY` or `GOOGLE_API_KEY` (required when provider is `gemini`)
- `GEMINI_MODEL` (default `gemini-2.5-flash`)

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
  internal/domain/
  internal/llm/
  internal/middleware/
```
