# README & Documentation Site Redesign

**Date:** 2026-02-14
**Status:** Approved

## Problem

The README has grown to ~444 lines, packing accurate but overwhelming information into a single document. A first-time visitor cannot quickly grasp what oastools offers or decide if it fits their needs. The GH Pages docs site homepage (`docs/index.md`) is a near-copy of the README, missing the opportunity to leverage its own navigation structure.

## Goals

1. **README as billboard** — An evaluator can understand what oastools is and does within 10 seconds of scrolling
2. **Docs homepage as launchpad** — Organized around three entry points (Library, CLI, MCP Server) that funnel visitors to the right "Get Started" path
3. **No information loss** — All current README depth migrates to the docs site, not deleted
4. **README target: 100–200 lines** (currently 444)

## Design Decisions

### Audience Model

| Surface | Primary Audience | Job to Do |
|---------|-----------------|-----------|
| GitHub README | Evaluators | "Should I use this?" → Yes/No in 30 seconds |
| Docs homepage | Adopters | "How do I get started for my use case?" |
| Docs subpages | Practitioners | "How do I do X?" (deep reference) |

### README Structure (~130–150 lines)

```
Banner SVG + badge row (kept as-is)
One-line pitch + bold tagline

## What It Does
  Compact grouped list showing 12-package landscape:
    Spec Lifecycle — parser · validator · fixer · converter
    Multi-Spec Ops — joiner · differ · overlay
    Code & Query  — generator · builder · walker
    Runtime       — httpvalidator · oaserrors
  Each package name links to pkg.go.dev.

## Highlights
  6 tight bullets (condensed from 11):
    - Minimal dependencies (yaml, x/tools, x/text)
    - Battle-tested (7,500+ tests, 10 production APIs)
    - Performance optimized (pre-parsed workflows 11–150x faster)
    - OAS 2.0 through 3.2 with full JSON Schema 2020-12
    - AI-ready (built-in MCP server for LLM agents)
    - Try it online (link to playground)

## Quick Start
  ### CLI (5–6 key commands, not 20)
    validate, convert, diff, fix, join, generate
  ### Library (one 5-line pipeline: parse → validate → fix)

## Installation
  CLI: brew / go install (compact)
  Library: go get

## Documentation
  3–4 links: Docs Site, CLI Reference, Developer Guide, pkg.go.dev

## Contributing (2 lines + link)
## License
```

**Removed from README (migrated to docs):**
- "Why oastools?" section (~120 lines): error handling examples, resource limits, HTTP client config, dep tree, corpus table, benchmark table, DeepCopy examples
- Package Ecosystem table (12-row detailed table with Try links)
- Deep Dive Guides table
- Examples table
- Supported OpenAPI Versions table (condensed to one Highlights bullet)
- MCP Server section with code blocks
- Full library Quick Start (7-package import block)

### Docs Site Homepage (`docs/index.md`)

Purpose-built landing page, **not** a README copy.

```
Banner + one-line pitch

Brief paragraph: what oastools is, who it's for

## Get Started
  Three entry-point cards:
    📦 Go Library → links to Developer Guide
    🖥️ CLI / CI/CD → links to CLI Reference
    🤖 MCP Server → links to MCP Server guide

## What's Included
  Full 12-package table (migrated from README)
  with descriptions and "Try Online" links

## Why oastools?
  Migrated highlights with depth, using Material's
  admonition/details blocks for collapsible sections:
    - Minimal Dependencies (with dep tree)
    - Battle-Tested (with corpus table)
    - Performance (with speedup table)
    - Enterprise Error Handling (with code examples)
    - Type-Safe Cloning (with DeepCopy examples)
    - Configurable Resource Limits (with code)
    - HTTP Client Configuration (with code)

## Try it Online
  Link to playground

Footer: GitHub, pkg.go.dev, contributing
```

### Docs Nav Changes (`mkdocs.yml`)

```yaml
nav:
  - Home: index.md                              # Purpose-built landing
  - "🌐 Try Online": https://oastools.robnrob.com
  - Get Started:                                 # NEW top-level section
    - Go Library: developer-guide.md
    - CLI / CI/CD: cli-reference.md
    - MCP Server: mcp-server.md
    - Claude Code Plugin: claude-code-plugin.md
  - Guides:
    - Why oastools?: why-oastools.md             # NEW: migrated from README
    - Detecting API Breaking Changes: breaking-changes.md  # RENAMED
    - Generator Features: generator_beyond_boilerplate.md
    - Benchmarks: benchmarks.md
  - Examples: ...                                # Unchanged
  - Package Deep Dives: ...                      # Unchanged
  - "📄 White Paper": whitepaper.md
  - Project Info: ...                            # Unchanged
  - API Documentation: https://pkg.go.dev/...
```

### New File: `docs/why-oastools.md`

Consolidates the "Why oastools?" content removed from the README:
- Minimal Dependencies (dep tree diagram)
- Battle-Tested Quality (corpus table)
- Performance (pre-parsed speedup table, benchmark link)
- Type-Safe Document Cloning (DeepCopy explanation + snippet)
- Enterprise-Grade Error Handling (oaserrors examples)
- Configurable Resource Limits (parser options snippet)
- HTTP Client Configuration (custom client snippet)

Uses Material's `??? note` / `!!! tip` admonition blocks for collapsible detail.

### Rename: `docs/breaking-changes.md`

Page title and nav entry renamed from "Breaking Changes" to "Detecting API Breaking Changes" to clarify this is about the differ's capabilities, not oastools' own changelog.

## Scope

| Item | Action |
|------|--------|
| `README.md` | Rewrite to ~130–150 lines |
| `docs/index.md` | Rewrite as purpose-built landing page |
| `docs/why-oastools.md` | New file: migrated depth from README |
| `docs/breaking-changes.md` | Rename title (h1) and nav entry |
| `mkdocs.yml` | Restructure nav with "Get Started" section |

## Non-Goals

- Rewriting the Developer Guide, CLI Reference, or any deep dive pages
- Changing the oastools-web playground
- Redesigning the banner SVG or badge selection
- Restructuring the Examples section in the nav