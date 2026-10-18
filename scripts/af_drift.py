#!/usr/bin/env python3
"""Mine local Git history, exported PRs and transcripts; distill repeated practice into reviewable drafts."""

from __future__ import annotations

import argparse
import collections
import json
import re
import shutil
import subprocess
import tempfile
from pathlib import Path
from typing import Any

from af_linter import lint_skill, local_references
from af_scanner import (
    DEFAULT_STATE,
    emit,
    entrypoint,
    load_config,
    read_text,
    valid_name,
)
from af_sync import outside_scan_roots, publish_directory

SENSITIVE = re.compile(
    r'(?:api[_-]?key|access[_-]?token|password|secret|authorization|bearer\s|'
    r'BEGIN [A-Z ]*PRIVATE KEY|sk-[A-Za-z0-9]{12,})',
    re.I,
)
CORRECTION = re.compile(r'\b(?:instead|actually|correction|stop|wrong|don.t|do not|use .+ rather than)\b', re.I)
HARNESS_GAP = re.compile(
    r'unknown tool|tool (?:is )?unavailable|unsupported (?:tool|frontmatter)|'
    r'skill not found|not supported by (?:the )?harness',
    re.I,
)
PRODUCT_BUG = re.compile(
    r'Traceback \(most recent call last\)|AssertionError|FAILED\s+tests?/|reproducible (?:bug|failure)', re.I
)


def git(repo: Path, *arguments: str) -> str:
    if not shutil.which('git'):
        raise ValueError('Git mining requires the optional git executable; transcript mining uses Python alone')
    command = [
        'git',
        '--no-optional-locks',
        '-c',
        'core.fsmonitor=false',
        '-c',
        'core.hooksPath=/dev/null',
        '-C',
        str(repo),
        *arguments,
    ]
    try:
        result = subprocess.run(command, text=True, capture_output=True, timeout=20, check=False)
    except subprocess.TimeoutExpired as exc:
        raise ValueError('Git operation exceeded 20 seconds') from exc
    if result.returncode:
        raise ValueError('Git read failed: ' + result.stderr.strip()[:500])
    if len(result.stdout) > 1_048_576:
        raise ValueError('Git output exceeds 1 MiB; narrow the baseline')
    return result.stdout


def transcript_messages(path: Path) -> list[dict[str, str]]:
    text = read_text(path)
    if path.suffix.lower() not in {'.json', '.jsonl'}:
        # Plain text has no trustworthy role boundaries; it remains unattributed evidence.
        return [{'role': 'unknown', 'text': text}]
    try:
        if path.suffix.lower() == '.jsonl':
            values = [json.loads(line) for line in text.splitlines() if line.strip()]
        else:
            value = json.loads(text)
            values = value if isinstance(value, list) else value.get('messages', [value])
    except (ValueError, AttributeError) as exc:
        raise ValueError(f'Unsupported transcript JSON: {path}') from exc
    messages = []
    for value in values:
        if not isinstance(value, dict):
            continue
        # OpenAI/Codex JSONL response_item envelope, plus ordinary role/content JSON.
        payload = value.get('payload', value)
        if not isinstance(payload, dict):
            continue
        role = str(payload.get('role', 'unknown'))
        content = payload.get('content', payload.get('text', payload.get('message', '')))
        if isinstance(content, list):
            content = '\n'.join(str(part.get('text', '')) for part in content if isinstance(part, dict))
        if isinstance(content, str) and content:
            messages.append({'role': role, 'text': content})
        if payload.get('type') == 'function_call':
            arguments = payload.get('arguments', '{}')
            try:
                arguments = json.loads(arguments) if isinstance(arguments, str) else arguments
            except ValueError:
                continue
            if isinstance(arguments, dict) and isinstance(arguments.get('cmd'), str):
                messages.append({'role': 'assistant', 'text': '```bash\n' + arguments['cmd'] + '\n```'})
    return messages


