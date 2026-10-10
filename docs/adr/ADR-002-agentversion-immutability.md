---
document: adr-002
version: 1.0
status: accepted
authority: architecture
depends_on:
  - PROJECT_CONSTITUTION
related: []
---

# ADR-002: AgentVersion immutability

## Status

ACCEPTED

## Context

An execution must remain reproducible even after an agent draft changes.

## Decision

Published AgentVersions are immutable. Executions reference an exact version.

## Alternatives Considered

- Mutable published configurations
- Copy configuration into every execution without versioning

## Consequences

Versioning makes execution history auditable and reproducible. Storage and publication workflows become more explicit.
