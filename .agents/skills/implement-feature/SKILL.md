---
name: implement-feature
description: >-
  Use this skill when implementing a new feature or modifying existing behavior
  in the AgentForge platform. Guides the agent through the full development
  workflow from requirement analysis to verified implementation.
---

# Feature Implementation Skill

This skill defines the end-to-end workflow for implementing features in
AgentForge. Follow these steps in order.

## Phase 1 — Understand

1. State the requested behavior in one paragraph.
2. Identify the affected bounded context(s) — activate the `context-loader` skill.
3. Identify the owning service(s) from `docs/02-architecture/SERVICE_BOUNDARIES.md`.
4. List affected data entities, APIs, and event contracts.
5. Check `docs/adr/` for relevant architectural decisions.

## Phase 2 — Classify

Determine the change type:

| Type | Criteria | Action |
|---|---|---|
| **Local** | Stays within one service boundary, no contract changes | Proceed to Phase 3 |
| **Contract** | Changes REST API shape, event payload, or error semantics | Requires versioning decision — confirm with user |
| **Architectural** | Changes service boundaries, data ownership, or communication patterns | Requires ADR — draft one before implementing |

## Phase 3 — Plan Tests

Before writing code, plan your tests:

- **Unit**: Domain rules, validation, state transitions
- **Integration**: Persistence, messaging, serialization (Testcontainers)
- **API**: Auth, tenant isolation, validation, error contracts
- **Contract**: REST/event compatibility
- **E2E**: Only for critical user journeys

## Phase 4 — Implement

Follow the layered architecture:

```
API (Controller) → Application (Service) → Domain (Entity/VO) → Infrastructure (Repository/Client)
```

### Checklist

- [ ] Constructor injection (no field injection)
- [ ] DTOs at API boundaries (no entity exposure)
- [ ] Explicit transaction boundaries
- [ ] Tenant context propagated
- [ ] Authorization checked server-side
- [ ] Timeouts on external calls
- [ ] Idempotent Kafka consumers
- [ ] Structured logging (no secrets)

## Phase 5 — Verify

1. Run the test suite.
2. Verify invariants from `PROJECT_CONSTITUTION.md`.
3. If contracts changed, update the contract document.
4. If behavior changed, update the relevant domain document.

## Reference

For detailed coding standards, see:
[CODING_STANDARDS.md](../../docs/08-engineering/CODING_STANDARDS.md)

For testing patterns, see:
[TESTING.md](../../docs/08-engineering/TESTING.md)
