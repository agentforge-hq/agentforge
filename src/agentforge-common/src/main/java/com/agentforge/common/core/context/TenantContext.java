package com.agentforge.common.core.context;

import java.util.Objects;
import java.util.Optional;
import java.util.Set;

import com.agentforge.common.core.domain.TenantId;

public record TenantContext(
        TenantId tenantId,
        String userId,
        Set<String> roles,
        Set<String> permissions,
        boolean isSystem) {

    public TenantContext {
        roles = (roles == null) ? Set.of() : Set.copyOf(roles);
        permissions = (permissions == null) ? Set.of() : Set.copyOf(permissions);
    }

    /**
     * Checks if this context is scoped to a specific tenant.
     */
    public boolean isTenantScoped() {
        return tenantId != null;
    }

    /**
     * Returns the TenantId if present.
     */
    public Optional<TenantId> getTenantId() {
        return Optional.ofNullable(tenantId);
    }

    /**
     * Returns the TenantId or throws an exception if the operation requires a
     * tenant.
     * Prevents cross-tenant leaks or operations running without tenant scope.
     */
    public TenantId getRequiredTenantId() {
        if (tenantId == null) {
            throw new IllegalStateException("Tenant context is required for this operation, but none was found.");
        }
        return tenantId;
    }

    public Optional<String> getUserId() {
        return Optional.ofNullable(userId);
    }

    public boolean hasRole(String role) {
        return roles.contains(role);
    }

    public boolean hasPermission(String permission) {
        return permissions.contains(permission);
    }

    // --- Factory Helpers ---

    /**
     * Factory for normal tenant-scoped user requests.
     */
    public static TenantContext of(TenantId tenantId, String userId, Set<String> roles, Set<String> permissions) {
        Objects.requireNonNull(tenantId, "tenantId must not be null for tenant-scoped context");
        return new TenantContext(tenantId, userId, roles, permissions, false);
    }

    /**
     * Factory for background jobs running within a specific tenant.
     */
    public static TenantContext forTenantSystem(TenantId tenantId) {
        Objects.requireNonNull(tenantId, "tenantId must not be null");
        return new TenantContext(tenantId, null, Set.of("SYSTEM"), Set.of("*"), true);
    }

    /**
     * Factory for global/platform system background jobs (no tenant).
     */
    public static TenantContext globalSystem() {
        return new TenantContext(null, null, Set.of("SYSTEM"), Set.of("*"), true);
    }
}
