package com.agentforge.identity.domain.model;

import com.agentforge.common.core.domain.Auditable;

import java.time.Instant;
import java.util.Objects;
import java.util.Optional;
import java.util.UUID;

/**
 * User represents an authenticated human identity.
 * Conforms to docs/03-domains/IDENTITY.md.
 */
public class User implements Auditable {

    private final UUID id;
    private final String email;
    private String passwordHash;
    private String fullName;
    private UserStatus status;
    private final Instant createdAt;
    private final String createdBy;
    private Instant updatedAt;
    private String updatedBy;

    public User(
            UUID id,
            String email,
            String passwordHash,
            String fullName,
            UserStatus status,
            Instant createdAt,
            String createdBy,
            Instant updatedAt,
            String updatedBy
    ) {
        this.id = Objects.requireNonNull(id, "User ID must not be null");
        this.email = validateEmail(email);
        this.passwordHash = Objects.requireNonNull(passwordHash, "passwordHash must not be null");
        this.fullName = validateNotBlank(fullName, "fullName must not be blank");
        this.status = Objects.requireNonNull(status, "UserStatus must not be null");
        this.createdAt = Objects.requireNonNull(createdAt, "createdAt must not be null");
        this.createdBy = Objects.requireNonNull(createdBy, "createdBy must not be null");
        this.updatedAt = updatedAt;
        this.updatedBy = updatedBy;
    }

    public static User register(String email, String passwordHash, String fullName) {
        Instant now = Instant.now();
        UUID userId = UUID.randomUUID();
        return new User(
                userId,
                email,
                passwordHash,
                fullName,
                UserStatus.ACTIVE,
                now,
                userId.toString(),
                now,
                userId.toString()
        );
    }

    public void changePassword(String newPasswordHash, String updatedBy) {
        this.passwordHash = Objects.requireNonNull(newPasswordHash, "newPasswordHash must not be null");
        this.updatedAt = Instant.now();
        this.updatedBy = Objects.requireNonNull(updatedBy, "updatedBy must not be null");
    }

    public void updateProfile(String newFullName, String updatedBy) {
        this.fullName = validateNotBlank(newFullName, "fullName must not be blank");
        this.updatedAt = Instant.now();
        this.updatedBy = Objects.requireNonNull(updatedBy, "updatedBy must not be null");
    }

    public void lock(String updatedBy) {
        this.status = UserStatus.LOCKED;
        this.updatedAt = Instant.now();
        this.updatedBy = Objects.requireNonNull(updatedBy, "updatedBy must not be null");
    }

    public void activate(String updatedBy) {
        this.status = UserStatus.ACTIVE;
        this.updatedAt = Instant.now();
        this.updatedBy = Objects.requireNonNull(updatedBy, "updatedBy must not be null");
    }

    public boolean isActive() {
        return this.status == UserStatus.ACTIVE;
    }

    public UUID getId() {
        return id;
    }

    public String getEmail() {
        return email;
    }

    public String getPasswordHash() {
        return passwordHash;
    }

    public String getFullName() {
        return fullName;
    }

    public UserStatus getStatus() {
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

    private static String validateEmail(String email) {
        String cleanEmail = validateNotBlank(email, "Email must not be blank").toLowerCase();
        if (!cleanEmail.contains("@") || !cleanEmail.contains(".")) {
            throw new IllegalArgumentException("Invalid email format: " + email);
        }
        return cleanEmail;
    }
}
