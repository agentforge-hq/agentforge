package com.agentforge.identity.infrastructure.persistence.repository;

import com.agentforge.identity.infrastructure.persistence.entity.MembershipEntity;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface MembershipRepository extends JpaRepository<MembershipEntity, UUID> {

    Optional<MembershipEntity> findByOrganizationIdAndUserId(UUID organizationId, UUID userId);

    List<MembershipEntity> findAllByOrganizationId(UUID organizationId);

    List<MembershipEntity> findAllByUserId(UUID userId);

    boolean existsByOrganizationIdAndUserId(UUID organizationId, UUID userId);

    @Query("SELECT m FROM MemberShipEntity m WHERE m.userId = :userId")
    Optional<MembershipEntity> findLatestJoinedByUserId(UUID userId);
}
