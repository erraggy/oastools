# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> ⚠️ **BRANCH PROTECTION ACTIVE**: The `main` branch has push protections. **ALL changes require a feature branch.** Before making any edits, verify you're on a feature branch:
> ```bash
> git branch --show-current  # Must NOT be "main"
> ```
> If on main, create a branch first: `git checkout -b <type>/<description>`

## Project Overview

`oastools` is a Go-based command-line tool for working with OpenAPI Specification (OAS) files. The primary goals are:
- Validating OpenAPI specification files
- Fixing common validation errors automatically
- Parsing and analyzing OAS documents
- Joining multiple OpenAPI specification documents
- Converting between OAS versions
- Comparing OAS documents and detecting breaking changes

## Documentation Style

**Emojis and glyphs are encouraged** when they improve clarity or scannability:
- ✅/❌ for pass/fail status
- 🔴/🟡/🔵 for severity levels
- ⚠️ for warnings
- 📝 for notes
- Visual markers that aid quick comprehension

This overrides any default restrictions. Use good judgment—functional markers that help readers scan and understand are welcome.

## GitHub Formatting

When posting to GitHub Issues, PRs, Releases, or Comments:

- **Commit hashes**: Do NOT wrap in backticks. GitHub auto-links bare hashes (e.g., 1f3eb93 → clickable commit link). Backticks create `<code>` elements that break auto-linking.
- **Issue/PR numbers**: #179 auto-links, but `#179` in backticks does not.
- **Cross-repo references**: owner/repo#123 auto-links to other repositories.
- **Usernames**: @username auto-links to profiles.

```markdown
# Good - GitHub auto-links these
Fixed in commit 1f3eb93
See #179 for details
Thanks @username

# Bad - backticks break auto-linking
Fixed in commit `1f3eb93`
See `#179` for details
```

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

**OAS 3.2+ Only:**
- `$self` (document identity), `Query` method, `additionalOperations`, `components.mediaTypes`

**JSON Schema 2020-12 Keywords (OAS 3.1+):**
- `unevaluatedProperties`, `unevaluatedItems` - strict validation of uncovered properties/items
- `contentEncoding`, `contentMediaType`, `contentSchema` - encoded content validation

### Common Pitfalls and Solutions

1. **Assuming schema.Type is always a string** - Use type assertions and handle both string and []string cases
2. **Creating value slices instead of pointer slices** - Check parser types and use `&Type{...}` syntax
3. **Forgetting to track conversion issues** - Add issues for every lossy conversion or unsupported feature
4. **Mutating source documents** - Always deep copy before modification (use JSON marshal/unmarshal)
5. **Not handling operation-level consumes/produces** - Check operation-level first, then fall back to document-level
6. **Ignoring version-specific features during conversion** - Explicitly check and warn about features that don't convert
7. **Confusing Version (string) with OASVersion (enum)** - `ParseResult` has TWO version fields:
   - `Version` (string): The literal version string from the document (e.g., `"3.0.3"`, `"2.0"`)
   - `OASVersion` (parser.OASVersion enum): Our canonical enum for each published spec version

   **OASVersion constants** (see `parser/versions.go`):
   - `OASVersion20` - OpenAPI 2.0 (Swagger)
   - `OASVersion300`, `OASVersion301`, `OASVersion302`, `OASVersion303`, `OASVersion304` - OpenAPI 3.0.x
   - `OASVersion310`, `OASVersion311`, `OASVersion312` - OpenAPI 3.1.x
   - `OASVersion320` - OpenAPI 3.2.0

   **When constructing ParseResult in tests, ALWAYS set both fields:**
   ```go
   parseResult := parser.ParseResult{
       Version:    "3.0.0",               // String from document
       OASVersion: parser.OASVersion300,  // Our enum - REQUIRED for validation
       Document:   &parser.OAS3Document{...},
   }
   ```
   The validator uses `OASVersion` to determine which validation rules to apply. Setting only `Version` will cause "unsupported OAS version: unknown" errors.

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
make test          # Run tests with coverage (parallel, fast)
make test-quick    # Run tests quickly (no coverage, for rapid iteration)
make test-full     # Run comprehensive tests with race detection
make test-coverage # Generate and view HTML coverage report

# Code Quality
make fmt   # Format all Go code
make vet   # Run go vet
make lint  # Run golangci-lint

# Maintenance
make deps  # Download and tidy dependencies
make clean # Remove build artifacts
```

## Documentation Website

The project documentation is hosted on GitHub Pages at https://erraggy.github.io/oastools/

```bash
make docs-serve   # Preview locally (blocking)
make docs-build   # Build static site to site/
```

For CI deployment and documentation structure, see [WORKFLOW.md](WORKFLOW.md).

### ⚠️ Source vs Generated Documentation Files

