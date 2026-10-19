# AgentForge

Keep agent skills useful as repositories, harnesses and working practices change.

AgentForge v0.1 is a local skill backed by one Go executable. It inventories skill
folders and eager rules, flags duplicate names, scores skill structure, compares
routing descriptions, exports portable bundles, quarantines public downloads,
and mines local evidence for maintenance proposals. The executable has no Python,
Node.js, Go-toolchain or third-party package requirement. Git is optional for
history analysis. The source uses only Go's standard library.

## Install

Native ZIP bundles contain `SKILL.md`, references and the matching executable
under `scripts/`. CI builds Linux, macOS and Windows bundles for amd64 and arm64;
download the `agentforge-native-bundles` artifact from a successful
[Skill checks run](https://github.com/agentforge-hq/agentforge/actions/workflows/test.yml).
Verify the archive against its `CHECKSUMS.txt`, then extract the `agentforge` folder
into your skill root. No interpreter or compiler is needed to run that bundle.

To build a fresh installation from source, use Go 1.23+:

```sh
git clone https://github.com/agentforge-hq/agentforge.git ~/.agents/skills/agentforge
cd ~/.agents/skills/agentforge
go build -trimpath -o scripts/agentforge ./cmd/agentforge
```

If that directory already exists, keep it until you have reviewed and backed up
local changes. Harness discovery varies; consult [the topology](references/harness-topology.md).
Cloud products use manual exports and may not execute bundled scripts.

On Windows build `scripts/agentforge.exe` and use `./scripts/agentforge.exe` in
the commands below. Filesystem support for symlinks, hard links and rename
semantics depends on the OS, account permissions and volume. Cold moves require
the same filesystem; the tool rejects failed moves and retains recovery journals.

## Use

Ask a compatible agent to use `agentforge` to audit or maintain your skills.
For direct use, run these commands from this repository:

```sh
./scripts/agentforge audit --project /path/to/repo
./scripts/agentforge lint /path/to/skill --fail-under 80
./scripts/agentforge sync --help
./scripts/agentforge discover --help
./scripts/agentforge drift --help
```

Scanner and linter reports are JSON. Management commands preview by default;
`--apply` performs the selected local change. Discovery fetches public content
only when explicitly invoked, and leaves it in quarantine. Promotion requires a
reviewed fingerprint. Cold moves have transaction journals and restore support.
`scan` aliases `audit`; `drift automate-me` aliases `drift distill`. Shared
`--config`, `--project` and `--state-dir` options can appear before or after the
command. Reports use exit 1 for findings or failed gates and exit 2 for invalid
input or operational failure.

`audit` discovers configured home and project roots automatically. `--project`
sets the workspace scope and its Git ancestors; home roots remain included.
Use `audit --project "$HOME"` for home scope. Repeat `--root` for a custom
skill-only inventory. This replaces the configured roots and skips eager-rule
scanning. `coverage` shows the roots actually considered. The
[report schema](references/report-schema.md) defines JSON fields, including
`description`, `tokens_estimate` and `index_tokens_estimate`.

Cold-store several explicitly selected skills in one transaction:

```sh
./scripts/agentforge sync cold --skill /path/one --skill /path/two
# Review every item, then repeat the same command with --apply.
```

One transaction ID restores the batch. Deduplication also accepts repeated
`--duplicate` options with one `--canonical`. These commands preserve their
fingerprint checks and do not select skills for you. See the
[management playbook](references/healing-playbooks.md) for review and recovery.

## What the numbers mean

SDQS is a configurable structural heuristic, not measured task success.
TF-IDF highlights lexical overlap, not model routing accuracy. Token counts use
a documented UTF-8 byte proxy. Injection scans cannot establish trust.
Filesystem exposure does not prove a harness loaded every copy into its prompt.
The Codex audit includes the locally observed `.system` built-in directory;
cached or built-in files may still be absent from an active session catalog.
The linter distinguishes explicit resource dependencies from unverified prose
mentions. Its lexical trigger tests abstain on weak evidence using configurable
similarity and shared-term floors; this is still a routing proxy.
Cold storage changes future discovery; it cannot remove content already read
into a conversation. See [the rubric](references/eval-rubric.md) and
[maintenance playbooks](references/healing-playbooks.md) for verification boundaries.

This release does not include a registry, hosted service, GitHub App, autonomous
patching, full YAML parser, or a guarantee of complete compatibility with every
agent. It implements the commands documented by each script's `--help`.

## Develop

```sh
go test ./...
go vet ./...
go build -o scripts/agentforge ./cmd/agentforge
./scripts/agentforge lint . --fail-under 80
go run ./scripts/build.go --all
```

The test suite uses temporary directories and mocks for network-sensitive cases.
Python-generated golden scan and transaction fixtures verify migration
compatibility without requiring Python at test time. Cross-platform CI covers
Go 1.23 and 1.27 on Linux, macOS and Windows; symlink cases skip when the account
cannot create symlinks. The build helper writes six native ZIP bundles and
checksums under `dist/`. No API key is needed.

Go script syntax checks use Go's AST. Python script compatibility checks inspect
literal flags and balanced tokens and explicitly leave full Python syntax
unverified; inspected scripts are never imported or executed.

Please include a reproduction, the observed harness
version, a before/after result and focused verification with a proposed change.
Report private data or suspected vulnerabilities through GitHub's private
vulnerability reporting when available; do not paste credentials into an issue.

## License

[Apache-2.0](LICENSE). Copyright 2026 Pierre Brunelle.
