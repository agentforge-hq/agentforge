package com.agentforge.common.core.domain;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class TenantIdTest {

    @Test
    @DisplayName("should create TenantId from valid UUID")
    void shouldCreateFromUuid() {
        UUID uuid = UUID.randomUUID();
        TenantId tenantId = TenantId.of(uuid);

        assertThat(tenantId.value()).isEqualTo(uuid);
        assertThat(tenantId.asString()).isEqualTo(uuid.toString());
        assertThat(tenantId.toString()).isEqualTo(uuid.toString());
    }

    @Test
    @DisplayName("should parse TenantId from valid UUID string")
    void shouldParseFromString() {
        UUID uuid = UUID.randomUUID();
        TenantId tenantId = TenantId.fromString(uuid.toString());

        assertThat(tenantId.value()).isEqualTo(uuid);
    }

    @Test
    @DisplayName("should fail when UUID is null")
    void shouldThrowWhenNullUuid() {
        assertThatThrownBy(() -> TenantId.of(null))
                .isInstanceOf(NullPointerException.class)
                .hasMessageContaining("cannot be null");
    }

    @Test
    @DisplayName("should fail when UUID string is malformed")
    void shouldThrowWhenInvalidString() {
        assertThatThrownBy(() -> TenantId.fromString("not-a-uuid"))
                .isInstanceOf(IllegalArgumentException.class)
                .hasMessageContaining("Invalid UUID");
    }

    @Test
    @DisplayName("should generate random TenantId")
    void shouldGenerateRandom() {
        TenantId t1 = TenantId.random();
        TenantId t2 = TenantId.random();

        assertThat(t1).isNotNull();
        assertThat(t2).isNotNull();
        assertThat(t1).isNotEqualTo(t2);
    }
}