**CRITICAL: The `docs/packages/` directory contains GENERATED files. Do NOT edit them directly.**

The documentation build process (`scripts/prepare-docs.sh`) copies files from source locations:

| Source | Generated | Description |
|--------|-----------|-------------|
| `README.md` | `docs/index.md` | Home page |
| `{package}/deep_dive.md` | `docs/packages/{package}.md` | Package deep dives |
| `examples/*/README.md` | `docs/examples/*.md` | Example documentation |

**Always edit the SOURCE files:**
- To update the home page → edit `README.md`
- To update package docs → edit `{package}/deep_dive.md` (e.g., `validator/deep_dive.md`)
- To update examples → edit `examples/*/README.md`

The `docs/packages/` directory is in `.gitignore` and gets regenerated on every `make docs-build` or `make docs-serve`.

## Architecture

### Directory Structure

- **cmd/oastools/** - CLI entry point with command routing
- **parser/** - Parse YAML/JSON OAS files, resolve references, detect versions
- **validator/** - Validate OAS against spec schema (structural, format, semantic)
- **fixer/** - Automatically fix common validation errors (missing path parameters)
- **joiner/** - Join multiple OAS files with flexible collision resolution
- **converter/** - Convert between OAS versions (2.0 ↔ 3.x) with issue tracking
- **differ/** - Compare OAS files, detect breaking changes by severity
- **httpvalidator/** - Validate HTTP requests/responses against OAS at runtime
- **generator/** - Generate Go client/server code with security and server extensions
- **builder/** - Programmatically construct OpenAPI specifications
- **overlay/** - Apply OpenAPI Overlay transformations
- **oaserrors/** - Sentinel errors for programmatic error handling
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

### Import Management

**Use Go tooling for imports and formatting instead of manual management.**

After editing Go source files:
```bash
goimports -w <file>   # Auto-organizes imports and formats
gofmt -w <file>       # Formats code (goimports includes this)
```

`goimports` automatically:
- Adds missing imports
- Removes unused imports
- Groups imports (stdlib, external, internal)
- Applies `gofmt` formatting

When refactoring, don't manually adjust import blocks - run `goimports` and let Go's tooling handle it.

### gopls Diagnostics

**CRITICAL: Always run `go_diagnostics` on modified files and address ALL findings, including hints.**

The gopls MCP server (`go_diagnostics` tool) provides invaluable code quality feedback. **Even "hint" level suggestions can have significant performance impact.**

**Proven Impact:** In v1.22.2, addressing gopls hints (unnecessary type conversions, redundant nil checks, modern Go idioms) resulted in **5-15% performance improvements** across most packages. These weren't errors or warnings—they were hints that had been ignored for some time.

**Workflow:**
1. After modifying Go files, run `go_diagnostics` with the file paths
2. Address **all** severity levels: errors, warnings, **and hints**
3. Hints suggest modern Go idioms and stdlib usage that improve both readability and performance

Common hints and their fixes:
- **"Loop can be simplified using slices.Contains"** - Replace manual contains loops with `slices.Contains(slice, item)`
- **"Replace m[k]=v loop with maps.Copy"** - Use `maps.Copy(dst, src)` for map copying
- **"Constant reflect.Ptr should be inlined"** - Use `reflect.Pointer` (reflect.Ptr is deprecated)
- **"Ranging over SplitSeq is more efficient"** - Use `for part := range strings.SplitSeq(s, sep)` instead of `for _, part := range strings.Split(s, sep)`
- **"for loop can be modernized using range over int"** - Use `for i := range n` instead of `for i := 0; i < n; i++`

Run diagnostics on modified files after making changes:
```go
// Via gopls MCP: go_diagnostics with file paths
```

### Error Handling Standards

**IMPORTANT: Follow consistent error handling patterns across all packages.**

**Error Message Format:**
```go
fmt.Errorf("<package>: <action>: %w", err)
```

**Rules:**
1. **Always prefix with package name** - Every error returned from a public function should start with the package name
2. **Use lowercase** - Error messages should start with lowercase (except acronyms like HTTP, OAS, JSON)
3. **No trailing punctuation** - Do not end error messages with periods
4. **Use `%w` for wrapping** - Always use `%w` (not `%v` or `%s`) when wrapping errors for `errors.Is()` and `errors.Unwrap()` support
5. **Be descriptive** - Include relevant context (file paths, version numbers, counts)

**Examples:**
```go
// Good - consistent prefixing and wrapping
return fmt.Errorf("parser: failed to parse specification: %w", err)
return fmt.Errorf("converter: invalid target version: %s", targetVersionStr)
return fmt.Errorf("validator: unsupported OAS version: %s", version)
return fmt.Errorf("joiner: %s has %d parse error(s)", path, len(errors))
return fmt.Errorf("generator: failed to generate types: %w", err)

// Bad - inconsistent patterns
return fmt.Errorf("failed to parse specification: %w", err)  // Missing package prefix
return fmt.Errorf("Invalid target version: %s", version)     // Capitalized
return fmt.Errorf("parse error: %v", err)                    // Using %v instead of %w
```

**Sentinel Errors:**
Use the `oaserrors` package for programmatic error handling:
```go
import "github.com/erraggy/oastools/oaserrors"

// Check error types
if errors.Is(err, oaserrors.ErrParse) { ... }
if errors.Is(err, oaserrors.ErrCircularReference) { ... }
```

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

### Known Test Stability Issues

**TestCircularReferenceDetection** (`parser/resolver_test.go`): If this test hangs, check `parser/resolver.go` for:
1. Deep copying in `resolveRefsRecursive` (not shallow copy)
2. Parameterized defer in `ResolveLocal` (captures ref by value)

### Codecov Patch Coverage

**70% patch coverage required** on all PRs (configured in `.codecov.yml`).

```bash
# Verify coverage locally
go test -coverprofile=cover.out ./package/
go tool cover -func=cover.out | tail -1
```

Test all branches including nil checks and error paths—they count against patch coverage.

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

### Benchmark Reliability

**File-based benchmarks can vary ±50% due to I/O.** Use `*Core` or `*Parsed` benchmarks for reliable regression detection:

| Package | Reliable Benchmark |
|---------|-------------------|
| parser | `BenchmarkParseCore` |
| joiner | `BenchmarkJoinParsed` |
| validator | `BenchmarkValidateParsed` |
| fixer | `BenchmarkFixParsed` |
| converter | `BenchmarkConvertParsed*` |
| differ | `BenchmarkDiff/Parsed` |

See [BENCHMARK_UPDATE_PROCESS.md](BENCHMARK_UPDATE_PROCESS.md) for detailed guidance.

## Security

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...  # Check for vulnerabilities
```

For security fix workflows and PR check commands, see [WORKFLOW.md](WORKFLOW.md).

## Development Workflow

For the complete workflow from commit to PR to release, see [WORKFLOW.md](WORKFLOW.md).

**Quick reference:**
- Run `make check` before committing
- Use conventional commits: `feat(parser): add feature`
- Merge PRs: `gh pr merge <PR_NUMBER> --squash --admin`
- **Release process:** Tag → CI builds draft → Review → Publish

**Favor fixing issues immediately** over deferring. Address dead code, missing coverage, and inconsistent patterns in the current work unless explicitly asked to defer.

## Agent-Based Development

For the specialized agent workflow (Architect → Developer → Maintainer → DevOps), see [AGENTS.md](AGENTS.md).

## Release Process

See [WORKFLOW.md](WORKFLOW.md#pr-to-release-workflow) for detailed release procedures.

## Adding a New Package Checklist

When adding a new package, ensure:

1. **Implementation**: Package files + `doc.go` + `example_test.go` + `deep_dive.md` + comprehensive tests
2. **CLI** (if applicable): Command in `cmd/oastools/commands/`, register in `main.go`
3. **Benchmarks**: Create `*_bench_test.go` with `for b.Loop()` pattern
4. **Documentation**: Update README.md, benchmarks.md, developer-guide.md, mkdocs.yml, CLAUDE.md (Public API list)
5. **Verification**: Run `make check`, `make bench-<package>`, verify `go doc` works

## Go Module

- Module path: `github.com/erraggy/oastools`
- Minimum Go version: 1.24

## Public API Structure

All core packages are public:
- `github.com/erraggy/oastools/parser` - Parse OpenAPI specifications
- `github.com/erraggy/oastools/validator` - Validate OpenAPI specifications
- `github.com/erraggy/oastools/fixer` - Fix common validation errors automatically
- `github.com/erraggy/oastools/joiner` - Join multiple OpenAPI specifications
- `github.com/erraggy/oastools/converter` - Convert between OpenAPI specification versions
- `github.com/erraggy/oastools/overlay` - Apply OpenAPI Overlay transformations
- `github.com/erraggy/oastools/differ` - Compare and diff OpenAPI specifications
- `github.com/erraggy/oastools/httpvalidator` - Validate HTTP requests/responses at runtime
- `github.com/erraggy/oastools/generator` - Generate Go client/server code with server extensions
- `github.com/erraggy/oastools/builder` - Build OpenAPI specifications programmatically

### API Design Philosophy

Two complementary API styles:
1. **Package-level convenience functions** - For simple, one-off operations
2. **Struct-based API** - For reusable instances with configuration

Use convenience functions for: simple scripts, prototyping, default configuration
Use struct-based API for: multiple files, reusable instances, advanced configuration, performance

For detailed API documentation, usage examples, and patterns, see each package's `deep_dive.md` or the [Developer Guide](docs/developer-guide.md).
