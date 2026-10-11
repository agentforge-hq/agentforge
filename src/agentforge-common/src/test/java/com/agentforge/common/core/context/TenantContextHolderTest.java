package com.agentforge.common.core.context;

import com.agentforge.common.core.domain.TenantId;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.Set;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.atomic.AtomicReference;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class TenantContextHolderTest {

    private final TenantId tenantId = TenantId.random();
    private final TenantContext sampleContext = TenantContext.of(
            tenantId,
            "user-123",
            Set.of("ORG_ADMIN"),
            Set.of("agent:read", "agent:write")
    );

    @BeforeEach
    @AfterEach
    void cleanup() {
        TenantContextHolder.clear();
    }

    @Test
    @DisplayName("should set and retrieve context on current thread")
    void shouldSetAndGetContext() {
        TenantContextHolder.setContext(sampleContext);

        assertThat(TenantContextHolder.getContext()).contains(sampleContext);
        assertThat(TenantContextHolder.getRequiredContext()).isEqualTo(sampleContext);
        assertThat(TenantContextHolder.getRequiredTenantId()).isEqualTo(tenantId);
    }

    @Test
    @DisplayName("should clear context completely")
    void shouldClearContext() {
        TenantContextHolder.setContext(sampleContext);
        TenantContextHolder.clear();

        assertThat(TenantContextHolder.getContext()).isEmpty();
    }

    @Test
    @DisplayName("should throw IllegalStateException when required context is missing")
    void shouldThrowWhenRequiredContextMissing() {
        assertThatThrownBy(TenantContextHolder::getRequiredContext)
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("No TenantContext found");
    }

    @Test
    @DisplayName("should throw IllegalStateException when tenantId is required but context is global system")
    void shouldThrowWhenTenantIdMissingInGlobalContext() {
        TenantContext globalContext = TenantContext.globalSystem();
        TenantContextHolder.setContext(globalContext);

        assertThatThrownBy(TenantContextHolder::getRequiredTenantId)
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("Tenant context is required");
    }

    @Test
    @DisplayName("should execute action with runWith(Supplier) and clean up afterwards")
    void shouldRunWithSupplierAndCleanup() {
        String result = TenantContextHolder.runWith(sampleContext, () -> {
            assertThat(TenantContextHolder.getRequiredContext()).isEqualTo(sampleContext);
            return "SUCCESS";
        });

        assertThat(result).isEqualTo("SUCCESS");
        // Must be empty immediately after runWith completes
        assertThat(TenantContextHolder.getContext()).isEmpty();
    }

    @Test
    @DisplayName("should execute runnable with runWith(Runnable) and clean up afterwards")
    void shouldRunWithRunnableAndCleanup() {
        AtomicReference<TenantId> capturedTenantId = new AtomicReference<>();

        TenantContextHolder.runWith(sampleContext, () -> {
            capturedTenantId.set(TenantContextHolder.getRequiredTenantId());
        });

        assertThat(capturedTenantId.get()).isEqualTo(tenantId);
        assertThat(TenantContextHolder.getContext()).isEmpty();
    }

    @Test
    @DisplayName("should restore previous context when runWith is nested")
    void shouldRestorePreviousContextOnNestedRunWith() {
        TenantContext outerContext = sampleContext;
        TenantContext innerContext = TenantContext.forTenantSystem(TenantId.random());

        TenantContextHolder.setContext(outerContext);

        TenantContextHolder.runWith(innerContext, () -> {
            assertThat(TenantContextHolder.getRequiredContext()).isEqualTo(innerContext);
        });

        // Outer context must be restored after inner finishes
        assertThat(TenantContextHolder.getRequiredContext()).isEqualTo(outerContext);
    }

    @Test
    @DisplayName("should isolate context between different threads")
    void shouldIsolateContextBetweenThreads() throws ExecutionException, InterruptedException {
        TenantContextHolder.setContext(sampleContext);

        try (var executor = Executors.newSingleThreadExecutor()) {
            Future<Boolean> threadResult = executor.submit(() -> {
                // Different thread should NOT see Thread 1's context
                return TenantContextHolder.getContext().isEmpty();
            });

            assertThat(threadResult.get()).isTrue();
        }

        // Current thread still retains its own context
        assertThat(TenantContextHolder.getRequiredContext()).isEqualTo(sampleContext);
    }
}
