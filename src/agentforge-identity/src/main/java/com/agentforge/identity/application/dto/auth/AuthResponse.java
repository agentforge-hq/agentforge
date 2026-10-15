package com.agentforge.identity.application.dto.auth;

import java.util.Set;
import java.util.UUID;

public record AuthResponse(
        String token,
        UUID userId,
        UUID tenantId,
        String email,
        Set<String> roles
) {
}
