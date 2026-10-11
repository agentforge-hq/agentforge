# Task: TASK-003 - Implement Identity & Tenant Service

**Status:** In Progress  
**Depends On:** TASK-001, TASK-002  
**Target:** `agentforge-identity`

---

## 1. Description
Implement tenant management, user authentication, role-based authorization, and API key management following `docs/03-domains/IDENTITY.md` and `docs/02-architecture/SECURITY_ARCHITECTURE.md`.

---

## 2. Checklist
- [ ] Tenant & User domain entities and repository ports
- [ ] Flyway database migration scripts
- [ ] API Key hashing, generation, and validation
- [ ] Security filter chain & token authentication
- [ ] Integration tests using Testcontainers (PostgreSQL)