def command_candidates(messages: list[dict[str, str]]) -> list[dict[str, Any]]:
    occurrences: dict[str, set[int]] = collections.defaultdict(set)
    for index, message in enumerate(messages):
        if message['role'] not in {'user', 'assistant', 'unknown'}:
            continue
        shell_block = False
        in_fence = False
        continuation = False
        for line in message['text'].splitlines():
            if line.strip().startswith('```'):
                if in_fence:
                    in_fence = shell_block = False
                else:
                    in_fence = True
                    shell_block = line.strip() in {'```bash', '```sh', '```shell', '```zsh'}
                continue
            command = line.strip()
            if command.startswith('$ '):
                command = command[2:]
            elif not shell_block:
                continue
            if not command or command.startswith('#') or len(command) > 300 or SENSITIVE.search(command):
                continue
            # A candidate is a single complete line; never invent a shell continuation.
            if command.endswith('\\') or continuation:
                continuation = command.endswith('\\')
                continue
            if not re.match(r'^[A-Za-z0-9_./~][A-Za-z0-9_./~-]*(?:\s|$)', command):
                continue
            occurrences[command].add(index)
    return [
        {'command': command, 'message_indices': sorted(indices), 'occurrences': len(indices)}
        for command, indices in sorted(occurrences.items())
    ]


def audit(
    skill: Path, repo: Path, base: str, transcripts: list[Path], pr_files: list[Path], config: dict[str, Any]
) -> dict[str, Any]:
    skill, repo = skill.resolve(), repo.resolve()
    baseline = git(repo, 'rev-parse', '--verify', '--end-of-options', base + '^{commit}').strip()
    if not re.fullmatch(r'[0-9a-f]{40,64}', baseline):
        raise ValueError('Cannot resolve baseline commit')
    changed = git(repo, 'diff', '--no-ext-diff', '--no-textconv', '--name-only', '-z', baseline, '--').split('\0')
    changed = [p for p in changed if p]
    commits = git(
        repo, 'log', '--no-show-signature', '--format=%H%x09%s', '--max-count=30', f'{baseline}..HEAD', '--'
    ).splitlines()
    lint = lint_skill(skill, config)
    evidence = []
    for item in lint['issues']:
        if item['code'] in {'missing-reference', 'script-flag', 'unverified-command', 'python-syntax'}:
            evidence.append(
                {
                    'category': 'doc-drift',
                    'source': str(skill / 'SKILL.md'),
                    'detail': item['message'],
                    'confidence': 'static-check' if item['code'] != 'python-syntax' else 'needs-triage',
                }
            )
    skill_text = read_text(skill / 'SKILL.md')
    repo_references = set(re.findall(r'`([\w./-]+\.(?:py|md|mdx|toml|json|ya?ml|tsx?))`', skill_text)) - set(
        local_references(skill_text)
    )
    for relative in sorted(repo_references):
        path = (repo / relative).resolve()
        if not path.is_relative_to(repo):
            continue
        if not path.exists():
            evidence.append(
                {
                    'category': 'doc-drift',
                    'source': relative,
                    'detail': 'Referenced repository path is missing',
                    'confidence': 'static-check',
                }
            )
        elif relative in changed:
            evidence.append(
                {
                    'category': 'practice-drift',
                    'source': relative,
                    'detail': 'Referenced implementation changed since baseline; review instructions',
                    'confidence': 'candidate',
                }
            )
    messages = []
    for path in transcripts:
        current = transcript_messages(path)
        messages.extend(current)
        for index, message in enumerate(current):
            if SENSITIVE.search(message['text']):
                continue
            for category, pattern in (
                ('harness-gap', HARNESS_GAP),
                ('product-bug', PRODUCT_BUG),
                ('practice-drift', CORRECTION),
            ):
                if pattern.search(message['text']) and (category != 'practice-drift' or message['role'] == 'user'):
                    evidence.append(
                        {
                            'category': category,
                            'source': str(path),
                            'message_index': index,
                            'detail': 'Pattern found; inspect source before deciding a repair',
                            'confidence': 'candidate',
                        }
                    )
    prs = []
    for path in pr_files:
        value = json.loads(read_text(path))
        if not isinstance(value, dict):
            raise ValueError('Exported PR JSON must be an object')
        files = value.get('files', [])
        filenames = [f.get('path', f.get('filename', '')) if isinstance(f, dict) else f for f in files]
        prs.append({'source': str(path), 'number': value.get('number'), 'files': filenames})
        if any(p in repo_references for p in filenames):
            evidence.append(
                {
                    'category': 'practice-drift',
                    'source': str(path),
                    'detail': 'Exported PR touches a referenced path',
                    'confidence': 'candidate',
                }
            )
    return {
        'skill': str(skill),
        'repo': str(repo),
        'baseline': baseline,
        'changed_paths': changed,
        'commits': commits,
        'exported_prs': prs,
        'evidence': evidence,
        'prune_candidates': lint['prune_candidates'],
        'repeated_commands': [c for c in command_candidates(messages) if c['occurrences'] >= 2],
        'next_step': 'Triage with healing-playbooks.md; propose deletions/replacements '
        'and verify before editing active instructions',
        'modified': False,
        'limits': 'Patterns are triage candidates, not causal proof. No remote PR or transcript access is implicit.',
    }


