---
document: agent-builder
version: 1.0
status: active
authority: product
depends_on:
  - AGENT
  - MODEL
  - TOOL
  - KNOWLEDGE
---

# Agent Builder

The Agent Builder edits a draft configuration.

## Main sections

1. Identity
2. Instructions/system prompt
3. Model
4. Runtime limits
5. Tools
6. Knowledge
7. Permissions
8. Validation
9. Publish

## Draft model

Edits modify a draft only.

Publishing creates an immutable AgentVersion.

## Validation UX

Show validation errors before publication:

- missing required configuration
- unauthorized tool
- invalid model configuration
- invalid knowledge binding
- unsafe runtime limit
- invalid permission configuration

## Publish flow

```text
Edit Draft
 -> Validate
 -> Show validation result
 -> Confirm Publish
 -> Create immutable version
 -> Mark version published
```

The UI must never assume a draft change is immediately active.
