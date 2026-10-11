package com.agentforge.common.web.filter;

import com.agentforge.common.core.context.TenantContextHolder;
import com.agentforge.common.core.domain.TenantId;
import com.agentforge.common.core.exception.ValidationException;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.slf4j.MDC;
import org.springframework.mock.web.MockFilterChain;
import org.springframework.mock.web.MockHttpServletRequest;
import org.springframework.mock.web.MockHttpServletResponse;

import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class WebFilterTest {

    @BeforeEach
    @AfterEach
    void cleanup() {
        TenantContextHolder.clear();
        MDC.clear();
    }

    @Test
    @DisplayName("CorrelationIdFilter should propagate existing X-Correlation-Id header")
    void shouldPropagateExistingCorrelationId() throws Exception {
        CorrelationIdFilter filter = new CorrelationIdFilter();
        MockHttpServletRequest request = new MockHttpServletRequest();
        request.addHeader(CorrelationIdFilter.CORRELATION_ID_HEADER, "custom-corr-id");
        MockHttpServletResponse response = new MockHttpServletResponse();
        MockFilterChain filterChain = new MockFilterChain();

        filter.doFilter(request, response, filterChain);

        assertThat(response.getHeader(CorrelationIdFilter.CORRELATION_ID_HEADER)).isEqualTo("custom-corr-id");
        // Must be removed from MDC after request completes
        assertThat(MDC.get(CorrelationIdFilter.MDC_CORRELATION_ID_KEY)).isNull();
    }

    @Test
    @DisplayName("CorrelationIdFilter should generate X-Correlation-Id header when absent")
    void shouldGenerateCorrelationIdWhenAbsent() throws Exception {
        CorrelationIdFilter filter = new CorrelationIdFilter();
        MockHttpServletRequest request = new MockHttpServletRequest();
        MockHttpServletResponse response = new MockHttpServletResponse();
        MockFilterChain filterChain = new MockFilterChain();

        filter.doFilter(request, response, filterChain);

        String generatedId = response.getHeader(CorrelationIdFilter.CORRELATION_ID_HEADER);
        assertThat(generatedId).isNotBlank();
        assertThat(UUID.fromString(generatedId)).isNotNull();
    }

    @Test
    @DisplayName("TenantContextFilter should extract valid X-Tenant-Id header and bind to context and MDC")
    void shouldExtractTenantIdAndBindContext() throws Exception {
        TenantContextFilter filter = new TenantContextFilter();
        TenantId tenantId = TenantId.random();
        MockHttpServletRequest request = new MockHttpServletRequest();
        request.addHeader(TenantContextFilter.TENANT_ID_HEADER, tenantId.asString());
        MockHttpServletResponse response = new MockHttpServletResponse();

        MockFilterChain filterChain = new MockFilterChain() {
            @Override
            public void doFilter(jakarta.servlet.ServletRequest req, jakarta.servlet.ServletResponse res) {
                // Verify within filter chain execution:
                assertThat(TenantContextHolder.getRequiredTenantId()).isEqualTo(tenantId);
                assertThat(MDC.get(TenantContextFilter.MDC_TENANT_ID_KEY)).isEqualTo(tenantId.asString());
            }
        };

        filter.doFilter(request, response, filterChain);

        // Verify cleanup in finally block
        assertThat(TenantContextHolder.getContext()).isEmpty();
        assertThat(MDC.get(TenantContextFilter.MDC_TENANT_ID_KEY)).isNull();
    }

    @Test
    @DisplayName("TenantContextFilter should reject malformed X-Tenant-Id header with ValidationException")
    void shouldRejectMalformedTenantHeader() {
        TenantContextFilter filter = new TenantContextFilter();
        MockHttpServletRequest request = new MockHttpServletRequest();
        request.addHeader(TenantContextFilter.TENANT_ID_HEADER, "invalid-uuid");
        MockHttpServletResponse response = new MockHttpServletResponse();
        MockFilterChain filterChain = new MockFilterChain();

        assertThatThrownBy(() -> filter.doFilter(request, response, filterChain))
                .isInstanceOf(ValidationException.class)
                .hasMessageContaining("Invalid X-Tenant-Id header format");

        assertThat(TenantContextHolder.getContext()).isEmpty();
    }
}
