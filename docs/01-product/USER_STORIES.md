---
document: user-stories
version: 1.0
status: active
authority: product
depends_on:
  - PRODUCT_REQUIREMENTS
---

# User Stories

## Admin

- As an admin, I can create and manage an organization.
- As an admin, I can invite members.
- As an admin, I can assign roles.
- As an admin, I can review security-sensitive audit activity.

## Developer

- As a developer, I can create an agent draft.
- As a developer, I can configure its model and runtime limits.
- As a developer, I can bind approved tools and knowledge sources.
- As a developer, I can publish a validated agent version.
- As a developer, I can inspect execution traces.

## Analyst

- As an analyst, I can execute a published agent.
- As an analyst, I can see model and tool progress.
- As an analyst, I can inspect retrieved knowledge and citations.
- As an analyst, I can cancel a long-running execution.

## Normal user

- As a user, I can converse with an available agent.
- As a user, I can see streamed responses.
- As a user, I can review previous conversation messages.

## Platform operator

- As an operator, I can observe execution latency and failures.
- As an operator, I can inspect Kafka lag and dependency health.
- As an operator, I can correlate a user-visible execution with backend traces.