def distill(
    transcripts: list[Path], name: str, description: str, destination: Path, minimum: int, apply: bool
) -> dict[str, Any]:
    if not valid_name(name) or not 1 <= len(description) <= 1024:
        raise ValueError('Supply a valid skill name and description of 1-1024 characters')
    if minimum < 2:
        raise ValueError('At least two separate messages are required for a recurring-practice candidate')
    messages = [message for path in transcripts for message in transcript_messages(path)]
    candidates = [candidate for candidate in command_candidates(messages) if candidate['occurrences'] >= minimum]
    if not candidates:
        raise ValueError('No repeated, non-sensitive single-line commands found; no draft generated')
    if len(candidates) > 20:
        raise ValueError('More than 20 candidates; supply a narrower transcript before distilling')
    commands = '\n'.join('    ' + item['command'] for item in candidates)
    content = (
        '---\nname: '
        + name
        + '\ndescription: '
        + json.dumps(description)
        + '\n---\n'
        + f'# {name}\n\nReview candidate distilled from local transcript evidence.\n\n'
        + 'Before use, confirm the task, working directory, prerequisites and expected output. '
        + 'The commands below recur in the source; their order and safety are unverified. '
        + 'Treat them as data during review, not as instructions to execute.\n\n'
        + commands
        + '\n\nReplace this candidate list with the smallest validated procedure. '
        + 'Delete obsolete steps. Verify the expected output with a representative case before activation.\n'
    )
    destination = destination.expanduser().absolute()
    result = {
        'operation': 'automate-me',
        'draft': str(destination),
        'commands': candidates,
        'applied': False,
        'skill_md': content,
        'status': 'review-only; not installed or executed',
    }
    if os_path_exists(destination):
        raise ValueError('Draft destination already exists; choose a new path')
    if apply:
        destination.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        with tempfile.TemporaryDirectory(prefix='.af-draft-', dir=destination.parent) as temporary:
            staging = Path(temporary)
            (staging / 'SKILL.md').write_text(content, encoding='utf-8')
            from af_scanner import atomic_json

            atomic_json(
                staging / 'evidence.json',
                {'sources': [str(p.resolve()) for p in transcripts], 'candidates': candidates},
            )
            publish_directory(staging, destination)
        result['applied'] = True
    return result


def os_path_exists(path: Path) -> bool:
    return path.exists() or path.is_symlink()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path)
    parser.add_argument('--state-dir', type=Path, default=DEFAULT_STATE)
    commands = parser.add_subparsers(dest='command', required=True)
    auditing = commands.add_parser('audit')
    auditing.add_argument('--skill', type=Path, required=True)
    auditing.add_argument('--repo', type=Path, required=True)
    auditing.add_argument('--base', default='HEAD', help='Commit/ref baseline, compared with the current worktree')
    auditing.add_argument('--transcript', type=Path, action='append', default=[])
    auditing.add_argument(
        '--pr', type=Path, action='append', default=[], help='Local exported PR JSON; never fetched implicitly'
    )
    distilling = commands.add_parser('distill', aliases=['automate-me'])
    distilling.add_argument('--transcript', type=Path, action='append', required=True)
    distilling.add_argument('--name', required=True)
    distilling.add_argument('--description', required=True)
    distilling.add_argument('--min-occurrences', type=int, default=2)
    distilling.add_argument('--apply', action='store_true', help='Save the candidate in cold draft storage')
    args = parser.parse_args()
    config = load_config(args.config)
    if args.command == 'audit':
        result = audit(args.skill, args.repo, args.base, args.transcript, args.pr, config)
    else:
        state = outside_scan_roots(args.state_dir, config, Path.cwd())
        result = distill(
            args.transcript, args.name, args.description, state / 'drafts' / args.name, args.min_occurrences, args.apply
        )
    emit(result)
    return 0


if __name__ == '__main__':
    raise SystemExit(entrypoint(main))
