---
document: system-architecture
version: 1.0
status: active
authority: architecture
depends_on:
  - PROJECT_CONSTITUTION
  - PRODUCT_REQUIREMENTS
related:
  - SERVICE_BOUNDARIES
  - COMMUNICATION
  - DATA_OWNERSHIP
  - SECURITY_ARCHITECTURE
---

# System Architecture

## Logical architecture

```text
React + TypeScript
        |
        v
Spring Cloud Gateway
        |
  +-----+-------------------------------+
  |                                     |
  v                                     v
Identity / Tenant                 Application APIs
                                      |
                +---------------------+----------------------+
                |                     |                      |
                v                     v                      v
          Agent Service       Conversation Service     Model Gateway
                |                     |                      |
                +-----------> Agent Execution <-------------+
                                  |
                         +--------+--------+
                         |                 |
                         v                 v
                      Redis           PostgreSQL
                         |
                  asynchronous phase
                         |
                         v
                       Kafka
                         |
          +--------------+---------------+
          |              |               |
          v              v               v
       Workers       Knowledge       Notifications
                          |
                          v
                    Search abstraction
                          |
                     Vector store
```

## Logical vs deployment architecture

The architecture defines ownership boundaries first. MVP deployment may combine services where doing so reduces operational overhead, provided ownership and dependency direction remain explicit.

## Main flows

### Synchronous execution start

Client -> Gateway -> authorization -> Agent/Execution boundary -> validate AgentVersion -> create execution -> return execution ID.

### Asynchronous execution

Execution request -> Kafka -> worker -> runtime -> model/tool calls -> persistent steps -> execution state.

### Client streaming

Client -> Gateway -> execution stream endpoint -> server-side stream of authorized execution events/output.

The client-facing stream does not require Kafka to be directly exposed to browsers.

## Primary consistency rule

Agent configuration is strongly owned by Agent Service. Execution uses an immutable snapshot/reference to a published version so runtime behavior cannot change because a draft was edited later.
