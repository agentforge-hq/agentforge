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

/**
 * ApiKey represents a credential for programmatic access scoped to an organization.
 * Invariant: Revoked or expired API keys cannot authenticate.
 * Conforms to docs/03-domains/IDENTITY.md.
 */
public class ApiKey implements Auditable {

    private final UUID id;
    private final TenantId tenantId;
    private final String name;
    private final String keyPrefix;
    private final String hashedSecret;
    private final Set<String> permissions;
    private final Instant expiresAt;
    private Instant revokedAt;
    private String revokedBy;
    private final Instant createdAt;
    private final String createdBy;

    public ApiKey(
            UUID id,
            TenantId tenantId,
            String name,
            String keyPrefix,
            String hashedSecret,
            Set<String> permissions,
            Instant expiresAt,
            Instant revokedAt,
            String revokedBy,
            Instant createdAt,
            String createdBy
    ) {
        this.id = Objects.requireNonNull(id, "ApiKey ID must not be null");
        this.tenantId = Objects.requireNonNull(tenantId, "TenantId must not be null");
        this.name = validateNotBlank(name, "ApiKey name must not be blank");
        this.keyPrefix = validateNotBlank(keyPrefix, "keyPrefix must not be blank");
        this.hashedSecret = Objects.requireNonNull(hashedSecret, "hashedSecret must not be null");
        this.permissions = permissions != null ? new HashSet<>(permissions) : new HashSet<>();
        this.expiresAt = expiresAt;
        this.revokedAt = revokedAt;
        this.revokedBy = revokedBy;
        this.createdAt = Objects.requireNonNull(createdAt, "createdAt must not be null");
        this.createdBy = Objects.requireNonNull(createdBy, "createdBy must not be null");
    }

    public static ApiKey create(
            TenantId tenantId,
            String name,
            String keyPrefix,
            String hashedSecret,
            Set<String> permissions,
            Instant expiresAt,
            String createdBy
    ) {
        Instant now = Instant.now();
        return new ApiKey(
                UUID.randomUUID(),
                tenantId,
                name,
                keyPrefix,
                hashedSecret,
                permissions,
                expiresAt,
                null,
                null,
                now,
                createdBy
        );
    }

    public boolean isValid() {
        if (revokedAt != null) {
            return false;
        }
        return expiresAt == null || expiresAt.isAfter(Instant.now());
    }

    public void revoke(String revokedBy) {
        if (this.revokedAt != null) {
            throw new IllegalStateException("ApiKey is already revoked.");
        }
        this.revokedAt = Instant.now();
        this.revokedBy = Objects.requireNonNull(revokedBy, "revokedBy must not be null");
    }

    public boolean hasPermission(String permission) {
        return this.permissions.contains(permission) || this.permissions.contains("*");
    }

    public UUID getId() {
        return id;
    }

    public TenantId getTenantId() {
        return tenantId;
    }

    public String getName() {
        return name;
    }

    public String getKeyPrefix() {
        return keyPrefix;
    }

    public String getHashedSecret() {
        return hashedSecret;
    }

    public Set<String> getPermissions() {
        return Collections.unmodifiableSet(permissions);
    }

    public Optional<Instant> getExpiresAt() {
        return Optional.ofNullable(expiresAt);
    }

    public Optional<Instant> getRevokedAt() {
        return Optional.ofNullable(revokedAt);
    }

    public Optional<String> getRevokedBy() {
        return Optional.ofNullable(revokedBy);
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
        return Optional.ofNullable(revokedAt);
    }

    @Override
    public Optional<String> getUpdatedBy() {
        return Optional.ofNullable(revokedBy);
    }

    private static String validateNotBlank(String value, String message) {
        if (value == null || value.trim().isEmpty()) {
            throw new IllegalArgumentException(message);
        }
        return value.trim();
    }
}
