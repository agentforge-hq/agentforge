package com.agentforge.common.core.exception;

import com.agentforge.common.core.domain.TenantId;

import java.util.Map;

/**
 * Thrown when an authenticated principal attempts to access or modify resources belonging to another tenant.
 */
public class TenantAccessDeniedException extends AgentForgeException {

    public TenantAccessDeniedException(String message) {
        super(ErrorCode.TENANT_ACCESS_DENIED, message);
    }

    public TenantAccessDeniedException(TenantId targetTenantId) {
        super(ErrorCode.TENANT_ACCESS_DENIED,
                "Access denied for tenant: " + (targetTenantId != null ? targetTenantId.asString() : "unknown"),
                targetTenantId != null ? Map.of("targetTenantId", targetTenantId.asString()) : Map.of());
    }

    public TenantAccessDeniedException(String message, Map<String, Object> details) {
        super(ErrorCode.TENANT_ACCESS_DENIED, message, details);
    }
}
