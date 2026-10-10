---
document: security-architecture
version: 1.0
status: active
authority: security
depends_on:
  - PROJECT_CONSTITUTION
  - DATA_OWNERSHIP
related:
  - IDENTITY
  - TOOL
---

# Security Architecture

## Authentication

Spring Security integrates with OAuth2/OIDC. JWT access tokens are validated at the API boundary and relevant downstream services.

## Authorization

Authorization is evaluated server-side using:

- authenticated principal
- tenant membership
- role/permission
- resource ownership
- operation-specific policy

A valid JWT does not imply access to every resource.

## Service-to-service security

Internal calls must be authenticated. Service identity must be distinct from end-user identity where appropriate. User/tenant context is propagated only as trusted signed/authenticated metadata.

## Tenant isolation

Tenant ID is derived from trusted identity/context, not accepted as an arbitrary client-controlled authority.

Every repository query and cache key for tenant-owned data must include tenant scope.

Asynchronous messages include tenant metadata and consumers verify it before processing.

## API keys

API keys are stored hashed where possible and shown only at creation. They are scoped and revocable.

## Secrets

Secrets are externalized through environment/secret management. Never place secrets in source, Docker images, logs, or documentation examples.

## Tool security

### HTTP tools

Must enforce:

- outbound destination allowlists where possible
- SSRF protection
- private-network blocking
- DNS/IP validation
- redirect restrictions
- timeout
- response-size limits

### Database tools

Must not allow arbitrary SQL. Operations are defined through allowlisted queries/procedures or a constrained query interface.

### Prompt injection

Retrieved content and tool outputs are untrusted data. The runtime must not treat instructions inside external content as higher authority than the agent/system policy.

## Rate limiting

Apply limits at gateway and/or service boundaries according to tenant and operation.

## Audit

Record security-sensitive actions such as:

- login/security changes
- role changes
- API key creation/revocation
- agent publication
- tool permission changes
- sensitive execution operations
