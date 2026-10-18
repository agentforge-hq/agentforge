#!/usr/bin/env python3
"""Score skills, compare routing descriptions and verify documented argparse flags without executing code."""

from __future__ import annotations

import argparse
import ast
import collections
import itertools
import math
import re
import shlex
from pathlib import Path
from typing import Any

from af_scanner import (
    emit,
    entrypoint,
    estimated_tokens,
    frontmatter,
    load_config,
    read_text,
    scan,
    valid_name,
)

STOPWORDS = set('a an and are as at be by for from in is it of on or that the this to use when with do not'.split())
PORTABLE_FIELDS = {'name', 'description', 'license', 'compatibility', 'metadata', 'allowed-tools'}


def words(text: str) -> list[str]:
    return [w for w in re.findall(r'[a-z0-9]+', text.lower()) if w not in STOPWORDS and len(w) > 1]


def vectors(descriptions: list[str]) -> tuple[list[dict[str, float]], dict[str, float]]:
    tokens = [collections.Counter(words(d)) for d in descriptions]
    frequency = collections.Counter(w for counts in tokens for w in counts)
    idf = {w: math.log((1 + len(tokens)) / (1 + n)) + 1 for w, n in frequency.items()}
    result = []
    for counts in tokens:
        vector = {w: (1 + math.log(n)) * idf[w] for w, n in counts.items()}
        norm = math.sqrt(sum(v * v for v in vector.values())) or 1
        result.append({w: v / norm for w, v in vector.items()})
    return result, idf


def cosine(left: dict[str, float], right: dict[str, float]) -> float:
    return sum(value * right.get(word, 0) for word, value in left.items())


def local_references(text: str) -> list[str]:
    return sorted(set(re.findall(r'(?<![\w/])((?:scripts|references|assets)/[\w./-]+\.[A-Za-z0-9]+)', text)))


def argparse_flags(path: Path) -> tuple[set[str], bool, list[dict[str, Any]]]:
    """Collect literal add_argument options. Dynamic declarations remain explicitly unverified."""
    tree = ast.parse(read_text(path), filename=str(path))
    flags = {'-h', '--help'}
    dynamic = False
    declarations = []
    for node in ast.walk(tree):
        if (
            not isinstance(node, ast.Call)
            or not isinstance(node.func, ast.Attribute)
            or node.func.attr != 'add_argument'
        ):
            continue
        options = []
        for arg in node.args:
            if isinstance(arg, ast.Constant) and isinstance(arg.value, str):
                if arg.value.startswith('-'):
                    flags.add(arg.value)
                    options.append(arg.value)
            else:
                dynamic = True
        if any(kw.arg is None for kw in node.keywords):
            dynamic = True
        declarations.append({'line': node.lineno, 'options': options})
    return flags, dynamic, declarations


def command_flags(text: str) -> list[tuple[str, str, int]]:
    """Read shell-like lines containing a local Python script, never execute them."""
    found = []
    joined = text.replace('\\\n', ' ')
    for line_number, line in enumerate(joined.splitlines(), 1):
        if 'scripts/' not in line or '.py' not in line:
            continue
        try:
            tokens = shlex.split(line.strip().strip('`'))
        except ValueError:
            continue
        for index, token in enumerate(tokens):
            if not re.fullmatch(r'(?:\./)?scripts/[\w/-]+\.py', token):
                continue
            for option in tokens[index + 1 :]:
                if option in {';', '&&', '||', '|', '>', '>>', '#'}:
                    break
                if re.match(r'^--?[a-zA-Z]', option):
                    found.append((token.removeprefix('./'), option.split('=', 1)[0].rstrip('`.,'), line_number))
    return found


