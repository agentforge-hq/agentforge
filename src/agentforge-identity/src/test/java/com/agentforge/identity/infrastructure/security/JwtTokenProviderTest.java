package com.agentforge.identity.infrastructure.security;

import com.agentforge.common.core.context.TenantContext;
import com.agentforge.common.core.domain.TenantId;
import io.jsonwebtoken.Claims;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.security.core.Authentication;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;

class JwtTokenProviderTest {

    private JwtTokenProvider tokenProvider;

    @BeforeEach
    void setUp() {
        tokenProvider = new JwtTokenProvider("", "", "test-key-id", 60);
    }

    @Test
    @DisplayName("Should generate a valid RS256 token and successfully validate and parse it")
    void shouldGenerateAndValidateToken() {
        UUID userId = UUID.randomUUID();
        String email = "alice@example.com";
        TenantId tenantId = TenantId.random();
        Set<String> roles = Set.of("ORG_ADMIN");
        Set<String> permissions = Set.of("org:read", "org:update");

        String token = tokenProvider.generateToken(userId, email, tenantId, roles, permissions);

        assertThat(token).isNotBlank();
        assertThat(tokenProvider.validateToken(token)).isTrue();

        Claims claims = tokenProvider.parseClaims(token);
        assertThat(claims.getSubject()).isEqualTo(userId.toString());
        assertThat(claims.get("email", String.class)).isEqualTo(email);
        assertThat(claims.get("tenant_id", String.class)).isEqualTo(tenantId.asString());

        TenantContext tenantContext = tokenProvider.createTenantContext(claims);
        assertThat(tenantContext.tenantId()).isEqualTo(tenantId);
        assertThat(tenantContext.userId()).isEqualTo(userId.toString());
        assertThat(tenantContext.roles()).containsExactly("ORG_ADMIN");
        assertThat(tenantContext.permissions()).containsExactlyInAnyOrder("org:read", "org:update");

        Authentication authentication = tokenProvider.createAuthentication(claims);
        assertThat(authentication.getName()).isEqualTo(userId.toString());
        assertThat(authentication.getAuthorities())
                .extracting("authority")
                .contains("ROLE_ORG_ADMIN", "org:read", "org:update");
    }

    @Test
    @DisplayName("Should reject invalid or tampered tokens")
    void shouldRejectInvalidToken() {
        assertThat(tokenProvider.validateToken("invalid.token.string")).isFalse();
        assertThat(tokenProvider.validateToken("")).isFalse();
    }

    @Test
    @DisplayName("Should generate standard RFC 7517 JWKS output")
    @SuppressWarnings("unchecked")
    void shouldGenerateJwks() {
        Map<String, Object> jwks = tokenProvider.getJwkSet();

        assertThat(jwks).containsKey("keys");
        List<Map<String, Object>> keys = (List<Map<String, Object>>) jwks.get("keys");
        assertThat(keys).hasSize(1);

        Map<String, Object> key = keys.getFirst();
        assertThat(key.get("kty")).isEqualTo("RSA");
        assertThat(key.get("use")).isEqualTo("sig");
        assertThat(key.get("alg")).isEqualTo("RS256");
        assertThat(key.get("kid")).isEqualTo("test-key-id");
        assertThat(key.get("n")).isNotNull();
        assertThat(key.get("e")).isNotNull();
    }
}
