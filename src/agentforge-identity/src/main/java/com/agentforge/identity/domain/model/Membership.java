package com.agentforge.identity.domain.model;

import com.agentforge.common.core.domain.Auditable;
import com.agentforge.common.core.domain.TenantId;

import java.time.Instant;
import java.util.Collections;
import java.util.HashSet;
import java.util.Objects;
import java.util.Optional;
import java.util.Set;
import java.util.UUID;
import java.util.stream.Collectors;

/**
 * Membership associates a User with an Organization and holds role assignments.
 * Invariant: A membership belongs to exactly one organization.
 * Conforms to docs/03-domains/IDENTITY.md.
 */
public class Membership implements Auditable {

    private final UUID id;
    private final TenantId tenantId;
    private final UUID userId;
    private final Set<Role> roles;
    private final Instant createdAt;
    private final String createdBy;
    private Instant updatedAt;
    private String updatedBy;

    public Membership(
            UUID id,
            TenantId tenantId,
            UUID userId,
            Set<Role> roles,
            Instant createdAt,
            String createdBy,
            Instant updatedAt,
            String updatedBy
    ) {
        this.id = Objects.requireNonNull(id, "Membership ID must not be null");
        this.tenantId = Objects.requireNonNull(tenantId, "TenantId must not be null");
        this.userId = Objects.requireNonNull(userId, "UserId must not be null");
        this.roles = (roles == null || roles.isEmpty()) ? new HashSet<>(Set.of(Role.AGENT_USER)) : new HashSet<>(roles);
        this.createdAt = Objects.requireNonNull(createdAt, "createdAt must not be null");
        this.createdBy = Objects.requireNonNull(createdBy, "createdBy must not be null");
        this.updatedAt = updatedAt;
        this.updatedBy = updatedBy;
    }

    public static Membership create(TenantId tenantId, UUID userId, Set<Role> roles, String createdBy) {
        Instant now = Instant.now();
        return new Membership(
                UUID.randomUUID(),
                tenantId,
                userId,
                roles,
                now,
                createdBy,
                now,
                createdBy
        );
    }

    public void addRole(Role role, String updatedBy) {
        Objects.requireNonNull(role, "Role must not be null");
        this.roles.add(role);
        this.updatedAt = Instant.now();
        this.updatedBy = Objects.requireNonNull(updatedBy, "updatedBy must not be null");
    }

    public void removeRole(Role role, String updatedBy) {
        Objects.requireNonNull(role, "Role must not be null");
        if (this.roles.size() <= 1 && this.roles.contains(role)) {
            throw new IllegalStateException("Cannot remove the last role from a membership.");
        }
        this.roles.remove(role);
        this.updatedAt = Instant.now();
        this.updatedBy = Objects.requireNonNull(updatedBy, "updatedBy must not be null");
    }

    public boolean hasRole(Role role) {
        return this.roles.contains(role);
    }

    public boolean hasPermission(Permission permission) {
        return this.roles.stream().anyMatch(role -> role.hasPermission(permission));
    }

    public Set<Permission> getAllPermissions() {
        return this.roles.stream()
                .flatMap(role -> role.getPermissions().stream())
                .collect(Collectors.toUnmodifiableSet());
    }

    public UUID getId() {
        return id;
    }

    public TenantId getTenantId() {
        return tenantId;
    }

    public UUID getUserId() {
        return userId;
    }

    public Set<Role> getRoles() {
        return Collections.unmodifiableSet(roles);
    }

    @Override
    public Instant getCreatedAt() {
        return createdAt;
    }

    @Override
    public String getCreatedBy() {
        return createdBy;
    }

    @Override
    public Optional<Instant> getUpdatedAt() {
        return Optional.ofNullable(updatedAt);
    }

    @Override
    public Optional<String> getUpdatedBy() {
        return Optional.ofNullable(updatedBy);
    }
}
