#!/usr/bin/env python3
"""Fetch public GitHub/skills.sh skills to quarantine and inspect them without execution."""

from __future__ import annotations

import argparse
import hashlib
import io
import json
import re
import stat
import tempfile
import urllib.error
import urllib.parse
import urllib.request
import uuid
import zipfile
from pathlib import Path, PurePosixPath
from typing import Any

from af_scanner import (
    DEFAULT_STATE,
    atomic_json,
    emit,
    entrypoint,
    frontmatter,
    load_config,
    read_text,
    tree_digest,
    valid_name,
)
from af_sync import outside_scan_roots, portable_files, publish_directory

ALLOWED_HOSTS = {
    'api.github.com',
    'github.com',
    'codeload.github.com',
    'raw.githubusercontent.com',
    'skills.sh',
    'www.skills.sh',
}
PATTERNS = [
    (
        'instruction-override',
        'high',
        r'ignore\s+(?:(?:all|any)\s+)?(?:previous|prior|system|developer)\s+(?:instructions|rules|messages)',
    ),
    ('role-spoofing', 'high', r'<\|(?:im_start|system|developer)\|>|\[INST\]|<system>'),
    (
        'secret-request',
        'high',
        r'(?:reveal|print|upload|send|exfiltrat\w*)[^\n]{0,80}(?:api.key|secret|password|credential|private.key)',
    ),
    ('disable-controls', 'high', r'(?:disable|bypass|ignore)[^\n]{0,50}(?:safety|approval|sandbox|security.check)'),
    ('remote-execution', 'high', r'(?:curl|wget)[^\n]*\|\s*(?:sh|bash|python)'),
    ('upload-command', 'medium', r'curl[^\n]*(?:--data|-d\s|--upload-file|-T\s)|requests\.post\('),
    ('obfuscated-execution', 'high', r'(?:exec|eval)\s*\([^\n]*(?:b64decode|fromhex)|base64\s+-d[^\n]*\|'),
    ('bidi-control', 'medium', r'[\u202a-\u202e\u2066-\u2069]'),
]


def inspect_tree(root: Path, config: dict[str, Any] | None = None) -> dict[str, Any]:
    config = config or load_config()
    fingerprint = tree_digest(root, config['limits'])
    findings = []
    unscanned = []
    for path in portable_files(root):
        relative = path.relative_to(root).as_posix()
        try:
            text = read_text(path, config['limits']['file_bytes'])
            if '\0' in text:
                raise UnicodeError('Binary data')
        except (UnicodeError, ValueError) as exc:
            unscanned.append({'file': relative, 'reason': str(exc)})
            continue
        for number, line in enumerate(text.splitlines(), 1):
            for code, severity, pattern in PATTERNS:
                if re.search(pattern, line, re.I):
                    findings.append(
                        {
                            'code': code,
                            'severity': severity,
                            'file': relative,
                            'line': number,
                            'evidence': line.strip()[:180],
                        }
                    )
    return {
        'findings': findings,
        'unscanned': unscanned,
        'tree_sha256': fingerprint,
        'executed': False,
        'assessment': 'Heuristic triage only; no findings does not establish trust or rule out injection',
    }


def checked_url(url: str) -> str:
    parsed = urllib.parse.urlsplit(url)
    if (
        parsed.scheme != 'https'
        or parsed.hostname not in ALLOWED_HOSTS
        or parsed.port not in {None, 443}
        or parsed.username
    ):
        raise ValueError('Only HTTPS public GitHub and skills.sh hosts are allowed')
    return url


class RestrictedRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req: Any, fp: Any, code: int, msg: str, headers: Any, newurl: str) -> Any:
        checked_url(newurl)
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def download(url: str, limit: int) -> bytes:
    checked_url(url)
    request = urllib.request.Request(
        url, headers={'User-Agent': 'agentforge/0.1', 'Accept': 'application/vnd.github+json'}
    )
    try:
        with urllib.request.build_opener(RestrictedRedirect).open(request, timeout=20) as response:
            checked_url(response.url)
            data = response.read(limit + 1)
    except (urllib.error.URLError, TimeoutError) as exc:
        raise ValueError(f'Public fetch failed: {exc}') from exc
    if len(data) > limit:
        raise ValueError(f'Download exceeds {limit} bytes')
    return data


