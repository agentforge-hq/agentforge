---
document: architecture-scope
version: 1.0
status: active
authority: architecture
depends_on:
  - PRODUCT_REQUIREMENTS
  - SERVICE_BOUNDARIES
---

# Architecture Scope

## Logical target

AgentForge has the following logical boundaries:

- Identity/Tenant
- Agent
- Agent Execution
- Conversation
- Model Gateway
- Knowledge
- Search
- Notification
- API Gateway

## MVP deployment simplification

The MVP does not need one deployable per logical boundary.

Recommended initial deployment grouping:

```text
gateway
identity-api
agent-api
execution-api
model-gateway
postgres
redis
```

Conversation may initially be a module within the application boundary that owns it, provided its data ownership remains explicit.

## Post-MVP

Split workloads when there is evidence of:

- independent scaling
- independent failure domain
- independent deployment cadence
- security isolation requirement
- clear ownership

## Principle

Bounded contexts are logical architecture. Microservice deployment is an operational decision.
