---
document: ci-cd
version: 1.0
status: target
authority: engineering
depends_on:
  - TESTING
  - DOCKER
---

# CI/CD

## Pipeline

```text
Commit
 -> lint
 -> unit tests
 -> integration tests
 -> build
 -> container image
 -> security scan
 -> publish registry
 -> deploy
 -> smoke tests
```

## Candidate tools

- GitHub Actions
- SonarQube
- Trivy
- Testcontainers

Tool selection may change without changing pipeline responsibilities.

## Promotion

Artifacts should be immutable between environments.

## Deployment safety

- health checks
- rollback strategy
- backward-compatible database migrations
- smoke tests
- deployment observability

## Security

Build pipeline must detect:

- vulnerable dependencies
- container vulnerabilities
- leaked secrets where tooling supports detection
