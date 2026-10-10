---
document: testing
version: 1.0
status: active
authority: engineering
depends_on:
  - CODING_STANDARDS
  - PROJECT_CONSTITUTION
---

# Testing

## Unit tests

Test domain rules and application orchestration without infrastructure.

Examples:

- AgentVersion immutability
- publication validation
- execution state transitions
- authorization policies
- retry policy

## Integration tests

Use Testcontainers for:

- PostgreSQL
- Redis
- Kafka

Test real persistence, messaging, serialization, and transaction boundaries.

## API tests

Verify:

- authentication
- authorization
- tenant isolation
- validation
- error contracts
- idempotency

## Contract tests

Protect REST and event contracts against incompatible changes.

## E2E

Critical journey:

```text
Create Agent
 -> Publish Agent
 -> Execute Agent
 -> Model Call
 -> Tool Call
 -> Response
 -> Complete Execution
```

## Failure tests

Explicitly test:

- model timeout
- tool timeout
- duplicate event
- consumer restart
- database transient failure
- Redis unavailable
- cancellation
- execution limit

## Test ownership

Each service owns its unit/integration/API tests. Cross-service critical journeys belong to platform-level E2E tests.
