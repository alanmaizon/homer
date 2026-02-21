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
Provider (mock or OpenAI)
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

## Environment
Copy `.env.example` values into your shell/session:
- `PORT` (default `8080`)
- `LLM_PROVIDER` (`mock` or `openai`)
- `OPENAI_API_KEY` (required when provider is `openai`)
- `OPENAI_MODEL` (default `gpt-4o-mini`)

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
