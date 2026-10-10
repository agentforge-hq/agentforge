# Task: TASK-001 - Scaffold Backend Maven Multi-Module Source Base

**Status:** Completed  
**Owner:** User (Implementation), AI Agent (Review)  
**Created:** 2026-10-08  
**Completed:** 2026-10-10  
**Target:** Java 21, Spring Boot 4.1.1, Maven Monorepo  

---

## 1. Description
Initialize the root Maven multi-module structure for AgentForge adhering to clean hexagonal architecture (`API → Application → Domain → Infrastructure`) and strict service boundaries.

---

## 2. Relevant Documents & References
- `docs/00-constitution/PROJECT_CONSTITUTION.md`
- `docs/00-constitution/ENGINEERING_PRINCIPLES.md`
- `docs/02-architecture/SERVICE_BOUNDARIES.md`
- `docs/02-architecture/SYSTEM_ARCHITECTURE.md`
- `docs/08-engineering/CODING_STANDARDS.md`
- `docs/08-engineering/TESTING.md`

---

## 3. Recommended Folder & Package Structure

```text
agentforge/
├── pom.xml                                   # Root Parent POM
├── mvnw / mvnw.cmd / .mvn/                   # Maven Wrapper
├── docker-compose.yml                        # Local Postgres, Redis, Kafka
│
├── agentforge-common/                        # Shared kernel & cross-cutting utilities
│   ├── pom.xml
│   └── src/
│       ├── main/java/com/agentforge/common/
│       │   ├── core/                         # Pure Java 21 core kernel (NO Spring dependencies)
│       │   │   ├── context/                  # TenantContext, TenantContextHolder
│       │   │   ├── domain/                   # Base AggregateRoot, TenantId, EntityId VO
│       │   │   ├── event/                    # Base DomainEvent, CloudEvents envelope
│       │   │   └── exception/                # Global exceptions, ErrorCode
│       │   ├── web/                          # Spring Web REST infrastructure
│       │   │   ├── filter/                   # TenantContextFilter, CorrelationIdFilter
│       │   │   ├── handler/                  # GlobalExceptionHandler (@RestControllerAdvice)
│       │   │   └── response/                 # Standard ApiResponse<T>, ErrorResponse, PageResponse<T>
│       │   ├── kafka/                        # Kafka messaging contracts & envelopes
│       │   │   └── event/                    # Kafka headers, event envelopes, serialization
│       │   └── config/                       # Spring AutoConfigurations (Jackson, Web)
│       └── test/java/com/agentforge/common/
│
└── agentforge-<module>/                      # Domain Modules (e.g., identity, agent, execution...)
    ├── pom.xml
    └── src/
        ├── main/
        │   ├── java/com/agentforge/<module>/
        │   │   │
        │   │   ├── api/                      # REST Controllers, DTOs, Mappers
        │   │   │   ├── controller/           # Thin REST controllers
        │   │   │   ├── dto/
        │   │   │   │   ├── request/
        │   │   │   │   └── response/
        │   │   │   └── mapper/               # MapStruct mappers
        │   │   │
        │   │   ├── application/              # Use cases, application services
        │   │   │   ├── port/
        │   │   │   │   ├── in/               # Inbound use-case interfaces (Commands/Queries)
        │   │   │   │   └── out/              # Outbound SPI interfaces (Repositories, Clients, Events)
        │   │   │   └── service/              # Use case service implementations
        │   │   │
        │   │   ├── domain/                   # Pure Domain (POJOs/Records, NO Spring imports)
        │   │   │   ├── model/                # Aggregates, Entities, Value Objects
        │   │   │   ├── event/                # Domain events
        │   │   │   └── exception/            # Domain-specific business exceptions
        │   │   │
        │   │   └── infrastructure/           # Technical implementation of outbound ports
        │   │       ├── persistence/          # JPA/Hibernate Entities, Spring Data Repositories, Adapters
        │   │       │   ├── entity/
        │   │       │   ├── repository/
        │   │       │   └── adapter/          # Implements application.port.out.*Repository
        │   │       ├── messaging/            # Kafka Producers/Consumers, Event Listeners
        │   │       └── client/               # External HTTP / SDK clients
        │   │
        │   └── resources/
        │       ├── db/migration/             # Flyway SQL scripts (V1__..., V2__...)
        │       └── application-<module>.yml
        │
        └── test/
            ├── java/com/agentforge/<module>/
            │   ├── unit/                     # Unit tests (Domain & Application logic)
            │   └── integration/              # Testcontainers integration tests (DB, Kafka)
            └── resources/
                └── application-test.yml
```

---

## 4. Checklist & Deliverables

### Root Configuration
- [x] Root `pom.xml` with `<packaging>pom</packaging>`
- [x] Centralized `<dependencyManagement>` for Spring Boot (4.1.1) (BOM manages Spring, Kafka, Jackson, Postgres, Testcontainers)
- [x] Maven Wrapper (`mvnw`, `mvnw.cmd`, `.mvn/wrapper/`)
- [x] Local infrastructure configuration (`docker-compose.yml` with Postgres pgvector, Redis, Kafka KRaft)

### Core Modules Setup
- [x] `agentforge-common` (Scaffolded: directory structure created and reactor linked)
- [-] Domain modules (`identity`, `agent`, etc.) deferred to subsequent dedicated tasks (`TASK-003+`)

### Verification & Review
- [x] Verify `mvn clean compile` succeeds across all modules
- [x] Verify no cyclic module dependencies
- [x] Verify domain layer contains no framework-specific / Spring direct dependencies
- [x] AI Agent review completed
