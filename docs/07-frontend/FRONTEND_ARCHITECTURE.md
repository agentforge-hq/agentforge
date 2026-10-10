---
document: frontend-architecture
version: 1.0
status: active
authority: architecture
depends_on:
  - REST_API
  - SECURITY_ARCHITECTURE
---

# Frontend Architecture

## Stack

React + TypeScript.

## Structure

```text
features/
components/
hooks/
services/
stores/
types/
```

Feature modules own feature-specific UI/state/API logic where practical.

## Screens

- Dashboard
- Agents
- Agent Builder
- Knowledge
- Documents
- Conversations
- Executions
- Execution Trace
- Monitoring
- Team
- API Keys
- Audit Logs

## Authentication

The frontend does not implement authorization as the security boundary. UI permission checks improve UX; backend authorization remains authoritative.

## API access

Centralize API client behavior for:

- authentication
- correlation ID
- error normalization
- retries where safe

## State

Use local state for local UI concerns and a server-state mechanism for remote resources. Avoid putting every API response into a global store.
