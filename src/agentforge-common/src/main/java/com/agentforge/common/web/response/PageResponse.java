package com.agentforge.common.web.response;

import java.util.Collections;
import java.util.List;

/**
 * Standard pagination response envelope supporting both page-based and cursor-based pagination.
 * Conforms to docs/04-contracts/REST_API.md.
 *
 * @param <T> item type
 */
public record PageResponse<T>(
        List<T> items,
        int page,
        int size,
        long totalElements,
        int totalPages,
        boolean hasNext,
        String nextCursor
) {

    public PageResponse {
        items = (items == null) ? Collections.emptyList() : List.copyOf(items);
    }

    public static <T> PageResponse<T> of(
            List<T> items,
            int page,
            int size,
            long totalElements
    ) {
        int totalPages = size > 0 ? (int) Math.ceil((double) totalElements / size) : 0;
        boolean hasNext = page + 1 < totalPages;
        return new PageResponse<>(items, page, size, totalElements, totalPages, hasNext, null);
    }

    public static <T> PageResponse<T> ofCursor(
            List<T> items,
            int size,
            String nextCursor
    ) {
        boolean hasNext = nextCursor != null && !nextCursor.isBlank();
        return new PageResponse<>(items, 0, size, items.size(), 1, hasNext, nextCursor);
    }
}
