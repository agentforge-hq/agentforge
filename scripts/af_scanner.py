#!/usr/bin/env python3
"""Inventory skills and rule exposure. Also supplies shared, standard-library-only utilities."""

from __future__ import annotations

import sys

if sys.version_info < (3, 11):
    raise SystemExit('agentforge requires Python 3.11+; try python3.11')

import argparse
import glob
import hashlib
import json
import math
import os
import re
import stat
import tempfile
from collections.abc import Callable
from pathlib import Path
from typing import Any

VERSION = '0.1.0'
SKILL_ROOT = Path(__file__).resolve().parent.parent
DEFAULT_STATE = Path.home() / '.local/share/agentforge'
IGNORED = {'.git', '__pycache__', '.pytest_cache', '.ruff_cache', '.DS_Store'}
NAME_RE = re.compile(r'^[a-z0-9]+(?:-[a-z0-9]+)*$')


def emit(value: Any) -> None:
    print(json.dumps(value, indent=2, sort_keys=True, ensure_ascii=True))


def entrypoint(main: Callable[[], int | None]) -> int:
    try:
        return main() or 0
    except (OSError, ValueError, KeyError) as exc:
        print(json.dumps({'error': str(exc)}), file=sys.stderr)
        return 2


def read_text(path: Path, limit: int = 1_048_576) -> str:
    with path.open('rb') as stream:
        data = stream.read(limit + 1)
    if len(data) > limit:
        raise ValueError(f'File exceeds {limit} bytes: {path}')
    return data.decode('utf-8-sig')


def atomic_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, temporary = tempfile.mkstemp(prefix='.af-', dir=path.parent)
    try:
        with os.fdopen(fd, 'w', encoding='utf-8') as stream:
            json.dump(value, stream, indent=2, sort_keys=True)
            stream.write('\n')
        os.replace(temporary, path)
    finally:
        Path(temporary).unlink(missing_ok=True)


def estimated_tokens(text: str) -> int:
    """UTF-8 bytes/4 proxy, NOT a tokenizer or a model billing count."""
    return math.ceil(len(text.encode('utf-8')) / 4)


def valid_name(name: str) -> bool:
    return len(name) <= 64 and NAME_RE.fullmatch(name) is not None


def frontmatter(text: str) -> tuple[dict[str, Any], str, dict[str, str], list[str]]:
    """Read common YAML scalars without PyYAML; retain original top-level blocks losslessly.

    Nested mappings/lists are opaque blocks, not interpreted YAML. Unsupported routing
    scalars produce diagnostics instead of being silently guessed.
    """
    lines = text.splitlines(keepends=True)
    if not lines or lines[0].strip() != '---':
        return {}, text, {}, ['Missing YAML frontmatter']
    end = next((i for i in range(1, len(lines)) if lines[i].strip() == '---'), None)
    if end is None:
        return {}, text, {}, ['Unclosed YAML frontmatter']
    fields: dict[str, Any] = {}
    blocks: dict[str, str] = {}
    problems: list[str] = []
    starts = [i for i in range(1, end) if re.match(r'^[A-Za-z][\w-]*\s*:', lines[i])]
    for pos, start in enumerate(starts):
        stop = starts[pos + 1] if pos + 1 < len(starts) else end
        key, raw = lines[start].split(':', 1)
        key, raw = key.strip(), raw.strip()
        if key in fields:
            problems.append(f'Duplicate frontmatter key: {key}')
        block = ''.join(lines[start:stop])
        blocks[key] = block
        continuation = lines[start + 1 : stop]
        if re.fullmatch(r'[>|][+-]?(?:\s+#.*)?', raw):
            content = [line.strip() for line in continuation if not line.lstrip().startswith('#')]
            fields[key] = (' ' if raw.startswith('>') else '\n').join(content).strip()
        elif not raw:
            fields[key] = {'opaque_yaml': ''.join(continuation)}
        else:
            try:
                if raw.startswith('"'):
                    fields[key] = json.JSONDecoder().raw_decode(raw)[0]
                elif raw.startswith("'"):
                    match = re.fullmatch(r"'((?:[^']|'')*)'\s*(?:#.*)?", raw)
                    if not match:
                        raise ValueError('Unclosed YAML single-quoted string')
                    fields[key] = match.group(1).replace("''", "'")
                elif raw.startswith(('[', '{', '&', '*', '!')):
                    fields[key] = {'opaque_yaml': raw}
                    if key in {'name', 'description', 'compatibility'}:
                        problems.append(f'Unsupported YAML scalar for {key}; use a plain, quoted or block string')
                elif raw in {'true', 'false'}:
                    fields[key] = raw == 'true'
                else:
                    fields[key] = re.split(r'\s+#', raw, maxsplit=1)[0]
                    if continuation and key in {'name', 'description', 'compatibility'}:
                        content = [s.strip() for s in continuation if s.strip() and not s.lstrip().startswith('#')]
                        if content:
                            fields[key] += ' ' + ' '.join(content)
            except (ValueError, json.JSONDecodeError) as exc:
                fields[key] = ''
                problems.append(f'Cannot read {key}: {exc}')
    for line in lines[1:end]:
        if line.strip() and not line[0].isspace() and not line.startswith('#') and not re.match(r'^[\w-]+\s*:', line):
            problems.append('Unsupported top-level YAML syntax')
    return fields, ''.join(lines[end + 1 :]), blocks, problems


