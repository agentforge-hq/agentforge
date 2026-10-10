---
document: conversation-domain
version: 1.0
status: active
authority: domain
depends_on:
  - IDENTITY
  - AGENT
related:
  - AGENT_RUNTIME
---

# Conversation Domain

## Entities

### Conversation

Fields:

- id
- tenantId
- agentId
- participant metadata
- state
- createdAt
- updatedAt

### Message

Fields:

- id
- conversationId
- sequence
- role
- content
- createdAt
- metadata

Roles:

- SYSTEM
- USER
- ASSISTANT
- TOOL

### MessageAttachment

Metadata for files associated with a message.

### ConversationParticipant

Identifies participating users/system actors.

## Ordering

Messages use a server-assigned monotonic sequence within a conversation. Client timestamps are not authoritative for ordering.

## Context window

The runtime selects messages according to model context limits. Conversation history remains durable even when only a subset is sent to a model.

## Relationship with execution

An execution references the conversation and records which messages/context were used. Runtime state must not silently rewrite historical conversation messages.
