---
name: test-writer
description: >-
  Use this skill when writing or updating tests for AgentForge services.
  Covers unit, integration, API, contract, and E2E test patterns specific
  to the AgentForge tech stack (Java 21, Spring Boot, Testcontainers).
---

# Test Writer Skill

Write tests that match the AgentForge testing strategy.

## Test Categories

### Unit Tests
- **What**: Domain rules, application orchestration, validation
- **How**: Plain JUnit 5 + Mockito, no Spring context
- **Examples**:
  - AgentVersion immutability enforcement
  - Publication validation rules
  - Execution state machine transitions
  - Authorization policy logic
  - Retry policy calculations

### Integration Tests
- **What**: Real persistence, messaging, serialization, transactions
- **How**: `@SpringBootTest` + Testcontainers
- **Containers**: PostgreSQL, Redis, Kafka
- **Verify**: Data is correctly persisted, events are produced/consumed, transactions roll back on failure

### API Tests
- **What**: HTTP endpoints
- **How**: `@WebMvcTest` or `@SpringBootTest` with `TestRestTemplate`/`WebTestClient`
- **Verify**: Authentication, authorization, tenant isolation, validation, error contracts (see `docs/04-contracts/ERROR_CODES.md`), idempotency

### Contract Tests
- **What**: REST response shapes, event payloads
- **How**: Schema validation / consumer-driven contract tests
- **Verify**: No breaking changes to `docs/04-contracts/REST_API.md` or `docs/04-contracts/EVENTS.md`

### E2E Tests
- **What**: Critical user journeys
- **Flow**: `Create Agent → Publish → Execute → Model Call → Tool Call → Response → Complete`
- **How**: Full stack with all containers running

### Failure Tests
- **What**: Resilience under failure
- **Scenarios**: Model timeout, tool timeout, duplicate event, consumer restart, DB transient failure, Redis unavailable, cancellation, execution limit

## Naming Convention

```
<ClassName>Test.java          — unit tests
<ClassName>IntegrationTest.java — integration tests
<Endpoint>ApiTest.java         — API tests
<Contract>ContractTest.java    — contract tests
```

## Test Structure

Use Arrange-Act-Assert (AAA) or Given-When-Then.

## Key Rules

1. Every behavior change needs tests — no exceptions.
2. Test tenant isolation: verify cross-tenant access is rejected.
3. Test authorization: verify unauthenticated/unauthorized requests are rejected.
4. Test idempotency for Kafka consumers.
5. Never test Spring framework internals — test your business behavior.
