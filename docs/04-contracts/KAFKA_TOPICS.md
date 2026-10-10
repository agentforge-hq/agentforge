---
document: kafka-topics
version: 1.0
status: active
authority: contract
depends_on:
  - EVENTS
  - COMMUNICATION
---

# Kafka Topics

Topic names use:

`<domain>.<entity>.<purpose>.v<version>`

Initial target topics:

| Topic | Key | Purpose |
|---|---|---|
| `agent.execution.requests.v1` | executionId | execution commands |
| `agent.execution.events.v1` | executionId | execution lifecycle events |
| `knowledge.document.events.v1` | documentId | document pipeline |
| `notification.requests.v1` | notificationId | notification work |
| `audit.events.v1` | tenantId | audit integration stream |

## Partition strategy

Execution topics use `executionId` as key to preserve ordering for one execution.

Knowledge document pipeline uses `documentId`.

Audit events may use tenant ID where tenant-local ordering is valuable, accepting lower parallelism per tenant.

## Consumer groups

Consumer group names represent a logical processing function, not a deployment instance.

Examples:

- `agent-execution-worker`
- `knowledge-processor`
- `search-indexer`
- `notification-worker`

## Retry

Use bounded retry topics or consumer retry mechanisms. Poison messages eventually reach a DLQ.

Example:

`agent.execution.requests.dlq.v1`

DLQ messages retain original event metadata and failure information.

## Replay

Replay is an operational capability, not a normal business workflow. Consumers must be safe to replay where their semantics allow it.

## Exactly-once

The platform does not claim global exactly-once processing. Durable state transitions and idempotency provide effectively-once business outcomes where required.
