package com.agentforge.common.core.exception;

import java.util.Map;

/**
 * Thrown when an authenticated caller lacks the required role or permission (HTTP 403).
 */
public class PermissionDeniedException extends AgentForgeException {

    public PermissionDeniedException(String message) {
        super(ErrorCode.PERMISSION_DENIED, message);
    }

    public PermissionDeniedException(String message, Map<String, Object> details) {
        super(ErrorCode.PERMISSION_DENIED, message, details);
    }

    public PermissionDeniedException(ErrorCode errorCode, String message, Map<String, Object> details) {
        super(errorCode, message, details);
    }
}
