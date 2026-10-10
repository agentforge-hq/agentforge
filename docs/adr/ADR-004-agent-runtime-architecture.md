---
document: adr-004
version: 1.0
status: accepted
authority: architecture
depends_on:
  - PROJECT_CONSTITUTION
related: []
---

# ADR-004: Agent runtime architecture

## Status

ACCEPTED

## Context

Agent execution is stateful, iterative, and may involve model/tool calls, limits, retries, cancellation, and worker failure.

## Decision

Represent execution as a durable state machine with explicit steps. Use immutable AgentVersion configuration, durable execution state, worker ownership/lease, bounded retries, and idempotent processing.

## Alternatives Considered

- Stateless request handler containing the whole loop
- External workflow engine from day one
- In-memory execution state only

## Consequences

The runtime is more explicit to implement, but recovery, observability, and correctness are controllable without prematurely adding a workflow platform.
