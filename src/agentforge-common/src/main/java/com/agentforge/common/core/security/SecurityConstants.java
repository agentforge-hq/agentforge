package com.agentforge.common.core.security;

import java.util.Set;

/**
 * Platform-wide security roles and atomic permissions.
 * Conforms to docs/02-architecture/SECURITY_ARCHITECTURE.md and docs/03-domains/IDENTITY.md.
 */
public final class SecurityConstants {

    private SecurityConstants() {
    }

    // ==========================================
    // Roles
    // ==========================================

    /**
     * Account/Identity-level role for an authenticated user prior to or outside of any tenant context.
     */
    public static final String ROLE_USER = "USER";

    /**
     * Internal system principal (e.g. background processors, asynchronous event handlers).
     */
    public static final String ROLE_SYSTEM = "SYSTEM";

    // Tenant-scoped roles (assigned via Membership within an Organization)
    public static final String ROLE_ORG_ADMIN = "ORG_ADMIN";
    public static final String ROLE_AGENT_DEVELOPER = "AGENT_DEVELOPER";
    public static final String ROLE_AGENT_USER = "AGENT_USER";
    public static final String ROLE_AUDITOR = "AUDITOR";

    // ==========================================
    // Permissions (Atomic Authorization Identifiers)
    // ==========================================

    // Organization & Tenant management
    public static final String PERM_ORG_CREATE = "org:create";
    public static final String PERM_ORG_READ = "org:read";
    public static final String PERM_ORG_UPDATE = "org:update";
    public static final String PERM_ORG_DELETE = "org:delete";
    public static final String PERM_MEMBER_MANAGE = "member:manage";

    // API Key management
    public static final String PERM_API_KEY_MANAGE = "apikey:manage";

    // Agent management
    public static final String PERM_AGENT_READ = "agent:read";
    public static final String PERM_AGENT_WRITE = "agent:write";
    public static final String PERM_AGENT_PUBLISH = "agent:publish";

    // Execution operations
    public static final String PERM_EXECUTION_RUN = "execution:run";
    public static final String PERM_EXECUTION_CANCEL = "execution:cancel";

    // Knowledge & Vector operations
    public static final String PERM_KNOWLEDGE_READ = "knowledge:read";
    public static final String PERM_KNOWLEDGE_WRITE = "knowledge:write";

    // ==========================================
    // Default Role & Permission Sets
    // ==========================================

    /**
     * Default roles for an authenticated user without an active tenant context.
     */
    public static final Set<String> UNSCOPED_USER_ROLES = Set.of(ROLE_USER);

    /**
     * Default permissions for an authenticated user without an active tenant context.
     * Allows listing organizations, creating a new organization, and selecting an active tenant.
     */
    public static final Set<String> UNSCOPED_USER_PERMISSIONS = Set.of(PERM_ORG_CREATE, PERM_ORG_READ);
}
