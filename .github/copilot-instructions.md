# GitHub Copilot Instructions for oastools

This file provides guidance to GitHub Copilot when working with code in this repository.

## How to Use This File

GitHub Copilot uses these instructions to:
- Understand the project structure, conventions, and best practices
- Make informed decisions when generating or modifying code
- Follow the same standards that human developers follow
- Avoid common pitfalls specific to this codebase

Read through all sections before making changes. Pay special attention to:
- **Development Environment Setup** - Install golangci-lint v2 before running `make check`
- **Acceptance Criteria** - Know when a task is truly complete
- **Boundaries and Exclusions** - Files and directories you should never modify
- **Testing Requirements** - All exported functionality must have comprehensive tests
- **Benchmark Test Requirements** - Use Go 1.24+ `for b.Loop()` pattern

## Project Overview

`oastools` is a Go-based command-line tool for working with OpenAPI Specification (OAS) files. The primary goals are:
- Validating OpenAPI specification files
- Parsing and analyzing OAS documents
- Joining multiple OpenAPI specification documents
- Converting between OAS versions
- Comparing OAS documents and detecting breaking changes

## Specification References

This tool supports the following OpenAPI Specification versions:

- **OAS 2.0** (Swagger): https://spec.openapis.org/oas/v2.0.html
- **OAS 3.0.x**: https://spec.openapis.org/oas/v3.0.0.html through v3.0.4
- **OAS 3.1.x**: https://spec.openapis.org/oas/v3.1.0.html through v3.1.2
- **OAS 3.2.0**: https://spec.openapis.org/oas/v3.2.0.html

All OAS versions utilize the **JSON Schema Specification Draft 2020-12**: https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html

## Key OpenAPI Specification Concepts

### OAS Version Evolution

**OAS 2.0 (Swagger) → OAS 3.0:**
- **Servers**: `host`, `basePath`, and `schemes` → unified `servers` array with URL templates
- **Components**: `definitions`, `parameters`, `responses`, `securityDefinitions` → `components.*`
- **Request Bodies**: `consumes` + body parameter → `requestBody.content` with media types
- **Response Bodies**: `produces` + schema → `responses.*.content` with media types
- **Security**: `securityDefinitions` → `components.securitySchemes` with flows restructuring
- **New Features**: Links, callbacks, and more flexible parameter serialization

**OAS 3.0 → OAS 3.1:**
- **JSON Schema Alignment**: OAS 3.1 fully aligns with JSON Schema Draft 2020-12
- **Type Arrays**: `type` can be a string or array (e.g., `type: ["string", "null"]`)
- **Nullable Handling**: Deprecated `nullable: true` in favor of `type: ["string", "null"]`
- **Webhooks**: New top-level `webhooks` object for event-driven APIs
- **License**: Added `identifier` field to license object

### Critical Type System Considerations

**interface{} Fields:**
Several OAS 3.1+ fields use `interface{}` to support multiple types. Always use type assertions:
```go
if typeStr, ok := schema.Type.(string); ok {
    // Handle string type
} else if typeArr, ok := schema.Type.([]string); ok {
    // Handle array type
}
```

**Pointer vs Value Types:**
- `OAS3Document.Servers` uses `[]*parser.Server` (slice of pointers)
- Always use `&parser.Server{...}` for pointer semantics
- This pattern applies to other nested structures to avoid unexpected mutations

### Version-Specific Features

**OAS 2.0 Only:**
- `allowEmptyValue`, `collectionFormat`, single `host`/`basePath`/`schemes`

**OAS 3.0+ Only:**
- `requestBody`, `callbacks`, `links`, cookie parameters, `servers` array, TRACE method

**OAS 3.1+ Only:**
- `webhooks`, JSON Schema 2020-12 alignment, `type` as array, `license.identifier`

### Common Pitfalls and Solutions

