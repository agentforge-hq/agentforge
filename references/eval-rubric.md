# Evaluation and pruning

SDQS means **Skill Description and Quality Score**, a local v0.1 heuristic out of 100.
It is not a published benchmark or evidence of task success. The weights in
`config.default.json` are configurable and must sum to 100. Token estimates use
`ceil(UTF-8 bytes / 4)`. They are a budget proxy, not exact model tokenization.

| Dimension | Default | Observable checks |
| --- | ---: | --- |
| Structure | 20 | Required name/description, name-directory agreement, frontmatter syntax, compatibility length |
| Routing | 20 | Positive task trigger, specific vocabulary, description budget |
| Economy | 20 | Body/router budget, repeated instruction lines |
| Execution | 20 | Go AST syntax, literal flag declarations, Python compatibility inspection, documented script references |
| Maintenance | 10 | Existing local references and an observable verification step |
| Safety | 10 | Escaping script paths, instruction override and remote pipe-to-shell patterns |

Each applicable warning deducts 25% of that dimension, each error 50%, capped at
the dimension's weight. Informational boundaries and harness extensions do not
deduct points. Missing required metadata is still an error even if the overall
score is high. The linter exits 1 for errors, a missed score gate, failed trigger
cases or an incomplete inventory. Exit 2 means invalid input or an operational
failure. Scanner exits 1 for incomplete reads or requested budget gates. Other
commands use 0 for successful preview/application and 2 for rejected operations.
Findings are returned as JSON; an injection finding by itself is not an execution
failure, and an inspect exit code of 0 does not mean trustworthy.

## Run a focused audit

From the installed skill directory:

```sh
./scripts/agentforge audit --project /path/to/repo --fail-on-budget
./scripts/agentforge lint /path/to/skill --fail-under 80
./scripts/agentforge lint --root /path/to/skills --collision-threshold 0.55
./scripts/agentforge lint --root /path/to/skills --full-matrix
```

Defaults are embedded in the executable; the adjacent `config.default.json` is
a readable copy. JSON config overlays merge dictionaries; arrays replace arrays. Use an explicit
`--config /path/to/overlay.json` to change roots, budgets or weights. No commands
implicitly rewrite the default config. Shared `--config`, `--project` and
`--state-dir` options can appear before or after the command. Command-specific
options, including `--apply`, follow the selected command/subcommand.

## Script verification boundary

The Go runtime parses Go scripts with the standard Go AST parser and reads
literal flag declarations. Python compatibility checks tokenize common strings,
comments and balanced delimiters, then inspect literal `add_argument()` options.
Python inspection is not a full parser and cannot establish Python syntax
validity. It reports that boundary explicitly; unusual string forms, decorators
or dynamically built arguments may require manual review. The linter never
imports or runs inspected scripts. It pools flags across
subcommands and cannot establish that a flag is valid for a particular command,
that a positional argument is correct, or that the script works. Dynamic
argument declarations are marked for manual verification. It reads linked
Markdown references one level deep. A dependency is an explicit local Markdown
link outside a code fence, a direct instruction such as `Read` followed by an
inline resource path, or a shell script invocation. Missing dependencies remain
errors. Other resource mentions, including illustrative paths and generated
output paths, are retained in `unverified_references` without a score deduction.
Review these when a dependency is expressed indirectly; this is not a full
Markdown or natural-language parser. It supports common plain/quoted strings,
folded/literal YAML scalars and preserves nested YAML blocks opaquely. It is not
a general YAML parser. Anchors, tags and flow values in routing scalars require
manual review or rewriting as simple strings. This limitation is deliberate to
keep the runtime dependency-free.

## Collisions and A/B trigger testing

TF-IDF uses lowercase alphanumeric terms, a small stopword list, logarithmic term
frequency and smoothed inverse document frequency. Cosine pairs at or above the
configured threshold are emitted with both paths; no skill is rewritten.
Lexical overlap can miss synonyms and misread negation. A corpus of twenty skills
is not a validated universal routing limit.

Create a JSON list of representative positive, neighboring, negative and
ambiguous requests. Example (not an LLM judge):

```json
[
  {"prompt": "Audit duplicate skills across harness roots", "expected": ["agentforge"]},
  {"prompt": "Translate this sentence into French", "expected": []}
]
```

```sh
./scripts/agentforge lint --root /path/to/skills --trigger-tests /path/to/cases.json
```

A case passes when a unique eligible top lexical match is in `expected`;
an empty expected list requires abstention. Query normalization includes words
absent from the skill descriptions so incidental matches are not inflated.
The default `routing_tests` config requires cosine at least `0.1` and two
distinct shared terms. These are configurable heuristics, not calibrated
confidence levels. A one-word request abstains by default; rankings still show
its lexical evidence. Equal top eligible scores across names are ambiguous.
For example, an overlay can set:

```json
{"routing_tests": {"min_cosine": 0.2, "min_shared_terms": 2}}
```

Compare before and after reports with the same cases and corpus. Then test actual
harness selection in fresh sessions, logging false activations, missed
activations and task completion. Do not claim model routing improved based only
on the lexical proxy.

## SkillSV pruning discipline

Here SkillSV is a local skill simplification/verification checklist, not an
external algorithm. For each proposed instruction, record its beneficiary,
failure prevented, source evidence, and smallest observable check. Delete stale
examples, repeated constraints and generic advice. Replace a wrong step rather
than appending a contradictory exception. Move conditional detail to a linked
reference only when it remains useful. Preserve permission boundaries and
non-obvious invariants. Review reported duplicate lines; do not delete code or
examples solely because they repeat.

Measure before/after body and index proxies. A repair should normally keep or
reduce the relevant budget; justify growth with a demonstrated missing case.
Run representative positive/negative cases and the scripts' regression suite:

```sh
go test ./...
```
