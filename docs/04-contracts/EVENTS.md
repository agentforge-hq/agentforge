---
document: events
version: 1.0
status: active
authority: contract
depends_on:
  - COMMUNICATION
  - PROJECT_CONSTITUTION
related:
  - KAFKA_TOPICS
---

# Event Contracts

## Envelope

```json
{
  "eventId": "uuid",
  "eventType": "agent.execution.started.v1",
  "occurredAt": "2026-10-08T12:00:00Z",
  "tenantId": "uuid",
  "correlationId": "uuid",
  "causationId": "uuid",
  "producer": "agent-execution",
  "payload": {}
}
```

## Required semantics

- `eventId` uniquely identifies the event.
- `eventType` identifies semantic type and version.
- `tenantId` identifies tenant scope.
- `correlationId` connects a distributed workflow.
- `causationId` points to the triggering operation/event where available.

## Event candidates

### Execution

- `agent.execution.requested.v1`
- `agent.execution.started.v1`
- `agent.execution.step.started.v1`
- `agent.execution.step.completed.v1`
- `agent.execution.completed.v1`
- `agent.execution.failed.v1`

### Knowledge

- `document.uploaded.v1`
- `document.processed.v1`
- `document.chunked.v1`
- `document.embedded.v1`
- `document.indexed.v1`

### Platform

- `notification.requested.v1`
- `audit.event.created.v1`

## Evolution

Consumers must tolerate additive fields.

Breaking semantic changes require a new event version.

## Delivery

At-least-once delivery is the default assumption. Consumers must be idempotent.

## Ordering

Ordering is guaranteed only within a Kafka partition. Partition keys must be selected according to the state machine whose ordering matters.
