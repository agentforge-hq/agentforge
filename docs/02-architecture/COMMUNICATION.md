---
document: communication
version: 1.0
status: active
authority: architecture
depends_on:
  - SERVICE_BOUNDARIES
related:
  - EVENTS
  - KAFKA_TOPICS
---

# Communication

## Synchronous communication

Use HTTP/REST when:

- the caller needs an immediate response
- the operation is short-lived
- the dependency is required to make the current decision
- request/response semantics are natural

## Asynchronous communication

Use Kafka when:

- work may exceed HTTP lifecycle
- workers need independent scaling
- retries/replay are useful
- multiple consumers need the same integration event
- pipeline stages are naturally asynchronous

## Commands vs events

A command asks a component to do something.

An event states that something happened.

Naming examples:

- command: `agent.execution.requested.v1`
- event: `agent.execution.started.v1`
- event: `agent.execution.completed.v1`

Kafka may carry both, but semantics must remain explicit.

## Failure behavior

HTTP clients use bounded timeouts and explicit retry policy.

Kafka consumers use idempotency and bounded retries. Failed messages are routed to a DLQ after the configured retry policy.

## Outbox

When a durable database state transition must atomically produce an event, use a transactional outbox or equivalent reliable publication mechanism.

Do not claim atomic database+Kafka transactions across unrelated service databases.

## Correlation

Every request/event chain carries:

- correlationId
- causationId where applicable
- tenantId
- actor/user identity metadata where permitted
