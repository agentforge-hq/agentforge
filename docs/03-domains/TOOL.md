---
document: tool-domain
version: 1.0
status: active
authority: domain
depends_on:
  - IDENTITY
  - SECURITY_ARCHITECTURE
  - AGENT
related:
  - AGENT_RUNTIME
---

# Tool Domain

## Tool definition

```text
id
name
description
type
inputSchema
outputSchema
timeout
retryPolicy
permission
status
```

## Initial types

- HTTP
- DATABASE
- SEARCH
- KNOWLEDGE
- INTERNAL
- WEBHOOK

V1 explicitly excludes arbitrary code execution.

## Tool registry

The registry owns approved tool definitions and their policy.

Agent versions bind to tool definitions by immutable identifier/version where necessary. Publishing validates that bindings remain authorized.

## Authorization

A tool call requires:

1. authenticated execution context
2. tenant ownership/availability
3. agent permission to use the tool
4. actor permission where applicable
5. tool-specific policy

## Validation

Inputs are validated against the tool input schema before execution. Outputs are validated/normalized where an output schema exists.

## Retry

Retries are bounded and only enabled when the operation is safe or explicitly idempotent.

## Audit

Security-sensitive tool calls emit audit information without leaking secrets.

## Observability

Tool calls record duration, outcome, retry count, and safe metadata. Sensitive request/response values are redacted.
