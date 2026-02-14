# README & Docs Site Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Slim the README from ~444 lines to ~130 lines (evaluator-first billboard) and rebuild the docs site homepage as a use-case-oriented landing page with Library/CLI/MCP entry points.

**Architecture:** The README becomes a "billboard" that earns click-throughs. Depth migrates to a new `docs/why-oastools.md` page. The docs homepage (`docs/index.md`) is rewritten as a purpose-built landing page with three entry-point sections, not a README copy. The mkdocs nav is restructured with a "Get Started" top-level section.

**Tech Stack:** Markdown, MkDocs Material (admonitions, details, attr_list already configured)

---

### Task 1: Create `docs/why-oastools.md`

This new page receives all the "Why oastools?" depth removed from the README. It consolidates: dependency tree, corpus table, performance benchmarks, DeepCopy, error handling, resource limits, and HTTP client config.

**Files:**
- Create: `docs/why-oastools.md`

**Step 1: Write the file**

Create `docs/why-oastools.md` with this exact content:

```markdown
# Why oastools?

oastools is designed around three principles: **minimal dependencies**, **production-grade quality**, and **performance**. This page explains what that means in practice.

## Minimal Dependencies

```text
github.com/erraggy/oastools
├── go.yaml.in/yaml/v4  (YAML parsing)
├── golang.org/x/text   (Title casing)
└── golang.org/x/tools  (Code generation — imports analysis)
```

Unlike many OpenAPI tools that pull in dozens of transitive dependencies, oastools is designed to be self-contained. The `stretchr/testify` dependency is test-only and not included in your production builds. The CLI adds the [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) for `oastools mcp`.

## Battle-Tested Quality

The entire toolchain is validated against a corpus of 10 real-world production APIs:

| Domain          | APIs                                    |
|-----------------|-----------------------------------------|
| FinTech         | Stripe, Plaid                           |
| Developer Tools | GitHub, DigitalOcean                    |
| Communications  | Discord (OAS 3.1)                       |
| Enterprise      | Microsoft Graph (34MB, 18k+ operations) |
| Location        | Google Maps                             |
| Public          | US National Weather Service             |
| Reference       | Petstore (OAS 2.0)                      |
| Productivity    | Asana                                   |

This corpus spans OAS 2.0 through 3.1, JSON and YAML formats, and document sizes from 20KB to 34MB.

## Performance

Pre-parsed workflows eliminate redundant parsing when processing multiple operations:

| Method             | Speedup      |
|--------------------|--------------|
| `ValidateParsed()` | 31x faster   |
| `ConvertParsed()`  | ~50x faster  |
| `JoinParsed()`     | 150x faster  |
| `DiffParsed()`     | 81x faster   |
| `FixParsed()`      | ~60x faster  |
| `ApplyParsed()`    | ~11x faster  |

JSON marshaling is optimized for 25-32% better performance with 29-37% fewer allocations. See [Benchmarks](benchmarks.md) for detailed analysis.

## Type-Safe Document Cloning

All parser types include generated `DeepCopy()` methods for safe document mutation. Unlike JSON marshal/unmarshal approaches used by other tools, oastools provides:

- **Type preservation** — Polymorphic fields maintain their actual types (e.g., `Schema.Type` as `string` vs `[]string` for OAS 3.1)
- **Version-aware copying** — Handles OAS version differences correctly (`ExclusiveMinimum` as bool in 3.0 vs number in 3.1)
- **Extension preservation** — All `x-*` extension fields are deep copied
- **Performance** — Direct struct copying without serialization overhead

```go
// Safe mutation without affecting the original
copy := result.OAS3Document.DeepCopy()
copy.Info.Title = "Modified API"
```

All OAS types also provide `Equals()` methods for structural comparison.

## Enterprise-Grade Error Handling

The `oaserrors` package provides structured error types that work with Go's standard `errors.Is()` and `errors.As()`:

```go
import (
    "errors"
    "github.com/erraggy/oastools/oaserrors"
    "github.com/erraggy/oastools/parser"
)