def merge_config(base: dict[str, Any], patch: dict[str, Any]) -> dict[str, Any]:
    for key, value in patch.items():
        if isinstance(value, dict) and isinstance(base.get(key), dict):
            merge_config(base[key], value)
        else:
            base[key] = value
    return base


def load_config(path: Path | None = None) -> dict[str, Any]:
    config = json.loads(read_text(SKILL_ROOT / 'config.default.json'))
    if path:
        merge_config(config, json.loads(read_text(path)))
    weights = config['sdqs_weights']
    if not all(isinstance(v, (int, float)) and v >= 0 for v in weights.values()) or sum(weights.values()) != 100:
        raise ValueError('SDQS weights must be nonnegative numbers summing to 100')
    for section in ('budgets', 'limits'):
        if not all(isinstance(v, int) and v > 0 for v in config[section].values()):
            raise ValueError(f'{section} must contain positive integers')
    return config


def render_pattern(pattern: str, project: Path) -> str:
    values = {
        'home': str(Path.home()),
        'project': str(project),
        'codex_home': os.environ.get('CODEX_HOME', str(Path.home() / '.codex')),
        'xdg_config_home': os.environ.get('XDG_CONFIG_HOME', str(Path.home() / '.config')),
    }
    return os.path.expanduser(pattern.format(**values))


def expand(pattern: str, project: Path) -> list[Path]:
    return [Path(p).absolute() for p in sorted(glob.glob(render_pattern(pattern, project), recursive=True))]


def project_scopes(project: Path) -> list[Path]:
    """Walk session ancestors only inside the nearest Git worktree; no implicit home-wide scan."""
    if not project.is_dir():
        raise ValueError(f'Project directory does not exist: {project}')
    chain = [project, *project.parents]
    boundary = next((i for i, p in enumerate(chain) if (p / '.git').exists()), None)
    return chain[: boundary + 1] if boundary is not None else [project]


def scoped_paths(pattern: str, scopes: list[Path]) -> list[Path]:
    locations = scopes if '{project}' in pattern else scopes[:1]
    return sorted({p for project in locations for p in expand(pattern, project)})


def walk_skills(root: Path, max_dirs: int = 10000) -> tuple[list[Path], list[str]]:
    found: list[Path] = []
    errors: list[str] = []
    pending = [(root, frozenset())]
    visited_count = 0
    while pending:
        current, ancestors = pending.pop()
        visited_count += 1
        if visited_count > max_dirs:
            errors.append(f'Directory limit reached in {root}')
            break
        try:
            real = current.resolve()
            if real in ancestors:
                continue
            entries = sorted(current.iterdir(), key=lambda p: p.name)
            if (current / 'SKILL.md').is_file():
                found.append(current / 'SKILL.md')
            # Ancestry, rather than a global visited set, retains sibling alias exposure.
            pending.extend(
                (p, ancestors | {real})
                for p in reversed(entries)
                if not p.name.startswith('.') and p.name not in IGNORED and p.is_dir()
            )
        except (OSError, RuntimeError) as exc:
            errors.append(str(exc))
    return found, errors


