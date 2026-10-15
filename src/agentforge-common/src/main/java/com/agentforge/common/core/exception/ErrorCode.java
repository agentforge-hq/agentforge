package com.agentforge.common.core.exception;

/**
 * Platform-wide machine-readable error codes.
 * Conforms strictly to docs/04-contracts/ERROR_CODES.md.
 */
public enum ErrorCode {

    // --- Identity ---
    AUTHENTICATION_REQUIRED(401, "Authentication is required to perform this operation."),
    UNAUTHORIZED(401, "Authentication failed or credentials are invalid."),
    TOKEN_INVALID(401, "The provided authentication token is invalid or expired."),
    PERMISSION_DENIED(403, "Caller lacks required permission for this operation."),
    TENANT_ACCESS_DENIED(403, "Access to the requested tenant is denied."),
    API_KEY_INVALID(401, "The provided API key is invalid or revoked."),
    USER_ALREADY_EXISTS(409, "User with the specified email already exists."),
    USER_NOT_FOUND(404, "User was not found."),
    ORGANIZATION_ALREADY_EXISTS(409, "Organization with the specified slug already exists."),

    // --- Agent ---
    AGENT_NOT_FOUND(404, "Agent was not found."),
    AGENT_VERSION_NOT_FOUND(404, "Agent version was not found."),
    AGENT_VERSION_IMMUTABLE(409, "Published agent version is immutable and cannot be modified."),
    AGENT_PUBLICATION_INVALID(400, "Agent version does not meet criteria for publication."),
    AGENT_NOT_PUBLISHED(409, "Agent has not been published."),

    // --- Execution ---
    EXECUTION_NOT_FOUND(404, "Execution was not found."),
    EXECUTION_ALREADY_COMPLETED(409, "Execution has already reached a completed state."),
    EXECUTION_CANCELLED(409, "Execution has been cancelled."),
    EXECUTION_LIMIT_EXCEEDED(429, "Execution rate or resource limit has been exceeded."),
    EXECUTION_TIMEOUT(504, "Execution timed out."),

    // --- Tool ---
    TOOL_NOT_FOUND(404, "Tool was not found."),
    TOOL_NOT_AUTHORIZED(403, "Tool execution is not authorized."),
    TOOL_INPUT_INVALID(400, "Tool input arguments are invalid."),
    TOOL_TIMEOUT(504, "Tool execution timed out."),
    TOOL_UNAVAILABLE(503, "Tool is currently unavailable."),

    // --- Knowledge ---
    KNOWLEDGE_BASE_NOT_FOUND(404, "Knowledge base was not found."),
    DOCUMENT_NOT_FOUND(404, "Document was not found."),
    DOCUMENT_PROCESSING_FAILED(500, "Document processing failed."),

    // --- Platform ---
    RATE_LIMITED(429, "Too many requests. Rate limit exceeded."),
    DEPENDENCY_UNAVAILABLE(503, "A required upstream dependency is unavailable."),
    VALIDATION_ERROR(400, "Request validation failed."),
    INTERNAL_ERROR(500, "An unexpected internal server error occurred.");

    private final int httpStatus;
    private final String defaultMessage;

    ErrorCode(int httpStatus, String defaultMessage) {
        this.httpStatus = httpStatus;
        this.defaultMessage = defaultMessage;
    }

    public int getHttpStatus() {
        return httpStatus;
    }

    public String getDefaultMessage() {
        return defaultMessage;
    }
}