result, err := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
if err != nil {
    // Check error category with errors.Is()
    if errors.Is(err, oaserrors.ErrPathTraversal) {
        log.Fatal("Security: path traversal attempt blocked")
    }

    // Extract details with errors.As()
    var refErr *oaserrors.ReferenceError
    if errors.As(err, &refErr) {
        log.Printf("Failed to resolve: %s (type: %s)", refErr.Ref, refErr.RefType)
    }
}
```

Error types include `ParseError`, `ReferenceError`, `ValidationError`, `ResourceLimitError`, `ConversionError`, and `ConfigError`.

## Configurable Resource Limits

Protect against resource exhaustion with configurable limits:

```go
result, err := parser.ParseWithOptions(
    parser.WithFilePath("api.yaml"),
    parser.WithMaxRefDepth(50),           // Max $ref nesting (default: 100)
    parser.WithMaxCachedDocuments(200),   // Max cached external docs (default: 100)
    parser.WithMaxFileSize(20*1024*1024), // Max file size in bytes (default: 10MB)
)
```

## HTTP Client Configuration

For advanced scenarios like custom timeouts, proxies, or authentication:

```go
// Custom timeout for slow networks
client := &http.Client{Timeout: 120 * time.Second}
result, _ := parser.ParseWithOptions(
    parser.WithFilePath("https://api.example.com/openapi.yaml"),
    parser.WithHTTPClient(client),
)

// Corporate proxy
proxyURL, _ := url.Parse("http://proxy.corp:8080")
client := &http.Client{
    Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
}
result, _ := parser.ParseWithOptions(
    parser.WithFilePath("https://internal-api.corp/spec.yaml"),
    parser.WithHTTPClient(client),
)
```

When a custom client is provided, `InsecureSkipVerify` is ignored — configure TLS on your client's transport instead.
```

**Step 2: Verify the file is valid markdown**

Run: `head -5 docs/why-oastools.md`
Expected: Shows `# Why oastools?` as the first line.

**Step 3: Commit**

```bash
git add docs/why-oastools.md
git commit -m "docs: add why-oastools page with migrated README depth"
```

---

### Task 2: Rename breaking-changes page title

The current title "Breaking Change Semantics" could be mistaken for oastools' own breaking changes. Rename to clarify it's about the differ's capabilities.

**Files:**
- Modify: `docs/breaking-changes.md:1` (h1 title only)

**Step 1: Update the title**

Change line 1 of `docs/breaking-changes.md` from:
```markdown
# Breaking Change Semantics
```
to:
```markdown
# Detecting API Breaking Changes
```

**Step 2: Commit**

```bash
git add docs/breaking-changes.md
git commit -m "docs: rename breaking-changes title to clarify it's about differ capabilities"
```

---

### Task 3: Rewrite `README.md`

Slim from ~444 lines to ~130 lines. Evaluator-first billboard.

**Files:**
- Modify: `README.md` (full rewrite)

**Step 1: Replace README.md with this content**