def tree_digest(root: Path, limits: dict[str, int] | None = None) -> str:
    """Hash paths, bytes, executable bits and symlink targets. Never follow tree symlinks."""
    limits = limits or {
        'file_bytes': 1_048_576,
        'hash_file_bytes': 134_217_728,
        'tree_bytes': 268_435_456,
        'tree_files': 5000,
    }
    if not root.is_dir():
        raise ValueError(f'Expected a readable directory: {root}')
    digest = hashlib.sha256()
    total = count = 0

    def onerror(exc: OSError) -> None:
        raise exc

    for directory, dirs, files in os.walk(root, followlinks=False, onerror=onerror):
        dirs[:] = sorted(d for d in dirs if d not in IGNORED)
        for name in dirs:
            if not (Path(directory) / name).is_symlink():
                count += 1
                if count > limits['tree_files']:
                    raise ValueError(f'Too many entries in {root}')
                relative_dir = (Path(directory) / name).relative_to(root).as_posix().encode()
                digest.update(b'D' + len(relative_dir).to_bytes(8, 'big') + relative_dir)
        for name in sorted(files + [d for d in dirs if (Path(directory) / d).is_symlink()]):
            path = Path(directory) / name
            relative = path.relative_to(root).as_posix().encode()
            count += 1
            if count > limits['tree_files']:
                raise ValueError(f'Too many files in {root}')
            if path.is_symlink():
                payload = b'L' + os.fsencode(os.readlink(path))
                total += len(payload)
                if total > limits['tree_bytes']:
                    raise ValueError(f'Tree too large to fingerprint: {root}')
                digest.update(len(relative).to_bytes(8, 'big') + relative)
                digest.update(len(payload).to_bytes(8, 'big') + payload)
            else:
                before = path.stat()
                if not stat.S_ISREG(before.st_mode):
                    raise ValueError(f'Not a regular file: {path}')
                if before.st_size > limits.get('hash_file_bytes', limits['file_bytes']):
                    raise ValueError(f'File too large to fingerprint: {path}')
                header = b'F' + str(before.st_mode & 0o111).encode() + b'\0'
                total += len(header) + before.st_size
                if total > limits['tree_bytes']:
                    raise ValueError(f'Tree too large to fingerprint: {root}')
                digest.update(len(relative).to_bytes(8, 'big') + relative)
                digest.update((len(header) + before.st_size).to_bytes(8, 'big') + header)
                actual = 0
                with path.open('rb') as stream:
                    while chunk := stream.read(1_048_576):
                        actual += len(chunk)
                        if actual > before.st_size:
                            raise ValueError(f'File grew while fingerprinting: {path}')
                        digest.update(chunk)
                after = path.stat()
                if actual != before.st_size or (before.st_ino, before.st_mtime_ns, before.st_size) != (
                    after.st_ino,
                    after.st_mtime_ns,
                    after.st_size,
                ):
                    raise ValueError(f'File changed while fingerprinting: {path}')
    return digest.hexdigest()


def skill_record(
    path: Path, config: dict[str, Any], fingerprints: dict[Path, tuple[str | None, str | None]] | None = None
) -> dict[str, Any]:
    text = read_text(path, config['limits']['file_bytes'])
    fields, body, _, problems = frontmatter(text)
    description = fields.get('description', '')
    description = description if isinstance(description, str) else ''
    name = fields.get('name', path.parent.name)
    name = name if isinstance(name, str) else path.parent.name
    fingerprints = fingerprints if fingerprints is not None else {}
    real_root = path.parent.resolve()
    if real_root not in fingerprints:
        try:
            fingerprints[real_root] = (tree_digest(path.parent, config['limits']), None)
        except (OSError, ValueError) as exc:
            fingerprints[real_root] = (None, str(exc))
    fingerprint, fingerprint_error = fingerprints[real_root]
    return {
        'path': str(path.absolute()),
        'real_path': str(path.resolve()),
        'name': name,
        'description': description,
        'tokens_estimate': estimated_tokens(body),
        'index_tokens_estimate': estimated_tokens(name + '\n' + description + '\n' + str(path)),
        'tree_sha256': fingerprint,
        'fingerprint_error': fingerprint_error,
        'diagnostics': problems,
    }


