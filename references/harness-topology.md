# Harness topology and export behavior

Checked against the primary sources linked below on 2026-10-18. Versions,
plugins, organization settings and additional roots can change discovery. This
is a configured filesystem audit, not an introspector of a running harness.
`coverage` lists the actual roots scanned, including missing patterns. Cache
presence does not imply that a plugin is enabled. Use config overlays and
additional `--root` scans for installation-specific paths. Do not edit caches as
canonical sources; an updater can replace them.

The brief lists more than ten products. The ten profiles below cover nine local
filesystems and Muse.ai, identified by the user as a cloud target. Claude Chat,
ChatGPT, Grok, DeepSeek and Muse.ai are manual export targets. These exports
create a ZIP only; they do not upload it or assert local filesystem discovery,
script execution support, account availability or instruction semantics.

| Profile | Default skill roots (home `~`; project relative) | Rule exposure / isolation |
| --- | --- | --- |
| Claude Code | `~/.claude/skills`, `.claude/skills`, candidate plugin cache skill folders | `CLAUDE.md` eager; `.claude/rules` may be conditional. Verified skill isolation: `context: fork`; optional `agent` selects a subagent type. |
| Codex | `~/.agents/skills`, `.agents/skills`, `/etc/codex/skills`; legacy `~/.codex/skills`, observed built-ins in `~/.codex/skills/.system` and plugin caches audited as candidates | `AGENTS.md`/override precedence depends on scope. No portable per-skill isolation frontmatter. |
| Cursor | `.agents/skills`, `.cursor/skills`, `.claude/skills`, `.codex/skills` and home equivalents | `.cursor/rules/*.mdc`: only `alwaysApply: true` is classified always; other rules are conditional. `paths` scopes skills. |
| Devin | `.agents/skills`, `.devin/skills`, `.github/skills`, `.claude/skills`, `.cognition/skills`, `.windsurf/skills` | Repository roots are documented; local CLI/global extras need verification. `triggers: ["user"]` disables automatic invocation. |
| Gemini CLI | `~/.agents/skills`, `~/.gemini/skills`, `.agents/skills`, `.gemini/skills` | `GEMINI.md` eager. `.agents` wins within a tier; workspace skills override user skills. No mapped skill isolation flag. |
| Antigravity | `.agents/skills`, legacy `.agent/skills`; App/IDE `~/.gemini/config/skills`, CLI `~/.gemini/antigravity-cli/skills`, CLI plugins; legacy `~/.gemini/antigravity/skills` | Rules and legacy global rules are candidates; surface/version matters. No mapped isolation flag. |
| Hermes | `~/.hermes/skills`; shared `~/.agents/skills` and project roots are configuration-dependent candidates | External directories and project precedence are configurable. No mapped skill isolation flag. |
| Pi | `~/.agents/skills`, `.agents/skills`; native `~/.pi/agent/skills`, `.pi/skills` are candidates | Package/additional roots configurable. `disable-model-invocation` is supported. No mapped isolation flag. |
| OpenCode | `.opencode/skills`, `~/.config/opencode/skills` (respects `XDG_CONFIG_HOME`), `.agents/skills`, `.claude/skills` and home equivalents | `AGENTS.md` instruction candidates depend on configuration. Command/subagent settings are not SKILL frontmatter. |
| Muse.ai | No local skill root assumed | Manual ZIP export only; native skill ingestion and script execution are unverified. Desktop UI presence does not establish local skill discovery. |

The machine-readable map is in `config.default.json`. Discovery precedence is
not simulated. An alias/duplicate finding means repeated filesystem exposure,
not proof of repeated prompt injection. Some harnesses shadow or replace names.
The explicit Codex `.system` root comes from local observation on 2026-10-18;
normal recursive walking still skips other hidden directories. A plugin cache
or built-in file can exist without appearing in the current session's skill
catalog. Compare a supplied session catalog separately when it is available.
The scanner walks session ancestors to the nearest Git worktree root. It does
not expand arbitrary Markdown imports or inspect every child project automatically;
scan the session directory and add relevant roots.
Always/conditional totals exclude built-in prompts and remote account skills.
No script unloads a skill already read into a conversation. Cold storage only
removes future filesystem discovery. Documented delegation can reduce future
context; it does not retroactively evict prior content.

## Frontmatter and translation

The [open specification](https://agentskills.io/specification) defines `name`,
`description`, optional `license`, `compatibility`, `metadata`, and experimental
`allowed-tools`. Match name to directory. The config lists recognized target
extensions; it is not a promise that every target enforces every field.
Nested metadata is retained byte-for-byte. Exports preserve bodies, references,
scripts and assets. Unknown fields are listed as losses; applying such an
export requires `--allow-lossy`. Tool restrictions are never silently dropped.
Supported fields can still have different enforcement semantics across tools;
review the target's permission model. Unknown `subagent: true` or
`invoke_subagent` fields are not converted into fictional equivalents.

From this skill directory:

```sh
./scripts/agentforge sync export --skill /path/to/skill-name --harness claude-code --isolate --destination /path/to/target/skill-name
./scripts/agentforge sync export --skill /path/to/skill-name --harness codex --destination /path/to/target/skill-name
./scripts/agentforge sync export --skill /path/to/skill --harness cursor --format cursor-rule --destination /path/to/rule.mdc
./scripts/agentforge sync export --skill /path/to/skill --harness chatgpt --destination /path/to/bundle.zip
```

Preview first; repeat with `--apply` within the authorized scope. `cursor-rule`
produces a small, non-always rule pointing to the source skill. It is local and
depends on that source path; move the source only after updating the pointer.
The destination is exact and must not exist. Portable directory exports should
use the skill name as the final directory. Exports reject symlinks; explicitly
review/materialize dependencies before sharing. Isolation is mapped only for
Claude Code. For other harnesses, pass file pointers or consult their current
delegation documentation.

## Primary sources

- [Claude Code skills](https://code.claude.com/docs/en/skills)
- [Codex/local skill locations](https://learn.chatgpt.com/docs/build-skills)
- [Cursor skills](https://cursor.com/docs/skills)
- [Devin skills](https://docs.devin.ai/product-guides/skills)
- [Gemini CLI skills](https://geminicli.com/docs/cli/skills/)
- [Antigravity skills](https://antigravity.google/docs/skills)
- [Hermes skills](https://hermes-agent.nousresearch.com/docs/user-guide/features/skills/)
- [Pi skills](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/skills.md)
- [OpenCode skills](https://opencode.ai/docs/skills/)
- [Muse.ai, user-selected cloud target](https://muse.ai) (public documentation could not be verified beyond its sign-in page)
- [skills.sh directory](https://skills.sh/docs)
