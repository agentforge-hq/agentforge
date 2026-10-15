package com.agentforge.identity.infrastructure.security;

import com.agentforge.common.core.context.TenantContext;
import com.agentforge.common.core.context.TenantContextHolder;
import com.agentforge.identity.domain.model.ApiKey;
import com.agentforge.identity.infrastructure.persistence.entity.ApiKeyEntity;
import com.agentforge.identity.infrastructure.persistence.repository.ApiKeyRepository;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.slf4j.MDC;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.HexFormat;
import java.util.Optional;
import java.util.Set;

@Component
public class ApiKeyAuthenticationFilter extends OncePerRequestFilter {

    private static final String API_KEY_HEADER = "X-API-Key";
    private final ApiKeyRepository apiKeyRepository;

    public ApiKeyAuthenticationFilter(ApiKeyRepository apiKeyRepository) {
        this.apiKeyRepository = apiKeyRepository;
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain filterChain) throws ServletException, IOException {
        // If already authenticated by JWT, skip
        if (SecurityContextHolder.getContext().getAuthentication() != null) {
            filterChain.doFilter(request, response);
            return;
        }

        // Extract header
        String rawApiKey = request.getHeader(API_KEY_HEADER);
        if (rawApiKey == null || rawApiKey.isBlank()) {
            filterChain.doFilter(request, response);
            return;
        }

        // Hash the key
        String hashedKey = hashSha256(rawApiKey);

        Optional<ApiKeyEntity> entityOpt = apiKeyRepository.findByHashedSecret(hashedKey);
        if (entityOpt.isEmpty()) {
            // Key not found -> let chain continue or reject as 401
            filterChain.doFilter(request, response);
            return;
        }

        ApiKey apiKey = entityOpt.get().toDomain();

        // Check if valid (not revoked and not expired)
        if (!apiKey.isValid()) {
            filterChain.doFilter(request, response);
            return;
        }

        // Establish TenantContext & SecurityContext from the DB record
        String userId = "api-key:" + apiKey.getId();
        TenantContext tenantContext = TenantContext.of(
                apiKey.getTenantId(),
                userId,
                Set.of(),
                apiKey.getPermissions());
        TenantContextHolder.setContext(tenantContext);
        MDC.put("tenantId", apiKey.getTenantId().asString());
        MDC.put("userId", userId);

        // Set Spring Security Authentication with permissions
        Authentication auth = new UsernamePasswordAuthenticationToken(
                apiKey.getId().toString(),
                null,
                apiKey.getPermissions().stream().map((SimpleGrantedAuthority::new)).toList());
        SecurityContextHolder.getContext().setAuthentication(auth);

        try {
            filterChain.doFilter(request, response);
        } finally {
            TenantContextHolder.clear();
            MDC.remove("tenantId");
            MDC.remove("userId");
        }
    }

    private static String hashSha256(String rawApiKey) {
        try {
            MessageDigest messageDigest = MessageDigest.getInstance("SHA-256");
            byte[] hash = messageDigest.digest(rawApiKey.getBytes(StandardCharsets.UTF_8));
            return HexFormat.of().formatHex(hash);
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException("SHA-256 algorithm not available", e);
        }
    }
}
