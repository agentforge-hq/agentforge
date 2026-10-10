---
name: code-review
description: >-
  Use this skill when performing code review on AgentForge changes. Provides
  the complete review checklist derived from the project's coding standards,
  constitution, and engineering principles.
---

# Code Review Skill

Perform code review against the AgentForge quality bar.

## Review Checklist

### 1. Invariant Compliance
- [ ] Services own their data (no cross-service DB access)
- [ ] Published AgentVersions are immutable
- [ ] Domain logic doesn't depend on Spring infrastructure
- [ ] Controllers are thin adapters
- [ ] Persistence entities not exposed as API DTOs
- [ ] External calls have bounded timeouts
- [ ] Retries are bounded with explicit policy
- [ ] Kafka consumers are idempotent
- [ ] Redis is never the authoritative source of durable data
- [ ] No hard-coded secrets

### 2. Security & Tenant Isolation
- [ ] Server-side authorization on every route
- [ ] Tenant isolation enforced — never trust client-supplied tenant ID
- [ ] Tenant context propagated in async messages/traces
- [ ] Auth checked before business operations
- [ ] SSRF defenses on HTTP tools
- [ ] Database tools use allowlisted operations

### 3. Architecture Compliance
- [ ] Follows `API → Application → Domain → Infrastructure`
- [ ] Constructor injection (no field injection)
- [ ] DTOs at API boundaries
- [ ] Explicit transaction boundaries
- [ ] Immutable objects where practical
- [ ] Names express business intent (no generic Manager/Helper/Util)

### 4. Contract Compatibility
- [ ] REST response shape unchanged (or explicitly versioned)
- [ ] REST error semantics unchanged
- [ ] Event envelope unchanged
- [ ] Event payload unchanged
- [ ] Kafka topic semantics unchanged
- [ ] Public state machine values unchanged

### 5. Observability
- [ ] Structured logs present
- [ ] No secrets in logs
- [ ] Critical operations emit enough telemetry for reconstruction
- [ ] Correlation IDs propagated

### 6. Testing
- [ ] Unit tests for domain/application logic
- [ ] Integration tests for persistence/messaging
- [ ] API tests for endpoints
- [ ] Contract tests if contracts were touched
- [ ] Failure scenarios tested
- [ ] Tenant isolation tested
- [ ] Authorization tested

### 7. Documentation
- [ ] Contract documents updated if contracts changed
- [ ] Domain documents updated if behavior changed
- [ ] ADR created if architecture changed

## Severity Levels

| Level | Meaning | Action |
|---|---|---|
| 🔴 **Blocker** | Violates constitution invariant | Must fix before merge |
| 🟠 **Major** | Security gap, missing test, contract break | Must fix before merge |
| 🟡 **Minor** | Style issue, naming, missing observability | Should fix |
| 🔵 **Suggestion** | Alternative approach, optimization | Optional |

## Output Format

For each finding, report:

```
[SEVERITY] File:Line — Description
  Invariant: <which invariant is affected>
  Suggestion: <how to fix>
```
