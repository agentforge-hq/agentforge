---
document: local-development
version: 1.0
status: active
authority: engineering
depends_on:
  - DATABASE
  - REDIS
---

# Local Development

## Goals

A developer should be able to run the MVP dependencies locally with minimal setup.

## MVP dependencies

- PostgreSQL
- Redis
- application services
- gateway

Kafka is introduced when the asynchronous phase is implemented.

## Configuration

Configuration is externalized.

Use local environment files/secrets only for development and never commit credentials.

## Database

Run migrations automatically or through an explicit developer command. Local schema must match the versioned migrations.

## Testcontainers

Integration tests should provision disposable PostgreSQL/Redis/Kafka dependencies rather than relying on a developer's long-running environment.

## Developer workflow

1. start dependencies
2. apply migrations
3. start services
4. authenticate
5. create tenant/agent
6. execute agent
7. inspect logs/traces
