# Task: TASK-002 - Implement Common Kernel & Tenant Context

**Status:** Completed  
**Owner:** User (Implementation), AI Agent (Review)  
**Depends On:** TASK-001  
**Target:** `agentforge-common`  
**Framework Version:** Java 21, Spring Boot 4.1.1  

---

## 1. Description
Implement shared domain building blocks, error response schemas, tenant context propagation (`TenantContextHolder`), and base repository/event interfaces for the `agentforge-common` module.

---

## 2. Technical Notes & Invariants

- **Spring Boot Version:** Target is **Spring Boot 4.1.1** (confirmed released in 10/2026; do not downgrade).
- **Parent Version Alignment:** Inherit `1.0.0-SNAPSHOT` from `agentforge-parent` (remove redundant `<groupId>` and `<version>1.0-SNAPSHOT</version>` from `agentforge-common/pom.xml`).
- **Domain Purity (Mandatory Invariant):** All classes under `com.agentforge.common.core.domain` must be pure POJOs / Records. **No direct Spring framework dependencies** in the domain layer.
- **Tenant Isolation:** Enforce mandatory tenant extraction and context validation; never trust client-supplied tenant IDs directly without server validation.
- **Contract Source of Truth:**
  - Errors: `docs/04-contracts/ERROR_CODES.md`
  - REST schema & envelopes: `docs/04-contracts/REST_API.md`
  - Domain events: `docs/04-contracts/EVENTS.md`

---

## 3. Checklist & Deliverables

### POM Configuration
- [x] Add baseline dependencies to `agentforge-common/pom.xml`: `jakarta.validation-api`, `jackson-databind`, `jackson-datatype-jsr310`, `slf4j-api`, `spring-web`, `jakarta.servlet-api`

### Core Context (`com.agentforge.common.core.context`)
- [x] `TenantContext`: Immutable record holding `TenantId`, `UserId`, roles/scopes
- [x] `TenantContextHolder`: ThreadLocal holder with safe lifecycle cleanup & `runWith` execution

### Core Domain Base Primitives (`com.agentforge.common.core.domain`)
- [x] Strongly typed ID Value Objects: `TenantId`, `EntityId`
- [x] Pragmatic domain contracts: `Auditable`, `SoftDeletable`

### Core Events (`com.agentforge.common.core.event`)
- [x] Base `DomainEvent` record adhering to CloudEvents envelope specification
- [x] Event metadata: `eventId`, `eventType`, `tenantId`, `occurredAt`, `correlationId`, `causationId`, `producer`

### Core Errors & Exceptions (`com.agentforge.common.core.exception`)
- [x] Base `AgentForgeException` (runtime exception hierarchy)
- [x] Domain-specific exceptions (`EntityNotFoundException`, `TenantAccessDeniedException`, `ValidationException`, `ConflictException`, `AuthenticationRequiredException`, `PermissionDeniedException`)
- [x] `ErrorCode` enum matching `docs/04-contracts/ERROR_CODES.md`

### Web Envelopes & Response Models (`com.agentforge.common.web.response`)
- [x] `ApiResponse<T>`: Unified response wrapper (`data`, `timestamp`, `correlationId`)
- [x] `ErrorResponse`: Standard error payload matching contract (`code`, `message`, `correlationId`, `details`)
- [x] `PageResponse<T>`: Standard pagination envelope

### Web Filters (`com.agentforge.common.web.filter`)
- [x] `TenantContextFilter`: Web filter extracting and validating `X-Tenant-Id` header into `TenantContextHolder`
- [x] `CorrelationIdFilter`: Web filter extracting/generating `X-Correlation-Id` and setting logging MDC

### Web Exception Handling (`com.agentforge.common.web.handler`)
- [x] `GlobalExceptionHandler`: Controller advice mapping exceptions to standard `ErrorResponse` schemas

### Auto-Configuration (`com.agentforge.common.config`)
- [x] `AgentForgeJacksonAutoConfiguration`: Omit null values (`NON_NULL`), configure ISO-8601 UTC dates, register `JavaTimeModule`
- [x] `AgentForgeWebAutoConfiguration`: Conditional web auto-configuration registering filters and exception handling
- [x] Register in `META-INF/spring/org.springframework.boot.autoconfigure.AutoConfiguration.imports`


