---
description: Draft a new AgentForge ADR as PROPOSED for human review
agent: doc-updater
subtask: true
---
Use the `adr-writer` skill to draft an ADR for: $ARGUMENTS

- Create the file at `docs/adr/ADR-NNN-<slug>.md` with `status: proposed` only.
- Update the index in `docs/adr/README.md`.
- Do NOT mark it ACCEPTED — the human accepts decisions.
- Do not begin implementation until the human has accepted the ADR.
