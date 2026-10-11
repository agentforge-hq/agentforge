package com.agentforge.common.core.event;

import com.agentforge.common.core.domain.TenantId;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.time.Instant;
import java.util.Map;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class DomainEventTest {

    @Test
    @DisplayName("should create DomainEvent with factory method")
    void shouldCreateDomainEvent() {
        TenantId tenantId = TenantId.random();
        Map<String, String> payload = Map.of("key", "value");

        DomainEvent<Map<String, String>> event = DomainEvent.of(
                "agent.execution.started.v1",
                tenantId,
                "agent-execution",
                "corr-123",
                payload
        );

        assertThat(event.eventId()).isNotNull();
        assertThat(event.eventType()).isEqualTo("agent.execution.started.v1");
        assertThat(event.occurredAt()).isBeforeOrEqualTo(Instant.now());
        assertThat(event.tenantId()).isEqualTo(tenantId);
        assertThat(event.correlationId()).isEqualTo("corr-123");
        assertThat(event.causationId()).isNull();
        assertThat(event.producer()).isEqualTo("agent-execution");
        assertThat(event.payload()).isEqualTo(payload);
    }

    @Test
    @DisplayName("should enforce required fields in envelope")
    void shouldEnforceRequiredFields() {
        assertThatThrownBy(() -> new DomainEvent<>(
                null,
                "type",
                Instant.now(),
                TenantId.random(),
                "corr",
                null,
                "producer",
                "payload"
        )).isInstanceOf(NullPointerException.class).hasMessageContaining("eventId");
    }
}
