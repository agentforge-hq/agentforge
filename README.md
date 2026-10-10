# AgentForge

AgentForge is a production-oriented, multi-tenant AI Agent Platform.

This repository is the **architecture and product Source of Truth**. It is intentionally implementation-light: architecture, domain invariants, contracts, operational rules, and engineering workflow are defined here before application code is written.

## Source-of-truth hierarchy

1. Security requirements
2. Domain invariants
3. Project Constitution
4. Architecture Decision Records
5. Domain specifications
6. API/Event contracts
7. Implementation
8. AI-generated suggestions

When implementation conflicts with a higher layer, the higher layer wins unless an explicit architectural change is accepted.

## Documentation navigation

Start with:

1. `AGENTS.md`
2. `docs/00-constitution/PROJECT_CONSTITUTION.md`
3. `docs/00-constitution/ENGINEERING_PRINCIPLES.md`
4. Relevant architecture/domain document
5. Relevant contracts
6. Relevant ADRs

Do not read the entire repository for every task.

## Repository status

This repository describes the target architecture and MVP boundary. It does not claim that the implementation already exists.

## Target stack

- Java 21+
- Spring Boot
- Spring Security
- Spring Cloud Gateway
- Spring Kafka
- Spring AI
- PostgreSQL
- Redis
- Vector database behind a Search Service abstraction
- React + TypeScript
- Docker
- Kubernetes
- Micrometer
- Prometheus
- OpenTelemetry
- Grafana
- JUnit + Testcontainers
- GitHub Actions