1. **Assuming schema.Type is always a string** - Use type assertions and handle both string and []string cases
2. **Creating value slices instead of pointer slices** - Check parser types and use `&Type{...}` syntax
3. **Forgetting to track conversion issues** - Add issues for every lossy conversion or unsupported feature
4. **Mutating source documents** - Always deep copy before modification (use JSON marshal/unmarshal)
5. **Not handling operation-level consumes/produces** - Check operation-level first, then fall back to document-level
6. **Ignoring version-specific features during conversion** - Explicitly check and warn about features that don't convert

## Development Commands

### Development Environment Setup

**CRITICAL: Install golangci-lint v2 before running `make check`**

The repository uses golangci-lint v2 for linting. Install it before making any code changes:

```bash
# Install golangci-lint v2.1.0
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.1.0

# Add GOPATH/bin to PATH if not already present
export PATH=$PATH:$(go env GOPATH)/bin

# Verify installation
golangci-lint version
# Expected output: golangci-lint has version 2.1.0 built with go1.24.2...
```

**Without golangci-lint v2 installed, `make check` and `make lint` will fail.**

### Recommended Workflow

After making changes to Go source files:
```bash
make check  # Runs all quality checks (tidy, fmt, lint, test) and shows git status
```

### Common Commands

```bash
# Building
make build    # Build binary to bin/oastools
make install  # Install to $GOPATH/bin

# Testing
make test          # Run all tests with race detection and coverage
make test-coverage # Generate and view HTML coverage report

# Code Quality
make fmt   # Format all Go code
make vet   # Run go vet
make lint  # Run golangci-lint (requires golangci-lint v2 installation)

# Maintenance
make deps  # Download and tidy dependencies
make clean # Remove build artifacts
```

## Architecture

### Directory Structure

