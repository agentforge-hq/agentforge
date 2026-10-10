---
document: data-ownership
version: 1.0
status: active
authority: architecture
depends_on:
  - SERVICE_BOUNDARIES
  - PROJECT_CONSTITUTION
---

# Data Ownership

| Data | Owner |
|---|---|
| User | Identity |
| Organization/Tenant | Identity |
| Membership | Identity |
| Role/Permission | Identity |
| Agent | Agent |
| AgentVersion | Agent |
| Agent bindings | Agent |
| Execution | Execution |
| ExecutionStep | Execution |
| Conversation | Conversation |
| Message | Conversation |
| KnowledgeBase | Knowledge |
| Document | Knowledge |
| Chunk metadata | Knowledge |
| Retrieval index | Search |
| Model provider configuration | Model Gateway |
| Notification delivery state | Notification |

Services may maintain local read models derived from events. A read model does not transfer ownership.

## Database strategy

The preferred production boundary is database-per-service. For MVP, a PostgreSQL cluster may host multiple service databases/schemas while preserving logical ownership and access boundaries. Shared tables across services are prohibited.

## Cross-domain data

If a service needs another domain's data, prefer:

1. API for current authoritative state
2. event-derived local projection for high-volume/read-heavy access
3. explicit integration contract

Never use another service's database as a shortcut.
