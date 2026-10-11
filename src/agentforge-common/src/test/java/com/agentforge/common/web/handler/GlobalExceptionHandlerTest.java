package com.agentforge.common.web.handler;

import com.agentforge.common.core.exception.EntityNotFoundException;
import com.agentforge.common.core.exception.ErrorCode;
import com.agentforge.common.core.exception.TenantAccessDeniedException;
import com.agentforge.common.web.filter.CorrelationIdFilter;
import com.agentforge.common.web.response.ErrorResponse;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.slf4j.MDC;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;

import static org.assertj.core.api.Assertions.assertThat;

class GlobalExceptionHandlerTest {

    private final GlobalExceptionHandler handler = new GlobalExceptionHandler();

    @AfterEach
    void cleanup() {
        MDC.clear();
    }

    @Test
    @DisplayName("should map EntityNotFoundException to 404 ErrorResponse")
    void shouldHandleEntityNotFoundException() {
        MDC.put(CorrelationIdFilter.MDC_CORRELATION_ID_KEY, "test-corr-id");
        EntityNotFoundException ex = new EntityNotFoundException(ErrorCode.AGENT_NOT_FOUND, "Agent not found");

        ResponseEntity<ErrorResponse> response = handler.handleAgentForgeException(ex);

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.NOT_FOUND);
        assertThat(response.getBody()).isNotNull();
        assertThat(response.getBody().code()).isEqualTo(ErrorCode.AGENT_NOT_FOUND.name());
        assertThat(response.getBody().message()).isEqualTo("Agent not found");
        assertThat(response.getBody().correlationId()).isEqualTo("test-corr-id");
    }

    @Test
    @DisplayName("should map TenantAccessDeniedException to 403 ErrorResponse")
    void shouldHandleTenantAccessDeniedException() {
        TenantAccessDeniedException ex = new TenantAccessDeniedException("Access denied to tenant");

        ResponseEntity<ErrorResponse> response = handler.handleAgentForgeException(ex);

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.FORBIDDEN);
        assertThat(response.getBody()).isNotNull();
        assertThat(response.getBody().code()).isEqualTo(ErrorCode.TENANT_ACCESS_DENIED.name());
    }

    @Test
    @DisplayName("should map unhandled Exception to 500 without leaking stack trace")
    void shouldHandleUnhandledExceptionSafely() {
        Exception ex = new RuntimeException("Database connection timeout or internal detail");

        ResponseEntity<ErrorResponse> response = handler.handleUnhandledException(ex);

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.INTERNAL_SERVER_ERROR);
        assertThat(response.getBody()).isNotNull();
        assertThat(response.getBody().code()).isEqualTo(ErrorCode.INTERNAL_ERROR.name());
        assertThat(response.getBody().message()).doesNotContain("Database connection");
        assertThat(response.getBody().details()).isEmpty();
    }
}
