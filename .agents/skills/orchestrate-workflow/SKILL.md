---
name: orchestrate-workflow
description: >-
  Use this skill when orchestrating a full development workflow that involves
  multiple subagents. Defines how the primary agent coordinates context loading,
  implementation, testing, code review, and documentation updates using the
  specialized AgentForge subagents.
---

# Workflow Orchestration Skill

This skill defines how to coordinate the AgentForge subagent team for
end-to-end development workflows.

## Available Subagents

| Subagent | Model | Role | Write Tools |
|---|---|---|---|
| `agentforge-context-researcher` | `flash` | Load & summarize source-of-truth docs | No |
| `agentforge-test-runner` | `flash` | Write & run tests | Yes |
| `agentforge-code-reviewer` | `pro` | Deep code review against invariants | No |
| `agentforge-doc-updater` | `flash` | Update docs when contracts/behavior change | Yes |

## Orchestration Patterns

### Pattern 1: Feature Implementation (Full Pipeline)

```
┌─────────────┐
│   User       │
│  Request     │
└──────┬──────┘
       │
       ▼
┌──────────────────────┐
│  1. Context Research  │  ← agentforge-context-researcher (flash)
│  Load relevant docs   │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│  2. Primary Agent     │  ← You (orchestrator)
│  Plan & Implement     │
└──────┬───────────────┘
       │
       ├────────────────────────┐
       ▼                        ▼
┌──────────────────┐  ┌──────────────────┐
│ 3a. Test Writer   │  │ 3b. Code Reviewer │  ← parallel
│ (flash)           │  │ (pro)             │
└──────┬───────────┘  └──────┬───────────┘
       │                      │
       ▼                      ▼
┌──────────────────────────────────┐
│  4. Fix issues from review       │  ← You (orchestrator)
└──────┬───────────────────────────┘
       │
       ▼
┌──────────────────────┐
│  5. Doc Updater       │  ← agentforge-doc-updater (flash)
│  Update if needed     │
└──────────────────────┘
```

### Pattern 2: Quick Change (Skip Full Pipeline)

For small, local changes within a single service:

1. **You** (orchestrator): Load context yourself (constitution triad + 1-2 domain docs)
2. **You**: Implement the change
3. **agentforge-test-runner**: Write/run tests
4. Done — no full review needed for trivial changes

### Pattern 3: Architecture Decision

1. **agentforge-context-researcher**: Load relevant ADRs and architecture docs
2. **You**: Analyze and draft ADR (or use `adr-writer` skill)
3. **agentforge-code-reviewer**: Review the ADR
4. **agentforge-doc-updater**: Update affected source-of-truth docs
5. Then implement using Pattern 1

## Token Optimization Rules

1. **Never load all 40+ docs** — use context-researcher to load the minimum slice
2. **Parallelize where possible** — test writing and code review can run simultaneously
3. **Use `flash` for mechanical tasks** — test writing, doc updates, file lookups
4. **Use `pro` for reasoning tasks** — code review needs deep invariant analysis
5. **Primary agent stays lean** — delegate heavy reading to subagents
6. **Subagents share the same workspace** — they can read each other's output files

## Invocation Examples

### Invoke context researcher:
```
invoke_subagent(
  TypeName="agentforge-context-researcher",
  Model="flash",
  Prompt="Load context for the Agent bounded context. The task is: <description>"
)
```

### Invoke test runner:
```
invoke_subagent(
  TypeName="agentforge-test-runner",
  Model="flash",
  Prompt="Write unit and integration tests for <class/feature>. Files changed: <list>"
)
```

### Invoke code reviewer:
```
invoke_subagent(
  TypeName="agentforge-code-reviewer",
  Model="pro",
  Prompt="Review the following changes: <diff or file list>. Focus on: <specific concerns>"
)
```

### Invoke doc updater:
```
invoke_subagent(
  TypeName="agentforge-doc-updater",
  Model="flash",
  Prompt="Update docs/04-contracts/REST_API.md to reflect: <change description>"
)
```
