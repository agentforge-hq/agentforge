---
document: documentation-rules
version: 1.0
status: active
authority: constitution
depends_on:
  - PROJECT_CONSTITUTION
---

# Documentation Rules

## Authority

Each document declares an authority level.

## Required metadata

Major documents contain front matter with:

- document
- version
- status
- authority
- depends_on
- related

## Change discipline

When changing a contract or invariant, update dependent documents.

## No fake history

ADRs describe decisions being established by this repository. They must not imply that a historical meeting or incident happened unless documented elsewhere.

## Status

`active` means the document is currently authoritative.

`target` means the architecture is approved as a future capability but may not be implemented yet.
