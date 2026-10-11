package com.agentforge.common.core.exception;

import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Objects;

/**
 * Base runtime exception for all AgentForge domain and platform errors.
 */
public class AgentForgeException extends RuntimeException {

    private final ErrorCode errorCode;
    private final Map<String, Object> details;
    private String correlationId;

    public AgentForgeException(ErrorCode errorCode) {
        this(errorCode, errorCode.getDefaultMessage(), Collections.emptyMap(), null);
    }

    public AgentForgeException(ErrorCode errorCode, String message) {
        this(errorCode, message, Collections.emptyMap(), null);
    }

    public AgentForgeException(ErrorCode errorCode, String message, Throwable cause) {
        this(errorCode, message, Collections.emptyMap(), cause);
    }

    public AgentForgeException(ErrorCode errorCode, String message, Map<String, Object> details) {
        this(errorCode, message, details, null);
    }

    public AgentForgeException(ErrorCode errorCode, String message, Map<String, Object> details, Throwable cause) {
        super(message, cause);
        this.errorCode = Objects.requireNonNull(errorCode, "errorCode must not be null");
        this.details = details != null ? Collections.unmodifiableMap(new LinkedHashMap<>(details)) : Collections.emptyMap();
    }

    public ErrorCode getErrorCode() {
        return errorCode;
    }

    public int getHttpStatus() {
        return errorCode.getHttpStatus();
    }

    public Map<String, Object> getDetails() {
        return details;
    }

    public String getCorrelationId() {
        return correlationId;
    }

    public AgentForgeException withCorrelationId(String correlationId) {
        this.correlationId = correlationId;
        return this;
    }
}
