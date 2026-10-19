# Lint codes and actionability

Every issue includes `code`, `severity`, `meaning`, `actionability`, `actionable`
and `action`. `actionable` is true for `fix` and `review`. It does not authorize
edits or mean that a heuristic finding is a confirmed defect.

- `fix` identifies a concrete format, dependency or script problem.
- `review` requires judgment or representative task verification.
- `conditional` needs attention only when the stated condition applies.
- `informational` states a verification boundary.
- `upstream` means the package is managed. Contact its owner or use the harness
  manager. Do not edit the cached, built-in or synced copy in place.

Managed packages retain the original issue severity for portable validation,
but local actions and pruning suggestions are suppressed. An error can still
make lint exit 1; this is not proof the owning harness cannot use that package.
Exit 2 indicates invalid input or an operational failure.

| Code | Meaning | User-owned actionability |
| --- | --- | --- |
| `frontmatter` | YAML frontmatter is missing, malformed or unsupported. | fix |
| `name` | Name is not lowercase kebab-case of 1-64 characters. | fix |
| `name-directory` | Name differs from its installation directory; align a portable export after checking references. | fix |
| `description` | Description is absent or outside 1-1024 characters. | fix |
| `compatibility` | Compatibility is not a supported string of 1-500 characters. | fix |
| `harness-extensions` | Extra frontmatter fields require a target-support check before export; valid harness fields need not be removed. | conditional |
| `positive-trigger` | Description lacks an explicit positive routing trigger. | review |
| `routing-boundary` | No exclusion clause was found; inspect only when neighboring skills overlap. | conditional |
| `vague-description` | Description has few specific routing terms. | review |
| `description-budget` | Description exceeds its configured byte-based token budget. | review |
| `body-budget` | Root body exceeds its configured token estimate budget. | review |
| `router-budget` | AgentForge's root exceeds its configured router budget. | fix |
| `repeated-instructions` | Long instruction lines repeat after text normalization. | review |
| `escaping-reference` | A dependency resolves outside the package. | fix |
| `missing-reference` | An explicit local dependency is missing. | fix |
| `escaping-script` | A script resolves outside the package. | fix |
| `go-syntax` | A Go helper fails Go AST parsing. | fix |
| `python-lexical-error` | A Python helper has malformed quoted or balanced tokens. | fix |
| `unverified-command` | A documented command lacks an inspectable script. | fix |
| `script-flag` | A flag lacks a verified literal declaration; dynamic declarations need manual inspection. | review |
| `python-verification-boundary` | Full Python syntax was not validated. | informational |
| `script-routing` | Root instructions do not explain how to invoke helpers. | review |
| `verification` | No observable output-verification step was found. | review |
| `unsafe-instruction` | An instruction-override or remote-execution pattern needs inspection; this is not a trust verdict. | review |

When a directory lacks `SKILL.md`, CLI lint reports it in `skipped` and discovers
its child skills. Empty containers produce no scored skill. Missing paths,
unreadable directories and malformed discovered skills remain errors.
