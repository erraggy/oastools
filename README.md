# oastools

OpenAPI Specification (OAS) tools for validating, parsing, converting, diffing, joining, and building specs; as well as generating client/servers/types _from_ specs.

[![CI: Go](https://github.com/erraggy/oastools/actions/workflows/go.yml/badge.svg)](https://github.com/erraggy/oastools/actions/workflows/go.yml)
[![CI: golangci-lint](https://github.com/erraggy/oastools/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/erraggy/oastools/actions/workflows/golangci-lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/erraggy/oastools)](https://goreportcard.com/report/github.com/erraggy/oastools)
[![codecov](https://codecov.io/gh/erraggy/oastools/graph/badge.svg?token=T8768QXQAX)](https://codecov.io/gh/erraggy/oastools)
[![Go Reference](https://pkg.go.dev/badge/github.com/erraggy/oastools.svg)](https://pkg.go.dev/github.com/erraggy/oastools)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Parse** - Parse and analyze OpenAPI specifications (all versions 2.0 through 3.2.0)
- **Validate** - Validate OpenAPI specification files for correctness
- **Convert** - Convert between OpenAPI versions (2.0 ↔ 3.x) with transparent issue tracking
- **Join** - Merge multiple OpenAPI specifications with flexible collision resolution
- **Diff** - Compare OpenAPI specifications and detect breaking changes
- **Generate** - Create idiomatic Go code for API clients and server stubs from OpenAPI specs
- **Build** - Programmatically construct OpenAPI specifications with reflection-based schema generation
- **Library** - Use as a Go library with both simple and advanced APIs

## Installation

### CLI Tool

#### Homebrew (macOS and Linux)

```bash
brew tap erraggy/oastools
brew install oastools
```

To upgrade to the latest version:

```bash
brew upgrade oastools
```

#### Other Methods

```bash
# Using Go
go install github.com/erraggy/oastools/cmd/oastools@latest

# From source
git clone https://github.com/erraggy/oastools.git
cd oastools
make install

# Download pre-built binaries
# Available for macOS (Intel/ARM), Linux (x86_64/ARM64/i386), and Windows (x86_64/i386)
# https://github.com/erraggy/oastools/releases/latest
```

### Go Library

```bash
go get github.com/erraggy/oastools@latest
```

## Quick Start

### Command-Line Interface

```bash
# Validate an OpenAPI spec (from file or URL)
oastools validate openapi.yaml
oastools validate https://example.com/api/openapi.yaml

# Parse and analyze
oastools parse openapi.yaml

# Convert between versions
oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml

# Join multiple specs
oastools join -o merged.yaml base.yaml extensions.yaml

# Compare specs and detect breaking changes
oastools diff --breaking api-v1.yaml api-v2.yaml

# Generate Go client and server code
oastools generate --client --server -o ./generated -p api openapi.yaml

# Pipeline support (stdin with quiet mode)
cat swagger.yaml | oastools convert -q -t 3.0.3 - > openapi.yaml
curl -s https://example.com/openapi.yaml | oastools validate -q -

# Structured output for scripting
oastools validate --format json openapi.yaml | jq '.valid'
oastools diff --format json --breaking v1.yaml v2.yaml | jq '.HasBreakingChanges'

# Show all commands
oastools help
```

**CLI Features**:
- **Load from files, URLs, or stdin** (HTTP/HTTPS) for parse, validate, convert, diff, and generate commands
- **Pipeline support** with stdin (`-`) and quiet mode (`-q`) for shell scripting
- **Structured output** with `--format json/yaml` for programmatic processing
- Automatic format detection (JSON/YAML)
- Format preservation (JSON input → JSON output, YAML → YAML)
- Detailed validation error messages with spec references
- **Path validation** with REST best practice warnings (trailing slashes, malformed templates)
- External reference resolution (local files only)
- Collision resolution strategies for joining (AcceptLeft, AcceptRight, Error)
- **Code generation** for HTTP clients and server interfaces (client, server, or types-only)

### Library Usage

The library provides two API styles:

#### Simple API (Convenience Functions)

For quick, one-off operations:

```go
import (
    "net/http"

    "github.com/erraggy/oastools/parser"
    "github.com/erraggy/oastools/validator"
    "github.com/erraggy/oastools/converter"
    "github.com/erraggy/oastools/joiner"
    "github.com/erraggy/oastools/differ"
    "github.com/erraggy/oastools/generator"
    "github.com/erraggy/oastools/builder"
)

// Parse
result, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithValidateStructure(true),
)

// Validate
vResult, err := validator.ValidateWithOptions(
    validator.WithFilePath("openapi.yaml"),
    validator.WithIncludeWarnings(true),
)

// Convert
cResult, err := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)

// Join
jResult, err := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
    joiner.WithConfig(joiner.DefaultConfig()),
)

// Diff
dResult, err := differ.DiffWithOptions(
    differ.WithSourceFilePath("api-v1.yaml"),
    differ.WithTargetFilePath("api-v2.yaml"),
)

// Generate Go code (client, server, or types)
gResult, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("petstore"),
    generator.WithClient(true),
)
if err == nil {
    gResult.WriteFiles("./generated")  // Write generated files
}

// Build (programmatic construction)
type User struct {
    ID   int64  `json:"id" oas:"description=Unique identifier"`
    Name string `json:"name" oas:"minLength=1"`
}

spec := builder.New(parser.OASVersion320).
    SetTitle("User API").
    SetVersion("1.0.0").
    AddOperation(http.MethodGet, "/users",
        builder.WithOperationID("listUsers"),
        builder.WithResponse(http.StatusOK, []User{}),
    )
doc, err := spec.BuildOAS3()  // Returns *parser.OAS3Document
```

#### Advanced API (Reusable Instances)

For processing multiple files with the same configuration:

```go
// Create reusable instances
p := parser.New()
p.ResolveRefs = false
p.ValidateStructure = true

v := validator.New()
v.IncludeWarnings = true

c := converter.New()
c.StrictMode = false

config := joiner.DefaultConfig()
j := joiner.New(config)

d := differ.New()
d.Mode = differ.ModeBreaking

// Process multiple files efficiently
result1, _ := p.Parse("api1.yaml")
result2, _ := p.Parse("api2.yaml")
result3, _ := p.Parse("api3.yaml")

v.ValidateParsed(result1)  // 30x faster than Validate
c.ConvertParsed(result2, "3.0.3")  // 9x faster than Convert
j.JoinParsed([]parser.ParseResult{result1, result2})  // 154x faster than Join
d.DiffParsed(result1, result2)  // Faster than Diff

// Generator processes one spec at a time
gen := generator.New()
gen.PackageName = "myapi"
gen.GenerateClient = true
gen.GenerateServer = true
genResult, _ := gen.Generate("api.yaml")
genResult.WriteFiles("./generated")
```

**Library Features**:
- **Parser**: Parse from files, readers, or bytes; optional reference resolution and structure validation
- **Validator**: Structural, format, and semantic validation; configurable warnings and strict mode
- **Converter**: Convert OAS 2.0 ↔ 3.x with severity-tracked issues (Info, Warning, Critical)
- **Joiner**: Flexible collision strategies, array merging, tag deduplication
- **Differ**: Compare specs with simple or breaking change detection; severity-based classification (Critical, Error, Warning, Info)
- **Generator**: Create idiomatic Go code for HTTP clients, server interfaces, or type-only generation
- **Builder**: Programmatically construct OAS documents with reflection-based schema generation from Go types
- **Performance**: ParseOnce pattern enables efficient workflows (parse once, validate/convert/join/diff many times)

See [pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools) for complete API documentation.

## Benchmarks

The library includes comprehensive performance benchmarking (100+ benchmarks across all packages). As of v1.9.1, significant optimizations have been implemented:

**Performance Highlights**:
- **25-32% faster** JSON marshaling (v1.7.0 optimization)
- **29-37% fewer** memory allocations
- **31x faster** validation with `ValidateParsed` vs `Validate` (parse once, validate many)
- **9x faster** conversion with `ConvertParsed` vs `Convert` (parse once, convert many)
- **150x faster** joining with `JoinParsed` vs `Join` (parse once, join many)
- **81x faster** diffing with `DiffParsed` vs `Diff` (parse once, diff many)

**Document Processing Performance** (Apple M4, Go 1.24):

| Operation        | Small (~60 lines) | Medium (~570 lines) | Large (~6000 lines) |
|------------------|-------------------|---------------------|---------------------|
| Parse            | 138 μs            | 1,119 μs            | 13,880 μs           |
| Validate         | 139 μs            | 1,133 μs            | 14,409 μs           |
| Convert (OAS2→3) | 148 μs            | 1,184 μs            | -                   |
| Join (2 docs)    | 101 μs            | -                   | -                   |
| Diff (2 docs)    | 448 μs            | -                   | -                   |

For detailed performance analysis, methodology, and optimization strategies, see [benchmarks.md](benchmarks.md).

## Supported OpenAPI Versions

All official OpenAPI Specification releases are supported:

| Version   | Specification                                                        |
|-----------|----------------------------------------------------------------------|
| **2.0**   | [OAS 2.0](https://spec.openapis.org/oas/v2.0.html)                   |
| **3.0.x** | [3.0.0](https://spec.openapis.org/oas/v3.0.0.html) - [3.0.4](https://spec.openapis.org/oas/v3.0.4.html) |
| **3.1.x** | [3.1.0](https://spec.openapis.org/oas/v3.1.0.html) - [3.1.2](https://spec.openapis.org/oas/v3.1.2.html) |
| **3.2.0** | [OAS 3.2.0](https://spec.openapis.org/oas/v3.2.0.html)               |

> **Note:** Release candidate versions (e.g., `3.0.0-rc0`) are detected but not officially supported.

## Format Preservation

When converting or joining, oastools automatically preserves the input file format:

- **JSON input → JSON output**
- **YAML input → YAML output**

This ensures format consistency across your toolchain.

```bash
# JSON file produces JSON output
oastools convert -t 3.0.3 swagger.json -o openapi.json

# YAML file produces YAML output
oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml

# First file's format determines output (for joining)
oastools join -o merged.json api1.json api2.json
```

## Validation Example

```bash
$ oastools validate testdata/invalid-oas3.yaml
OpenAPI Specification Validator
================================

File: testdata/invalid-oas3.yaml
Version: 3.0.3

Errors (12):
  ✗ document: oas 3.0.3: missing required field 'info.version': Info object must have a version string per spec
  ✗ document: oas 3.0.3: invalid path pattern 'paths.items': path must begin with '/'
  ✗ document: oas 3.0.3: missing required field 'paths.items.get.responses': Operation must have a responses object
  ✗ info.version: Info object must have a version
    Spec: https://spec.openapis.org/oas/v3.0.3.html#info-object
  ✗ paths.items: Path must start with '/'
    Spec: https://spec.openapis.org/oas/v3.0.3.html#paths-object
  ...

Warnings (3):
  ⚠ paths.items.post: Operation should have a description or summary for better documentation
    Spec: https://spec.openapis.org/oas/v3.0.3.html#operation-object
  ...

✗ Validation failed: 12 error(s), 3 warning(s)
```

## Limitations

### External References

- **HTTP(S) References**: Supported via `--resolve-http-refs` flag (opt-in for security). Use `--insecure` for self-signed certificates.
- **Security**: External file references are restricted to the base directory and subdirectories to prevent path traversal attacks
- **URL-loaded Specs**: When loading a spec from a URL, relative `$ref` paths resolve against the current directory, not relative to the URL (known limitation)

## Development

### Prerequisites

- Go 1.24 or higher
- make (optional, but recommended)

### Commands

```bash
# Build and test
make build          # Build binary (output: bin/oastools)
make test           # Run all tests with coverage
make test-coverage  # Generate HTML coverage report

# Code quality
make fmt            # Format code
make lint           # Run golangci-lint
make check          # Run all quality checks (tidy, fmt, lint, test)

# Benchmarks
make bench-parser   # Benchmark parser package
make bench-baseline # Save benchmark baseline

# Other
make clean          # Remove build artifacts
make help           # Show all available commands
```

### Integration Testing

The project includes comprehensive integration tests using 10 real-world public OpenAPI specifications spanning OAS 2.0, 3.0.x, and 3.1.0. These tests validate the full pipeline: parser, validator, converter, joiner, and differ.

```bash
# Download corpus specs (one-time setup)
make corpus-download

# Run integration tests (excludes large >5MB specs)
make test-corpus-short

# Run all integration tests including large specs
make test-corpus
```

**Corpus specifications include:** Petstore (2.0), Discord (3.1.0), Stripe, GitHub, Microsoft Graph, DigitalOcean, Google Maps, Asana, Plaid, and US National Weather Service.

For detailed information about the corpus selection methodology and validation results, see the [OAS Corpus Research](planning/Top10-Public-OAS-Docs-CombinedSummary.md) documentation.

### Project Structure

```
.
├── cmd/oastools/       # CLI entry point
├── parser/             # OpenAPI parsing library (public API)
├── validator/          # OpenAPI validation library (public API)
├── converter/          # OpenAPI conversion library (public API)
├── joiner/             # OpenAPI joining library (public API)
├── differ/             # OpenAPI diffing library (public API)
├── generator/          # OpenAPI code generation library (public API)
├── builder/            # OpenAPI builder library (public API)
├── internal/           # Internal shared utilities
│   ├── corpusutil/     # Corpus management for integration tests
│   ├── httputil/       # HTTP validation constants
│   ├── severity/       # Severity levels for issues
│   ├── issues/         # Unified issue type
│   └── testutil/       # Test fixtures and helpers
├── testdata/           # Test fixtures and sample specs
└── planning/           # Research docs (OAS corpus selection)
```

All seven main packages (parser, validator, converter, joiner, differ, generator, builder) are public and can be imported directly.

## Documentation

- **[Developer Guide](docs/developer-guide.md)** - Comprehensive guide for library and CLI usage
- **[CLI Reference](docs/cli-reference.md)** - Complete command-line reference with examples
- **[Breaking Changes Guide](docs/breaking-changes.md)** - Understanding breaking change semantics
- **[Performance Benchmarks](benchmarks.md)** - Detailed performance analysis
- **[API Reference](https://pkg.go.dev/github.com/erraggy/oastools)** - GoDoc API documentation

## Contributing

We welcome contributions! Please see:

- **[AGENTS.md](AGENTS.md)** - Quick reference guide for AI coding agents (GitHub Copilot, etc.)
- **[WORKFLOW.md](WORKFLOW.md)** - Development workflow from commit to PR to release
- **[CLAUDE.md](CLAUDE.md)** - Project guidance for Claude Code
- **[CONTRIBUTORS.md](CONTRIBUTORS.md)** - List of contributors

**Quick Start for Contributors:**
1. Read [AGENTS.md](AGENTS.md) for quick setup and commands (AI agents start here)
2. Read [WORKFLOW.md](WORKFLOW.md) for the complete development process
3. Run `make check` before committing
4. Follow conventional commit format (e.g., `feat(parser): add feature`)
5. Create detailed pull requests with testing checklist
6. All new exported functionality must have comprehensive tests

## License

MIT

_Note: All code generated by Claude Code using claude-4-5-sonnet with minor edits and full control by [@erraggy](https://github.com/erraggy)_
