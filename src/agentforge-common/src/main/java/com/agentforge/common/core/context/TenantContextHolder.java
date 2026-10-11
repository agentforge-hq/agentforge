package com.agentforge.common.core.context;

import java.util.Objects;
import java.util.Optional;
import java.util.function.Supplier;

import com.agentforge.common.core.domain.TenantId;

public final class TenantContextHolder {

    private static final ThreadLocal<TenantContext> CONTEXT = new ThreadLocal<>();

    private TenantContextHolder() {
        // Prevent instantiation of utility class
    }

    /**
     * Sets the context for the current thread.
     */
    public static void setContext(TenantContext context) {
        CONTEXT.set(context);
    }

    /**
     * Gets the context for the current thread.
     */
    public static Optional<TenantContext> getContext() {
        return Optional.ofNullable(CONTEXT.get());
    }

    /**
     * Returns the current context or throws an exception if none is present.
     */
    public static TenantContext getRequiredContext() {
        TenantContext context = CONTEXT.get();
        if (context == null) {
            throw new IllegalStateException("No TenantContext found in the current thread.");
        }
        return context;
    }

    /**
     * Shortcut to retrieve the required TenantId directly from the current thread.
     */
    public static TenantId getRequiredTenantId() {
        return getRequiredContext().getRequiredTenantId();
    }

    /**
     * Clears the context from the current thread.
     * Must be called in servlet filters or worker finally blocks.
     */
    public static void clear() {
        CONTEXT.remove();
    }

    /**
     * Executes an action within the specified context and restores the previous
     * context upon completion.
     */
    public static <T> T runWith(TenantContext context, Supplier<T> action) {
        Objects.requireNonNull(action, "action must not be null");
        TenantContext previous = CONTEXT.get();
        try {
            setContext(context);
            return action.get();
        } finally {
            if (previous != null) {
                setContext(previous);
            } else {
                clear();
            }
        }
    }

    /**
     * Executes a runnable action within the specified context and restores the
     * previous context upon completion.
     */
    public static void runWith(TenantContext context, Runnable action) {
        Objects.requireNonNull(action, "action must not be null");
        runWith(context, () -> {
            action.run();
            return null;
        });
    }
}
