package com.agentforge.common.core.exception;

import java.util.Map;

/**
 * Thrown when an unauthenticated caller attempts to access a protected resource
 * or when provided credentials are invalid or expired (HTTP 401).
 */
public class AuthenticationRequiredException extends AgentForgeException {

    public AuthenticationRequiredException(String message) {
        super(ErrorCode.AUTHENTICATION_REQUIRED, message);
    }

    public AuthenticationRequiredException(ErrorCode errorCode, String message) {
        super(errorCode, message);
    }

    public AuthenticationRequiredException(ErrorCode errorCode, String message, Map<String, Object> details) {
        super(errorCode, message, details);
    }
}
