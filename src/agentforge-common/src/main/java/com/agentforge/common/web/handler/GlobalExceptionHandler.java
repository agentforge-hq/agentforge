package com.agentforge.common.web.handler;

import com.agentforge.common.core.exception.AgentForgeException;
import com.agentforge.common.core.exception.ErrorCode;
import com.agentforge.common.web.filter.CorrelationIdFilter;
import com.agentforge.common.web.response.ErrorResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.MDC;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.validation.FieldError;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Global exception handler transforming exceptions into standard ErrorResponse payloads.
 * Conforms to docs/04-contracts/REST_API.md and ERROR_CODES.md.
 */
@RestControllerAdvice
public class GlobalExceptionHandler {

    private static final Logger log = LoggerFactory.getLogger(GlobalExceptionHandler.class);

    @ExceptionHandler(AgentForgeException.class)
    public ResponseEntity<ErrorResponse> handleAgentForgeException(AgentForgeException ex) {
        String correlationId = resolveCorrelationId(ex.getCorrelationId());
        log.warn("Handled business exception [{}]: {} (correlationId={})",
                ex.getErrorCode(), ex.getMessage(), correlationId);

        ErrorResponse errorResponse = ErrorResponse.of(
                ex.getErrorCode(),
                ex.getMessage(),
                correlationId,
                ex.getDetails()
        );

        return ResponseEntity.status(ex.getHttpStatus()).body(errorResponse);
    }

    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ResponseEntity<ErrorResponse> handleMethodArgumentNotValid(MethodArgumentNotValidException ex) {
        String correlationId = resolveCorrelationId(null);
        Map<String, Object> fieldErrors = new LinkedHashMap<>();

        for (FieldError fieldError : ex.getBindingResult().getFieldErrors()) {
            fieldErrors.put(fieldError.getField(), fieldError.getDefaultMessage());
        }

        log.warn("Request validation failed: {} (correlationId={})", fieldErrors, correlationId);

        ErrorResponse errorResponse = ErrorResponse.of(
                ErrorCode.VALIDATION_ERROR,
                "Request validation failed.",
                correlationId,
                fieldErrors
        );

        return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(errorResponse);
    }

    @ExceptionHandler(IllegalArgumentException.class)
    public ResponseEntity<ErrorResponse> handleIllegalArgumentException(IllegalArgumentException ex) {
        String correlationId = resolveCorrelationId(null);
        log.warn("Illegal argument: {} (correlationId={})", ex.getMessage(), correlationId);

        ErrorResponse errorResponse = ErrorResponse.of(
                ErrorCode.VALIDATION_ERROR,
                ex.getMessage(),
                correlationId,
                Map.of()
        );

        return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(errorResponse);
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<ErrorResponse> handleUnhandledException(Exception ex) {
        String correlationId = resolveCorrelationId(null);
        // Log full stack trace server-side for observability, but NEVER leak it to client
        log.error("Unhandled server exception (correlationId={}): ", correlationId, ex);

        ErrorResponse errorResponse = ErrorResponse.of(
                ErrorCode.INTERNAL_ERROR,
                ErrorCode.INTERNAL_ERROR.getDefaultMessage(),
                correlationId,
                Map.of()
        );

        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).body(errorResponse);
    }

    private String resolveCorrelationId(String explicitCorrelationId) {
        if (explicitCorrelationId != null && !explicitCorrelationId.isBlank()) {
            return explicitCorrelationId;
        }
        String mdcCorrelationId = MDC.get(CorrelationIdFilter.MDC_CORRELATION_ID_KEY);
        return mdcCorrelationId != null ? mdcCorrelationId : "unknown";
    }
}
