package com.agentforge.common.core.exception;

import java.util.Map;

/**
 * Thrown when domain or input validation rules fail.
 */
public class ValidationException extends AgentForgeException {

    public ValidationException(String message) {
        super(ErrorCode.VALIDATION_ERROR, message);
    }

    public ValidationException(String message, Map<String, Object> details) {
        super(ErrorCode.VALIDATION_ERROR, message, details);
    }

    public ValidationException(ErrorCode errorCode, String message, Map<String, Object> details) {
        super(errorCode, message, details);
    }
}
