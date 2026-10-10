---
document: redis
version: 1.0
status: active
authority: architecture
depends_on:
  - PROJECT_CONSTITUTION
---

# Redis

## Role

Redis provides low-latency ephemeral/distributed capabilities.

Legitimate uses:

- cache
- distributed lock where justified
- rate limiting
- idempotency records with explicit TTL
- short-lived execution coordination
- session-related data if a session model is used

## Source of truth

Redis is not durable business truth.

Every cache entry must have an invalidation/TTL strategy.

## Key design

Keys must include tenant scope when data is tenant-owned.

Example:

`agentforge:{tenantId}:agent:{agentId}:...`

## Failure

The application must define whether Redis failure:

- degrades to database
- temporarily disables an optimization
- fails a specific operation

Do not silently treat Redis loss as loss of business state.
