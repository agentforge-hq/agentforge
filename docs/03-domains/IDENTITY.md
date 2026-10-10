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

External identity provider authenticates the user. AgentForge validates the resulting token and maps claims to an application principal.

## Tenant context

The trusted organization context is established before domain authorization.

## Role examples

- `ORG_ADMIN`
- `AGENT_DEVELOPER`
- `AGENT_USER`
- `AUDITOR`

Roles are implementation conveniences; permissions are the authorization contract.
