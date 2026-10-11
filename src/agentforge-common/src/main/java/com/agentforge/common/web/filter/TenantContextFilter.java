package com.agentforge.common.web.filter;

import com.agentforge.common.core.context.TenantContext;
import com.agentforge.common.core.context.TenantContextHolder;
import com.agentforge.common.core.domain.TenantId;
import com.agentforge.common.core.exception.ErrorCode;
import com.agentforge.common.core.exception.ValidationException;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.slf4j.MDC;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;
import java.util.Map;
import java.util.Set;

/**
 * Filter that extracts the X-Tenant-Id header, populates TenantContextHolder and logging MDC,
 * and guarantees thread cleanup in finally.
 */
public class TenantContextFilter extends OncePerRequestFilter {

    public static final String TENANT_ID_HEADER = "X-Tenant-Id";
    public static final String MDC_TENANT_ID_KEY = "tenantId";

    @Override
    protected void doFilterInternal(
            HttpServletRequest request,
            HttpServletResponse response,
            FilterChain filterChain
    ) throws ServletException, IOException {
        String tenantHeader = request.getHeader(TENANT_ID_HEADER);

        if (tenantHeader != null && !tenantHeader.isBlank()) {
            TenantId tenantId;
            try {
                tenantId = TenantId.fromString(tenantHeader);
            } catch (IllegalArgumentException ex) {
                throw new ValidationException(
                        ErrorCode.VALIDATION_ERROR,
                        "Invalid " + TENANT_ID_HEADER + " header format: " + tenantHeader,
                        Map.of("header", TENANT_ID_HEADER, "value", tenantHeader)
                );
            }

            TenantContext context = TenantContext.of(tenantId, null, Set.of(), Set.of());
            TenantContextHolder.setContext(context);
            MDC.put(MDC_TENANT_ID_KEY, tenantId.asString());
        }

        try {
            filterChain.doFilter(request, response);
        } finally {
            TenantContextHolder.clear();
            MDC.remove(MDC_TENANT_ID_KEY);
        }
    }
}
