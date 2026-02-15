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

**Spec Lifecycle** — [parser](https://pkg.go.dev/github.com/erraggy/oastools/parser) · [validator](https://pkg.go.dev/github.com/erraggy/oastools/validator) · [fixer](https://pkg.go.dev/github.com/erraggy/oastools/fixer) · [converter](https://pkg.go.dev/github.com/erraggy/oastools/converter)<br>
**Multi-Spec Ops** — [joiner](https://pkg.go.dev/github.com/erraggy/oastools/joiner) · [differ](https://pkg.go.dev/github.com/erraggy/oastools/differ) · [overlay](https://pkg.go.dev/github.com/erraggy/oastools/overlay)<br>
**Code & Query** — [generator](https://pkg.go.dev/github.com/erraggy/oastools/generator) · [builder](https://pkg.go.dev/github.com/erraggy/oastools/builder) · [walker](https://pkg.go.dev/github.com/erraggy/oastools/walker)<br>
**Runtime** — [httpvalidator](https://pkg.go.dev/github.com/erraggy/oastools/httpvalidator) · [oaserrors](https://pkg.go.dev/github.com/erraggy/oastools/oaserrors)

12 packages covering the full OpenAPI lifecycle. [See full details →](https://erraggy.github.io/oastools/)

## Highlights

- 📦 **Minimal Dependencies** — Only [`go.yaml.in/yaml`](https://pkg.go.dev/go.yaml.in/yaml/v4), [`golang.org/x/tools`](https://pkg.go.dev/golang.org/x/tools), [`golang.org/x/text`](https://pkg.go.dev/golang.org/x/text), and the [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) at runtime
- ✅ **Battle-Tested** — 8,000+ tests against 10 production APIs (Stripe, GitHub, Discord, MS Graph 34MB)
- ⚡ **Performance** — Pre-parsed workflows 11–150x faster; 340+ benchmarks
- 📋 **OAS 2.0–3.2** — Full JSON Schema Draft 2020-12 for OAS 3.1+; automatic format detection and preservation
- 🤖 **AI-Ready** — Built-in [MCP server](https://erraggy.github.io/oastools/mcp-server/) exposes all capabilities to LLM agents
- 🌐 **Try Online** — [oastools.robnrob.com](https://oastools.robnrob.com) — no install required

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
brew install erraggy/oastools/oastools                   # Homebrew (macOS/Linux)
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
