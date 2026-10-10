---
document: glossary
version: 1.0
status: active
authority: constitution
depends_on: []
related:
  - PROJECT_CONSTITUTION
---

# Glossary

| Term | Meaning |
|---|---|
| Tenant | An isolated customer/organization boundary. |
| Organization | Product representation of a tenant and its members. |
| Agent | Logical AI assistant definition owned by a tenant. |
| AgentVersion | Immutable snapshot of an agent configuration. |
| Execution | One runtime attempt using one exact AgentVersion. |
| Step | One meaningful runtime operation such as model call or tool call. |
| Tool | Controlled capability exposed to an agent runtime. |
| KnowledgeBase | Tenant-owned collection of documents used for retrieval. |
| Model Gateway | Provider-neutral abstraction for model invocation. |
| Conversation | Ordered interaction history between a user and an agent. |
| Integration Event | Event emitted to communicate a completed or meaningful fact. |
| Command | Request asking a component to perform work. |
| Tenant Context | Trusted runtime identity of the tenant associated with an operation. |
| Correlation ID | Identifier connecting related work across components. |
| Causation ID | Identifier of the operation/event that caused the current event. |
| RAG | Retrieval-Augmented Generation. |
| DLQ | Dead-letter topic for messages that cannot be processed normally. |
| Published Version | AgentVersion that has passed validation and is immutable. |