```markdown
<p align="center">
  <img src="docs/img/banner.svg" alt="oastools - for validating, parsing, fixing, converting, diffing, joining, and building specs" width="100%">
</p>

A complete, self-contained OpenAPI toolkit for Go with minimal dependencies.

[![CI: Go](https://github.com/erraggy/oastools/actions/workflows/go.yml/badge.svg)](https://github.com/erraggy/oastools/actions/workflows/go.yml)
[![CI: golangci-lint](https://github.com/erraggy/oastools/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/erraggy/oastools/actions/workflows/golangci-lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/erraggy/oastools)](https://goreportcard.com/report/github.com/erraggy/oastools)
[![codecov](https://codecov.io/gh/erraggy/oastools/graph/badge.svg?token=T8768QXQAX)](https://codecov.io/gh/erraggy/oastools)
[![Go Reference](https://pkg.go.dev/badge/github.com/erraggy/oastools.svg)](https://pkg.go.dev/github.com/erraggy/oastools)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Try it Online](https://img.shields.io/badge/Try_it-Online-blue)](https://oastools.robnrob.com)

**Parse, validate, fix, convert, diff, join, generate, and build OpenAPI specs (2.0–3.2) — all in one tool.**

## What It Does

**Spec Lifecycle** — [parser](https://pkg.go.dev/github.com/erraggy/oastools/parser) · [validator](https://pkg.go.dev/github.com/erraggy/oastools/validator) · [fixer](https://pkg.go.dev/github.com/erraggy/oastools/fixer) · [converter](https://pkg.go.dev/github.com/erraggy/oastools/converter)
**Multi-Spec Ops** — [joiner](https://pkg.go.dev/github.com/erraggy/oastools/joiner) · [differ](https://pkg.go.dev/github.com/erraggy/oastools/differ) · [overlay](https://pkg.go.dev/github.com/erraggy/oastools/overlay)
**Code & Query** — [generator](https://pkg.go.dev/github.com/erraggy/oastools/generator) · [builder](https://pkg.go.dev/github.com/erraggy/oastools/builder) · [walker](https://pkg.go.dev/github.com/erraggy/oastools/walker)
**Runtime** — [httpvalidator](https://pkg.go.dev/github.com/erraggy/oastools/httpvalidator) · [oaserrors](https://pkg.go.dev/github.com/erraggy/oastools/oaserrors)

12 packages covering the full OpenAPI lifecycle. [See full details →](https://erraggy.github.io/oastools/)

## Highlights

- **Minimal Dependencies** — Only [`go.yaml.in/yaml`](https://pkg.go.dev/go.yaml.in/yaml/v4), [`golang.org/x/tools`](https://pkg.go.dev/golang.org/x/tools), and [`golang.org/x/text`](https://pkg.go.dev/golang.org/x/text) at runtime
- **Battle-Tested** — 7,500+ tests against 10 production APIs (Stripe, GitHub, Discord, MS Graph 34MB)
- **Performance** — Pre-parsed workflows 11–150x faster; 340+ benchmarks
- **OAS 2.0–3.2** — Full JSON Schema Draft 2020-12 for OAS 3.1+; automatic format detection and preservation
- **AI-Ready** — Built-in [MCP server](https://erraggy.github.io/oastools/mcp-server/) exposes all capabilities to LLM agents
- **Try Online** — [oastools.robnrob.com](https://oastools.robnrob.com) — no install required

## Quick Start

### CLI

```bash
oastools validate openapi.yaml                          # Validate a spec
oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml  # Convert versions
oastools diff --breaking v1.yaml v2.yaml                 # Detect breaking changes
oastools fix api.yaml -o fixed.yaml                      # Auto-fix errors
oastools join -o merged.yaml base.yaml ext.yaml          # Merge specs
oastools generate --client --server -o ./gen -p api openapi.yaml  # Generate Go code
```

### Library

```go
// Parse, validate, and fix — consistent functional options API
result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
vResult, _ := validator.ValidateWithOptions(validator.WithDocument(result))
fResult, _ := fixer.FixWithOptions(fixer.WithDocument(result))
```

[Full library guide →](https://erraggy.github.io/oastools/developer-guide/)

## Installation

### CLI

```bash
brew tap erraggy/oastools && brew install oastools       # Homebrew (macOS/Linux)
go install github.com/erraggy/oastools/cmd/oastools@latest  # Go install
```

Pre-built binaries for macOS, Linux, and Windows on the [Releases](https://github.com/erraggy/oastools/releases/latest) page.

### Library

```bash
go get github.com/erraggy/oastools@latest
```

Requires Go 1.24+.

## Documentation

📚 **[Documentation Site](https://erraggy.github.io/oastools/)** — Guides, examples, and package deep dives

📖 **[CLI Reference](https://erraggy.github.io/oastools/cli-reference/)** — All commands, flags, and output formats

🤖 **[MCP Server](https://erraggy.github.io/oastools/mcp-server/)** — LLM agent integration via Model Context Protocol

📦 **[API Reference](https://pkg.go.dev/github.com/erraggy/oastools)** — Go package documentation with runnable examples

## Contributing

Fork, branch, `make check`, [conventional commits](https://conventionalcommits.org). See [WORKFLOW.md](WORKFLOW.md) for the full process.

## License

MIT

_All code generated by Claude Code using claude-4-5-sonnet/opus with minor edits and full control by [@erraggy](https://github.com/erraggy)_
```

