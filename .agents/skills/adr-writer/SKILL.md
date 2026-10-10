---
name: adr-writer
description: >-
  Use this skill when a proposed change requires an Architecture Decision Record.
  Triggers when changes affect service boundaries, data ownership, communication
  patterns, security model, or runtime semantics.
---

# ADR Writer Skill

## When to Write an ADR

An ADR is required when changing:
- Service boundaries
- Data ownership
- Major communication patterns
- Security model
- Runtime semantics

## ADR Template

Create the file at `docs/adr/ADR-NNN-<slug>.md` where NNN is the next available number.

```markdown
---
adr: NNN
title: <Decision Title>
status: proposed | accepted | deprecated | superseded
date: YYYY-MM-DD
decision_makers: []
---

# ADR-NNN: <Decision Title>

## Status

Proposed

## Context

<What is the issue? What forces are at play?>

## Decision

<What is the change we are proposing?>

## Consequences

### Positive
- <benefit>

### Negative
- <cost>

### Neutral
- <observation>

## Alternatives Considered

### <Alternative 1>
- Pros: ...
- Cons: ...
- Why rejected: ...
```

## Current ADR Sequence

| Number | Status | Topic |
|--------|--------|-------|
| 001 | Accepted | Event-driven architecture |
| 002 | Accepted | AgentVersion immutability |
| 003 | Accepted | Multi-tenancy strategy |
| 004 | Accepted | Agent runtime architecture |
| 005 | Accepted | Model gateway abstraction |
| 006 | Accepted | SSE for client-facing execution streaming |

Next available: **ADR-007**

## After Creating the ADR

1. Update `docs/adr/README.md` with the new entry
2. Update any affected source-of-truth documents
3. Only then proceed with implementation
