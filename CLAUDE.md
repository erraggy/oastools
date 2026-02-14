# CLAUDE.md

> ⚠️ **BRANCH PROTECTION**: Never commit directly to main. A PreToolUse hook enforces this automatically.

## Project Overview

`oastools` is a Go CLI for OpenAPI Specification files. Validates, fixes, joins, converts, diffs, walks, generates, and builds OAS 2.0-3.2.

## Style

- **Emojis encouraged** (✅❌🔴🟡⚠️) for scannability
- **GitHub formatting**: Bare hashes/issues auto-link; backticks break linking
  - Good: `Fixed in commit 1f3eb93` → clickable
  - Bad: `Fixed in commit \`1f3eb93\`` → not clickable

## Quick Reference

- `make check` before committing
- Conventional commits: `feat(parser): add feature`
- See [WORKFLOW.md](WORKFLOW.md) for PR/release process
- See [AGENTS.md](AGENTS.md) for agent workflow

## Architecture

| Package | Purpose |
|---------|---------|
| cmd/oastools/ | CLI entry point |
| parser/ | Parse YAML/JSON OAS, resolve refs, detect versions |
| validator/ | Validate against spec schema |
| fixer/ | Auto-fix common errors |
| joiner/ | Join multiple OAS files |
| converter/ | Convert between OAS versions |
| differ/ | Compare specs, detect breaking changes |
| httpvalidator/ | Runtime HTTP validation |
| generator/ | Generate Go client/server |
| builder/ | Programmatic spec construction |
| overlay/ | Apply Overlay transformations |
| walker/ | Traverse with typed handlers |

## Key Patterns

- **Format preserved**: JSON/YAML auto-detected from extension or content
- **Use constants**: `httputil.MethodGet`, `severity.SeverityError`
- **Always run `go_diagnostics`** after edits—hints improve perf 5-15%
- **Favor fixing immediately** over deferring issues
- **Deep copy**: Use generated `doc.DeepCopy()` methods, **never** JSON marshal/unmarshal (loses `interface{}` types, drops `json:"-"` fields)
- **`make check` before pushing** — not just `go test`; catches lint, formatting, and trailing whitespace
- **`docs/` is generated**: Files are copied by mkdocs build scripts — always edit source files at repo root

## Orchestrator Mode

**Default behavior**: Act as an orchestrator, not an implementer.

### When to Delegate
| Task Type | Agent |
|-----------|-------|
| Research/exploration | `general-purpose` |
| Architecture/planning | `architect` |
| Implementation/coding | `developer` |
| Code review/security | `maintainer` |
| Release/deployment | `devops-engineer` |

### When to Handle Directly
- Simple questions answerable from context
- Clarifying user intent
- Synthesizing agent results
- Coordinating multi-agent workflows

## Deep Dives (read when needed)

| Topic | File |
|-------|------|
| OAS concepts & pitfalls | `.claude/docs/oas-concepts.md` |
| Error handling patterns | `.claude/docs/error-handling.md` |
| Testing requirements | `.claude/docs/testing-requirements.md` |
| Benchmark guide | `.claude/docs/benchmark-guide.md` |
| gopls workflow | `.claude/docs/gopls-workflow.md` |
| New package checklist | `.claude/docs/new-package-checklist.md` |
| Make commands | `.claude/docs/make-commands.md` |
| Docs website | `.claude/docs/docs-website.md` |

## Go Module

- **Module**: `github.com/erraggy/oastools`
- **Minimum Go**: 1.24
