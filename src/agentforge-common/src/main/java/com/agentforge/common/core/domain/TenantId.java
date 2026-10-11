package com.agentforge.common.core.domain;

import java.util.Objects;
import java.util.UUID;

public record TenantId(UUID value) implements EntityId, Comparable<TenantId> {

    public TenantId {
        Objects.requireNonNull(value, "TenantId value cannot be null");
    }

    public static TenantId of(UUID value) {
        return new TenantId(value);
    }

    public static TenantId fromString(String raw) {
        Objects.requireNonNull(raw, "TenantId string must not be null");
        try {
            return new TenantId(UUID.fromString(raw));
        } catch (IllegalArgumentException e) {
            throw new IllegalArgumentException("Invalid UUID string for TenantId: " + raw, e);
        }
    }

    public static TenantId random() {
        return new TenantId(UUID.randomUUID());
    }

    @Override
    public String asString() {
        return value.toString();
    }

    @Override
    public String toString() {
        return value.toString();
    }

    @Override
    public int compareTo(TenantId other) {
        return this.value.compareTo(other.value);
    }
}
