package com.agentforge.identity.infrastructure.security;

import com.agentforge.common.core.context.TenantContext;
import com.agentforge.common.core.domain.TenantId;
import io.jsonwebtoken.Claims;
import io.jsonwebtoken.ExpiredJwtException;
import io.jsonwebtoken.JwtException;
import io.jsonwebtoken.Jwts;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.stereotype.Component;

import java.math.BigInteger;
import java.security.KeyFactory;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.NoSuchAlgorithmException;
import java.security.interfaces.RSAPrivateKey;
import java.security.interfaces.RSAPublicKey;
import java.security.spec.PKCS8EncodedKeySpec;
import java.security.spec.X509EncodedKeySpec;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.Base64;
import java.util.Collection;
import java.util.Date;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.UUID;
import java.util.stream.Collectors;

/**
 * Utility for generating, validating, and extracting claims from JWT tokens using Asymmetric RSA (RS256).
 * Signs with RSAPrivateKey, verifies with RSAPublicKey, and provides JWKS metadata.
 * Conforms to docs/02-architecture/SECURITY_ARCHITECTURE.md.
 */
@Component
public class JwtTokenProvider {

    private static final Logger log = LoggerFactory.getLogger(JwtTokenProvider.class);

    private final RSAPrivateKey privateKey;
    private final RSAPublicKey publicKey;
    private final String keyId;
    private final long expirationMinutes;

    public JwtTokenProvider(
            @Value("${agentforge.jwt.private-key:}") String privateKeyPem,
            @Value("${agentforge.jwt.public-key:}") String publicKeyPem,
            @Value("${agentforge.jwt.key-id:agentforge-identity-key-1}") String keyId,
            @Value("${agentforge.jwt.expiration-minutes:1440}") long expirationMinutes
    ) {
        this.keyId = keyId;
        this.expirationMinutes = expirationMinutes;

        if (privateKeyPem != null && !privateKeyPem.isBlank() && publicKeyPem != null && !publicKeyPem.isBlank()) {
            this.privateKey = parsePrivateKey(privateKeyPem);
            this.publicKey = parsePublicKey(publicKeyPem);
            log.info("Initialized JwtTokenProvider with configured RSA key pair (kid: {})", keyId);
        } else {
            KeyPair keyPair = generateRsaKeyPair();
            this.privateKey = (RSAPrivateKey) keyPair.getPrivate();
            this.publicKey = (RSAPublicKey) keyPair.getPublic();
            log.info("Initialized JwtTokenProvider with dynamically generated RSA 2048 key pair (kid: {})", keyId);
        }
    }

    public String generateToken(
            UUID userId,
            String email,
            TenantId tenantId,
            Set<String> roles,
            Set<String> permissions
    ) {
        Instant now = Instant.now();
        Instant validity = now.plus(expirationMinutes, ChronoUnit.MINUTES);

        return Jwts.builder()
                .header()
                    .keyId(keyId)
                    .and()
                .subject(userId.toString())
                .claim("email", email)
                .claim("tenant_id", tenantId != null ? tenantId.asString() : null)
                .claim("roles", roles)
                .claim("permissions", permissions)
                .issuedAt(Date.from(now))
                .expiration(Date.from(validity))
                .signWith(privateKey, Jwts.SIG.RS256)
                .compact();
    }

    public boolean validateToken(String token) {
        try {
            Jwts.parser()
                    .verifyWith(publicKey)
                    .build()
                    .parseSignedClaims(token);
            return true;
        } catch (ExpiredJwtException e) {
            log.warn("JWT token expired: {}", e.getMessage());
        } catch (JwtException | IllegalArgumentException e) {
            log.warn("Invalid JWT token: {}", e.getMessage());
        }
        return false;
    }

    public Claims parseClaims(String token) {
        return Jwts.parser()
                .verifyWith(publicKey)
                .build()
                .parseSignedClaims(token)
                .getPayload();
    }

