package com.agentforge.identity.api.controller;

import com.agentforge.identity.infrastructure.security.JwtTokenProvider;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

/**
 * Controller exposing JSON Web Key Set (JWKS) endpoint.
 * Conforms to RFC 7517.
 * Allows resource servers and API gateways to retrieve the public key for verifying JWTs.
 */
@RestController
public class JwksController {

    private final JwtTokenProvider jwtTokenProvider;

    public JwksController(JwtTokenProvider jwtTokenProvider) {
        this.jwtTokenProvider = jwtTokenProvider;
    }

    @GetMapping(value = {"/.well-known/jwks.json", "/api/v1/auth/.well-known/jwks.json"}, produces = MediaType.APPLICATION_JSON_VALUE)
    public ResponseEntity<Map<String, Object>> getJwks() {
        return ResponseEntity.ok(jwtTokenProvider.getJwkSet());
    }
}
