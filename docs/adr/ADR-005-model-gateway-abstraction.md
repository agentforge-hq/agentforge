---
document: adr-005
version: 1.0
status: accepted
authority: architecture
depends_on:
  - PROJECT_CONSTITUTION
related: []
---

# ADR-005: Model Gateway abstraction

## Status

ACCEPTED

## Context

Provider-specific model SDKs change and Agent domain behavior should remain provider-neutral.

## Decision

All model calls go through a Model Gateway with provider adapters and normalized usage/errors/streaming semantics.

## Alternatives Considered

- Direct provider SDK use inside Agent Runtime
- One provider hard-coded as the platform model

## Consequences

Adds an abstraction boundary and adapter code, but enables provider changes and consistent operational policy.
