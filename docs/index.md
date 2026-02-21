<p align="center">
  <img src="img/banner.svg" alt="oastools" width="100%">
</p>

# oastools

A complete, self-contained OpenAPI toolkit for Go — parse, validate, fix, convert, diff, join, walk, generate, and build OpenAPI specs (2.0–3.2) with an MCP server for AI-assisted development.

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
vResult, _ := validator.ValidateWithOptions(validator.WithParsed(*result))
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

Connect oastools to Claude Code, Cursor, VS Code, or any MCP-compatible AI agent. All 17 tools are available over stdio via the Model Context Protocol.

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

    Only [`go.yaml.in/yaml`](https://pkg.go.dev/go.yaml.in/yaml/v4), [`golang.org/x/tools`](https://pkg.go.dev/golang.org/x/tools), [`golang.org/x/text`](https://pkg.go.dev/golang.org/x/text), and the [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) at runtime. No sprawling dependency trees.

??? note "Battle-Tested"

    8,000+ tests validated against 10 production APIs — Stripe, GitHub, Discord, Microsoft Graph (34MB), and more. Spans OAS 2.0 through 3.1, JSON and YAML, 20KB to 34MB.

??? note "Performance"

    Pre-parsed workflows are 11–150x faster. JSON marshaling optimized for 25-32% better performance. 340+ benchmarks track regressions. See the [whitepaper performance section](whitepaper.md#17-performance-analysis) for detailed analysis.

??? note "Enterprise-Ready"

    Structured errors with `errors.Is()`/`errors.As()`, configurable resource limits, pluggable HTTP clients, deterministic output ordering, and generated `DeepCopy()` methods for safe document mutation.

---

## Installation

### CLI

```bash
brew install erraggy/oastools/oastools                   # Homebrew (macOS/Linux)
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
