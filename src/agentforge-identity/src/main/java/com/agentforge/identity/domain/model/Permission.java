package com.agentforge.identity.domain.model;

import com.agentforge.common.core.security.SecurityConstants;

import java.util.Arrays;
import java.util.Optional;

/**
 * Atomic authorization permissions.
 * Stable public security identifiers conforming to docs/03-domains/IDENTITY.md.
 * Values are bound directly to com.agentforge.common.core.security.SecurityConstants.
 */
public enum Permission {

    // Agent operations
    AGENT_READ(SecurityConstants.PERM_AGENT_READ),
    AGENT_WRITE(SecurityConstants.PERM_AGENT_WRITE),
    AGENT_PUBLISH(SecurityConstants.PERM_AGENT_PUBLISH),

    // Execution operations
    EXECUTION_RUN(SecurityConstants.PERM_EXECUTION_RUN),
    EXECUTION_CANCEL(SecurityConstants.PERM_EXECUTION_CANCEL),

    // Knowledge & Vector operations
    KNOWLEDGE_READ(SecurityConstants.PERM_KNOWLEDGE_READ),
    KNOWLEDGE_WRITE(SecurityConstants.PERM_KNOWLEDGE_WRITE),

    // Organization & Tenant operations
    ORG_READ(SecurityConstants.PERM_ORG_READ),
    ORG_UPDATE(SecurityConstants.PERM_ORG_UPDATE),
    ORG_CREATE(SecurityConstants.PERM_ORG_CREATE),
    ORG_DELETE(SecurityConstants.PERM_ORG_DELETE),
    MEMBER_MANAGE(SecurityConstants.PERM_MEMBER_MANAGE),

    // API Key management
    API_KEY_MANAGE(SecurityConstants.PERM_API_KEY_MANAGE);

    private final String value;

    Permission(String value) {
        this.value = value;
    }

    public String getValue() {
        return value;
    }

    public static Optional<Permission> fromValue(String value) {
        if (value == null) {
            return Optional.empty();
        }
        return Arrays.stream(values())
                .filter(p -> p.value.equalsIgnoreCase(value))
                .findFirst();
    }
}