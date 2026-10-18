#!/usr/bin/env python3
"""Preview or apply reversible deduplication, cold storage, exports and quarantine promotion."""

from __future__ import annotations

import argparse
import contextlib
import glob
import hashlib
import json
import os
import re
import shutil
import tempfile
import uuid
import zipfile
from collections.abc import Iterator
from pathlib import Path
from typing import Any

from af_scanner import (
    DEFAULT_STATE,
    IGNORED,
    SKILL_ROOT,
    atomic_json,
    emit,
    entrypoint,
    expand,
    frontmatter,
    load_config,
    project_scopes,
    read_text,
    render_pattern,
    tree_digest,
    valid_name,
)


def outside_scan_roots(state: Path, config: dict[str, Any], project: Path) -> Path:
    state = state.expanduser().resolve()
    for profile in config['harnesses'].values():
        for pattern in profile['skill_roots']:
            for scope in project_scopes(project):
                rendered = render_pattern(pattern, scope)
                roots = expand(pattern, scope) if glob.has_magic(rendered) else [Path(rendered)]
                for root in roots:
                    if state.is_relative_to(root.resolve()):
                        raise ValueError(f'State/quarantine must be outside skill scan roots: {root}')
    return state


@contextlib.contextmanager
def locked_state(state: Path) -> Iterator[None]:
    state.mkdir(parents=True, exist_ok=True, mode=0o700)
    lock = state / '.mutation.lock'
    try:
        fd = os.open(lock, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    except FileExistsError as exc:
        raise ValueError(
            f'Another mutation or interrupted run owns {lock}; inspect before removing a stale lock'
        ) from exc
    try:
        with os.fdopen(fd, 'w') as stream:
            stream.write(str(os.getpid()))
        yield
    finally:
        lock.unlink(missing_ok=True)


def node_digest(path: Path) -> str:
    if path.is_symlink():
        return hashlib.sha256(b'LINK\0' + os.fsencode(os.readlink(path))).hexdigest()
    return tree_digest(path)


def checked_skill(path: Path) -> Path:
    path = path.expanduser().absolute()
    path = path.parent.resolve() / path.name
    if not path.is_dir() or not (path / 'SKILL.md').is_file():
        raise ValueError(f'Expected a skill directory: {path}')
    fields, _, _, problems = frontmatter(read_text(path / 'SKILL.md'))
    description = fields.get('description')
    if (
        problems
        or not isinstance(fields.get('name'), str)
        or not valid_name(fields['name'])
        or not isinstance(description, str)
        or not 1 <= len(description) <= 1024
    ):
        raise ValueError(f'Cannot manage invalid skill metadata: {path}: {problems}')
    return path


def archive(paths: list[Path], state: Path, apply: bool, canonical: Path | None = None) -> dict[str, Any]:
    paths = [checked_skill(p) for p in paths]
    if len(set(paths)) != len(paths):
        raise ValueError('Duplicate input paths')
    canonical_hash = tree_digest(canonical) if canonical else None
    canonical_name = frontmatter(read_text(canonical / 'SKILL.md'))[0]['name'] if canonical else None
    for path in paths:
        if SKILL_ROOT.is_relative_to(path.resolve()):
            raise ValueError('Cannot archive the running agentforge installation')
        if state.is_relative_to(path.resolve()):
            raise ValueError('Cold storage must be outside every moved skill')
        if canonical:
            if path.absolute() == canonical.absolute() or canonical.absolute().is_relative_to(path.absolute()):
                raise ValueError('Cannot remove the canonical path or its ancestor')
            if not path.is_symlink() and canonical.resolve().is_relative_to(path.resolve()):
                raise ValueError('Cannot archive a directory containing the canonical target')
            if path.resolve() == canonical.resolve() and not path.is_symlink():
                raise ValueError('A canonical path alias through a symlinked parent cannot be archived')
            fields = frontmatter(read_text(path / 'SKILL.md'))[0]
            if fields['name'] != canonical_name or tree_digest(path) != canonical_hash:
                raise ValueError(f'Divergent skill; choose or merge it manually: {path}')
            if path.resolve() != canonical.resolve():
                try:
                    portable_files(canonical)
                    portable_files(path)
                except ValueError as exc:
                    raise ValueError(
                        'Distinct trees containing symlinks require manual dependency review before deduplication'
                    ) from exc
        for other in paths:
            if path != other and (
                path.absolute().is_relative_to(other.absolute())
                or (not other.is_symlink() and path.resolve().is_relative_to(other.resolve()))
            ):
                raise ValueError('Nested archive inputs are not supported')
    items = [
        {
            'source': str(p),
            'parent_real': str(p.parent.resolve()),
            'sha256': node_digest(p),
            'kind': 'symlink' if p.is_symlink() else 'directory',
            'status': 'pending',
        }
        for p in paths
    ]
    plan = {
        'operation': 'dedupe' if canonical else 'cold',
        'applied': False,
        'canonical': str(canonical) if canonical else None,
        'items': items,
    }
    if not apply:
        return plan
    with locked_state(state):
        transaction_id = uuid.uuid4().hex
        transaction = state / 'transactions' / transaction_id
        transaction.mkdir(parents=True)
        for item in items:
            if Path(item['source']).parent.stat().st_dev != transaction.stat().st_dev:
                raise ValueError(
                    'Cold storage must be on the same filesystem for atomic moves; choose --state-dir there'
                )
        manifest = {**plan, 'id': transaction_id, 'status': 'prepared'}
        for i, item in enumerate(items):
            item['stored'] = str(i)
        atomic_json(transaction / 'manifest.json', manifest)
        moved = []
        try:
            for item in items:
                source = Path(item['source'])
                if source.parent.resolve() != Path(item['parent_real']) or node_digest(source) != item['sha256']:
                    raise ValueError(f'Source changed after preview: {source}')
                if canonical and tree_digest(canonical) != canonical_hash:
                    raise ValueError('Canonical skill changed during deduplication')
                os.replace(source, transaction / item['stored'])
                moved.append(item)
                item['status'] = 'stored'
                atomic_json(transaction / 'manifest.json', manifest)
        except BaseException:
            for item in reversed(moved):
                source = Path(item['source'])
                if not os.path.lexists(source):
                    os.replace(transaction / item['stored'], source)
                    item['status'] = 'rolled-back'
            manifest['status'] = 'interrupted'
            atomic_json(transaction / 'manifest.json', manifest)
            raise
        manifest.update(applied=True, status='stored')
        atomic_json(transaction / 'manifest.json', manifest)
        return manifest


def restore(transaction_id: str, state: Path, apply: bool) -> dict[str, Any]:
    if not re.fullmatch(r'[0-9a-f]{32}', transaction_id):
        raise ValueError('Invalid transaction ID')
    transaction = state / 'transactions' / transaction_id
    manifest = json.loads(read_text(transaction / 'manifest.json'))
    items = manifest['items']
    for item in items:
        if item['status'] not in {'stored', 'restored'}:
            raise ValueError('Transaction was interrupted; inspect its journal before recovery')
        stored = transaction / item['stored']
        if stored.parent != transaction or stored.is_absolute() is False:
            raise ValueError('Invalid journal storage path')
        source = Path(item['source'])
        if not source.is_absolute() or source.parent.resolve() != Path(item['parent_real']):
            raise ValueError(f'Restore parent changed: {source}')
        if item['status'] == 'restored':
            if node_digest(source) != item['sha256']:
                raise ValueError(f'Previously restored item has changed: {source}')
            continue
        if os.path.lexists(source):
            raise ValueError(f'Refusing to overwrite restore destination: {source}')
        if node_digest(stored) != item['sha256']:
            raise ValueError(f'Cold-storage contents changed: {stored}')
    result = {'operation': 'restore', 'transaction': transaction_id, 'applied': False, 'items': items}
    if apply:
        with locked_state(state):
            for item in items:
                if item['status'] == 'restored':
                    continue
                source = Path(item['source'])
                if os.path.lexists(source) or source.parent.resolve() != Path(item['parent_real']):
                    raise ValueError(f'Restore destination changed: {source}')
                os.replace(transaction / item['stored'], source)
                item['status'] = 'restored'
                atomic_json(transaction / 'manifest.json', manifest)
            manifest['status'] = 'restored'
            atomic_json(transaction / 'manifest.json', manifest)
            result['applied'] = True
    return result


def portable_files(source: Path, include_directories: bool = False) -> list[Path]:
    result = []
    for directory, dirs, files in os.walk(source, followlinks=False):
        dirs[:] = sorted(d for d in dirs if d not in IGNORED)
        for name in dirs + sorted(files):
            path = Path(directory) / name
            if path.is_symlink():
                raise ValueError(f'Portable exports cannot contain symlinks: {path}')
        result.extend(Path(directory) / f for f in sorted(files) if f != '.DS_Store')
        if include_directories:
            result.extend(Path(directory) / d for d in dirs)
    return result


def publish_directory(staging: Path, destination: Path) -> None:
    """Reserve an absent destination. Never merge with or overwrite an existing tree."""
    destination.mkdir(mode=0o700)
    try:
        for child in staging.iterdir():
            shutil.move(str(child), destination / child.name)
    except BaseException:
        shutil.rmtree(destination)
        raise


def export_skill(
    source: Path,
    harness: str,
    destination: Path,
    config: dict[str, Any],
    apply: bool,
    isolate: bool = False,
    allow_lossy: bool = False,
    format_name: str = 'skill',
) -> dict[str, Any]:
    source = checked_skill(source).resolve()
    destination = destination.expanduser().absolute()
    if destination.resolve().is_relative_to(source):
        raise ValueError('Export destination must be outside the source skill')
    if os.path.lexists(destination):
        raise ValueError(f'Destination already exists: {destination}')
    fields, body, blocks, errors = frontmatter(read_text(source / 'SKILL.md'))
    if errors:
        raise ValueError(str(errors))
    manual = harness in config['manual_targets']
    if harness == 'portable' or manual:
        supported = {'name', 'description', 'license', 'compatibility', 'metadata', 'allowed-tools'}
    elif harness in config['harnesses']:
        profile = config['harnesses'][harness]
        if profile['status'].startswith('unverified'):
            raise ValueError('Configure and verify this harness before exporting')
        supported = set(profile['frontmatter'])
    else:
        raise ValueError(f'Unknown harness: {harness}')
    if not manual and format_name == 'skill' and destination.name != fields['name']:
        raise ValueError('Export destination directory must match the declared skill name')
    if isolate and harness != 'claude-code':
        raise ValueError('Only Claude Code has a verified skill-level isolation mapping (context: fork)')
    dropped = sorted(set(fields) - supported)
    content = '---\n' + ''.join(
        block for key, block in blocks.items() if key in supported and not (isolate and key == 'context')
    )
    if isolate:
        content += 'context: fork\n'
    content += '---\n' + body
    if format_name == 'cursor-rule':
        if harness != 'cursor' or isolate:
            raise ValueError('cursor-rule format requires --harness cursor')
        # A small pointer avoids copying the entire skill into an eagerly available rule.
        content = (
            '---\ndescription: '
            + json.dumps(str(fields['description']))
            + '\nalwaysApply: false\n---\n'
            + f'Read `{source / "SKILL.md"}` when this task applies. Run its scripts from `{source}`.\n'
        )
        dropped = sorted(set(fields) - {'name', 'description'})
    files = portable_files(source, include_directories=True)
    result = {
        'operation': 'export',
        'harness': harness,
        'format': format_name,
        'destination': str(destination),
        'applied': False,
        'dropped_fields': dropped,
        'isolation': 'context: fork' if isolate else None,
        'files': [str(p.relative_to(source)) for p in files],
        'manual_upload': manual,
        'note': 'Local export only. Tool permissions and loading semantics remain harness-specific.',
    }
    if apply:
        if dropped and not allow_lossy:
            raise ValueError(f'Export drops fields {dropped}; review preview and use --allow-lossy if intended')
        expected = tree_digest(source)
        destination.parent.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(prefix='.af-export-', dir=destination.parent) as temporary:
            staging = Path(temporary)
            if manual:
                archive_path = staging / 'bundle.zip'
                with zipfile.ZipFile(archive_path, 'w', zipfile.ZIP_DEFLATED) as bundle:
                    for path in files:
                        relative = path.relative_to(source)
                        if path.is_dir():
                            bundle.writestr(f'{fields["name"]}/{relative.as_posix()}/', b'')
                            continue
                        data = content.encode() if relative.as_posix() == 'SKILL.md' else path.read_bytes()
                        bundle.writestr(f'{fields["name"]}/{relative.as_posix()}', data)
                if tree_digest(source) != expected:
                    raise ValueError('Source changed during export')
                os.link(archive_path, destination)
            elif format_name == 'cursor-rule':
                candidate = staging / 'rule.mdc'
                candidate.write_text(content, encoding='utf-8')
                os.link(candidate, destination)
            else:
                for path in files:
                    relative = path.relative_to(source)
                    target = staging / relative
                    if path.is_dir():
                        target.mkdir(parents=True, exist_ok=True)
                        continue
                    target.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(path, target)
                (staging / 'SKILL.md').write_text(content, encoding='utf-8')
                if tree_digest(source) != expected:
                    raise ValueError('Source changed during export')
                publish_directory(staging, destination)
        result['applied'] = True
    return result


def promote(
    quarantine: Path, destination: Path, reviewed: str | None, accept_findings: bool, apply: bool
) -> dict[str, Any]:
    quarantine = quarantine.expanduser().resolve()
    manifest = json.loads(read_text(quarantine / 'manifest.json'))
    source = checked_skill(quarantine / 'skill')
    digest = tree_digest(source)
    if digest != manifest['tree_sha256']:
        raise ValueError('Quarantine contents changed; fetch/inspect again')
    from af_discover import inspect_tree
    from af_linter import lint_skill

    inspection = inspect_tree(source)
    lint = lint_skill(source, load_config())
    blocking = [
        i
        for i in lint['issues']
        if i['severity'] == 'error' and i['code'] not in {'name-directory', 'unsafe-instruction'}
    ]
    result = {
        'operation': 'promote',
        'applied': False,
        'destination': str(destination),
        'tree_sha256': digest,
        'inspection': inspection,
        'lint_issues': lint['issues'],
    }
    if apply:
        if blocking:
            raise ValueError(f'Static validation failed before promotion: {blocking}')
        if reviewed != digest:
            raise ValueError('Promotion requires --review-sha256 matching the inspected tree fingerprint')
        if (inspection['findings'] or inspection['unscanned']) and not accept_findings:
            raise ValueError('Review findings/unscanned files and use --accept-findings if intended')
        fields = frontmatter(read_text(source / 'SKILL.md'))[0]
        if destination.name != fields['name']:
            raise ValueError('Promotion destination directory must match the skill name')
        result['export'] = export_skill(source, 'portable', destination, load_config(), True)
        result['applied'] = True
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path)
    parser.add_argument('--project', type=Path, default=Path.cwd())
    parser.add_argument('--state-dir', type=Path, default=DEFAULT_STATE)
    commands = parser.add_subparsers(dest='command', required=True)
    dedupe = commands.add_parser('dedupe', help='Archive selected identical copies; retain the canonical path')
    dedupe.add_argument('--canonical', type=Path, required=True)
    dedupe.add_argument('--duplicate', type=Path, action='append', required=True)
    cold = commands.add_parser('cold', help='Remove one skill from discovery into restorable storage')
    cold.add_argument('--skill', type=Path, required=True)
    undo = commands.add_parser('restore')
    undo.add_argument('--transaction', required=True)
    export = commands.add_parser('export')
    export.add_argument('--skill', type=Path, required=True)
    export.add_argument('--harness', required=True)
    export.add_argument(
        '--destination', type=Path, required=True, help='Exact new skill directory, .mdc file or .zip file'
    )
    export.add_argument('--isolate', action='store_true')
    export.add_argument('--allow-lossy', action='store_true')
    export.add_argument('--format', choices=['skill', 'cursor-rule'], default='skill')
    promotion = commands.add_parser('promote')
    promotion.add_argument('--quarantine', type=Path, required=True)
    promotion.add_argument('--destination', type=Path, required=True)
    promotion.add_argument('--review-sha256')
    promotion.add_argument('--accept-findings', action='store_true')
    for command in (dedupe, cold, undo, export, promotion):
        command.add_argument('--apply', action='store_true', help='Perform the previewed local mutation')
    args = parser.parse_args()
    config = load_config(args.config)
    state = outside_scan_roots(args.state_dir, config, args.project.resolve())
    if args.command == 'dedupe':
        result = archive(args.duplicate, state, args.apply, checked_skill(args.canonical))
    elif args.command == 'cold':
        result = archive([args.skill], state, args.apply)
    elif args.command == 'restore':
        result = restore(args.transaction, state, args.apply)
    elif args.command == 'export':
        result = export_skill(
            args.skill, args.harness, args.destination, config, args.apply, args.isolate, args.allow_lossy, args.format
        )
    else:
        result = promote(args.quarantine, args.destination, args.review_sha256, args.accept_findings, args.apply)
    emit(result)
    return 0


if __name__ == '__main__':
    raise SystemExit(entrypoint(main))