- **cmd/oastools/** - CLI entry point with command routing
- **parser/** - Parse YAML/JSON OAS files, resolve references, detect versions
- **validator/** - Validate OAS against spec schema (structural, format, semantic)
- **joiner/** - Join multiple OAS files with flexible collision resolution
- **converter/** - Convert between OAS versions (2.0 ↔ 3.x) with issue tracking
- **differ/** - Compare OAS files, detect breaking changes by severity
- **internal/** - Shared utilities (httputil, severity, issues, testutil)
- **testdata/** - Test fixtures including sample OAS files

Each public package includes `doc.go` (package docs) and `example_test.go` (godoc examples).

### Design Patterns

**IMPORTANT: The parser, converter, and joiner automatically preserve the input file format (JSON or YAML).**

Format detection:
- From file extension (`.json`, `.yaml`, `.yml`)
- From content (JSON starts with `{` or `[`)
- Default to YAML if unknown
- First file determines output format for joiner

**IMPORTANT: Use package-level constants instead of string literals.**
- HTTP Methods: `httputil.MethodGet`, `httputil.MethodPost`, etc.
- HTTP Status Codes: `httputil.ValidateStatusCode()`, `httputil.StandardHTTPStatusCodes`
- Severity Levels: `severity.SeverityError`, `severity.SeverityWarning`, etc.

### Testing Requirements

**CRITICAL: All exported functionality MUST have comprehensive test coverage.**

Test coverage must include:
1. **Exported Functions** - Package-level convenience functions and struct methods
2. **Exported Types** - Struct initialization, fields, type conversions
3. **Exported Constants** - Verify expected values

Coverage types:
- **Positive Cases**: Valid inputs work correctly
- **Negative Cases**: Error handling with invalid inputs, missing files, malformed data
- **Edge Cases**: Boundary conditions, empty inputs, nil values
- **Integration**: Components working together (parse then validate, parse then join)

### Benchmark Test Requirements

**CRITICAL: Use the Go 1.24+ `for b.Loop()` pattern for all benchmarks.**

Correct pattern:
```go
func BenchmarkOperation(b *testing.B) {
    // Setup (parsing, creating instances, etc.)
    source, _ := parser.ParseWithOptions(
        parser.WithFilePath("file.yaml"),
        parser.WithValidateStructure(true),
    )

    for b.Loop() {  // ✅ Modern Go 1.24+ pattern
        _, err := Operation(source)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

**DO NOT:**
- Use `for i := 0; i < b.N; i++` (old pattern)
- Call `b.ReportAllocs()` manually (handled by `b.Loop()`)
- Call `b.ResetTimer()` for trivial setup

## Security

### Common Security Alert Fixes

**Size computation for allocation may overflow (CWE-190):**
```go
// Safe pattern - use uint64 for arithmetic, check fits in int
capacity := 0
sum := uint64(len(a)) + uint64(len(b))
if sum <= uint64(math.MaxInt) {
    capacity = int(sum)
}
result := make([]string, 0, capacity)
```

**Workflow permissions (CWE-275):**
Add minimal `permissions` block to GitHub Actions workflows:
```yaml
permissions:
  contents: read  # Minimal permissions following principle of least privilege
```

## Code Quality Standards

**Before Submitting:**
1. **Install golangci-lint v2** - Required for `make check` and `make lint` (see Development Environment Setup)
2. Run `make check` - all code formatted, lints/tests pass
3. Run `make test-coverage` - review coverage report
4. Verify all new exported functionality has tests
5. Update benchmarks with `make bench-save` if changes affect performance
6. Check for security vulnerabilities: `govulncheck`

**Never submit code with:**
- Untested exported functions, methods, or types
- Tests that only cover the "happy path" without error cases
- Performance regressions without documented justification
- Linting errors (all golangci-lint checks must pass)

**Commit Message Format:**
- First line: Conventional commit message within 72 characters
- Body: Simply formatted (max 100 columns), basic reasoning and changes
- PR: Same title as commit, detailed markdown with reasoning, changes, and context

## Acceptance Criteria

A task is considered complete when ALL of the following are met:

1. **Code Changes**: All required functionality is implemented
2. **Tests**: New/modified exported functions have comprehensive tests (positive, negative, edge cases)
3. **Build**: `make build` succeeds without errors or warnings
4. **Tests Pass**: `make test` passes with no failures
5. **Code Quality**: Code is formatted (`make fmt`) and follows existing patterns
6. **Documentation**: Public APIs have godoc comments; non-trivial changes update relevant docs
7. **No Regressions**: Existing tests still pass; no breaking changes to public APIs (unless intentional)
8. **Security**: No new security vulnerabilities introduced (verify with `govulncheck` if dependencies change)

For documentation-only changes, only items 3, 6, and 7 apply.

## Boundaries and Exclusions

**DO NOT modify these files/directories:**

- `.github/workflows/` - CI/CD workflows (except when specifically requested)
- `testdata/` - Test fixtures (except when adding new test cases)
- `vendor/` - External dependencies (managed by Go modules)
- `bin/`, `dist/` - Build artifacts (generated by build tools)
- `.git/` - Version control internals
- `go.mod`, `go.sum` - Only modify when explicitly adding/removing dependencies
- `.goreleaser.yaml` - Release configuration (managed separately)
- `benchmarks/` - Benchmark data (except when updating benchmarks per [BENCHMARK_UPDATE_PROCESS.md](../BENCHMARK_UPDATE_PROCESS.md))

**DO NOT:**
- Add dependencies without checking for security vulnerabilities first
- Modify benchmark test patterns (must use Go 1.24 `for b.Loop()` pattern)
- Remove or weaken existing test coverage
- Change public API signatures without documenting breaking changes
- Commit secrets, credentials, or sensitive data
- Create temporary files in the repository root (use `/tmp` instead)

## Go Module

- Module path: `github.com/erraggy/oastools`
- Minimum Go version: 1.24

## Public API Structure

All core packages are public:
- `github.com/erraggy/oastools/parser` - Parse OpenAPI specifications
- `github.com/erraggy/oastools/validator` - Validate OpenAPI specifications
- `github.com/erraggy/oastools/joiner` - Join multiple OpenAPI specifications
- `github.com/erraggy/oastools/converter` - Convert between OpenAPI specification versions
- `github.com/erraggy/oastools/differ` - Compare and diff OpenAPI specifications

### API Design Philosophy

Two complementary API styles:
1. **Package-level convenience functions** - For simple, one-off operations
2. **Struct-based API** - For reusable instances with configuration

Use convenience functions for: simple scripts, prototyping, default configuration
Use struct-based API for: multiple files, reusable instances, advanced configuration, performance

### Key API Features

**Parser Package:**
- Functional options: `parser.ParseWithOptions(parser.WithFilePath(...), parser.WithResolveRefs(...), ...)`
- Struct-based: `parser.New()`, `Parser.Parse()`, `Parser.ParseReader()`, `Parser.ParseBytes()`
- `ParseResult.SourcePath` tracks source (`"ParseReader.yaml"` for readers, `"ParseBytes.yaml"` for bytes)

**Validator Package:**
- Functional options: `validator.ValidateWithOptions(validator.WithFilePath(...), validator.WithIncludeWarnings(...), ...)`
- Struct-based: `validator.New()`, `Validator.Validate()`, `Validator.ValidateParsed()`

**Joiner Package:**
- Functional options: `joiner.JoinWithOptions(joiner.WithFilePaths(...), joiner.WithPathStrategy(...), ...)`
- Struct-based: `joiner.New()`, `Joiner.Join()`, `Joiner.JoinParsed()`, `Joiner.WriteResult()`
- All input documents must be pre-validated (Errors slice must be empty)

**Converter Package:**
- Functional options: `converter.ConvertWithOptions(converter.WithFilePath(...), converter.WithTargetVersion(...), ...)`
- Struct-based: `converter.New()`, `Converter.Convert()`, `Converter.ConvertParsed()`
- Configuration: `StrictMode`, `IncludeInfo`
- Returns ConversionResult with severity-tracked issues (Info, Warning, Critical)

**Differ Package:**
- Functional options: `differ.DiffWithOptions(differ.WithSourceFilePath(...), differ.WithTargetFilePath(...), differ.WithMode(...), ...)`
- Struct-based: `differ.New()`, `Differ.Diff()`, `Differ.DiffParsed()`
- Returns DiffResult with changes categorized by severity (Critical, Error, Warning, Info)

### Usage Examples

**Quick operations:**
```go
// Parse
result, _ := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithValidateStructure(true),
)

// Validate
result, _ := validator.ValidateWithOptions(
    validator.WithFilePath("openapi.yaml"),
    validator.WithIncludeWarnings(true),
)

// Join
result, _ := joiner.JoinWithOptions(
    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
    joiner.WithConfig(joiner.DefaultConfig()),
)

// Convert
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)

// Diff
result, _ := differ.DiffWithOptions(
    differ.WithSourceFilePath("v1.yaml"),
    differ.WithTargetFilePath("v2.yaml"),
)
```

**Reusable instances:**
```go
// Parser for multiple files
p := parser.New()
p.ResolveRefs = false
result1, _ := p.Parse("api1.yaml")
result2, _ := p.Parse("api2.yaml")

// Validator with config
v := validator.New()
v.IncludeWarnings = true
result1, _ := v.Validate("api1.yaml")

// Joiner with strategy
j := joiner.New(config)
result1, _ := j.Join([]string{"api1-base.yaml", "api1-ext.yaml"})

// Converter with settings
c := converter.New()
c.StrictMode = false
result1, _ := c.Convert("swagger-v1.yaml", "3.0.3")

// Differ for multiple comparisons
d := differ.New()
result1, _ := d.Diff("api-v1.yaml", "api-v2.yaml")
result2, _ := d.Diff("api-v2.yaml", "api-v3.yaml")
```
