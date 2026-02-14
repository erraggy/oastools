# Documentation Website

The project documentation is hosted on GitHub Pages at https://erraggy.github.io/oastools/

## Dependencies

```bash
pip install mkdocs mkdocs-material mkdocs-exclude
```

The `mkdocs-exclude` plugin is required to exclude the `docs/plans/` directory from builds (used by superpowers plugin for implementation plans).

## Commands

```bash
make docs-serve   # Preview locally (blocking)
make docs-build   # Build static site to site/
```

For CI deployment and documentation structure, see [WORKFLOW.md](../../WORKFLOW.md).

## Source vs Generated Documentation Files

**CRITICAL: The `docs/packages/` directory contains GENERATED files. Do NOT edit them directly.**

The documentation build process (`scripts/prepare-docs.sh`) copies files from source locations:

| Source | Generated | Description |
|--------|-----------|-------------|
| `docs/index.md` | (checked in) | Home page (hand-crafted landing page) |
| `{package}/deep_dive.md` | `docs/packages/{package}.md` | Package deep dives |
| `examples/*/README.md` | `docs/examples/*.md` | Example documentation |

## Editing Documentation

**Always edit the SOURCE files:**
- To update the home page → edit `docs/index.md` directly (it is checked into git, not generated)
- To update package docs → edit `{package}/deep_dive.md` (e.g., `validator/deep_dive.md`)
- To update examples → edit `examples/*/README.md`

The `docs/packages/` directory is in `.gitignore` and gets regenerated on every `make docs-build` or `make docs-serve`.
