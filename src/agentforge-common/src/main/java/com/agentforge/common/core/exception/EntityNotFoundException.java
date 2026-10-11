package com.agentforge.common.core.exception;

import java.util.Map;

/**
 * Thrown when a requested domain entity cannot be found.
 */
public class EntityNotFoundException extends AgentForgeException {

    public EntityNotFoundException(ErrorCode errorCode, String message) {
        super(errorCode, message);
    }

    public EntityNotFoundException(ErrorCode errorCode, String entityType, String identifier) {
        super(errorCode, String.format("%s was not found with identifier: %s", entityType, identifier),
                Map.of("entityType", entityType, "identifier", identifier));
    }

    public EntityNotFoundException(String message) {
        super(ErrorCode.AGENT_NOT_FOUND, message);
    }
}