    @SuppressWarnings("unchecked")
    public TenantContext createTenantContext(Claims claims) {
        String tenantIdStr = claims.get("tenant_id", String.class);
        TenantId tenantId = tenantIdStr != null ? TenantId.fromString(tenantIdStr) : null;
        String userId = claims.getSubject();

        List<String> rolesList = claims.get("roles", List.class);
        Set<String> roles = rolesList != null ? Set.copyOf(rolesList) : Set.of();

        List<String> permList = claims.get("permissions", List.class);
        Set<String> permissions = permList != null ? Set.copyOf(permList) : Set.of();

        return new TenantContext(tenantId, userId, roles, permissions, false);
    }

    public Authentication createAuthentication(Claims claims) {
        String userId = claims.getSubject();
        TenantContext context = createTenantContext(claims);

        Collection<SimpleGrantedAuthority> authorities = context.roles().stream()
                .map(role -> new SimpleGrantedAuthority("ROLE_" + role))
                .collect(Collectors.toList());

        context.permissions().forEach(perm ->
                authorities.add(new SimpleGrantedAuthority(perm))
        );

        return new UsernamePasswordAuthenticationToken(userId, null, authorities);
    }

    public RSAPublicKey getPublicKey() {
        return publicKey;
    }

    public String getKeyId() {
        return keyId;
    }

    /**
     * Produces standard RFC 7517 JWKS representation for public key verification.
     */
    public Map<String, Object> getJwkSet() {
        Map<String, Object> jwk = new LinkedHashMap<>();
        jwk.put("kty", "RSA");
        jwk.put("use", "sig");
        jwk.put("alg", "RS256");
        jwk.put("kid", keyId);
        jwk.put("n", encodeBigInteger(publicKey.getModulus()));
        jwk.put("e", encodeBigInteger(publicKey.getPublicExponent()));

        return Map.of("keys", List.of(jwk));
    }

    private static String encodeBigInteger(BigInteger value) {
        byte[] bytes = value.toByteArray();
        if (bytes.length > 1 && bytes[0] == 0) {
            byte[] unsigned = new byte[bytes.length - 1];
            System.arraycopy(bytes, 1, unsigned, 0, unsigned.length);
            bytes = unsigned;
        }
        return Base64.getUrlEncoder().withoutPadding().encodeToString(bytes);
    }

    private static KeyPair generateRsaKeyPair() {
        try {
            KeyPairGenerator generator = KeyPairGenerator.getInstance("RSA");
            generator.initialize(2048);
            return generator.generateKeyPair();
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException("RSA algorithm not available", e);
        }
    }

    private static RSAPrivateKey parsePrivateKey(String pem) {
        try {
            String clean = cleanPem(pem, "PRIVATE KEY");
            byte[] encoded = Base64.getDecoder().decode(clean);
            PKCS8EncodedKeySpec keySpec = new PKCS8EncodedKeySpec(encoded);
            KeyFactory keyFactory = KeyFactory.getInstance("RSA");
            return (RSAPrivateKey) keyFactory.generatePrivate(keySpec);
        } catch (Exception e) {
            throw new IllegalArgumentException("Failed to parse RSA private key PEM", e);
        }
    }

    private static RSAPublicKey parsePublicKey(String pem) {
        try {
            String clean = cleanPem(pem, "PUBLIC KEY");
            byte[] encoded = Base64.getDecoder().decode(clean);
            X509EncodedKeySpec keySpec = new X509EncodedKeySpec(encoded);
            KeyFactory keyFactory = KeyFactory.getInstance("RSA");
            return (RSAPublicKey) keyFactory.generatePublic(keySpec);
        } catch (Exception e) {
            throw new IllegalArgumentException("Failed to parse RSA public key PEM", e);
        }
    }

    private static String cleanPem(String pem, String type) {
        return pem.replace("-----BEGIN " + type + "-----", "")
                .replace("-----END " + type + "-----", "")
                .replaceAll("\\s+", "");
    }
}
