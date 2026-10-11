package com.agentforge.common.core.event;

import com.agentforge.common.core.domain.TenantId;

import java.io.Serializable;
import java.time.Instant;
import java.util.Objects;
import java.util.UUID;

/**
 * Standard CloudEvents-compliant domain event envelope.
 * Conforms to docs/04-contracts/EVENTS.md.
 *
 * @param <T> the type of the domain event payload
 */
public record DomainEvent<T>(
        UUID eventId,
        String eventType,
        Instant occurredAt,
        TenantId tenantId,
        String correlationId,
        String causationId,
        String producer,
        T payload
) implements Serializable {

    public DomainEvent {
        Objects.requireNonNull(eventId, "eventId must not be null");
        Objects.requireNonNull(eventType, "eventType must not be null");
        Objects.requireNonNull(occurredAt, "occurredAt must not be null");
        Objects.requireNonNull(correlationId, "correlationId must not be null");
        Objects.requireNonNull(producer, "producer must not be null");
        Objects.requireNonNull(payload, "payload must not be null");
    }

    /**
     * Factory method creating a new DomainEvent with generated eventId and UTC occurredAt.
     */
    public static <T> DomainEvent<T> of(
            String eventType,
            TenantId tenantId,
            String producer,
            String correlationId,
            String causationId,
            T payload
    ) {
        return new DomainEvent<>(
                UUID.randomUUID(),
                eventType,
                Instant.now(),
                tenantId,
                correlationId,
                causationId,
                producer,
                payload
        );
    }

    /**
     * Factory method creating a new DomainEvent without causationId.
     */
    public static <T> DomainEvent<T> of(
            String eventType,
            TenantId tenantId,
            String producer,
            String correlationId,
            T payload
    ) {
        return of(eventType, tenantId, producer, correlationId, null, payload);
    }
}
