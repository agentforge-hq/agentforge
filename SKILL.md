---
name: agentforge
license: Apache-2.0
description: Audit, lint, deduplicate and port agent skills; profile eager rules; quarantine new skills; distill recurring practice from local evidence. Use for skill maintenance and cross-harness context budgets. Not for ordinary application coding or editing unrelated prompts.
---
# agentforge

Work from this skill directory. Use the bundled `./scripts/agentforge` (`./scripts/agentforge.exe` on Windows). Native bundles need neither Python nor Go installed. For a source checkout, build as described in [README.md](README.md). Start with the command's `--help`; read only the reference needed for the job.

1. **Measure.** Run `./scripts/agentforge audit --project /path/to/repo`. Known roots are automatic; repeat `--root` for a custom skill-only inventory. Read [report-schema.md](references/report-schema.md) when interpreting JSON. Report index estimates, always/conditional rules, aliases, identical copies and divergent names separately. Estimates are not measured harness context.
2. **Lint.** Run `./scripts/agentforge lint /path/to/skill --fail-under 80`. Read [eval-rubric.md](references/eval-rubric.md) for SDQS, pruning and A/B trigger cases. Read [lint-codes.md](references/lint-codes.md) when interpreting a finding. Static scores and lexical collisions require behavioral follow-up.
3. **Manage.** Use `./scripts/agentforge compare --canonical /path/to/skill` to inspect its name group; add `--copy` for explicit copies. Normalized paths are diagnostic only. Use `sync` subcommands: `dedupe`, `cold`, `restore`, `export`, `promote`. Read [harness-topology.md](references/harness-topology.md) before choosing a target. Keep a user-owned canonical source. Managed copies require their harness manager. Cold storage affects future discovery, never claims to erase a transcript.
4. **Discover.** Use `./scripts/agentforge discover search`, `fetch` or `inspect`. Downloads stay in external quarantine. Never execute fetched scripts during review. Inspect instructions and all supporting files; heuristic injection scans cannot certify trust. Promotion requires the reviewed fingerprint.
5. **Heal.** Use `./scripts/agentforge drift audit` with local Git/PR/transcript evidence, or `distill` (`automate-me`) for recurring commands. Read [healing-playbooks.md](references/healing-playbooks.md). Repair the demonstrated cause. Keep distillation in drafts until validated.

Do not skip previews before `--apply`, fingerprint review before promotion, or representative verification before activation. Apply only within existing user authorization. Never resolve divergent copies by deleting one arbitrarily, edit plugin caches as canonical sources, or prune eager rules automatically. Preserve tool restrictions and report lossy exports. Prefer a deletion or replacement over accumulating instructions; record the evidence and before/after budget.

Return the result, verification and remaining uncertainty. If a harness lacks a verified isolation mapping, pass file pointers or use its documented delegation mechanism; never invent `subagent` flags.
