---
document: agent-domain
version: 1.0
status: active
authority: domain
depends_on:
  - PROJECT_CONSTITUTION
  - IDENTITY
related:
  - AGENT_RUNTIME
  - TOOL
  - KNOWLEDGE
  - MODEL
---

# Agent Domain

## Entities

### Agent

Mutable logical definition.

Fields conceptually include:

- id
- tenantId
- name
- description
- status
- currentDraftVersion
- createdBy
- timestamps

### AgentVersion

Immutable configuration snapshot.

Includes:

- versionId
- agentId
- version number
- system prompt
- model configuration
- runtime limits
- permissions
- tool bindings
- knowledge bindings
- configuration metadata
- createdAt
- publishedAt

## Lifecycle

```text
DRAFT
  |
  v
VALIDATING
  |
  +---- invalid ---> DRAFT
  |
  v
PUBLISHED
  |
  v
ACTIVE
  |
  v
ARCHIVED
```

A simpler deployment may use `PUBLISHED` as the executable state; `ACTIVE` represents a version selected for default execution.

## Invariants

1. Published AgentVersions are immutable.
2. A new configuration requires a new version.
3. An execution references an exact AgentVersion.
4. Agent draft mutation cannot mutate a published version.
5. Tool and knowledge bindings are resolved against authorized resources.
6. A version cannot be published if required validation fails.

## Publication

Publishing validates:

- configuration completeness
- model configuration
- tool permissions
- knowledge references
- runtime limits
- tenant authorization

Publication produces an immutable snapshot.
