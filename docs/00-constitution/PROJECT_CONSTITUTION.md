---
document: project-constitution
version: 1.0
status: active
authority: constitution
depends_on: []
related:
  - ENGINEERING_PRINCIPLES
  - SERVICE_BOUNDARIES
---

# Project Constitution

## Architectural invariants

1. Services own their data.
2. No service directly accesses another service's database.
3. Tenant isolation is mandatory for every tenant-owned resource.
4. Authorization is enforced server-side.
5. Published agent versions are immutable.
6. Every execution references an exact immutable AgentVersion.
7. Domain logic does not depend directly on infrastructure.
8. Controllers remain thin.
9. Persistence entities are never exposed as public API contracts.
10. External calls have bounded timeouts.
11. Retries are bounded and use an explicit policy.
12. Kafka consumers are idempotent.
13. Redis is never the authoritative source of durable business truth.
14. Secrets are never committed to source control.
15. Public API and event contracts do not change silently.
16. New services require architectural justification.
17. AI-generated code is not a source of truth.
18. Observability is part of production behavior, not an optional add-on.
19. Security and tenant checks occur before protected business operations.
20. V1 tools cannot execute arbitrary code.

## Architecture style

The target architecture is service-oriented and event-driven where asynchronous processing provides a concrete benefit. The MVP may deploy multiple bounded contexts together where operational simplicity is preferable; logical ownership remains explicit.

## Change rule

Architectural changes require an ADR. A coding agent must not silently override an invariant.
