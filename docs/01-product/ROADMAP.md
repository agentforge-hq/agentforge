---
document: roadmap
version: 1.0
status: active
authority: product
depends_on:
  - PRODUCT_REQUIREMENTS
---

# Roadmap

## Phase 0 — Foundation

- repository structure
- source-of-truth documentation
- CI baseline
- local development conventions

## Phase 1 — MVP

- Identity
- Tenant
- Agent CRUD
- AgentVersion
- publishing
- Model Gateway
- Conversation
- basic Agent Runtime
- execution persistence
- SSE streaming
- PostgreSQL
- Redis
- Docker

## Phase 2 — Platform capabilities

- Tool registry
- tool authorization
- Kafka execution/document pipelines
- KnowledgeBase
- document processing
- vector search
- citations
- advanced execution recovery

## Phase 3 — Production platform

- Kubernetes
- HPA
- OpenTelemetry
- Prometheus/Grafana
- security hardening
- rate limiting
- cost tracking
- CI/CD hardening

## Phase 4 — Scale and resilience

- load testing
- fault injection
- replay tooling
- advanced model routing
- operational automation

Roadmap phases do not override domain boundaries established by the architecture.