def parse_source(source: str, ref: str | None = None, skill_path: str | None = None) -> dict[str, Any]:
    slug = None
    if '://' in source:
        checked_url(source)
        parsed = urllib.parse.urlsplit(source)
        pieces = [urllib.parse.unquote(p) for p in parsed.path.strip('/').split('/')]
        if parsed.hostname in {'skills.sh', 'www.skills.sh'}:
            if len(pieces) != 3:
                raise ValueError('Use a skills.sh/owner/repo/skill-slug URL')
            owner, repo, slug = pieces
        elif parsed.hostname == 'github.com':
            if len(pieces) < 2:
                raise ValueError('GitHub URL must name an owner and repository')
            owner, repo = pieces[:2]
            if len(pieces) > 2:
                if len(pieces) < 4 or pieces[2] not in {'tree', 'blob'}:
                    raise ValueError('Use a repository URL or /tree/REF/path URL')
                url_ref = pieces[3]
                if ref and ref != url_ref:
                    raise ValueError('For refs containing slashes use a repository URL with --ref and --skill-path')
                ref = ref or url_ref
                skill_path = skill_path or '/'.join(pieces[4:])
                if skill_path.endswith('/SKILL.md'):
                    skill_path = skill_path[:-9]
        else:
            raise ValueError('Source must be a GitHub repository or skills.sh skill page')
    else:
        pieces = source.strip('/').split('/')
        if len(pieces) < 2:
            raise ValueError('Expected owner/repo[/path]')
        owner, repo = pieces[:2]
        skill_path = skill_path if skill_path is not None else '/'.join(pieces[2:]) or None
    repo = repo.removesuffix('.git')
    if not re.fullmatch(r'[A-Za-z0-9_.-]+', owner) or not re.fullmatch(r'[A-Za-z0-9_.-]+', repo):
        raise ValueError('Invalid GitHub repository name')
    if owner in {'.', '..'} or repo in {'.', '..'}:
        raise ValueError('Invalid repository path')
    if skill_path is not None:
        safe_relative(skill_path)
    return {'owner': owner, 'repo': repo, 'ref': ref or 'HEAD', 'skill_path': skill_path, 'slug': slug}


def safe_relative(name: str) -> PurePosixPath:
    path = PurePosixPath(name)
    if not name or path.is_absolute() or '\\' in name or ':' in name or any(p in {'..', ''} for p in name.split('/')):
        raise ValueError(f'Unsafe archive/path entry: {name}')
    return path


def unpack_skill(data: bytes, selection: dict[str, Any], destination: Path, config: dict[str, Any]) -> str:
    """Validate every ZIP member before extracting a selected skill. No extractall()."""
    limits = config['limits']
    try:
        archive = zipfile.ZipFile(io.BytesIO(data))
    except zipfile.BadZipFile as exc:
        raise ValueError('Invalid GitHub ZIP archive') from exc
    with archive:
        infos = archive.infolist()
        if len(infos) > limits['tree_files']:
            raise ValueError('Repository archive has too many entries')
        names = set()
        for info in infos:
            safe_relative(info.filename.rstrip('/'))
            if info.filename in names:
                raise ValueError('Duplicate ZIP entry')
            names.add(info.filename)
        roots = {info.filename.split('/')[0] for info in infos}
        if len(roots) != 1:
            raise ValueError('Expected a single GitHub archive root')
        prefix = next(iter(roots)) + '/'
        candidates = []
        for info in infos:
            if not info.filename.endswith('/SKILL.md'):
                continue
            relative = info.filename[len(prefix) : -9].rstrip('/')
            if selection['skill_path'] is not None and relative != selection['skill_path'].rstrip('/'):
                continue
            if selection['slug']:
                if info.file_size > limits['file_bytes']:
                    raise ValueError('Skill manifest exceeds read limits')
                fields, _, _, problems = frontmatter(archive.read(info).decode('utf-8-sig'))
                if problems or fields.get('name') != selection['slug']:
                    continue
            candidates.append(relative)
        if len(candidates) != 1:
            raise ValueError(f'Expected exactly one skill, found {len(candidates)}; specify --skill-path')
        selected = prefix + (candidates[0] + '/' if candidates[0] else '')
        total = 0
        for info in infos:
            inside = info.filename.startswith(selected)
            ancestor = selected.startswith(info.filename.rstrip('/') + '/')
            if not inside and not ancestor:
                continue
            mode = info.external_attr >> 16
            if stat.S_ISLNK(mode) or (stat.S_IFMT(mode) not in {0, stat.S_IFREG, stat.S_IFDIR}):
                raise ValueError('Selected skill contains a symlink or special file')
            if inside:
                total += info.file_size
                if info.file_size > limits['file_bytes'] or total > limits.get('extract_bytes', limits['tree_bytes']):
                    raise ValueError('Selected skill exceeds expanded size limits')
        destination.mkdir()
        for info in infos:
            if info.is_dir() or not info.filename.startswith(selected):
                continue
            relative = safe_relative(info.filename[len(selected) :])
            target = destination.joinpath(*relative.parts)
            target.parent.mkdir(parents=True, exist_ok=True)
            with archive.open(info) as stream:
                payload = stream.read(limits['file_bytes'] + 1)
            if len(payload) != info.file_size or len(payload) > limits['file_bytes']:
                raise ValueError('ZIP member size mismatch')
            target.write_bytes(payload)
            # Preserve executable bits for fingerprinting; downloaded scripts are NEVER run.
            target.chmod(0o755 if (info.external_attr >> 16) & 0o111 else 0o644)
    return candidates[0]


