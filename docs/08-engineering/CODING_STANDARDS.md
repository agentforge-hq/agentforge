---
document: coding-standards
version: 1.0
status: active
authority: engineering
depends_on:
  - PROJECT_CONSTITUTION
---

# Coding Standards

## Backend

- Java 21+
- Spring Boot
- constructor injection
- no field injection
- controllers remain thin
- domain logic outside controllers
- explicit transaction boundaries
- immutable objects where practical
- DTOs at API boundaries
- no persistence entity exposure
- no direct cross-service DB access

## Layering

```text
API
 ↓
Application
 ↓
Domain
 ↓
Infrastructure
```

Infrastructure implements interfaces required by application/domain layers.

## Naming

Names should express business intent. Avoid generic `Manager`, `Helper`, or `Util` classes unless the abstraction is genuinely generic.

## Exceptions

Use stable application/domain error categories. Do not expose infrastructure exception types through public APIs.

## Configuration

Externalize environment-specific values.

## Logging

Structured logs. No secrets.

## Code review

Reviewers verify:

- invariants
- authorization
- tenant isolation
- transaction boundaries
- retries/timeouts
- observability
- tests
- contract compatibility
