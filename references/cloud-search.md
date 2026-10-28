# Discover and adopt a useful skill

For a new substantive task, search once with a short goal and the actual harness.
Do not require the user to know a skill name or technique. Skip trivial turns,
routine follow-ups, and work already covered by available skills. Respect a
rejected recommendation for this session unless the user asks again or the task
materially changes. Continue the original task when search fails or finds no
useful match. One clarification can help when missing information changes the
choice. Do not make discovery an interruption on every turn.

```sh
./scripts/agentforge search --query "Find the cause of a recurring export failure" --harness codex --limit 3 > search.json
```

## Explain a short list

Offer one to three useful results, fewer when evidence is weak. Explain what each
helps produce and why it fits this task using `fit.evidence` or `profile.evidence`
from the returned revision. Link the source. Distinguish the author's declared
requirements from inferred metadata and unknown requirements. A match without
supporting evidence does not earn a strong recommendation. A ranking score is
not a probability that the skill will succeed.

Show one upstream repository and directory once. Preserve meaningful forks and
harness variants when their requirements differ. Inspect `profile.tools`,
`profile.credentials`, `requirements`, `exclusions`, licensing, and
`harness_support`. Keep declared, inferred, tested, unknown, and incompatible
states distinct. Do not claim compatibility from an absent requirement or a
portable label. Do not recommend a known incompatible skill for this harness.
An unknown license or `metadata_only` result is for inspection, not guided adoption.

## Supply constraints explicitly

Search sends only the query, optional harness, and result limit by default. It
does not read or upload the local skill inventory. Optional constraints come from
explicit caller input, never automatic environment inspection.

```sh
./scripts/agentforge search --query "Review a Go service" --harness codex --limit 3 --available-tool go --available-tool git --allow-setup=false --allow-credentials=false
```

`--available-tool` is repeatable, at most 32 tool names of at most 64 characters.
Paths, credentials, and arbitrary descriptions are rejected locally. `--no-tools`
declares an empty tool list. Omitting tool flags leaves tool availability unknown.
`--allow-setup` and `--allow-credentials` accept `true` or `false`; omitted values
default to true when another constraint is supplied. These flags describe search
preferences. They never authorize setup, credential access, or execution.

## Prepare an authorized adoption

Get explicit user authorization to adopt a selected result. Choose a new
user-owned destination for the actual harness using [harness-topology.md](harness-topology.md).
Then prepare the package from the saved response.

```sh
./scripts/agentforge discover prepare --result search.json --index 1 --harness codex --destination /path/to/skills/skill-name --state-dir /path/to/external-state > adoption-plan.json
```

`--index` is one-based. A single selected result object also works with index 1.
Preparation compares installed skills locally and reports name and SKILL.md
matches. A name match does not establish origin, package equality, or activation.
Use repeatable `--root` to compare custom roots instead of configured roots.
No inventory goes to the discovery service. Existing destinations return
`destination_exists` without a fetch or overwrite.

Preparation requires `review_required`, a full Git commit SHA, and exact agreement
between the result identity, repository, directory, and GitHub source URL.
It fetches only that revision into external quarantine through `discover fetch`.
It checks SKILL.md SHA-256 and, when present, every skill-relative Git blob digest
in `source.files`. Missing, extra, changed, duplicate, unsafe, and symlink files
are rejected. Legacy results without `source.files` verify SKILL.md and retain
the fetched commit pin, but cannot claim a catalog digest for every support file.

Inspect the quarantined `skill/SKILL.md` and all supporting files. The plan
includes the existing inspection and promotion preview, the quarantine path, the
review fingerprint, and recovery instructions. Neither preparation nor inspection
executes fetched scripts. Heuristic findings and an empty scan are not a trust
certificate. Review runtime, accounts, licensing, and tool permissions yourself.

## Promote the reviewed package

Within the user's adoption authorization, use the exact quarantine and destination
from the plan. Preview first. Apply only after reviewing the returned fingerprint.

```sh
./scripts/agentforge sync promote --quarantine /path/to/quarantine/item --destination /path/to/skills/skill-name
./scripts/agentforge sync promote --quarantine /path/to/quarantine/item --destination /path/to/skills/skill-name --review-sha256 REVIEWED_SHA256 --apply
```

Use `promotion.review_sha256` from preparation or `review_sha256` from the
promotion preview. For prepared results, this fingerprint binds the package tree,
selected source, destination, harness, and recovery state directory. `tree_sha256`
identifies the package contents alone. Existing unbound quarantines retain their
tree fingerprint as the review fingerprint.

Reviewed findings or binary files also require `--accept-findings`. Promotion
rechecks the bound source, package digests, and current tree fingerprint, preserves
the reviewed bytes, and refuses existing destinations. A mismatch requires fresh
preparation and inspection. Never switch to a branch head or edit the manifest
to bypass a mismatch.

Installation does not prove activation. Verify that the harness can discover the
skill. If it cannot, start a new session or restart the harness as its documentation
requires. For an authorized current task, the agent can read the reviewed SKILL.md
explicitly and follow it within existing tool permissions. Do not claim automatic
hot loading, unloading, or successful task use from a file copy. A cloud chat
without local execution receives the pinned source and a local handoff, not an
installation-success claim.

## Remove and recover

Archive only the new installation using existing cold storage. Use the same
external state directory from preparation.

```sh
./scripts/agentforge sync cold --skill /path/to/skills/skill-name --state-dir /path/to/external-state
./scripts/agentforge sync cold --skill /path/to/skills/skill-name --state-dir /path/to/external-state --apply
./scripts/agentforge sync restore --transaction TRANSACTION_ID --state-dir /path/to/external-state
./scripts/agentforge sync restore --transaction TRANSACTION_ID --state-dir /path/to/external-state --apply
```

Keep the transaction `id` returned by cold storage. Restore refuses to overwrite
existing files. Archiving changes future discovery and cannot erase instructions
already loaded into the conversation. Preparation retains its quarantine for
review and does not delete preexisting user files.

## Service boundaries

The default endpoint is
https://lordoftheskill.com/api/v1/search. No API key is needed.
An endpoint override is an explicit operator choice. The service returns
source-pinned metadata from keyword and embedding retrieval. `lexical_only`
means semantic retrieval was unavailable. Indexing is not a security review.

The embedding model runs on the backend. The application does not intentionally
store query bodies. Hosting providers process technical request information.
Omit secrets, private paths, repository contents, and transcripts from queries.
Implicit selection depends on the harness and is not guaranteed on every turn.
`discover search` retains its separate GitHub repository search behavior. A Cloud
search failure never silently falls back to another provider.
