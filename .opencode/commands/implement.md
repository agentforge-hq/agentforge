---
description: Run the AgentForge feature-implementation workflow (context, classify, test plan, implement, review, docs)
---
Load the `implement-feature` and `context-loader` skills and follow them for: $ARGUMENTS

OpenCode orchestration rules:
- Delegate document loading to `context-researcher`, tests to `test-runner`, review to `code-reviewer`, and contract/domain doc updates to `doc-updater` via the Task tool.
- At most ONE write-capable agent may edit at a time; never run two writers concurrently over the same files. Read-only agents may run in parallel with a single writer.
- Governance: you may propose decisions as PROPOSED only; never mark anything ACCEPTED or VERIFIED. "Implementation complete" is never "verified".
- Do not commit or push; report status and leave history changes to the human.
