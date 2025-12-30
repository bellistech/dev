#!/usr/bin/env python3
# init_python_project.py
# Modern Python scaffold: src/ layout + pyproject.toml.
# Features via --features: ruff,pytest,mypy,precommit,github,docker,compose,mkdocs,makefile,all

from __future__ import annotations

import argparse
import re
import shutil
import subprocess
from pathlib import Path


def sh(cmd: list[str], cwd: Path | None = None) -> None:
    subprocess.run(cmd, cwd=str(cwd) if cwd else None, check=True)


def w(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding='utf-8')


def slug_to_pkg(name: str) -> str:
    s = re.sub(r'[^a-zA-Z0-9]+', '_', name.strip()).strip('_').lower()
    if not s:
        return 'app'
    if s[0].isdigit():
        s = f'app_{s}'
    return s


def main() -> int:
    ap = argparse.ArgumentParser(description='Modern Python project scaffold (src/ layout + pyproject.toml).')
    ap.add_argument('-n', '--name', required=True, help='Project dir name (hyphens ok)')
    ap.add_argument('-p', '--package', default='', help='Import package name (default derived)')
    ap.add_argument('-o', '--out', default='.', help='Parent output dir')
    ap.add_argument('-g', '--git', action='store_true', help='git init')
    ap.add_argument('-f', '--force', action='store_true', help='overwrite existing dir')
    ap.add_argument('--features', default='', help='comma list: ruff,pytest,mypy,precommit,github,docker,compose,mkdocs,makefile,all')
    a = ap.parse_args()

    name = a.name.strip()
    pkg = a.package.strip() or slug_to_pkg(name)

    feats = {x.strip().lower() for x in a.features.split(',') if x.strip()}
    all_on = 'all' in feats
    def on(k: str) -> bool:
        return all_on or (k in feats)

    use_ruff = on('ruff')
    use_pytest = on('pytest')
    use_mypy = on('mypy')
    use_precommit = on('precommit')
    use_github = on('github')
    use_compose = on('compose')
    use_docker = on('docker') or use_compose
    use_mkdocs = on('mkdocs')
    use_makefile = on('makefile')

    target = (Path(a.out).expanduser().resolve() / name)
    if target.exists():
        if a.force:
            shutil.rmtree(target)
        else:
            raise SystemExit(f'ERROR: target exists: {target} (use -f/--force)')

    (target / 'src' / pkg).mkdir(parents=True, exist_ok=True)
    (target / 'scripts').mkdir(parents=True, exist_ok=True)
    if use_pytest:
        (target / 'tests').mkdir(parents=True, exist_ok=True)
    if use_mkdocs:
        (target / 'docs').mkdir(parents=True, exist_ok=True)
    if use_github:
        (target / '.github' / 'workflows').mkdir(parents=True, exist_ok=True)

    w(target / 'src' / pkg / '__init__.py', "__all__ = []\n__version__ = '0.1.0'\n")

    cli = (
        'from __future__ import annotations\n\n'
        'import argparse\n\n'
        f"def main(argv: list[str] | None = None) -> int:\n"
        f"    p = argparse.ArgumentParser(prog='{name}', description='{name} CLI')\n"
        "    p.add_argument('--version', action='store_true')\n"
        "    ns = p.parse_args(argv)\n"
        "    if ns.version:\n"
        f"        from {pkg} import __version__\n"
        "        print(__version__)\n"
        "        return 0\n"
        f"    print('✅ {name} is alive')\n"
        "    return 0\n\n"
        "if __name__ == '__main__':\n"
        "    raise SystemExit(main())\n"
    )
    w(target / 'src' / pkg / 'cli.py', ''.join(cli))

    readme = f"""# {name}

## Quickstart
```bash
python -m venv .venv
source .venv/bin/activate   # Windows: .venv\\Scripts\\activate
python -m pip install -U pip
pip install -e \".[dev]\"
{name} --help
```

- package: `{pkg}`
- entrypoint: `{name}` -> `{pkg}.cli:main`
"""
    w(target / 'README.md', readme + '\n')

    w(target / '.gitignore', '.venv/\n__pycache__/\n*.pyc\n.pytest_cache/\n.mypy_cache/\n.ruff_cache/\n'                            'dist/\nbuild/\n*.egg-info/\n.coverage\nhtmlcov/\n.DS_Store\n.env\n')

    if use_pytest:
        w(target / 'tests' / 'test_smoke.py', 'def test_smoke():\n    assert True\n')

    if use_mkdocs:
        w(target / 'mkdocs.yml', f'site_name: {name}\nnav:\n  - Home: index.md\n')
        w(target / 'docs' / 'index.md', f'# {name}\n')

    dev_deps: list[str] = []
    tool = ''
    if use_ruff:
        dev_deps.append('ruff>=0.6.0')
        tool += '[tool.ruff]\nline-length = 100\ntarget-version = "py311"\nfix = true\n\n'                '[tool.ruff.lint]\nselect = ["E","F","I","B","UP","SIM"]\n'
    if use_pytest:
        dev_deps += ['pytest>=8.0.0', 'pytest-cov>=5.0.0']
        tool += '[tool.pytest.ini_options]\ntestpaths = ["tests"]\naddopts = "-q"\n'
    if use_mypy:
        dev_deps += ['mypy>=1.10.0', 'types-setuptools']
        tool += '[tool.mypy]\npython_version = "3.11"\ndisallow_untyped_defs = true\n'
    dev_deps = list(dict.fromkeys(dev_deps))

    pyproject = (
        '[build-system]\nrequires = ["hatchling>=1.25.0"]\nbuild-backend = "hatchling.build"\n\n'
        f'[project]\nname = "{name}"\nversion = "0.1.0"\nrequires-python = ">=3.11"\ndependencies = []\n\n'
        f'[project.optional-dependencies]\ndev = {dev_deps if dev_deps else []}\n\n'
        f'[project.scripts]\n{name} = "{pkg}.cli:main"\n\n'
        f'[tool.hatch.build.targets.wheel]\npackages = ["src/{pkg}"]\n\n'
        tool
    )
    w(target / 'pyproject.toml', ''.join(pyproject))

    if use_precommit:
        pc = 'repos:\n- repo: https://github.com/pre-commit/pre-commit-hooks\n  rev: v4.6.0\n  hooks:\n'
        pc += '    - id: end-of-file-fixer\n    - id: trailing-whitespace\n    - id: check-yaml\n    - id: check-toml\n'
        if use_ruff:
            pc += '- repo: https://github.com/astral-sh/ruff-pre-commit\n  rev: v0.6.9\n  hooks:\n'
            pc += '    - id: ruff\n      args: ["--fix"]\n    - id: ruff-format\n'
        w(target / '.pre-commit-config.yaml', pc)

    if use_github:
        steps = []
        if use_ruff:
            steps += ['      - run: python -m pip install -U pip ruff', '      - run: ruff check .', '      - run: ruff format --check .']
        if use_pytest:
            steps += ['      - run: python -m pip install -U pip ".[dev]"', '      - run: pytest']
        if not steps:
            steps = ['      - run: python -c "print(\'CI ok\')"']
        ci = (
            'name: ci\non: [push, pull_request]\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n'
            '      - uses: actions/checkout@v4\n'
            '      - uses: actions/setup-python@v5\n        with:\n          python-version: "3.11"\n'
            '\n'.join(steps) + '\n'
        )
        w(target / '.github' / 'workflows' / 'ci.yml', ''.join(ci))

    if use_docker:
        w(target / 'Dockerfile', f'FROM python:3.12-slim\nWORKDIR /app\nCOPY pyproject.toml README.md /app/\nCOPY src /app/src\n'                               f'RUN python -m pip install -U pip && pip install .\nCMD ["{name}", "--help"]\n')
        if use_compose:
            w(target / 'docker-compose.yaml', 'services:\n  app:\n    build: .\n')

    if use_makefile:
        mk = f'run:\n\tpython -m {pkg}.cli\n'
        if use_ruff:
            mk += 'lint:\n\truff check .\nfmt:\n\truff format .\n'
        if use_pytest:
            mk += 'test:\n\tpytest\n'
        w(target / 'Makefile', mk)

    if a.git:
        sh(['git', 'init'], cwd=target)

    print(f'✅ Created: {target}')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
