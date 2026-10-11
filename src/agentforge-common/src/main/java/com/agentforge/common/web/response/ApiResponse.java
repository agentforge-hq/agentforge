package com.agentforge.common.web.response;

import java.time.Instant;

/**
 * Standard unified API response wrapper.
 * Conforms to docs/04-contracts/REST_API.md.
 *
 * @param <T> payload type
 */
public record ApiResponse<T>(
        T data,
        Instant timestamp,
        String correlationId
) {

    public static <T> ApiResponse<T> ok(T data, String correlationId) {
        return new ApiResponse<>(data, Instant.now(), correlationId);
    }

    public static <T> ApiResponse<T> ok(T data) {
        return new ApiResponse<>(data, Instant.now(), null);
    }
}
