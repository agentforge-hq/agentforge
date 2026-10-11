package com.agentforge.common.core.domain;

import java.time.Instant;
import java.util.Optional;

public interface Auditable {
    Instant getCreatedAt();

    String getCreatedBy();

    Optional<Instant> getUpdatedAt();

    Optional<String> getUpdatedBy();
}