**Step 2: Verify line count**

Run: `wc -l README.md`
Expected: ~105-115 lines (within the 100-200 target).

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: slim README to evaluator-first billboard (~110 lines)

Move 'Why oastools?' depth, package ecosystem table, examples table,
deep dive guides table, and extended Quick Start to the docs site.
Replace with compact grouped package list, 6 highlights, and focused
CLI + library Quick Start."
```

---

### Task 4: Rewrite `docs/index.md`

Replace the README copy with a purpose-built landing page organized around three entry points: Library, CLI, MCP Server.

**Files:**
- Modify: `docs/index.md` (full rewrite)

**Step 1: Replace docs/index.md with this content**

```markdown
<p align="center">
  <img src="img/banner.svg" alt="oastools" width="100%">
</p>

# oastools

A complete, self-contained OpenAPI toolkit for Go — parse, validate, fix, convert, diff, join, generate, and build OpenAPI specs (2.0–3.2) with minimal dependencies.

[![CI: Go](https://github.com/erraggy/oastools/actions/workflows/go.yml/badge.svg)](https://github.com/erraggy/oastools/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/erraggy/oastools/graph/badge.svg?token=T8768QXQAX)](https://codecov.io/gh/erraggy/oastools)
[![Go Reference](https://pkg.go.dev/badge/github.com/erraggy/oastools.svg)](https://pkg.go.dev/github.com/erraggy/oastools)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Get Started

### 📦 Go Library

Import oastools packages into your Go application for parsing, validation, conversion, code generation, and more. All packages use a consistent functional options API.

```go
result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
vResult, _ := validator.ValidateWithOptions(validator.WithDocument(result))
```

→ **[Developer Guide](developer-guide.md)** — Complete library usage with examples for all 12 packages

### 🖥️ CLI / CI/CD

Run oastools from the terminal, in shell scripts, or as a CI/CD pipeline step. Supports file and URL inputs, stdin/stdout piping, and JSON output for machine consumption.

```bash
oastools validate openapi.yaml
oastools diff --breaking --format json v1.yaml v2.yaml | jq
```

→ **[CLI Reference](cli-reference.md)** — All commands, flags, and output formats

### 🤖 MCP Server

Connect oastools to Claude Code, Cursor, VS Code, or any MCP-compatible AI agent. All 15 tools are available over stdio via the Model Context Protocol.

```bash
oastools mcp  # Start the MCP server
```

→ **[MCP Server Guide](mcp-server.md)** — Setup and tool reference
→ **[Claude Code Plugin](claude-code-plugin.md)** — One-command setup for Claude Code

---

## 🌐 Try it Online

No installation required — use oastools directly in your browser:

**[oastools.robnrob.com](https://oastools.robnrob.com)** — Validate, convert, diff, fix, join, and apply overlays.

---

## What's Included

| Package | Description | Try |
|---------|-------------|:---:|
| [parser](https://pkg.go.dev/github.com/erraggy/oastools/parser) | Parse & analyze OAS files from files, URLs, or readers | |
| [validator](https://pkg.go.dev/github.com/erraggy/oastools/validator) | Validate specs with structural & semantic checks | [🌐](https://oastools.robnrob.com/validate) |
| [fixer](https://pkg.go.dev/github.com/erraggy/oastools/fixer) | Auto-fix common validation errors | [🌐](https://oastools.robnrob.com/fix) |
| [httpvalidator](https://pkg.go.dev/github.com/erraggy/oastools/httpvalidator) | Validate HTTP requests/responses against OAS at runtime | |
| [converter](https://pkg.go.dev/github.com/erraggy/oastools/converter) | Convert between OAS 2.0 and 3.x | [🌐](https://oastools.robnrob.com/convert) |
| [joiner](https://pkg.go.dev/github.com/erraggy/oastools/joiner) | Merge multiple OAS documents with schema deduplication | [🌐](https://oastools.robnrob.com/join) |
| [overlay](https://pkg.go.dev/github.com/erraggy/oastools/overlay) | Apply OpenAPI Overlay v1.0.0 with JSONPath targeting | [🌐](https://oastools.robnrob.com/overlay) |
| [differ](https://pkg.go.dev/github.com/erraggy/oastools/differ) | Detect breaking changes between versions | [🌐](https://oastools.robnrob.com/diff) |
| [generator](https://pkg.go.dev/github.com/erraggy/oastools/generator) | Generate Go client/server code with security support | |
| [builder](https://pkg.go.dev/github.com/erraggy/oastools/builder) | Programmatically construct OAS documents with deduplication | |
| [walker](https://pkg.go.dev/github.com/erraggy/oastools/walker) | Traverse OAS documents with typed handlers and flow control | |
| [oaserrors](https://pkg.go.dev/github.com/erraggy/oastools/oaserrors) | Structured error types for programmatic handling | |

All packages include comprehensive documentation with runnable examples. See individual package pages on [pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools) for API details.

---

## Why oastools?

→ **[Full details](why-oastools.md)**

??? note "Minimal Dependencies"

    Only [`go.yaml.in/yaml`](https://pkg.go.dev/go.yaml.in/yaml/v4), [`golang.org/x/tools`](https://pkg.go.dev/golang.org/x/tools), and [`golang.org/x/text`](https://pkg.go.dev/golang.org/x/text) at runtime. The CLI adds the [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) for `oastools mcp`. No sprawling dependency trees.

??? note "Battle-Tested"

    7,500+ tests validated against 10 production APIs — Stripe, GitHub, Discord, Microsoft Graph (34MB), and more. Spans OAS 2.0 through 3.1, JSON and YAML, 20KB to 34MB.

??? note "Performance"

    Pre-parsed workflows are 11–150x faster. JSON marshaling optimized for 25-32% better performance. 340+ benchmarks track regressions. See [Benchmarks](benchmarks.md).

??? note "Enterprise-Ready"

    Structured errors with `errors.Is()`/`errors.As()`, configurable resource limits, pluggable HTTP clients, deterministic output ordering, and generated `DeepCopy()` methods for safe document mutation.

---

## Installation

### CLI

```bash
brew tap erraggy/oastools && brew install oastools       # Homebrew (macOS/Linux)
go install github.com/erraggy/oastools/cmd/oastools@latest  # Go install
```

Pre-built binaries for macOS, Linux, and Windows on the [Releases](https://github.com/erraggy/oastools/releases/latest) page.

### Library

```bash
go get github.com/erraggy/oastools@latest
```

Requires Go 1.24+.

---

## Examples

Explore complete, runnable examples demonstrating the full oastools ecosystem:

| Category | Examples | Time |
|----------|----------|------|
| **Getting Started** | [Quickstart](examples/quickstart.md), [Validation Pipeline](examples/validation-pipeline.md) | 2-5 min |
| **Workflows** | [Validate & Fix](examples/workflows/validate-and-fix.md), [Version Conversion](examples/workflows/version-conversion.md), [Multi-API Merge](examples/workflows/multi-api-merge.md), [Breaking Changes](examples/workflows/breaking-change-detection.md), [Overlays](examples/workflows/overlay-transformations.md), [HTTP Validation](examples/workflows/http-validation.md) | 3-5 min each |
| **Programmatic** | [Builder](examples/programmatic-api/builder.md) with ServerBuilder | 5 min |
| **Code Generation** | [Petstore](examples/petstore/index.md) (stdlib & chi router) | 10 min |
| **Walker** | [API Statistics](examples/walker/api-statistics.md), [Security Audit](examples/walker/security-audit.md), [Public API Filter](examples/walker/public-api-filter.md) | 3-5 min each |

See [all examples](examples/index.md) for the full list.

---

## Supported OpenAPI Versions

| Version        | Specification                                      |
|----------------|----------------------------------------------------|
| 2.0 (Swagger)  | [spec](https://spec.openapis.org/oas/v2.0.html)   |
| 3.0.0 – 3.0.4 | [spec](https://spec.openapis.org/oas/v3.0.4.html) |
| 3.1.0 – 3.1.2 | [spec](https://spec.openapis.org/oas/v3.1.2.html) |
| 3.2.0          | [spec](https://spec.openapis.org/oas/v3.2.0.html) |

Automatic format detection and preservation (JSON/YAML), external reference resolution, JSON Pointer array index support, and full JSON Schema Draft 2020-12 compliance for OAS 3.1+.

---

## Contributing

1. Fork and create a feature branch
2. Run `make check` before committing
3. Follow [conventional commits](https://conventionalcommits.org) (e.g., `feat(parser): add feature`)
4. Submit a PR

See [WORKFLOW.md](https://github.com/erraggy/oastools/blob/main/WORKFLOW.md) for guidelines.

## License

MIT

_All code generated by Claude Code using claude-4-5-sonnet/opus with minor edits and full control by [@erraggy](https://github.com/erraggy)_
```

**Step 2: Verify the file is valid markdown**

Run: `wc -l docs/index.md`
Expected: ~170-190 lines.

**Step 3: Commit**

```bash
git add docs/index.md
git commit -m "docs: rewrite docs homepage as use-case-oriented landing page

Replace README copy with purpose-built landing page featuring three
entry points (Library, CLI, MCP), full package table, collapsible
'Why oastools?' sections, and examples overview."
```

---

### Task 5: Update `mkdocs.yml` navigation

Restructure the nav to add a "Get Started" section and rename the breaking changes entry.

**Files:**
- Modify: `mkdocs.yml:58-119` (nav section)

**Step 1: Replace the nav section**

Replace the entire `nav:` section in `mkdocs.yml` (lines 58-119) with:

```yaml
nav:
  - Home: index.md
  - "🌐 Try Online": https://oastools.robnrob.com
  - Get Started:
    - Go Library: developer-guide.md
    - CLI / CI-CD: cli-reference.md
    - MCP Server: mcp-server.md
    - Claude Code Plugin: claude-code-plugin.md
  - Guides:
    - Why oastools?: why-oastools.md
    - Detecting API Breaking Changes: breaking-changes.md
    - Generator Features: generator_beyond_boilerplate.md
    - Benchmarks: benchmarks.md
  - Examples:
    - Overview: examples/index.md
    - Getting Started:
      - Quickstart: examples/quickstart.md
      - Validation Pipeline: examples/validation-pipeline.md
    - Workflows:
      - Overview: examples/workflows/index.md
      - Validate and Fix: examples/workflows/validate-and-fix.md
      - Version Conversion: examples/workflows/version-conversion.md
      - Version Migration: examples/workflows/version-migration.md
      - Multi-API Merge: examples/workflows/multi-api-merge.md
      - Collision Resolution: examples/workflows/collision-resolution.md
      - Schema Deduplication: examples/workflows/schema-deduplication.md
      - Schema Renaming: examples/workflows/schema-renaming.md
      - Fixer Showcase: examples/workflows/fixer-showcase.md
      - Pipeline Compositions: examples/workflows/pipeline-compositions.md
      - Breaking Change Detection: examples/workflows/breaking-change-detection.md
      - Overlay Transformations: examples/workflows/overlay-transformations.md
      - HTTP Validation: examples/workflows/http-validation.md
    - Programmatic API:
      - Overview: examples/programmatic-api/index.md
      - Builder: examples/programmatic-api/builder.md
    - Walker:
      - Overview: examples/walker/index.md
      - API Statistics: examples/walker/api-statistics.md
      - Security Audit: examples/walker/security-audit.md
      - Vendor Extensions: examples/walker/vendor-extensions.md
      - Public API Filter: examples/walker/public-api-filter.md
      - API Documentation: examples/walker/api-documentation.md
      - Reference Collector: examples/walker/reference-collector.md
    - Code Generation:
      - Overview: examples/petstore/index.md
      - Standard Library: examples/petstore/stdlib.md
      - Chi Router: examples/petstore/chi.md
  - Package Deep Dives:
    - Parser: packages/parser.md
    - Validator: packages/validator.md
    - Fixer: packages/fixer.md
    - Converter: packages/converter.md
    - Overlay: packages/overlay.md
    - Joiner: packages/joiner.md
    - Differ: packages/differ.md
    - HTTP Validator: packages/httpvalidator.md
    - Generator: packages/generator.md
    - Builder: packages/builder.md
    - Walker: packages/walker.md
  - "📄 White Paper": whitepaper.md
  - Project Info:
    - Contributing: CONTRIBUTORS.md
    - License: LICENSE.md
  - API Documentation: https://pkg.go.dev/github.com/erraggy/oastools
```

Key changes from the current nav:
- **Added** "Get Started" top-level section (Library, CLI, MCP, Plugin)
- **Moved** "Why oastools?" from nowhere → under Guides (new page)
- **Renamed** "Breaking Changes" → "Detecting API Breaking Changes"
- **Moved** "White Paper" from position 3 → near bottom (it's deep content, not a first visit)
- **Removed** "White Paper" from the prominent third position

**Step 2: Validate mkdocs config**

Run: `cd /Users/robbie/code/oastools && python3 -m mkdocs build --strict 2>&1 | head -20`
Expected: Build succeeds without errors. If `mkdocs` is not installed locally, skip this step — CI will validate.

**Step 3: Commit**

```bash
git add mkdocs.yml
git commit -m "docs: restructure mkdocs nav with Get Started section

Add top-level 'Get Started' section with Library/CLI/MCP entry points.
Move White Paper to bottom. Add why-oastools.md to Guides. Rename
breaking changes nav entry to clarify it's about differ capabilities."
```

---

### Task 6: Build verification and final review

Verify the docs site builds correctly and the README renders well.

**Step 1: Build docs site**

Run: `cd /Users/robbie/code/oastools && make docs-build 2>&1 | tail -20`
Expected: Build succeeds. If mkdocs is not available, verify manually that all linked files exist.

**Step 2: Verify all referenced files exist**

Run:
```bash
cd /Users/robbie/code/oastools
for f in docs/why-oastools.md docs/breaking-changes.md docs/developer-guide.md docs/cli-reference.md docs/mcp-server.md docs/claude-code-plugin.md docs/benchmarks.md docs/generator_beyond_boilerplate.md; do
  [ -f "$f" ] && echo "✅ $f" || echo "❌ MISSING: $f"
done
```
Expected: All files show ✅.

**Step 3: Verify README line count**

Run: `wc -l README.md`
Expected: 100-130 lines.

**Step 4: Visual review**

Manually review the README rendering on GitHub (after push) or locally to confirm:
- Banner and badges render correctly
- Package links work
- No broken markdown
- Reads well on mobile (short lines, no wide tables)
