---
document: service-boundaries
version: 1.0
status: active
authority: architecture
depends_on:
  - SYSTEM_ARCHITECTURE
  - PROJECT_CONSTITUTION
related:
  - DATA_OWNERSHIP
  - COMMUNICATION
---

# Service Boundaries

## Identity Service

Responsibility:

- users
- organizations/tenants
- memberships
- roles
- permissions
- authentication integration
- API keys

Non-responsibility:

- agent configuration
- conversation content
- execution state

## Agent Service

Responsibility:

- Agent
- AgentVersion
- agent configuration
- publishing validation
- tool/knowledge bindings

Non-responsibility:

- executing model calls
- conversation history
- vector indexing

## Agent Execution Service

Responsibility:

- execution lifecycle
- runtime state
- steps
- model/tool orchestration
- cancellation
- execution recovery
- streaming execution state

Non-responsibility:

- owning agent drafts
- owning user identity

## Conversation Service

Responsibility:

- conversations
- messages
- attachments metadata
- ordering
- conversation context

Non-responsibility:

- agent runtime policy
- model provider SDKs

## Model Gateway

Responsibility:

- provider-neutral model invocation
- model capabilities
- provider routing
- timeout/retry
- streaming
- usage/cost normalization

Non-responsibility:

- agent configuration
- tenant business rules

## Knowledge Service

Responsibility:

- knowledge bases
- documents
- ingestion lifecycle
- chunking metadata
- embedding/indexing orchestration

## Search Service

Responsibility:

- retrieval abstraction
- metadata filtering
- vector/hybrid retrieval
- reranking abstraction

It must not become the owner of knowledge domain metadata.

## Notification Service

Responsibility:

- delivery of user/system notifications

This service is optional for MVP and should not be introduced until a concrete notification workflow exists.

## API Gateway

Responsibility:

- routing
- edge authentication enforcement
- rate limiting where appropriate
- correlation IDs
- client-facing policy

The gateway does not own business state.

## Service count policy

These are logical boundaries, not a requirement for one Kubernetes deployment per service. MVP may combine Identity + Tenant, Agent + related application modules, or Conversation with other low-volume boundaries if deployment simplicity is more valuable. Splitting requires operational justification.
