---
document: model-domain
version: 1.0
status: active
authority: domain
depends_on:
  - PROJECT_CONSTITUTION
related:
  - AGENT_RUNTIME
  - SECURITY_ARCHITECTURE
---

# Model Domain

## Model Gateway purpose

Agent domains depend on a provider-neutral model capability, not provider-specific SDKs.

## Conceptual interface

The gateway supports:

- chat completion
- streaming completion
- structured output where supported
- tool-call interpretation
- usage reporting

## Provider abstraction

Provider adapters translate between internal request/response models and provider SDKs.

The Agent domain does not import provider SDK classes.

## Configuration

A model configuration may include:

- provider
- model
- temperature where supported
- max output tokens
- response format
- timeout
- retry policy
- fallback policy

## Failure

Normalize provider failures into stable categories:

- authentication/configuration
- rate limited
- timeout
- unavailable
- invalid request
- content/policy rejection
- unknown provider failure

## Fallback

Fallback is optional and policy-driven. It must not silently violate model capability requirements or cost constraints.

## Cost

Usage is normalized to a provider-neutral usage record. Pricing is configuration, not hard-coded domain behavior.
