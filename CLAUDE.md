# CLAUDE.md

> ⚠️ **BRANCH PROTECTION**: Never commit directly to main. A PreToolUse hook enforces this automatically.

## Project Overview

`oastools` is a Go CLI for OpenAPI Specification files. Validates, fixes, joins, converts, diffs, walks, generates, and builds OAS 2.0-3.2.

## Style

- **Emojis welcome** in PR descriptions and release notes, but not required in code or docs
- **GitHub formatting**: Bare hashes/issues auto-link; backticks break linking
  - Good: `Fixed in commit 1f3eb93` → clickable
  - Bad: `Fixed in commit \`1f3eb93\`` → not clickable

## Quick Reference

- `make check` before committing
- Conventional commits: `feat(parser): add feature`
- **Never amend a pushed PR commit once review has begun** — add a new commit. PRs squash on merge, so amending only breaks review diffs and comment anchors
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
- **Schema-or-bool fields are always promoted**: `Items`, `AdditionalProperties`, `AdditionalItems`, `UnevaluatedItems` and `UnevaluatedProperties` are `any`, but every decode path (JSON, YAML, `decodeFromMap`) yields `*Schema`, `[]*Schema` (OAS 2.0 tuple form), or `bool`. Type-assert to `*parser.Schema`; a `map[string]any` arm is dead for parsed documents
- **Discriminator has two dialects**: OAS 2.0 spells `discriminator` as a bare string, OAS 3.0+ as an object. Both decode into `parser.Discriminator`; `StringForm bool` records which. The parser accepts either (it cannot see the document version), the validator rejects the wrong one for the version, and the converter flips the flag in both directions. `StringForm` is excluded from JSON, YAML, and `equalDiscriminator` — it is spelling, not meaning
- **`make check` before pushing** — not just `go test`; catches lint, formatting, and trailing whitespace
- **`docs/` is mixed source + generated**: Source files (`index.md`, `mcp-server.md`, `cli-reference.md`, etc.) are edited directly in `docs/`. Generated files (`docs/packages/`, `docs/examples/`) come from `{package}/deep_dive.md` and `examples/*/README.md` — see `.claude/docs/docs-website.md`
- **MCP config via env vars**: The MCP server reads `OASTOOLS_*` env vars for configuration (cache TTLs, walk limits, join strategies, etc.). The Go MCP SDK doesn't support `initializationOptions`, so env vars are used instead. MCP clients set these via their `env` field in server config.
- **Component-name charset has one definition**: `internal/naming` owns it — `ComponentNamePattern` (string, for error messages), `IsComponentNameChar` (per-rune, for the fixer building replacements), `IsValidComponentName` (whole-name, for the validator). Never restate the pattern elsewhere; both `validator` and `fixer` consume `internal/naming`. A drift-guard test compares the compiled pattern against the predicate over every rune through U+0700
- **OAS versions disagree on schema-name legality, deliberately**: OAS 3.x Components keys are an *allowlist* (`^[a-zA-Z0-9._-]+$`), so a denylist can never keep up — `pkg/Pet`, `Pet@v1`, `pet~summary`, `Pét` are all illegal without sharing a character. OAS 2.0 places *no* charset constraint on `definitions` keys, so those names are valid and renaming them would rewrite a valid document; the fixer's character denylist applies there only. `fixer.charsetForVersion` is the switch
- **`$ref` tokens: look up exact-first, decoded-second**: Generators mix escaping conventions (e.g. percent-encoding brackets while leaving slashes raw — neither pure RFC 6901 nor pure percent-encoding). `pathutil.DecodeRefToken` reverses both, but it's lossy: a component genuinely named `Foo%20Bar` decodes to `Foo Bar` and stops matching itself. Index rename maps via `fixer.lookupRenamedRef` (checks the exact ref first, decoded second), and register decoded keys in sorted order — two names can share a decoded form without either being it, making map-range order-dependent

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
- **Minimum Go**: 1.25
