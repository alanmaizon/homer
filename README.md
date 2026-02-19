# Homer

## 1) Overview
Homer is a Microsoft Word AI task pane MVP built for the Agents League challenge. It delivers:
- Document summary
- Selection rewrite
- Multi-snippet summary input
- Foundry-inspired multi-agent orchestration

## 2) Why this is an Agent (not just an API call)
Homer is designed as explicit role-based agent collaboration:
- **Orchestrator Agent** receives user intent and context
- **Planner Agent** creates an execution plan
- **Executor Agent** performs plan steps through a provider abstraction
- **Critic Agent (optional)** validates/improves output when enabled

This means reasoning and execution are separated, making the system extensible beyond one-shot LLM calls.

## 3) Architecture Diagram (ASCII)
```text
User (Word Task Pane)
        ↓
Orchestrator Agent
        ↓
Planner Agent (task plan)
        ↓
Executor Agent (provider call)
        ↓
Optional Critic Agent (feature-flagged)
```

Designed following Microsoft Foundry multi-agent reasoning patterns: **Planner → Executor → Critic**.

## 4) Demo Flow (3-step)
1. User selects Summarize Document or Rewrite Selection in the task pane.
2. Backend `/api/task` orchestrates planner → executor (+ optional critic).
3. Output returns to the pane for preview, then Insert/Replace writes it back to Word.

## 5) Privacy-by-design
- Request IDs added for traceability.
- Document text is not logged in backend handlers.
- Provider selection and API keys are environment-variable based.
- Critic is optional and disabled by default.
- TODO: enterprise provider controls (tenant isolation, DLP checks, encryption-at-rest policy integration).

## 6) Roadmap
- Microsoft Graph multi-document ingestion
- Foundry SDK integration
- Copilot Studio deployment
- Citation validation agent

## Repository Structure
```text
/
  package.json
  pnpm-workspace.yaml
  README.md
  .env.example
  apps/
    addin-word/
    backend/
  packages/
    shared/
```

## Scripts
```bash
pnpm install
pnpm dev
pnpm build
pnpm typecheck
```
