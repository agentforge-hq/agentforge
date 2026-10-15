package com.agentforge.identity.domain.model;

import com.agentforge.common.core.domain.Auditable;
import com.agentforge.common.core.domain.TenantId;

import java.time.Instant;
import java.util.Locale;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/**
 * Organization represents a tenant boundary in AgentForge.
 * Conforms to docs/03-domains/IDENTITY.md.
 */
public class Organization implements Auditable {

    private final TenantId id;
    private String name;
    private final String slug;
    private OrganizationStatus status;
    private final Instant createdAt;
    private final String createdBy;
    private Instant updatedAt;
    private String updatedBy;

    public Organization(
            TenantId id,
            String name,
            String slug,
            OrganizationStatus status,
            Instant createdAt,
            String createdBy,
            Instant updatedAt,
            String updatedBy
    ) {
        this.id = Objects.requireNonNull(id, "TenantId must not be null");
        this.name = validateNotBlank(name, "Organization name must not be blank");
        this.slug = validateSlug(slug);
        this.status = Objects.requireNonNull(status, "OrganizationStatus must not be null");
        this.createdAt = Objects.requireNonNull(createdAt, "createdAt must not be null");
        this.createdBy = Objects.requireNonNull(createdBy, "createdBy must not be null");
        this.updatedAt = updatedAt;
        this.updatedBy = updatedBy;
    }

    public static Organization create(String name, String createdBy) {
        return create(name, slugify(name), createdBy);
    }
    
    public static Organization create(String name, String slug, String createdBy) {
        Instant now = Instant.now();
        return new Organization(
                TenantId.random(),
                name,
                slug,
                OrganizationStatus.ACTIVE,
                now,
                createdBy,
                now,
                createdBy
        );
    }

    public static String slugify(String name) {
        if (name == null || name.isBlank()) {
            return String.format("org-%s", UUID.randomUUID().toString().substring(0, 8));
        }

        String slug = name.trim().toLowerCase(Locale.ENGLISH)
                .replaceAll("[^a-z0-9]+", "-")
                .replaceAll("^-+|-+$", "");

        if (slug.isBlank()) {
            return "org-" + UUID.randomUUID().toString().substring(0, 8);
        }

        return slug;
    }

    public void updateName(String newName, String updatedBy) {
        this.name = validateNotBlank(newName, "Organization name must not be blank");
        this.updatedAt = Instant.now();
        this.updatedBy = Objects.requireNonNull(updatedBy, "updatedBy must not be null");
    }

    public void suspend(String updatedBy) {
        this.status = OrganizationStatus.SUSPENDED;
        this.updatedAt = Instant.now();
        this.updatedBy = Objects.requireNonNull(updatedBy, "updatedBy must not be null");
    }

    public void activate(String updatedBy) {
        this.status = OrganizationStatus.ACTIVE;
        this.updatedAt = Instant.now();
        this.updatedBy = Objects.requireNonNull(updatedBy, "updatedBy must not be null");
    }

    public boolean isActive() {
        return this.status == OrganizationStatus.ACTIVE;
    }

    public TenantId getId() {
        return id;
    }

    public String getName() {
        return name;
    }

    public String getSlug() {
        return slug;
    }

    public OrganizationStatus getStatus() {
        return status;
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

    private static String validateNotBlank(String value, String message) {
        if (value == null || value.trim().isEmpty()) {
            throw new IllegalArgumentException(message);
        }
        return value.trim();
    }

    private static String validateSlug(String slug) {
        String cleanSlug = validateNotBlank(slug, "Organization slug must not be blank").toLowerCase();
        if (!cleanSlug.matches("^[a-z0-9]+(-[a-z0-9]+)*$")) {
            throw new IllegalArgumentException("Invalid slug format. Slug must contain only lowercase alphanumeric characters and hyphens.");
        }
        return cleanSlug;
    }
}
