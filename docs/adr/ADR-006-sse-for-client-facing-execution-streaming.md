---
document: adr-006
version: 1.0
status: accepted
authority: architecture
depends_on:
  - PROJECT_CONSTITUTION
related: []
---

# ADR-006: SSE for client-facing execution streaming

## Status

ACCEPTED

## Context

Browsers need incremental execution/output updates, while Kafka should remain an internal infrastructure concern.

## Decision

Use Server-Sent Events for the initial client-facing execution stream. The server maps authorized execution events/state to an SSE stream.

## Alternatives Considered

- Browser WebSocket connection
- Browser directly consuming Kafka
- Polling only

## Consequences

SSE is simple for server-to-client streaming and fits execution progress. Bidirectional realtime requirements may justify WebSockets later.
