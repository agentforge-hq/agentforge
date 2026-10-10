---
document: adr-001
version: 1.0
status: accepted
authority: architecture
depends_on:
  - PROJECT_CONSTITUTION
related: []
---

# ADR-001: Use event-driven architecture

## Status

ACCEPTED

## Context

Agent execution and document processing can outlive HTTP requests and need independent scaling, retry, and failure isolation.

## Decision

Use Kafka for asynchronous execution/document workflows and integration events. Keep synchronous REST for short request/response operations. Kafka is not introduced solely for ordinary CRUD.

## Alternatives Considered

- Fully synchronous HTTP orchestration
- Generic message broker without a concrete workload
- Kafka for every interaction

## Consequences

Adds operational complexity and eventual consistency to selected workflows, but provides durable asynchronous work, independent worker scaling, and replayable integration events.
