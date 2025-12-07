# AGENTS.md

This file provides quick, actionable guidance for AI coding agents working on the oastools project. For comprehensive details, see [.github/copilot-instructions.md](.github/copilot-instructions.md).

## Project Overview

`oastools` is a Go-based CLI tool for working with OpenAPI Specification (OAS) files. It supports parsing, validating, fixing, joining, converting, and comparing OAS documents across versions 2.0, 3.0.x, 3.1.x, and 3.2.0.

**Module:** `github.com/erraggy/oastools`  
**Go Version:** 1.24+

## Dev Environment Setup

**CRITICAL: Install golangci-lint v2 before making code changes:**
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.1.0
export PATH=$PATH:$(go env GOPATH)/bin
golangci-lint version  # Should show v2.1.0
```

## Quick Commands

```bash
# Build and test workflow (run after every code change)
make check          # Runs tidy, fmt, lint, test, and git status

# Individual commands
make build          # Build binary to bin/oastools
make test           # Run all tests with race detection
make test-coverage  # Generate and view HTML coverage report
make fmt            # Format all Go code
make lint           # Run golangci-lint (requires v2 installation)

# Benchmarks
make bench          # Run benchmarks
make bench-save     # Save new benchmark baseline (only if changes affect performance)
```

## Testing Requirements

**ALL exported functionality MUST have comprehensive tests:**
- Positive cases (valid inputs work)
- Negative cases (errors handled correctly)
- Edge cases (nil, empty, boundary conditions)
- Integration cases (components work together)

**Benchmark tests MUST use Go 1.24+ pattern:**
```go
func BenchmarkOperation(b *testing.B) {
    // Setup
    source, _ := parser.ParseWithOptions(...)
    
    for b.Loop() {  // ✅ Use b.Loop(), NOT for i := 0; i < b.N; i++
        _, err := Operation(source)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Code Style & Patterns

**Use package constants instead of string literals:**
- HTTP Methods: `httputil.MethodGet`, `httputil.MethodPost`, etc.
- HTTP Status Codes: `httputil.ValidateStatusCode()`, `httputil.StandardHTTPStatusCodes`
- Severity Levels: `severity.SeverityError`, `severity.SeverityWarning`, etc.

**Format preservation:**
- Parser, converter, and joiner preserve input file format (JSON or YAML)
- Format detected from file extension or content
- First file determines output format for joiner

**Type handling in OAS 3.1+:**
```go
// schema.Type can be string or []string - always use type assertions
if typeStr, ok := schema.Type.(string); ok {
    // Handle string type
} else if typeArr, ok := schema.Type.([]string); ok {
    // Handle array type
}
```

**Pointer semantics:**
```go
// Use pointer slices for nested structures
servers := []*parser.Server{
    &parser.Server{URL: "https://api.example.com"},  // ✅ Correct
}
```

## Security Fixes

**Size computation overflow (CWE-190):**
```go
capacity := 0
sum := uint64(len(a)) + uint64(len(b))
if sum <= uint64(math.MaxInt) {
    capacity = int(sum)
}
result := make([]string, 0, capacity)
```

**Always check for vulnerabilities:**
```bash
govulncheck ./...
```

## Boundaries - DO NOT Modify

- `.github/workflows/` - CI/CD workflows (unless specifically requested)
- `testdata/` - Test fixtures (unless adding new test cases)
- `vendor/`, `bin/`, `dist/` - Generated artifacts
- `go.mod`, `go.sum` - Only modify when explicitly adding/removing dependencies
- `.goreleaser.yaml` - Release configuration
- `benchmarks/` - Benchmark data (see BENCHMARK_UPDATE_PROCESS.md)

## Acceptance Criteria

A task is complete when:
1. ✅ All required functionality is implemented
2. ✅ New/modified exported functions have comprehensive tests
3. ✅ `make build` succeeds without errors
4. ✅ `make test` passes with no failures
5. ✅ Code is formatted and follows existing patterns
6. ✅ Public APIs have godoc comments
7. ✅ No regressions in existing tests
8. ✅ No new security vulnerabilities (`govulncheck`)

For documentation-only changes, only items 3, 6, and 7 apply.

## Commit & PR Format

**Commit messages:**
- First line: Conventional commit format (max 72 chars)
- Examples: `feat: add webhook support to parser`, `fix: handle nil pointer in converter`

**PR format:**
- Title: Same as commit message
- Body: Detailed markdown explaining reasoning, changes, and context

## Common Pitfalls

1. **Type assertions:** Don't assume `schema.Type` is always a string - check both string and []string
2. **Pointer slices:** Use `&Type{...}` for pointer semantics, not value types
3. **Conversion issues:** Track all lossy conversions or unsupported features
4. **Deep copy:** Never mutate source documents - use JSON marshal/unmarshal
5. **Operation-level overrides:** Check operation-level fields before falling back to document-level
6. **Version features:** Explicitly handle version-specific features during conversion

## Architecture

```
cmd/oastools/     - CLI entry point
parser/           - Parse YAML/JSON OAS files, resolve refs
validator/        - Validate OAS against spec schema
fixer/            - Automatically fix common validation errors
converter/        - Convert between OAS versions
joiner/           - Join multiple OAS files
differ/           - Compare and diff OAS files
generator/        - Generate Go code from OAS files
builder/          - Build OAS documents programmatically
internal/         - Shared utilities (httputil, severity, issues, testutil)
testdata/         - Test fixtures
```

Each public package includes `doc.go` (package docs) and `example_test.go` (godoc examples).

## Resources

- Full instructions: [.github/copilot-instructions.md](.github/copilot-instructions.md)
- OAS 2.0 Spec: https://spec.openapis.org/oas/v2.0.html
- OAS 3.0 Spec: https://spec.openapis.org/oas/v3.0.0.html
- OAS 3.1 Spec: https://spec.openapis.org/oas/v3.1.0.html
- OAS 3.2 Spec: https://spec.openapis.org/oas/v3.2.0.html
- JSON Schema 2020-12: https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html
