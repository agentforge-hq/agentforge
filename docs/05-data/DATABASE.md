---
document: database
version: 1.0
status: active
authority: architecture
depends_on:
  - DATA_OWNERSHIP
  - PROJECT_CONSTITUTION
---

# PostgreSQL

## Role

PostgreSQL is the primary durable transactional store.

## Ownership

Each service owns its own tables/schema/database boundary. A shared PostgreSQL cluster is acceptable operationally; cross-service table access is not.

## UUID

Use UUID identifiers for externally visible entities.

## Transactions

Transactions should match a business consistency boundary. Avoid distributed transactions across services.

## Optimistic locking

Use versioning/optimistic concurrency where concurrent updates to mutable resources are possible, especially agent drafts and execution state.

## Indexing

Indexes must be justified by query patterns. Tenant-scoped queries commonly begin with tenant/resource identifiers.

## Migrations

Use Flyway or an equivalent explicit migration mechanism.

Rules:

- migrations are ordered
- applied migrations are immutable
- destructive changes use staged migration
- production migration must be backward compatible with the deployment transition when rolling upgrades require it

## Outbox

Use transactional outbox when database state and event publication must be reliably coordinated.

## No shared domain tables

The database may physically host multiple schemas, but domain ownership remains separate.
