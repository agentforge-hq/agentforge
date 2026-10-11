package com.agentforge.common.core.domain;

import java.io.Serializable;
import java.util.UUID;

/**
 * Marker and contract interface for all strongly typed domain identifiers.
 */
public interface EntityId extends Serializable {

    /**
     * Returns the underlying UUID value.
     */
    UUID value();

    /**
     * Returns the canonical string representation of the identifier.
     */
    default String asString() {
        return value().toString();
    }
}
