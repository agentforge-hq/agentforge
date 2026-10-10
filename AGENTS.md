# AgentForge — AI Agent Instructions

## Mission

You are an implementation agent for the AgentForge platform. The source of truth is the `docs/` hierarchy — never invent architecture when an existing document answers the question.

## Context Loading Protocol

**Every task — read these first (in order):**

1. `docs/00-constitution/PROJECT_CONSTITUTION.md`
2. `docs/00-constitution/ENGINEERING_PRINCIPLES.md`
3. `docs/00-constitution/GLOSSARY.md`

Then identify the affected bounded context and load **only** the relevant slice:

| Context | Documents |
|---------|-----------|
| Agent | `docs/03-domains/AGENT.md`, `docs/03-domains/AGENT_RUNTIME.md`, `docs/02-architecture/SERVICE_BOUNDARIES.md`, `docs/04-contracts/REST_API.md` |
| Kafka / Events | `docs/02-architecture/COMMUNICATION.md`, `docs/04-contracts/EVENTS.md`, `docs/04-contracts/KAFKA_TOPICS.md` |
| Security / Identity | `docs/02-architecture/SECURITY_ARCHITECTURE.md`, `docs/03-domains/IDENTITY.md` |
| Knowledge / RAG | `docs/03-domains/KNOWLEDGE.md`, `docs/05-data/VECTOR_SEARCH.md`, `docs/02-architecture/DATA_OWNERSHIP.md` |
| Frontend | `docs/07-frontend/FRONTEND_ARCHITECTURE.md`, `docs/07-frontend/AGENT_BUILDER.md` |
| Infrastructure | `docs/06-infrastructure/DOCKER.md`, `docs/06-infrastructure/KUBERNETES.md`, `docs/06-infrastructure/OBSERVABILITY.md` |

Check `docs/adr/` for relevant ADRs before proposing architectural changes.

## Invariants (never violate)

- Services own their data — no cross-service DB access
- Tenant isolation is mandatory; never trust client-supplied tenant IDs
- Server-side authorization is mandatory on every route
- Published AgentVersions are immutable
- Kafka consumers must be idempotent
- Public API/event contracts never change silently
- No hard-coded secrets; no arbitrary SQL execution
- Domain code must not depend on Spring infrastructure directly

## Implementation Workflow

```
1. State the behavior
2. Identify bounded context + owning service
3. Identify affected data & contracts
4. Check ADRs
5. Determine local vs architectural change
6. Plan tests
7. Implement
8. Update docs if contracts/behavior changed
```

## Architecture

`API → Application → Domain → Infrastructure`

Controllers are thin adapters. Persistence entities ≠ API DTOs.

## Tech Stack

Java 21+ · Spring Boot · PostgreSQL · Redis (cache only) · Kafka · Testcontainers

## Testing Requirements

- Unit tests for domain/application logic
- Integration tests with Testcontainers (Postgres, Redis, Kafka)
- API/security tests for endpoints
- Contract tests for integration boundaries
- E2E for critical user journeys

## Code Review Checklist

Reviewers (human or AI) verify: invariants, authorization, tenant isolation, transaction boundaries, retries/timeouts, observability, tests, contract compatibility.

## Definition of Done

- Implementation matches source of truth
- Tests pass
- Security/tenant boundaries preserved
- Contracts preserved or explicitly versioned
- Observability adequate
- Documentation updated where required
