# AgentForge

Keep agent skills useful as repositories, harnesses and working practices change.

AgentForge v0.1 is a local skill with five Python scripts. It inventories skill
folders and eager rules, flags duplicate names, scores skill structure, compares
routing descriptions, exports portable bundles, quarantines public downloads,
and mines local evidence for maintenance proposals. Python 3.11+ and its standard
library are enough. Git is optional for history analysis.

## Install

For a fresh installation, clone the repository into your shared skill root:

```sh
git clone https://github.com/agentforge-hq/agentforge.git ~/.agents/skills/agentforge
```

If that directory already exists, keep it until you have reviewed and backed up
local changes. Harness discovery varies; consult [the topology](references/harness-topology.md).
Cloud products use manual exports and may not execute bundled scripts.

## Use

Ask a compatible agent to use `agentforge` to audit or maintain your skills.
For direct use, run these commands from this repository:

```sh
python3.11 scripts/af_scanner.py --project /path/to/repo
python3.11 scripts/af_linter.py /path/to/skill --fail-under 80
python3.11 scripts/af_sync.py --help
python3.11 scripts/af_discover.py --help
python3.11 scripts/af_drift.py --help
```

Scanner and linter reports are JSON. Management commands preview by default;
`--apply` performs the selected local change. Discovery fetches public content
only when explicitly invoked, and leaves it in quarantine. Promotion requires a
reviewed fingerprint. Cold moves have transaction journals and restore support.

## What the numbers mean

SDQS is a configurable structural heuristic, not measured task success.
TF-IDF highlights lexical overlap, not model routing accuracy. Token counts use
a documented UTF-8 byte proxy. Injection scans cannot establish trust.
Filesystem exposure does not prove a harness loaded every copy into its prompt.
Cold storage changes future discovery; it cannot remove content already read
into a conversation. See [the rubric](references/eval-rubric.md) and
[maintenance playbooks](references/healing-playbooks.md) for verification boundaries.

This release does not include a registry, hosted service, GitHub App, autonomous
patching, full YAML parser, or a guarantee of complete compatibility with every
agent. It implements the commands documented by each script's `--help`.

## Develop

```sh
python3.11 -S -m unittest discover -s tests -v
python3.11 scripts/af_linter.py . --fail-under 80
```

The test suite uses temporary directories and mocks for network-sensitive cases.
No API key is needed. Please include a reproduction, the observed harness
version, a before/after result and focused verification with a proposed change.
Report private data or suspected vulnerabilities through GitHub's private
vulnerability reporting when available; do not paste credentials into an issue.

## License

[Apache-2.0](LICENSE). Copyright 2026 Pierre Brunelle.
