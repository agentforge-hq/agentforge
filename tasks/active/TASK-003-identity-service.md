# Task: TASK-003 - Implement Identity & Tenant Service

**Status:** In Progress  
**Owner:** User (Implementation), AI Agent (Review)  
**Depends On:** TASK-001, TASK-002  
**Target:** `agentforge-identity`  
**Framework Version:** Java 21, Spring Boot 4.1.1  

---

## 1. Description
Implement tenant management, user authentication, role-based authorization, and API key management for the `agentforge-identity` service adhering to `docs/03-domains/IDENTITY.md` and `docs/02-architecture/SECURITY_ARCHITECTURE.md`.

---

## 2. Technical Notes & Invariants

- **Layering:** Follow `API -> Application -> Domain -> Infrastructure` (pragmatic layered architecture).
- **Controllers are Thin Adapters:** Controllers only parse HTTP requests, invoke application services, and return DTOs.
- **DTOs ≠ Persistence Entities:** Persistence JPA entities must never be exposed as public API contracts.
- **Tenant Isolation:** Every repository query for membership, API keys, and resources must include the `tenant_id` scope.
- **API Key Hashing:** Store only cryptographic hashes (SHA-256 or BCrypt) in PostgreSQL; raw secrets are returned only once upon creation.
- **Login Tenant Resolution & Switching:** Login (`/api/v1/auth/login`) takes `email` and `password` without requiring `tenantId`; the active tenant context defaults to the user's latest joined organization (`createdAt DESC`). Authenticated users switch workspaces via `/api/v1/auth/switch-tenant` without re-entering credentials.
- **Database Rules:** Opaque UUID primary keys, optimistic locking (`@Version`), and Flyway migrations.

---

## 3. Package Structure

```text
src/main/java/com/agentforge/identity/
│
├── api/                                # Inbound Web Adapters
│   ├── controller/                     # Thin REST controllers
│   │   ├── OrganizationController.java
│   │   ├── AuthController.java
│   │   └── ApiKeyController.java
│   └── dto/                            # API contracts (Request/Response DTOs)
│       ├── request/                    # CreateOrganizationRequest, LoginRequest, etc.
│       └── response/                   # OrganizationResponse, ApiKeyCreatedResponse, etc.
│
├── application/                        # Use Cases & Application Services
│   └── service/
│       ├── OrganizationService.java
│       ├── UserService.java
│       ├── MembershipService.java
│       ├── ApiKeyService.java
│       └── AuthService.java
│
├── domain/                             # Core Business Models & Invariants
│   ├── model/                          # Business entities & records
│   │   ├── Organization.java           # Tenant boundary
│   │   ├── User.java                   # Human principal
│   │   ├── Membership.java             # User <-> Org association
│   │   ├── ApiKey.java                 # Programmatic credential
│   │   └── Role.java / Permission.java # Stable security identifiers
│   └── exception/                      # Service-specific exceptions
│
├── infrastructure/                     # Technical Adapters & Frameworks
│   ├── persistence/
│   │   ├── entity/                     # JPA @Entity database classes
│   │   │   ├── OrganizationEntity.java
│   │   │   ├── UserEntity.java
│   │   │   ├── MembershipEntity.java
│   │   │   └── ApiKeyEntity.java
│   │   └── repository/                 # Spring Data JPA repositories
│   │       ├── OrganizationRepository.java
│   │       ├── UserRepository.java
│   │       ├── MembershipRepository.java
│   │       └── ApiKeyRepository.java
│   └── security/                       # Spring Security & Token Infrastructure
│       ├── JwtTokenProvider.java       # JWT token creation & parsing
│       ├── ApiKeyAuthenticationFilter.java
│       └── SecurityConfig.java         # Spring SecurityFilterChain
│
└── IdentityServiceApplication.java     # Spring Boot entry point

src/main/resources/
├── db/migration/
│   └── V1__init_identity_schema.sql    # Flyway database migration
└── application.yml                     # Service properties (Postgres, JWT, etc.)
```

---

## 4. Checklist & Deliverables

### Module & POM Setup
- [x] Add `<module>agentforge-identity</module>` to root `src/pom.xml`
- [x] Create `agentforge-identity/pom.xml` inheriting from `agentforge-parent`
- [x] Add dependencies: `agentforge-common`, `spring-boot-starter-web`, `spring-boot-starter-data-jpa`, `postgresql`, `flyway-core`, `spring-boot-starter-security`, `spring-boot-starter-test`, `testcontainers`

### Domain Models (`com.agentforge.identity.domain.model`)
- [x] `Organization`: Tenant boundary with `TenantId`, name, status
- [x] `User`: Human identity with email, password hash, status
- [x] `Membership`: Associates `User` with `Organization` and roles
- [x] `ApiKey`: Programmatic credential with hashed secret, tenant ID, and scopes
- [x] `Role` & `Permission` enums / constants

### Database Migrations (`src/main/resources/db/migration`)
- [x] `V1__init_identity_schema.sql`: Tables for `organizations`, `users`, `memberships`, `api_keys` with UUID primary keys and indexes

### Persistence (`com.agentforge.identity.infrastructure.persistence`)
- [x] JPA Entities: `OrganizationEntity`, `UserEntity`, `MembershipEntity`, `ApiKeyEntity`
- [x] Spring Data JPA repositories with tenant-scoped query methods

### Security & Authentication (`com.agentforge.identity.infrastructure.security`)
- [x] `JwtTokenProvider`: Asymmetric RSA (RS256) token signing, verification, and RFC 7517 JWKS output
- [x] `SecurityConfig`: Spring Security filter chain setup
- [x] `JwksController`: Public `/.well-known/jwks.json` endpoint for downstream microservices and gateways
- [ ] `ApiKeyAuthenticationFilter`: Validates hashed API keys for programmatic access

### Application Services (`com.agentforge.identity.application.service`)
- [ ] `OrganizationService`: Tenant creation and management
- [ ] `AuthService`: User registration, login (latest joined tenant resolution), switch tenant, JWT token issuance
- [ ] `MembershipService`: Adding/removing users from organizations
- [ ] `ApiKeyService`: Generation (returns raw key once) and hash validation

### API Controllers & DTOs (`com.agentforge.identity.api`)
- [ ] `AuthController`: `/api/v1/auth/register`, `/api/v1/auth/login`, `/api/v1/auth/switch-tenant`
- [ ] `OrganizationController`: `/api/v1/organizations`
- [ ] `ApiKeyController`: `/api/v1/api-keys`

### Testing
- [ ] Unit tests for domain models & service logic
- [ ] Integration tests with Testcontainers (PostgreSQL)
