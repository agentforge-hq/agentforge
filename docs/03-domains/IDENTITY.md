---
document: identity-domain
version: 1.0
status: active
authority: domain
depends_on:
  - PROJECT_CONSTITUTION
  - SECURITY_ARCHITECTURE
---

# Identity Domain

## Entities

### User

Represents an authenticated human identity.

### Organization

Represents a tenant boundary.

### Membership

Associates a User with an Organization and role assignments.

### Role

Named bundle of permissions.

### Permission

Atomic operation authorization such as `agent:read` or `agent:publish`.

### APIKey

Credential for programmatic access, scoped to an organization and permission set.

## Invariants

- A membership belongs to exactly one organization.
- A resource access check requires organization membership.
- Revoked API keys cannot authenticate.
- Permission names are stable public security identifiers.

## Authentication model

External identity provider or native authentication validates the user. AgentForge validates credentials/tokens and maps claims to an application principal.
At login (`/api/v1/auth/login`), credentials (email/password) are validated without requiring a `tenantId`. The active tenant context is automatically resolved to the latest organization joined by the user (`createdAt DESC`).

## Tenant context & switching

The trusted organization context is established before domain authorization.
Users belonging to multiple organizations can switch their active tenant via `/api/v1/auth/switch-tenant`, which verifies membership in the target organization and issues a new tenant-scoped access token without requiring re-authentication.

## Role examples

- `ORG_ADMIN`
- `AGENT_DEVELOPER`
- `AGENT_USER`
- `AUDITOR`

Roles are implementation conveniences; permissions are the authorization contract.
