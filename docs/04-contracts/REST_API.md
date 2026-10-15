---
document: rest-api
version: 1.0
status: active
authority: contract
depends_on:
  - PROJECT_CONSTITUTION
  - SECURITY_ARCHITECTURE
related:
  - ERROR_CODES
  - EVENTS
---

# REST API Contract

## Base path

`/api/v1`

## Conventions

- JSON request/response unless explicitly documented.
- Resource IDs are opaque UUIDs.
- ISO-8601 timestamps in UTC.
- Pagination uses cursor-based pagination for large collections.
- Filtering/sorting are explicit query parameters.
- Authorization is enforced server-side.
- `Idempotency-Key` is accepted for operations where duplicate submission is harmful.
- `X-Correlation-Id` may be supplied and is propagated; the server generates one if absent.

## Representative endpoints

### Authentication & Tenant

`POST /api/v1/auth/register`

Register a new user and create their initial organization (assigns `ORG_ADMIN`).

`POST /api/v1/auth/login`

Authenticate with email/password (no `tenantId` required; resolves active tenant to latest joined organization).

`POST /api/v1/auth/switch-tenant`

Switch active tenant context for authenticated user by issuing a new JWT scoped to the requested organization.

### Agents

`POST /api/v1/agents`

Create draft.

`GET /api/v1/agents/{agentId}`

Retrieve authorized agent.

`POST /api/v1/agents/{agentId}/versions`

Create draft version.

`POST /api/v1/agents/{agentId}/publish`

Validate and publish selected draft version.

### Execution

`POST /api/v1/agents/{agentId}/execute`

Create an execution against the selected published version.

`GET /api/v1/executions/{executionId}`

Retrieve execution state.

`GET /api/v1/executions/{executionId}/stream`

Stream authorized execution events using SSE.

`POST /api/v1/executions/{executionId}/cancel`

Request cancellation.

### Conversations

`POST /api/v1/conversations`

Create conversation.

`GET /api/v1/conversations/{conversationId}`

Retrieve conversation.

`POST /api/v1/conversations/{conversationId}/messages`

Append a user message.

## Error format

```json
{
  "code": "AGENT_NOT_FOUND",
  "message": "Agent was not found.",
  "correlationId": "uuid",
  "details": {}
}
```

Do not expose stack traces.

## HTTP status conventions

- 200 successful retrieval/update
- 201 creation
- 202 accepted asynchronous work
- 204 successful operation without body
- 400 validation error
- 401 unauthenticated
- 403 unauthorized
- 404 resource not visible/found
- 409 state/conflict
- 429 rate limited
- 500 unexpected server failure
- 502/503 dependency or service availability where appropriate

## Contract evolution

Prefer additive changes. Breaking changes require a new API version or explicitly accepted migration strategy.
