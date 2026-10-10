# AgentForge — Custom Subagent Definitions

This document describes the custom subagents available in the AgentForge
workspace. These are defined using Antigravity's `define_subagent` and
`invoke_subagent` system.

## Subagent Roster

### 1. `agentforge-test-runner` — Test Writer & Runner
- **Model**: `flash` (fast turnaround for test generation, doesn't need deep reasoning)
- **Purpose**: Write and run tests across all test categories
- **Has write tools**: Yes

### 2. `agentforge-code-reviewer` — Code Reviewer
- **Model**: `pro` (needs deep reasoning for thorough review)
- **Purpose**: Review code changes against the full AgentForge quality checklist
- **Has write tools**: No (read-only, reports findings)

### 3. `agentforge-context-researcher` — Context Researcher
- **Model**: `flash` (fast lookup, many file reads)
- **Purpose**: Load and summarize relevant source-of-truth documents
- **Has write tools**: No (read-only research)

### 4. `agentforge-doc-updater` — Documentation Updater
- **Model**: `flash` (straightforward doc updates)
- **Purpose**: Update source-of-truth documents when implementation changes contracts or behavior
- **Has write tools**: Yes

## Usage

The orchestrator (primary agent) delegates to these subagents during the
implementation workflow. The primary agent coordinates, subagents execute
their specialized tasks in parallel where possible.
