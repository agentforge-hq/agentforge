package com.agentforge.identity.infrastructure.persistence.entity;

import com.agentforge.common.core.domain.TenantId;
import com.agentforge.identity.domain.model.Organization;
import com.agentforge.identity.domain.model.OrganizationStatus;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import jakarta.persistence.Version;

import java.time.Instant;
import java.util.UUID;

@Entity
@Table(name = "organizations")
public class OrganizationEntity {

    @Id
    private UUID id;

    @Column(nullable = false)
    private String name;

    @Column(nullable = false, unique = true)
    private String slug;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private OrganizationStatus status;

    @Version
    private Long version;

    @Column(name = "created_at", nullable = false, updatable = false)
    private Instant createdAt;

    @Column(name = "created_by", nullable = false, updatable = false)
    private String createdBy;

    @Column(name = "updated_at")
    private Instant updatedAt;

    @Column(name = "updated_by")
    private String updatedBy;

    protected OrganizationEntity() {
    }

    public OrganizationEntity(
            UUID id,
            String name,
            String slug,
            OrganizationStatus status,
            Instant createdAt,
            String createdBy,
            Instant updatedAt,
            String updatedBy
    ) {
        this.id = id;
        this.name = name;
        this.slug = slug;
        this.status = status;
        this.createdAt = createdAt;
        this.createdBy = createdBy;
        this.updatedAt = updatedAt;
        this.updatedBy = updatedBy;
    }

    public static OrganizationEntity fromDomain(Organization org) {
        return new OrganizationEntity(
                org.getId().value(),
                org.getName(),
                org.getSlug(),
                org.getStatus(),
                org.getCreatedAt(),
                org.getCreatedBy(),
                org.getUpdatedAt().orElse(null),
                org.getUpdatedBy().orElse(null)
        );
    }

    public Organization toDomain() {
        return new Organization(
                TenantId.of(this.id),
                this.name,
                this.slug,
                this.status,
                this.createdAt,
                this.createdBy,
                this.updatedAt,
                this.updatedBy
        );
    }

    public UUID getId() {
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

    public Long getVersion() {
        return version;
    }

    public Instant getCreatedAt() {
        return createdAt;
    }

    public String getCreatedBy() {
        return createdBy;
    }

    public Instant getUpdatedAt() {
        return updatedAt;
    }

    public String getUpdatedBy() {
        return updatedBy;
    }
}
