package com.agentforge.identity.infrastructure.security;

import com.agentforge.common.core.context.TenantContext;
import com.agentforge.common.core.context.TenantContextHolder;
import io.jsonwebtoken.Claims;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.slf4j.MDC;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;

/**
 * Filter that intercepts incoming HTTP requests, validates the Bearer JWT token,
 * establishes SecurityContextHolder, and sets the platform TenantContextHolder.
 */
@Component
public class JwtAuthenticationFilter extends OncePerRequestFilter {

    private static final String AUTHORIZATION_HEADER = "Authorization";
    private static final String BEARER_PREFIX = "Bearer ";

    private final JwtTokenProvider jwtTokenProvider;

    public JwtAuthenticationFilter(JwtTokenProvider jwtTokenProvider) {
        this.jwtTokenProvider = jwtTokenProvider;
    }

    @Override
    protected void doFilterInternal(
            HttpServletRequest request,
            HttpServletResponse response,
            FilterChain filterChain
    ) throws ServletException, IOException {
        String token = resolveToken(request);

        if (token != null && jwtTokenProvider.validateToken(token)) {
            Claims claims = jwtTokenProvider.parseClaims(token);
            Authentication authentication = jwtTokenProvider.createAuthentication(claims);
            SecurityContextHolder.getContext().setAuthentication(authentication);

            TenantContext tenantContext = jwtTokenProvider.createTenantContext(claims);
            TenantContextHolder.setContext(tenantContext);

            if (tenantContext.isTenantScoped()) {
                MDC.put("tenantId", tenantContext.getRequiredTenantId().asString());
            }
            tenantContext.getUserId().ifPresent(uid -> MDC.put("userId", uid));
        }

        try {
            filterChain.doFilter(request, response);
        } finally {
            TenantContextHolder.clear();
            MDC.remove("userId");
        }
    }

    private String resolveToken(HttpServletRequest request) {
        String bearerToken = request.getHeader(AUTHORIZATION_HEADER);
        if (bearerToken != null && bearerToken.startsWith(BEARER_PREFIX)) {
            return bearerToken.substring(BEARER_PREFIX.length()).trim();
        }
        return null;
    }
}