def scan(config: dict[str, Any], project: Path, roots: list[Path] | None = None) -> dict[str, Any]:
    records: dict[str, dict[str, Any]] = {}
    rules: dict[str, dict[str, Any]] = {}
    errors: list[dict[str, str]] = []
    coverage: list[dict[str, Any]] = []
    fingerprints: dict[Path, tuple[str | None, str | None]] = {}
    scopes = project_scopes(project)
    profiles = (
        config['harnesses'] if roots is None else {'custom': {'skill_roots': [str(p) for p in roots], 'rules': []}}
    )
    for harness, profile in profiles.items():
        for pattern in profile['skill_roots']:
            resolved = scoped_paths(pattern, scopes)
            coverage.append(
                {
                    'harness': harness,
                    'pattern': pattern,
                    'matches': [str(p) for p in resolved],
                    'status': profile.get('status', 'custom'),
                }
            )
            for root in resolved:
                paths, issues = walk_skills(root, config['limits']['scan_dirs'])
                errors.extend({'path': str(root), 'error': issue} for issue in issues)
                for path in paths:
                    key = str(path.absolute())
                    if key not in records:
                        try:
                            records[key] = skill_record(path, config, fingerprints)
                            records[key]['harnesses'] = []
                            if records[key]['fingerprint_error']:
                                errors.append(
                                    {'path': key, 'error': records[key]['fingerprint_error'], 'stage': 'fingerprint'}
                                )
                        except (OSError, ValueError, UnicodeError) as exc:
                            errors.append({'path': key, 'error': str(exc)})
                            continue
                    if harness not in records[key]['harnesses']:
                        records[key]['harnesses'].append(harness)
        for rule in profile.get('rules', []):
            for path in scoped_paths(rule['path'], scopes):
                if not path.is_file():
                    continue
                key = str(path)
                try:
                    text = read_text(path, config['limits']['file_bytes'])
                    mode = rule.get('load', 'candidate')
                    if mode == 'frontmatter':
                        fields, _, _, _ = frontmatter(text)
                        mode = 'always' if fields.get('alwaysApply') is True else 'conditional'
                    if harness == 'codex':
                        if path.name == 'AGENTS.override.md':
                            mode = 'always'
                        elif path.name == 'AGENTS.md' and (path.parent / 'AGENTS.override.md').is_file():
                            mode = 'shadowed'
                    record = rules.setdefault(
                        key, {'path': key, 'tokens_estimate': estimated_tokens(text), 'harnesses': {}}
                    )
                    record['harnesses'][harness] = mode
                except (OSError, ValueError, UnicodeError) as exc:
                    errors.append({'path': key, 'error': str(exc)})
    groups: dict[str, list[dict[str, Any]]] = {}
    for record in records.values():
        groups.setdefault(record['name'], []).append(record)
    duplicates = []
    for name, copies in sorted(groups.items()):
        if len(copies) < 2:
            continue
        physical = {r['real_path'] for r in copies}
        hashes = {r['tree_sha256'] for r in copies}
        kind = (
            'aliases'
            if len(physical) == 1
            else 'unverified'
            if None in hashes
            else 'identical'
            if len(hashes) == 1
            else 'divergent'
        )
        duplicates.append(
            {
                'name': name,
                'kind': kind,
                'paths': [r['path'] for r in copies],
                'harness_exposure': {h: sum(h in r['harnesses'] for r in copies) for h in profiles},
            }
        )
    summaries = {}
    for harness in profiles:
        selected = [r for r in records.values() if harness in r['harnesses']]
        always = sum(r['tokens_estimate'] for r in rules.values() if r['harnesses'].get(harness) == 'always')
        conditional = sum(
            r['tokens_estimate'] for r in rules.values() if r['harnesses'].get(harness) in {'conditional', 'candidate'}
        )
        index = sum(r['index_tokens_estimate'] for r in selected)
        summaries[harness] = {
            'skill_occurrences': len(selected),
            'unique_physical_skills': len({r['real_path'] for r in selected}),
            'index_tokens_estimate': index,
            'always_rule_tokens_estimate': always,
            'conditional_rule_tokens_estimate': conditional,
            'over_index_budget': index > config['budgets']['index_tokens'],
            'over_eager_budget': always > config['budgets']['eager_tokens'],
        }
    return {
        'version': VERSION,
        'project': str(project),
        'project_scopes': [str(p) for p in scopes],
        'token_method': 'ceil(UTF-8 bytes / 4); heuristic, not exact',
        'scope': 'Configured paths only; exposure estimates, not observed harness prompt contents or precedence',
        'skills': sorted(records.values(), key=lambda r: r['path']),
        'rules': sorted(rules.values(), key=lambda r: r['path']),
        'duplicates': duplicates,
        'summaries': summaries,
        'coverage': coverage,
        'errors': errors,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, help='JSON overlay for config.default.json')
    parser.add_argument('--project', type=Path, default=Path.cwd())
    parser.add_argument('--root', type=Path, action='append', help='Scan only these roots; repeatable')
    parser.add_argument('--fail-on-budget', action='store_true')
    args = parser.parse_args()
    result = scan(load_config(args.config), args.project.resolve(), args.root)
    emit(result)
    return (
        1
        if result['errors']
        or (
            args.fail_on_budget
            and any(s['over_index_budget'] or s['over_eager_budget'] for s in result['summaries'].values())
        )
        else 0
    )


if __name__ == '__main__':
    raise SystemExit(entrypoint(main))
