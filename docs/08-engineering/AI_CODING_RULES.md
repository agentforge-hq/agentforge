---
document: ai-coding-rules
version: 1.0
status: active
authority: engineering
depends_on:
  - PROJECT_CONSTITUTION
  - AGENTS.md
related:
  - ADR-README
---

# AI Coding Rules

## AI is an implementation assistant

AI suggestions are not architectural authority.

## Required workflow

```text
Understand requirement
 -> inspect source of truth
 -> inspect code
 -> identify impact
 -> propose plan
 -> implement
 -> test
 -> review
 -> update docs
 -> verify
```

## Never

- invent a service because it is convenient
- access another service's database
- bypass authorization
- remove tenant filtering
- silently change API/event contracts
- introduce infrastructure without justification
- add dependencies merely because they are popular

## When blocked

If requirements conflict or are incomplete:

- identify the ambiguity
- propose alternatives
- choose only when the existing authority permits it
- otherwise request/record a decision

## Architectural changes

Any change to service boundaries, data ownership, major communication patterns, security model, or runtime semantics requires an ADR.

## Generated code review

Generated code is reviewed like human-written code.
