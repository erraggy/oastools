# oastools

OpenAPI Specification (OAS) tools for validation, parsing, converting, joining, and comparing.

[![Go Reference](https://pkg.go.dev/badge/github.com/erraggy/oastools.svg)](https://pkg.go.dev/github.com/erraggy/oastools)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Parse** - Parse and analyze OpenAPI specifications (all versions 2.0 through 3.2.0)
- **Validate** - Validate OpenAPI specification files for correctness
- **Convert** - Convert between OpenAPI versions (2.0 ↔ 3.x) with transparent issue tracking
- **Join** - Merge multiple OpenAPI specifications with flexible collision resolution
- **Diff** - Compare OpenAPI specifications and detect breaking changes
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

# Show all commands
oastools help
```

**CLI Features**:
- **Load from files or URLs** (HTTP/HTTPS) for parse, validate, and convert commands
- Automatic format detection (JSON/YAML)
- Format preservation (JSON input → JSON output, YAML → YAML)
- Detailed validation error messages with spec references
- **Path validation** with REST best practice warnings (trailing slashes, malformed templates)
- External reference resolution (local files only)
- Collision resolution strategies for joining (AcceptLeft, AcceptRight, Error)

### Library Usage

The library provides two API styles:

#### Simple API (Convenience Functions)

For quick, one-off operations:

```go
import (
    "github.com/erraggy/oastools/parser"
    "github.com/erraggy/oastools/validator"
    "github.com/erraggy/oastools/converter"
    "github.com/erraggy/oastools/joiner"
    "github.com/erraggy/oastools/differ"
)

// Parse
result, err := parser.Parse("openapi.yaml", false, true)

// Validate
vResult, err := validator.Validate("openapi.yaml", true, false)

// Convert
cResult, err := converter.Convert("swagger.yaml", "3.0.3")

// Join
config := joiner.DefaultConfig()
jResult, err := joiner.Join([]string{"base.yaml", "ext.yaml"}, config)

// Diff
dResult, err := differ.Diff("api-v1.yaml", "api-v2.yaml")
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
```

**Library Features**:
- **Parser**: Parse from files, readers, or bytes; optional reference resolution and structure validation
- **Validator**: Structural, format, and semantic validation; configurable warnings and strict mode
- **Converter**: Convert OAS 2.0 ↔ 3.x with severity-tracked issues (Info, Warning, Critical)
- **Joiner**: Flexible collision strategies, array merging, tag deduplication
- **Differ**: Compare specs with simple or breaking change detection; severity-based classification (Critical, Error, Warning, Info)
- **Performance**: ParseOnce pattern enables efficient workflows (parse once, validate/convert/join/diff many times)

See [pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools) for complete API documentation.

## Benchmarks

The library includes comprehensive performance benchmarking (60+ benchmarks across all packages). As of v1.7.0, significant optimizations have been implemented:

**Performance Highlights**:
- **25-32% faster** JSON marshaling (v1.7.0 optimization)
- **29-37% fewer** memory allocations
- **30x faster** validation with `ValidateParsed` vs `Validate` (parse once, validate many)
- **9x faster** conversion with `ConvertParsed` vs `Convert` (parse once, convert many)
- **154x faster** joining with `JoinParsed` vs `Join` (parse once, join many)

**Document Processing Performance** (Apple M4, Go 1.24):

| Operation        | Small (~60 lines) | Medium (~570 lines) | Large (~6000 lines) |
|------------------|-------------------|---------------------|---------------------|
| Parse            | 142 μs            | 1,130 μs            | 14,131 μs           |
| Validate         | 143 μs            | 1,160 μs            | 14,635 μs           |
| Convert (OAS2→3) | 153 μs            | 1,314 μs            | -                   |
| Join (2 docs)    | 115 μs            | -                   | -                   |

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

- **HTTP(S) References Not Supported**: Only local file references for `$ref` values are supported
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

### Project Structure

```
.
├── cmd/oastools/       # CLI entry point
├── parser/             # OpenAPI parsing library (public API)
├── validator/          # OpenAPI validation library (public API)
├── converter/          # OpenAPI conversion library (public API)
├── joiner/             # OpenAPI joining library (public API)
├── differ/             # OpenAPI diffing library (public API)
├── internal/           # Internal shared utilities
│   ├── httputil/       # HTTP validation constants
│   ├── severity/       # Severity levels for issues
│   ├── issues/         # Unified issue type
│   └── testutil/       # Test fixtures and helpers
└── testdata/           # Test fixtures and sample specs
```

All five main packages (parser, validator, converter, joiner, differ) are public and can be imported directly.

## License

MIT

_Note: All code generated by Claude Code using claude-4-5-sonnet with minor edits and full control by [@erraggy](https://github.com/erraggy)_
