---
document: engineering-principles
version: 1.0
status: active
authority: constitution
depends_on:
  - PROJECT_CONSTITUTION
related:
  - CODING_STANDARDS
  - TESTING
---

# Engineering Principles

## Prefer explicit boundaries

Dependencies should point toward stable contracts. Domain behavior should not be coupled to vendor SDKs.

## Optimize for correctness first

Distributed systems behavior must be explicit about:

- ownership
- consistency
- retries
- failure
- idempotency
- ordering

## Use asynchronous processing deliberately

An operation should become asynchronous when it is long-running, independently scalable, failure-isolatable, or naturally event-driven.

## Make state transitions explicit

Important lifecycle transitions are modeled as named states and validated transitions, not inferred from incidental database fields.

## Make failures normal

Every external dependency is assumed capable of:

- timeout
- unavailable state
- partial response
- duplicate delivery
- delayed delivery
- restart

## Observability is part of the contract

Critical operations emit enough structured telemetry to reconstruct what happened without inspecting application internals.

## Boring infrastructure wins

Prefer mature, understandable primitives over infrastructure introduced solely for architectural appearance.

## Backward compatibility matters

Public APIs and events should evolve additively whenever practical.
