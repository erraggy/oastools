# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
make lint  # Run golangci-lint

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
    source, _ := parser.Parse("file.yaml", false, true)

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

### Checking for Security Vulnerabilities

```bash
# List all code scanning alerts
gh api /repos/erraggy/oastools/code-scanning/alerts

# Check for Go vulnerabilities
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

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

### Security Fix Workflow

1. Retrieve alerts: `gh api /repos/erraggy/oastools/code-scanning/alerts`
2. Review details and implement fixes
3. Run `make test` and `govulncheck`
4. Commit with security-focused message
5. Verify alerts closed after push

### Retrieving PR Check Results

```bash
gh pr checks <PR_NUMBER>           # Check status of all PR checks
gh run view <RUN_ID>                # View workflow run details
gh run watch <RUN_ID>               # Monitor running workflow
gh pr view <PR_NUMBER> --comments   # Get all PR comments (including bot reviews)
```

## Submitting Changes

**Before Submitting:**
1. Run `make check` - all code formatted, lints/tests pass
2. Run `make test-coverage` - review coverage report
3. Verify all new exported functionality has tests
4. Update benchmarks with `make bench-save` if changes affect performance
5. Check for security vulnerabilities: `govulncheck`

**Never submit a PR with:**
- Untested exported functions, methods, or types
- Tests that only cover the "happy path" without error cases
- Performance regressions without documented justification

**Commit Message Format:**
- First line: Conventional commit message within 72 characters
- Body: Simply formatted (max 100 columns), basic reasoning and changes
- PR: Same title as commit, detailed markdown with reasoning, changes, and context

## Creating a New Release

### Prerequisites

1. On `main` branch, up-to-date with `origin/main`
2. All tests pass: `make check`
3. Update benchmark results per [BENCHMARK_UPDATE_PROCESS.md](BENCHMARK_UPDATE_PROCESS.md)
4. Review merged PRs since last release

### Semantic Versioning

- **PATCH** (`v1.6.0` → `v1.6.1`): Bug fixes, docs, small refactors without API changes
- **MINOR** (`v1.6.0` → `v1.7.0`): New features, optimizations, new public APIs (backward compatible)
- **MAJOR** (`v1.6.0` → `v2.0.0`): Breaking changes to public APIs (rare)

### GitHub PAT Setup (Required)

**REQUIRED:** Create `HOMEBREW_TAP_TOKEN` secret with `repo` scope to push to homebrew-oastools repository.

1. GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate token with `repo` scope
3. Add to repository: Settings → Secrets and variables → Actions
4. Name: `HOMEBREW_TAP_TOKEN`

Verify setup: `gh secret list --repo erraggy/oastools`

### GitHub Repository Settings

1. **Release immutability** - Currently DISABLED (not compatible with current workflow)
2. **Workflow permissions** - "Read and write permissions" required (Settings → Actions → General)
3. **Branch protection** - Can be enabled (applies to branches, not tags)

### Release Process

⚠️ **IMPORTANT**: Release immutability is DISABLED. Use direct published releases (not drafts).

**Why not use drafts?**
- Draft releases don't create/push tags
- No tag = no workflow trigger
- Publishing draft creates tag, but release is then immutable → 422 errors
- See [planning/release-issues.md](planning/release-issues.md) for full analysis

**Step 1:** Test locally (optional but recommended)
```bash
make release-test
```

**Step 2:** Create published GitHub Release
```bash
gh release create v1.7.1 \
  --title "v1.7.1 - Brief summary within 72 chars" \
  --notes "$(cat <<'EOF'
## Summary
High-level overview of what this release delivers.

## What's New
- Feature 1: Description
- Feature 2: Description

## Changes
- Change 1
- Change 2

## Related PRs
- #17 - PR title

## Installation
### Homebrew
```bash
brew tap erraggy/oastools
brew install oastools
```

### Binary Download
Download the appropriate binary for your platform from the assets below.
EOF
)"
```

This creates git tag, published release, and triggers GitHub Actions workflow immediately.

**Step 3:** Monitor automated build
- Workflow: https://github.com/erraggy/oastools/actions
- Release: https://github.com/erraggy/oastools/releases

**Step 4:** Verify release has assets
```bash
gh release view v1.7.1 --json assets
```

Confirm all platform binaries are attached (Darwin, Linux, Windows).

**Step 5:** Test installation
```bash
brew tap erraggy/oastools
brew install oastools
oastools --version
```

### Troubleshooting

- **GoReleaser can't push**: Verify `homebrew-oastools` repo exists, token has `repo` scope, commit author email verified
- **Build fails**: Review GitHub Actions logs, check CGO dependencies, test with `make release-test`
- **Formula doesn't work**: Verify formula in homebrew-oastools, test in clean environment

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
- Convenience: `parser.Parse()`, `ParseReader()`, `ParseBytes()`
- Struct-based: `parser.New()`, `Parser.Parse()`, `Parser.ParseReader()`, `Parser.ParseBytes()`
- `ParseResult.SourcePath` tracks source (`"ParseReader.yaml"` for readers, `"ParseBytes.yaml"` for bytes)

**Validator Package:**
- Convenience: `validator.Validate()`, `ValidateParsed()`
- Struct-based: `validator.New()`, `Validator.Validate()`, `Validator.ValidateParsed()`

**Joiner Package:**
- Convenience: `joiner.Join()`, `JoinParsed()`
- Struct-based: `joiner.New()`, `Joiner.Join()`, `Joiner.JoinParsed()`, `Joiner.WriteResult()`
- All input documents must be pre-validated (Errors slice must be empty)

**Converter Package:**
- Convenience: `converter.Convert()`, `ConvertParsed()`
- Struct-based: `converter.New()`, `Converter.Convert()`, `Converter.ConvertParsed()`
- Configuration: `StrictMode`, `IncludeInfo`
- Returns ConversionResult with severity-tracked issues (Info, Warning, Critical)

**Differ Package:**
- Convenience: `differ.Diff()`, `DiffParsed()`
- Struct-based: `differ.New()`, `Differ.Diff()`, `Differ.DiffParsed()`
- Returns DiffResult with changes categorized by severity (Critical, Error, Warning, Info)

### Usage Examples

**Quick operations:**
```go
// Parse
result, _ := parser.Parse("openapi.yaml", false, true)

// Validate
result, _ := validator.Validate("openapi.yaml", true, false)

// Join
result, _ := joiner.Join([]string{"base.yaml", "ext.yaml"}, config)

// Convert
result, _ := converter.Convert("swagger.yaml", "3.0.3")

// Diff
result, _ := differ.Diff("v1.yaml", "v2.yaml")
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
