package com.agentforge.common.core.domain;

import java.time.Instant;
import java.util.Optional;

public interface SoftDeletable {
    default boolean isDeleted() {
        return getDeletedAt().isPresent();
    }

    Optional<Instant> getDeletedAt();

    Optional<String> getDeletedBy();
}
