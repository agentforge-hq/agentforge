package com.agentforge.identity.infrastructure.persistence.entity;

import com.agentforge.common.core.domain.TenantId;
import com.agentforge.identity.domain.model.ApiKey;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Id;
import jakarta.persistence.Table;

import java.time.Instant;
import java.util.Arrays;
import java.util.Set;
import java.util.UUID;
import java.util.stream.Collectors;

@Entity
@Table(name = "api_keys")
public class ApiKeyEntity {

    @Id
    private UUID id;

    @Column(name = "organization_id", nullable = false)
    private UUID organizationId;

    @Column(nullable = false)
    private String name;

    @Column(name = "key_prefix", nullable = false)
    private String keyPrefix;

    @Column(name = "hashed_secret", nullable = false)
    private String hashedSecret;

    @Column(nullable = false, columnDefinition = "TEXT")
    private String permissions;

    @Column(name = "expires_at")
    private Instant expiresAt;

    @Column(name = "revoked_at")
    private Instant revokedAt;

    @Column(name = "revoked_by")
    private String revokedBy;

    @Column(name = "created_at", nullable = false, updatable = false)
    private Instant createdAt;

    @Column(name = "created_by", nullable = false, updatable = false)
    private String createdBy;

    protected ApiKeyEntity() {
    }

    public ApiKeyEntity(
            UUID id,
            UUID organizationId,
            String name,
            String keyPrefix,
            String hashedSecret,
            String permissions,
            Instant expiresAt,
            Instant revokedAt,
            String revokedBy,
            Instant createdAt,
            String createdBy) {
        this.id = id;
        this.organizationId = organizationId;
        this.name = name;
        this.keyPrefix = keyPrefix;
        this.hashedSecret = hashedSecret;
        this.permissions = permissions;
        this.expiresAt = expiresAt;
        this.revokedAt = revokedAt;
        this.revokedBy = revokedBy;
        this.createdAt = createdAt;
        this.createdBy = createdBy;
    }

    public static ApiKeyEntity fromDomain(ApiKey apiKey) {
        String permissionsString = String.join(",", apiKey.getPermissions());
        return new ApiKeyEntity(
                apiKey.getId(),
                apiKey.getTenantId().value(),
                apiKey.getName(),
                apiKey.getKeyPrefix(),
                apiKey.getHashedSecret(),
                permissionsString,
                apiKey.getExpiresAt().orElse(null),
                apiKey.getRevokedAt().orElse(null),
                apiKey.getRevokedBy().orElse(null),
                apiKey.getCreatedAt(),
                apiKey.getCreatedBy());
    }

    public ApiKey toDomain() {
        Set<String> perms = (this.permissions == null || this.permissions.isBlank())
                ? Set.of()
                : Arrays.stream(this.permissions.split(","))
                        .map(String::trim)
                        .filter(s -> !s.isEmpty())
                        .collect(Collectors.toSet());

        return new ApiKey(
                this.id,
                TenantId.of(this.organizationId),
                this.name,
                this.keyPrefix,
                this.hashedSecret,
                perms,
                this.expiresAt,
                this.revokedAt,
                this.revokedBy,
                this.createdAt,
                this.createdBy);
    }

    public UUID getId() {
        return id;
    }

    public UUID getOrganizationId() {
        return organizationId;
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

    public String getPermissions() {
        return permissions;
    }

    public Instant getExpiresAt() {
        return expiresAt;
    }

    public Instant getRevokedAt() {
        return revokedAt;
    }

    public String getRevokedBy() {
        return revokedBy;
    }

    public Instant getCreatedAt() {
        return createdAt;
    }

    public String getCreatedBy() {
        return createdBy;
    }
}
