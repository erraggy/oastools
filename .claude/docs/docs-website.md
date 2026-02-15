# Documentation Website

The project documentation is hosted on GitHub Pages at https://erraggy.github.io/oastools/

## Dependencies

```bash
pip install mkdocs mkdocs-material mkdocs-exclude
```

The `mkdocs-exclude` plugin is configured to exclude the `docs/plans/` directory from builds.

## Commands

```bash
make docs-prepare  # Run prepare-docs.sh (copies generated files)
make docs-serve    # Preview locally (blocking, includes prepare)
make docs-start    # Start server in background (includes prepare)
make docs-stop     # Stop background server
make docs-build    # Build static site to site/ (includes prepare)
make docs-clean    # Remove generated files (site/, docs/packages/, docs/examples/, etc.)
```

## Deployment

Docs deploy automatically on push to `main` via `.github/workflows/docs.yml`. The workflow runs `prepare-docs.sh` then `mkdocs gh-deploy --force`.

## Source vs Generated Files

**CRITICAL: `docs/` contains BOTH source files and generated files. Know which is which before editing.**

### Source files (edit directly)

These are checked into git and are the canonical source:

| File | Description |
|------|-------------|
| `docs/index.md` | Home page (hand-crafted landing page, NOT a README copy) |
| `docs/mcp-server.md` | MCP server guide and tool reference |
| `docs/claude-code-plugin.md` | Claude Code plugin setup |
| `docs/cli-reference.md` | CLI commands and flags |
| `docs/developer-guide.md` | Go library usage guide |
| `docs/why-oastools.md` | Feature depth content |
| `docs/whitepaper.md` | Technical white paper |
| `docs/breaking-changes.md` | Breaking change detection guide |
| `docs/generator_beyond_boilerplate.md` | Generator features guide |
| `mkdocs.yml` | Site navigation and theme config |

### Generated files (edit the SOURCE, not the copy)

These are created by `scripts/prepare-docs.sh` and should NOT be edited directly:

| Source | Generated | Description |
|--------|-----------|-------------|
| `{package}/deep_dive.md` | `docs/packages/{package}.md` | Package deep dives |
| `examples/*/README.md` | `docs/examples/*.md` | Example documentation |
| `CONTRIBUTORS.md` | `docs/CONTRIBUTORS.md` | Contributors list |
| `LICENSE` | `docs/LICENSE.md` | License file |
| `benchmarks.md` | `docs/benchmarks.md` | Benchmark results |

Generated directories (`docs/packages/`, `docs/examples/`) are in `.gitignore`.

### Historical note

Before PR #314, `docs/index.md` was generated from `README.md` with sed transformations. PR #314 ("docs: redesign README and docs site for clarity") decoupled them — `README.md` is now a concise GitHub billboard, while `docs/index.md` is a purpose-built docs site landing page. The prepare-docs script explicitly skips `index.md`.