def fetch(
    source: str,
    state: Path,
    config: dict[str, Any],
    ref: str | None = None,
    skill_path: str | None = None,
    expected_archive: str | None = None,
) -> dict[str, Any]:
    selection = parse_source(source, ref, skill_path)
    owner, repo = selection['owner'], selection['repo']
    encoded_ref = urllib.parse.quote(selection['ref'], safe='')
    commit = json.loads(
        download(f'https://api.github.com/repos/{owner}/{repo}/commits/{encoded_ref}', config['limits']['file_bytes'])
    )
    sha = commit.get('sha', '')
    if not re.fullmatch(r'[0-9a-f]{40}', sha):
        raise ValueError('GitHub did not return a pinned commit SHA')
    data = download(f'https://codeload.github.com/{owner}/{repo}/zip/{sha}', config['limits']['download_bytes'])
    archive_sha = hashlib.sha256(data).hexdigest()
    if expected_archive and archive_sha != expected_archive:
        raise ValueError('Archive checksum does not match --archive-sha256')
    quarantine_root = state / 'quarantine'
    quarantine_root.mkdir(parents=True, exist_ok=True, mode=0o700)
    with tempfile.TemporaryDirectory(prefix='.fetch-', dir=quarantine_root) as temporary:
        staging = Path(temporary)
        selected = unpack_skill(data, selection, staging / 'skill', config)
        fields, _, _, problems = frontmatter(read_text(staging / 'skill/SKILL.md'))
        if problems or not isinstance(fields.get('name'), str) or not valid_name(fields['name']):
            raise ValueError('Downloaded skill has invalid or unsupported metadata')
        inspection = inspect_tree(staging / 'skill', config)
        manifest = {
            'source': source,
            'repository': f'{owner}/{repo}',
            'commit': sha,
            'skill_path': selected,
            'name': fields['name'],
            'archive_sha256': archive_sha,
            'tree_sha256': inspection['tree_sha256'],
            'inspection': inspection,
            'status': 'quarantined',
            'executed': False,
        }
        atomic_json(staging / 'manifest.json', manifest)
        destination = quarantine_root / (fields['name'] + '-' + uuid.uuid4().hex)
        publish_directory(staging, destination)
    return {**manifest, 'quarantine': str(destination)}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path)
    parser.add_argument('--project', type=Path, default=Path.cwd())
    parser.add_argument('--state-dir', type=Path, default=DEFAULT_STATE)
    commands = parser.add_subparsers(dest='command', required=True)
    inspection = commands.add_parser('inspect', help='Read-only heuristic scan of a local skill')
    inspection.add_argument('--path', type=Path, required=True)
    fetching = commands.add_parser('fetch', help='Explicit network operation; writes only to quarantine')
    fetching.add_argument('source')
    fetching.add_argument('--ref')
    fetching.add_argument('--skill-path')
    fetching.add_argument('--archive-sha256')
    search = commands.add_parser('search', help='Explicit public GitHub repository search; no installation')
    search.add_argument('--query', required=True)
    args = parser.parse_args()
    config = load_config(args.config)
    if args.command == 'inspect':
        result = inspect_tree(args.path.expanduser().resolve(), config)
    elif args.command == 'search':
        query = urllib.parse.urlencode({'q': args.query + ' agent skills', 'per_page': 10})
        data = json.loads(
            download('https://api.github.com/search/repositories?' + query, config['limits']['file_bytes'])
        )
        result = {
            'repositories': [
                {'name': item['full_name'], 'url': item['html_url'], 'description': item.get('description')}
                for item in data.get('items', [])
            ],
            'skills_sh_catalog': 'https://skills.sh/',
        }
    else:
        state = outside_scan_roots(args.state_dir, config, args.project.resolve())
        result = fetch(args.source, state, config, args.ref, args.skill_path, args.archive_sha256)
    emit(result)
    return 0


if __name__ == '__main__':
    raise SystemExit(entrypoint(main))