def lint_skill(root: Path, config: dict[str, Any]) -> dict[str, Any]:
    root = root.resolve()
    text = read_text(root / 'SKILL.md', config['limits']['file_bytes'])
    fields, body, _, parse_errors = frontmatter(text)
    issues: list[dict[str, Any]] = []

    def issue(code: str, message: str, severity: str = 'warning', **location: Any) -> None:
        issues.append({'code': code, 'message': message, 'severity': severity, **location})

    for error in parse_errors:
        issue('frontmatter', error, 'error')
    name, description = fields.get('name'), fields.get('description')
    if not isinstance(name, str) or not valid_name(name):
        issue('name', 'Name must be lowercase kebab-case, 1-64 characters', 'error')
    elif name != root.name:
        issue('name-directory', 'Name must match its directory for portable installation', 'error')
    if not isinstance(description, str) or not 1 <= len(description) <= 1024:
        issue('description', 'Description must be a string of 1-1024 characters', 'error')
        description = ''
    if 'compatibility' in fields and (
        not isinstance(fields['compatibility'], str) or not 1 <= len(fields['compatibility']) <= 500
    ):
        issue('compatibility', 'Compatibility must be a string of 1-500 characters', 'error')
    extensions = sorted(set(fields) - PORTABLE_FIELDS)
    if extensions:
        issue('harness-extensions', f'Harness-specific fields require an export check: {", ".join(extensions)}', 'info')
    if not re.search(r'\b(when|for|use|if)\b', description, re.I):
        issue('positive-trigger', 'Description lacks a clear positive routing trigger')
    if not re.search(r'\b(not for|do not|don.t use|exclude|except|instead)\b', description, re.I):
        issue('routing-boundary', 'Consider a negative trigger if a neighboring skill overlaps', 'info')
    if len(words(description)) < 5:
        issue('vague-description', 'Description has too few specific terms for reliable routing')
    if estimated_tokens(description) > config['budgets']['description_tokens']:
        issue('description-budget', 'Description exceeds the configured index token proxy budget')
    body_tokens = estimated_tokens(body)
    if body_tokens > config['budgets']['skill_tokens']:
        issue('body-budget', 'Move conditional detail into linked references')
    if name == 'agentforge' and estimated_tokens(text) >= config['budgets']['router_tokens']:
        issue('router-budget', 'agentforge root must stay below its router budget', 'error')
    normalized: dict[str, int] = {}
    prune = []
    for number, line in enumerate(body.splitlines(), 1):
        key = re.sub(r'\W+', ' ', line.lower()).strip()
        if len(key) < 45 or line.lstrip().startswith(('#', '```', '|')):
            continue
        if key in normalized:
            prune.append({'line_in_body': number, 'duplicate_of': normalized[key], 'text': line.strip()})
        else:
            normalized[key] = number
    if prune:
        issue('repeated-instructions', f'{len(prune)} duplicate instruction lines; review deletion candidates')
    documents = {'SKILL.md': text}
    for relative in local_references(text):
        target = (root / relative).resolve()
        if not target.is_relative_to(root):
            issue('escaping-reference', f'Reference escapes skill: {relative}', 'error')
        elif not target.is_file():
            issue('missing-reference', f'Missing file: {relative}', 'error')
        elif target.suffix == '.md':
            documents[relative] = read_text(target, config['limits']['file_bytes'])
    scripts = {}
    for path in sorted((root / 'scripts').rglob('*.py')) if (root / 'scripts').exists() else []:
        if not path.resolve().is_relative_to(root):
            issue('escaping-script', f'Script resolves outside skill: {path}', 'error')
            continue
        relative = path.relative_to(root).as_posix()
        try:
            flags, dynamic, declarations = argparse_flags(path)
            scripts[relative] = {'flags': sorted(flags), 'dynamic_options': dynamic, 'declarations': declarations}
        except (SyntaxError, UnicodeError) as exc:
            issue('python-syntax', f'{relative}: {exc}', 'error')
    for relative, document in documents.items():
        for script, flag, line in command_flags(document):
            if script not in scripts:
                issue(
                    'unverified-command', f'No inspectable Python script: {script}', 'error', file=relative, line=line
                )
            elif flag not in scripts[script]['flags']:
                dynamic = scripts[script]['dynamic_options']
                issue(
                    'script-flag',
                    f'{script} does not declare {flag}'
                    + ('; dynamic declarations need manual verification' if dynamic else ''),
                    'warning' if dynamic else 'error',
                    file=relative,
                    line=line,
                )
    if scripts and not re.search(r'--help|python\d*|scripts/', body):
        issue('script-routing', 'Root does not tell the agent how to use its scripts')
    if not re.search(r'\b(test|verif|validat|check|eval)', body, re.I):
        issue('verification', 'No observable verification step in the root playbook')
    if re.search(r'curl[^\n]*\|\s*(?:sh|bash)|ignore (?:all )?(?:previous|system) instructions', body, re.I):
        issue('unsafe-instruction', 'Review instruction override or remote execution pattern', 'error')
    penalties = {
        'structure': {'frontmatter', 'name', 'name-directory', 'description', 'compatibility'},
        'routing': {'positive-trigger', 'vague-description', 'description-budget'},
        'economy': {'body-budget', 'router-budget', 'repeated-instructions'},
        'execution': {'python-syntax', 'script-flag', 'unverified-command', 'script-routing'},
        'maintenance': {'missing-reference', 'escaping-reference', 'verification'},
        'safety': {'unsafe-instruction', 'escaping-script'},
    }
    dimensions = {}
    for dimension, codes in penalties.items():
        matches = [i for i in issues if i['code'] in codes and i['severity'] != 'info']
        loss = sum(0.5 if i['severity'] == 'error' else 0.25 for i in matches)
        dimensions[dimension] = round(config['sdqs_weights'][dimension] * max(0, 1 - loss), 2)
    return {
        'path': str(root),
        'name': name,
        'description': description,
        'sdqs': round(sum(dimensions.values()), 2),
        'dimensions': dimensions,
        'body_tokens_estimate': body_tokens,
        'issues': issues,
        'prune_candidates': prune,
        'scripts': scripts,
        'limits': 'Heuristic SDQS and lexical routing; AST flags are pooled across subcommands, '
        'not execution validation',
    }


