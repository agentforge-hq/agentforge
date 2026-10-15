package com.agentforge.identity.domain.model;

import java.util.Collections;
import java.util.Set;

/**
 * Named bundles of permissions.
 * Implementation conveniences; permissions are the true authorization contract.
 */
public enum Role {

    ORG_ADMIN(Set.of(Permission.values())),

    AGENT_DEVELOPER(Set.of(
            Permission.AGENT_READ,
            Permission.AGENT_WRITE,
            Permission.AGENT_PUBLISH,
            Permission.EXECUTION_RUN,
            Permission.EXECUTION_CANCEL,
            Permission.KNOWLEDGE_READ,
            Permission.KNOWLEDGE_WRITE,
            Permission.ORG_READ
    )),

    AGENT_USER(Set.of(
            Permission.AGENT_READ,
            Permission.EXECUTION_RUN,
            Permission.KNOWLEDGE_READ,
            Permission.ORG_READ
    )),

    AUDITOR(Set.of(
            Permission.AGENT_READ,
            Permission.ORG_READ,
            Permission.KNOWLEDGE_READ
    ));

    private final Set<Permission> permissions;

    Role(Set<Permission> permissions) {
        this.permissions = Collections.unmodifiableSet(permissions);
    }

    public Set<Permission> getPermissions() {
        return permissions;
    }

    public boolean hasPermission(Permission permission) {
        return permissions.contains(permission);
    }
}
