package com.agentforge.identity.infrastructure.persistence.repository;

import com.agentforge.identity.infrastructure.persistence.entity.ApiKeyEntity;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface ApiKeyRepository extends JpaRepository<ApiKeyEntity, UUID> {

    List<ApiKeyEntity> findAllByOrganizationId(UUID organizationId);

    Optional<ApiKeyEntity> findByOrganizationIdAndId(UUID organizationId, UUID id);

    Optional<ApiKeyEntity> findByHashedSecret(String hashedSecret);
}