def routing_report(
    skills: list[dict[str, Any]], threshold: float, cases: list[dict[str, Any]] | None = None, full_matrix: bool = False
) -> dict[str, Any]:
    descriptions = [s['description'] for s in skills]
    matrix, idf = vectors(descriptions)
    collisions = []
    for i, j in itertools.combinations(range(len(skills)), 2):
        score = cosine(matrix[i], matrix[j])
        if score >= threshold:
            collisions.append(
                {
                    'left': skills[i]['name'],
                    'right': skills[j]['name'],
                    'left_path': skills[i]['path'],
                    'right_path': skills[j]['path'],
                    'cosine': round(score, 4),
                }
            )
    tests = []
    for case in cases or []:
        counts = collections.Counter(words(case['prompt']))
        query = {w: (1 + math.log(n)) * idf[w] for w, n in counts.items() if w in idf}
        norm = math.sqrt(sum(v * v for v in query.values())) or 1
        query = {w: v / norm for w, v in query.items()}
        ranked = sorted(
            [(cosine(v, query), str(s['name'])) for v, s in zip(matrix, skills)], key=lambda p: (-p[0], p[1])
        )
        # Equal positive top scores are ambiguous; zero means no lexical evidence.
        winners = sorted({name for score, name in ranked if score > 0 and abs(score - ranked[0][0]) < 1e-10})
        expected = case.get('expected', [])
        expected = [expected] if isinstance(expected, str) else expected
        passed = (len(winners) == 1 and winners[0] in expected) if expected else not winners
        tests.append(
            {
                'prompt': case['prompt'],
                'expected': expected,
                'winners': winners,
                'passed': passed,
                'ranking': [{'name': name, 'score': round(score, 4)} for score, name in ranked[:3]],
            }
        )
    result = {
        'method': 'TF-IDF cosine; lexical proxy, not an LLM trigger evaluation',
        'threshold': threshold,
        'collisions': sorted(collisions, key=lambda c: -c['cosine']),
        'trigger_tests': tests,
    }
    if full_matrix:
        result['matrix'] = {
            'paths': [s['path'] for s in skills],
            'cosine': [[round(cosine(a, b), 4) for b in matrix] for a in matrix],
        }
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        'skills', type=Path, nargs='*', help='Skill directories or SKILL.md files; default: configured inventory'
    )
    parser.add_argument('--root', type=Path, action='append')
    parser.add_argument('--config', type=Path)
    parser.add_argument('--project', type=Path, default=Path.cwd())
    parser.add_argument('--collision-threshold', type=float)
    parser.add_argument('--full-matrix', action='store_true', help='Include the full pairwise cosine matrix in JSON')
    parser.add_argument('--trigger-tests', type=Path, help='JSON list of {prompt, expected: [skill-name]}')
    parser.add_argument('--fail-under', type=float, default=0)
    args = parser.parse_args()
    config = load_config(args.config)
    threshold = config['collision_threshold'] if args.collision_threshold is None else args.collision_threshold
    if not 0 <= threshold <= 1 or not 0 <= args.fail_under <= 100:
        raise ValueError('Threshold must be 0..1 and fail-under 0..100')
    errors = []
    if args.skills:
        paths = [p.parent if p.name == 'SKILL.md' else p for p in args.skills]
    else:
        inventory = scan(config, args.project.resolve(), args.root)
        errors.extend(inventory['errors'])
        paths = [Path(s['real_path']).parent for s in inventory['skills']]
    skills = []
    for path in sorted(set(p.resolve() for p in paths)):
        try:
            skills.append(lint_skill(path, config))
        except (OSError, ValueError, UnicodeError) as exc:
            errors.append({'path': str(path), 'error': str(exc)})
    cases = None
    if args.trigger_tests:
        import json

        cases = json.loads(read_text(args.trigger_tests))
        if not isinstance(cases, list) or any(
            not isinstance(c, dict) or not isinstance(c.get('prompt'), str) for c in cases
        ):
            raise ValueError('Trigger tests must be a JSON list containing prompt strings')
    routing = routing_report(skills, threshold, cases, args.full_matrix)
    emit({'skills': skills, 'routing': routing, 'errors': errors})
    return int(
        bool(errors)
        or any(s['sdqs'] < args.fail_under or any(i['severity'] == 'error' for i in s['issues']) for s in skills)
        or any(not t['passed'] for t in routing['trigger_tests'])
    )


if __name__ == '__main__':
    raise SystemExit(entrypoint(main))
