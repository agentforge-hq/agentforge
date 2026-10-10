---
document: agent-runtime
version: 1.0
status: active
authority: domain
depends_on:
  - AGENT
  - MODEL
  - TOOL
  - PROJECT_CONSTITUTION
related:
  - EVENTS
  - KAFKA_TOPICS
---

# Agent Runtime

The runtime is the central execution engine.

## Conceptual loop

```text
Input
  |
  v
Load immutable AgentVersion
  |
  v
Build Context
  |
  v
Call Model
  |
  +---- final answer ------> Complete
  |
  +---- tool call ---------> Authorize
                               |
                               v
                           Validate Input
                               |
                               v
                           Execute Tool
                               |
                               v
                         Append Tool Result
                               |
                               v
                           Call Model
```

## Execution states

```text
REQUESTED
  -> RUNNING
  -> COMPLETED

REQUESTED -> CANCELLED
RUNNING   -> CANCELLED

RUNNING -> FAILED
REQUESTED -> FAILED
```

A worker restart must not leave an execution permanently invisible. Recoverable `RUNNING` executions require lease/heartbeat or equivalent stale-worker detection.

## Step model

Each step has:

- stepId
- executionId
- sequence
- type
- state
- startedAt
- completedAt
- input metadata
- output metadata
- error metadata
- trace identifiers

Step types:

- CONTEXT_BUILD
- MODEL_CALL
- TOOL_AUTHORIZATION
- TOOL_CALL
- RETRIEVAL
- FINAL_RESPONSE

## Limits

Every execution has explicit:

- maximum steps
- model token budget
- wall-clock timeout
- tool timeout
- output-size limits

The runtime must fail deterministically when a limit is exceeded.

## Model timeout

Record failed model step, apply bounded retry only if the model policy permits it, otherwise fail execution.

## Tool timeout

Record timeout. Retry only according to tool retry policy. Never retry indefinitely.

## Duplicate event

The execution consumer uses an idempotency key/eventId and durable state checks. A duplicate request must not create a second logical execution when the original request is already accepted.

## Cancellation

Cancellation changes durable execution state. Workers periodically check cancellation before starting expensive work and at safe boundaries. An in-flight provider call is cancelled when the provider/client supports cancellation.

## Redis unavailable

Redis is not authoritative. Features depending on Redis must either degrade safely or fail explicitly according to operation criticality. Execution state remains in PostgreSQL.

## Database unavailable

Do not acknowledge durable work that could not be persisted. Kafka-based workers allow redelivery after transient database failure.

## Recovery

A worker claims an execution with a lease/heartbeat. If the lease expires, another worker may recover the execution only when the state machine proves the previous worker no longer owns it. Recovery must be idempotent.

## Streaming

Client-facing streaming uses SSE in the initial architecture. Stream events are derived from authorized execution state/events; Kafka is not exposed directly to browsers.

## Persistence

At minimum persist:

- execution
- exact agentVersion reference
- tenant
- actor
- state
- timestamps
- steps
- final result metadata
- failure metadata
- usage/cost metadata

Sensitive prompts/results must follow data-retention and redaction policy.

## Concurrency

An execution is one logical state machine. Concurrent workers must not mutate the same execution without ownership/compare-and-set protection.

## Failure matrix

| Failure | Expected behavior |
|---|---|
| Model timeout | bounded retry or fail |
| Tool timeout | bounded retry or fail |
| Kafka consumer crash | message redelivery |
| Redis unavailable | degrade/fail safely; DB remains truth |
| DB unavailable | do not acknowledge durable work |
| Duplicate event | idempotent no-op or state-aware handling |
| Step limit exceeded | fail execution |
| Cancellation | transition to CANCELLED at safe boundary |
