---
document: adr-003
version: 1.0
status: accepted
authority: architecture
depends_on:
  - PROJECT_CONSTITUTION
related: []
---

# ADR-003: Multi-tenancy strategy

## Status

ACCEPTED

## Context

AgentForge serves multiple organizations and must prevent cross-tenant access.

## Decision

Use logical tenant isolation enforced in identity context, authorization, persistence queries, cache keys, Kafka metadata, vector filters, and observability controls. Production may use database-per-service while MVP can use schemas/databases in one PostgreSQL cluster.

## Alternatives Considered

- Separate deployment/database for every tenant
- Application-only tenant filtering
- Shared database without mandatory tenant predicates

## Consequences

Logical isolation supports efficient multi-tenancy while defense-in-depth reduces leakage risk. The application must treat tenant context as security-critical.
