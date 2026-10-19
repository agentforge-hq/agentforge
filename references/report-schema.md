# Audit JSON fields

This reference describes `agentforge audit` and its `scan` alias in v0.1.
Numbers use `ceil(UTF-8 bytes / 4)`. They estimate file content, not observed
harness prompt tokens, latency or task quality. There is no `tokens` field.
A consumer must handle a missing field explicitly instead of substituting zero.

## Scope and top-level fields

| Field | Type | Meaning |
| --- | --- | --- |
| `version` | string | AgentForge version |
| `project` | string | Resolved session directory |
| `project_scopes` | string array | Session-to-Git-root scopes, or only the session directory when no Git root exists |
| `token_method`, `scope` | string | Estimate method and interpretation boundary |
| `skills` | object array | Discovered path occurrences, with each lexical path represented once |
| `rules` | object array | Configured eager, conditional and candidate rule files |
| `duplicates` | object array | Groups sharing a declared or inferred skill name |
| `summaries` | object | Counts and estimates keyed by configured harness, or `custom` |
| `coverage` | object array | Harness, configured pattern, matched roots and profile status |
| `errors` | object array | Read, traversal or fingerprint errors. An incomplete scan exits 1 |

With no `--root`, the audit uses embedded configured home roots and workspace
scopes. `--config` supplies an explicit overlay. Repeated `--root` switches to
a `custom` skill-only inventory with no configured rules. Empty rule totals
in that mode do not establish that a harness has no eager rules.

## Skill entries

| Field | Type | Meaning |
| --- | --- | --- |
| `path` | string | Discovered `SKILL.md` path, including an alias when present |
| `real_path` | string | Resolved `SKILL.md` path. This does not choose a canonical source |
| `name` | string | Frontmatter name, falling back to the parent directory name |
| `description` | string | Parsed frontmatter description |
| `tokens_estimate` | integer | Root body bytes divided by four, rounded up. Excludes frontmatter and references |
| `index_tokens_estimate` | integer | Estimate for name, description and discovered path joined with newlines |
| `tree_sha256` | string or null | Strict package fingerprint, including supporting files, executable bits and link targets |
| `fingerprint_error` | string or null | Reason the fingerprint could not be verified |
| `diagnostics` | string array | Frontmatter parsing problems |
| `harnesses` | string array | Configured profiles exposing this path, not an observed active catalog |
| `managed` | boolean | Path or resolved target is a plugin cache, built-in or Claude-synced package |
| `managed_reason` | string | Manager classification; empty for user-owned paths |
| `cleanup_eligible` | boolean | False for managed paths; other paths still require metadata, fingerprint and dependency checks |

For example, read `skills[i].tokens_estimate` for the body estimate and
`skills[i].description` for triage text. Root body size is a lazy-load estimate,
so summing it across all installed skills does not measure startup context.

## Duplicate groups

Each group has `name`, `paths`, `kind` and `harness_exposure`. The last field
counts path occurrences per configured profile.

| `kind` | Meaning |
| --- | --- |
| `aliases` | All paths resolve to one physical skill |
| `identical` | Distinct physical skills have equal strict tree fingerprints |
| `divergent` | Verified fingerprints differ |
| `unverified` | At least one fingerprint failed, so distinct packages cannot be compared safely |

Mixed groups can also include `alias_groups` with `real_path` and `paths`,
and `identical_groups` with `tree_sha256` and `paths`. These preserve subsets
inside a name-level group.

Divergent groups also include `comparison_reference`, `difference_summaries`
and `comparisons_omitted`. The first discovered path is a diagnostic reference,
not a canonical choice. Up to ten other copies are summarized. Each summary
identifies differing relative paths and fields such as `bytes`,
`executable_bits`, `symlink_target`, `presence` and `type`.

`divergent` does not diagnose a broken or outdated skill. Different harness
paths and frontmatter remain different packages even when generated from one
version. Those differences can affect execution and permissions. Alias paths
also do not establish that a harness deduplicates its index. See the
[management playbook](healing-playbooks.md) for fingerprint exclusions and
safe handling.

## Explicit comparisons

`agentforge compare --canonical PATH` discovers the declared-name group.
Repeated `--copy` compares explicit paths instead. The result includes
`canonical`, `canonical_managed`, `comparisons`, `cleanup_candidates`,
`dedupe_preview_argv`, `errors` and `applied: false`.

A comparison reports strict `kind`, both fingerprints, `byte_content_equal`,
`content_relation` and bounded `differences`. The content relation is
`same-bytes`, `content-differs` or, with normalization,
`harness-path-variant`. Byte equality excludes executable-bit differences;
strict fingerprints retain them. Symlinks are compared by their targets.

`--normalize-harness-paths` replaces known self-root spellings in text only.
It preserves frontmatter and does not normalize other skills' paths.
`raw_lines` and `normalized_lines` count added and removed lines when within
the work limit. `--show-lines` includes bounded changed lines, with truncation
reported. Text inspection is limited to the configured per-file size and
8 MiB per package. Fingerprints still cover files outside the text budget.
Normalized equality is diagnostic and cannot make a divergent package eligible
for deduplication. Managed packages are excluded from the preview candidates.

## Summary and rule entries

Each summary contains `skill_occurrences`, `unique_physical_skills`,
`index_tokens_estimate`, `always_rule_tokens_estimate`,
`conditional_rule_tokens_estimate`, `over_index_budget` and `over_eager_budget`.
The boolean budget fields compare estimates with configured limits.

A rule entry has `path`, `tokens_estimate` and a `harnesses` object mapping
each profile to `always`, `conditional`, `candidate` or `shadowed`.
Always totals include only `always` rules. Conditional totals include
`conditional` and `candidate` rules. Neither total counts `shadowed` rules.
