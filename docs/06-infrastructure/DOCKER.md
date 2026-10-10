---
document: docker
version: 1.0
status: active
authority: engineering
depends_on:
  - LOCAL_DEVELOPMENT
---

# Docker

## Principles

- multi-stage builds where useful
- minimal runtime images
- non-root process where compatible
- immutable image tags
- no secrets baked into images
- deterministic dependency resolution

## Image

Application images contain only the application and runtime dependencies.

Environment-specific configuration is injected at runtime.

## Health

Containers expose application health through Spring Actuator endpoints as appropriate.

## Local compose

A Docker Compose environment may be used for local dependencies and MVP services. It is not the production orchestration contract.
