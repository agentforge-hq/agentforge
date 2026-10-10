---
name: context-loader
description: >-
  Use this skill when a task touches a specific AgentForge bounded context
  and you need to load the minimal, relevant source-of-truth documents into
  context. Prevents loading the entire docs/ tree and keeps token usage low.
---

# Context Loader Skill

This skill implements **progressive context loading** for the AgentForge
source-of-truth hierarchy. Instead of reading all 40+ documents, the agent
identifies which bounded context is affected and loads only the required slice.

## Step 1 — Always Load the Constitution Triad

Read these three files on every task (they are small and foundational):

1. [PROJECT_CONSTITUTION.md](../../docs/00-constitution/PROJECT_CONSTITUTION.md)
2. [ENGINEERING_PRINCIPLES.md](../../docs/00-constitution/ENGINEERING_PRINCIPLES.md)
3. [GLOSSARY.md](../../docs/00-constitution/GLOSSARY.md)

## Step 2 — Identify the Bounded Context

From the user's request, determine which context(s) are affected:

| Context ID | Keywords / Signals |
|---|---|
| `agent` | Agent, AgentVersion, publish, draft, tool binding, knowledge binding |
| `execution` | Execute, run, step, cancel, model call, tool call, streaming |
| `conversation` | Message, conversation, chat, context window, attachments |
| `identity` | User, tenant, organization, role, permission, API key, auth |
| `knowledge` | Knowledge base, document, ingest, chunk, embedding, RAG |
| `search` | Retrieval, vector search, hybrid search, reranking |
| `model` | Model provider, gateway, routing, usage, cost, timeout |
| `infrastructure` | Docker, Kubernetes, CI/CD, observability, Redis, database |
| `frontend` | UI, builder, React, components |
| `events` | Kafka, event, topic, consumer, producer |

## Step 3 — Load Context-Specific Documents

Based on the identified context, read **only** these documents:

### `agent`
- `docs/03-domains/AGENT.md`
- `docs/03-domains/AGENT_RUNTIME.md`
- `docs/02-architecture/SERVICE_BOUNDARIES.md`
- `docs/04-contracts/REST_API.md`

### `execution`
- `docs/03-domains/AGENT_RUNTIME.md`
- `docs/02-architecture/SERVICE_BOUNDARIES.md`
- `docs/04-contracts/REST_API.md`

### `conversation`
- `docs/03-domains/CONVERSATION.md`
- `docs/02-architecture/SERVICE_BOUNDARIES.md`

### `identity`
- `docs/03-domains/IDENTITY.md`
- `docs/02-architecture/SECURITY_ARCHITECTURE.md`

### `knowledge`
- `docs/03-domains/KNOWLEDGE.md`
- `docs/05-data/VECTOR_SEARCH.md`
- `docs/02-architecture/DATA_OWNERSHIP.md`
- `docs/04-contracts/EVENTS.md`

### `search`
- `docs/05-data/VECTOR_SEARCH.md`
- `docs/02-architecture/DATA_OWNERSHIP.md`

### `model`
- `docs/03-domains/MODEL.md`
- `docs/02-architecture/SERVICE_BOUNDARIES.md`

### `events`
- `docs/02-architecture/COMMUNICATION.md`
- `docs/04-contracts/EVENTS.md`
- `docs/04-contracts/KAFKA_TOPICS.md`

### `infrastructure`
- `docs/06-infrastructure/DOCKER.md`
- `docs/06-infrastructure/KUBERNETES.md`
- `docs/06-infrastructure/OBSERVABILITY.md`
- `docs/06-infrastructure/LOCAL_DEVELOPMENT.md`

### `frontend`
- `docs/07-frontend/FRONTEND_ARCHITECTURE.md`
- `docs/07-frontend/AGENT_BUILDER.md`

## Step 4 — Check ADRs

If the task may involve an architectural change, scan `docs/adr/` for relevant
ADRs. Current ADRs:

| ADR | Topic |
|---|---|
| ADR-001 | Event-driven architecture |
| ADR-002 | AgentVersion immutability |
| ADR-003 | Multi-tenancy strategy |
| ADR-004 | Agent runtime architecture |
| ADR-005 | Model gateway abstraction |
| ADR-006 | SSE for client-facing execution streaming |

## Step 5 — Summarize Loaded Context

Before proceeding, state:

1. Which bounded context(s) you identified
2. Which documents you loaded
3. Key invariants that apply
4. Whether any ADRs are relevant

This ensures traceability and prevents loading unnecessary documents.
