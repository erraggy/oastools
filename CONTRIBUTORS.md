# Contributing to oastools

Thank you for your interest in contributing to oastools! This document provides everything you need to know to contribute effectively.

## Table of Contents

- [Quick Start](#quick-start)
- [Development Workflow](#development-workflow)
- [Project Architecture](#project-architecture)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Submitting Changes](#submitting-changes)
- [CI/CD and Automation](#cicd-and-automation)
- [Getting Help](#getting-help)

## Quick Start

### Prerequisites

- **Go 1.24+** - Required for development
- **golangci-lint** - For linting (optional but recommended)
- **gotestsum** - For better test output formatting (optional)

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/erraggy/oastools.git
cd oastools

# Install dependencies
make deps

# Build the binary
make build

# Run tests
make test

# Run all quality checks
make check
```

## Development Workflow

### The Golden Rule: Always Run `make check`

After making changes, **always** run:

```bash
make check
```

This command runs:

1. `go mod tidy` - Clean up dependencies
2. `go fmt` - Format code
3. `golangci-lint run` - Lint code
4. `go test` with race detection - Run tests
5. `git status` - Show what changed

### Common Development Commands

```bash
# Build the binary (outputs to bin/oastools)
make build

# Install to $GOPATH/bin
make install

# Run tests with coverage
make test

# Generate HTML coverage report
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Clean build artifacts
make clean
```

### Development Loop

1. Make your changes
2. Run `make check` to validate
3. Fix any issues reported
4. Commit your changes
5. Create a pull request

## Project Architecture

### What is oastools?

oastools is a Go-based CLI tool and library for working with OpenAPI Specification (OAS) files. It provides:

- **Validation** - Ensure OAS files conform to specifications
- **Parsing** - Load and analyze OAS documents
- **Joining** - Combine multiple OAS files
- **Converting** - Transform between OAS versions (2.0 ↔ 3.x)
- **Diffing** - Compare specs and detect breaking changes

### Directory Structure

```
oastools/
├── cmd/oastools/       # CLI entry point
│   └── main.go         # Command dispatcher
├── parser/             # Parse OAS files (public API)
├── validator/          # Validate OAS files (public API)
├── joiner/             # Join multiple OAS files (public API)
├── converter/          # Convert between OAS versions (public API)
├── differ/             # Compare OAS files (public API)
├── internal/           # Internal utilities (not public API)
│   ├── httputil/       # HTTP constants and validation
│   ├── severity/       # Issue severity levels
│   ├── issues/         # Unified issue reporting
│   └── testutil/       # Test helpers
└── testdata/           # Test fixtures
```

### Public vs Internal Packages

**Public packages** (can be imported by external projects):

- `parser` - Parse OpenAPI specifications
- `validator` - Validate OpenAPI specifications
- `joiner` - Join multiple specifications
- `converter` - Convert between versions
- `differ` - Compare specifications

**Internal packages** (project-only):

- `internal/*` - Shared utilities not exposed to external users

### Design Principles

1. **Public API First** - Core functionality is exposed as importable Go packages
2. **Separation of Concerns** - Each package has one responsibility
3. **Format Preservation** - Input format (JSON/YAML) is automatically preserved
4. **Comprehensive Documentation** - Every public package has `doc.go` and `example_test.go`
5. **Testability** - High test coverage required for all exported functionality

## Code Standards

### Go Style Guidelines

- Follow standard Go conventions
- Use `gofmt` for formatting (run via `make fmt`)
- Pass `golangci-lint` checks (run via `make lint`)
- Use meaningful variable and function names
- Write self-documenting code; add comments only where logic isn't obvious

### Documentation Requirements

All exported functionality **must** have:

1. **Godoc comments** - Describe what it does
2. **Package-level docs** - Update `doc.go` if adding new public APIs
3. **Runnable examples** - Add to `example_test.go` for new features

Example:

```go
// Parse parses an OpenAPI specification file from the given path.
// It automatically detects the file format (JSON or YAML) and validates
// the document structure if validateStructure is true.
func Parse(specPath string, resolveRefs bool, validateStructure bool) (*ParseResult, error) {
    // Implementation
}
```

### Constant Usage

**Always use package-level constants instead of string literals:**

```go
// ❌ Bad - hardcoded strings
if method == "get" { ... }

// ✅ Good - use constants
if method == httputil.MethodGet { ... }
```

This ensures:

- Single source of truth
- Type safety
- Easy refactoring
- Clear intent

### Format Preservation

**IMPORTANT**: The parser, converter, and joiner automatically preserve input file format.

- Input JSON → Output JSON
- Input YAML → Output YAML

This is handled automatically via the `SourceFormat` field in `ParseResult`. When writing new features:

1. **Don't** manually choose output format
2. **Do** use `result.SourceFormat` to determine marshaling
3. **Test** both JSON and YAML format preservation

### Error Handling

- Return errors; don't panic (except for programmer errors)
- Use `fmt.Errorf` with `%w` for error wrapping
- Provide context in error messages

```go
if err != nil {
    return nil, fmt.Errorf("failed to parse spec at %s: %w", specPath, err)
}
```

## Testing Requirements

### Coverage Expectations

**All exported functionality MUST have comprehensive test coverage.**

This includes:

- ✅ Exported functions (e.g., `parser.Parse()`)
- ✅ Exported methods (e.g., `Parser.Parse()`)
- ✅ Exported types and their fields
- ✅ Exported constants and variables

### Test Types Required

1. **Positive tests** - Valid inputs produce expected outputs
2. **Negative tests** - Invalid inputs produce appropriate errors
3. **Edge cases** - Boundary conditions, empty inputs, nil values
4. **Integration tests** - Multiple components working together

### Test Naming Convention

```go
// Package-level convenience functions
func TestParseConvenience(t *testing.T) { ... }

// Struct methods
func TestParserParse(t *testing.T) { ... }

// Specific features
func TestJSONFormatPreservation(t *testing.T) { ... }
```

### Benchmark Tests

**Use the Go 1.24+ `for b.Loop()` pattern:**

```go
func BenchmarkParse(b *testing.B) {
    // Setup
    specPath := "testdata/petstore.yaml"

    // Benchmark loop
    for b.Loop() {
        _, err := Parse(specPath, false, true)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

**Don't use:**

- ❌ `for i := 0; i < b.N; i++` (old pattern)
- ❌ `b.ReportAllocs()` (handled automatically by `b.Loop()`)

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./parser/...

# Run specific test
go test ./parser -run TestParse

# Run benchmarks
go test -bench=. ./parser
```

## Submitting Changes

### Before You Commit

1. ✅ Run `make check` and ensure it passes
2. ✅ Add tests for new functionality
3. ✅ Update documentation (godoc, doc.go, example_test.go)
4. ✅ Verify test coverage is sufficient
5. ✅ Update benchmarks with `make bench-save` for performance-impacting changes

### Commit Message Format

Use conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**

- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only
- `test` - Adding/updating tests
- `refactor` - Code restructuring (no behavior change)
- `perf` - Performance improvements
- `chore` - Maintenance tasks

**Examples:**

```
feat(parser): add support for OAS 3.2.0

Implemented parsing logic for the new OAS 3.2.0 specification,
including support for the updated JSON Schema Draft 2020-12
alignment and new spec features.

- Added version detection for 3.2.0
- Updated schema validation
- Added test fixtures for 3.2.0
```

```
fix(converter): handle nullable types in OAS 3.1 conversion

Fixed conversion of OAS 3.1 nullable types that use type arrays
instead of the deprecated nullable field.

Fixes #123
```

```
chore: run go mod tidy

[skip-review] - automated dependency cleanup
```

### Skipping Automated Code Review

You can skip the Claude Code Review workflow for trivial commits by adding `[skip-review]` to your commit message:

```bash
# Example: Skip review for automated formatting
git commit -m "chore: run go fmt

[skip-review] - automated code formatting, no logic changes"

# Example: Skip review for dependency updates
git commit -m "chore: update dependencies

[skip-review] - go mod tidy only"
```

**When to use `[skip-review]`:**

- Automated formatting (`go fmt`, `gofmt`)
- Dependency updates (`go mod tidy`)
- Minor documentation typos
- Whitespace or comment-only changes

**When NOT to use `[skip-review]`:**

- Any logic changes
- New features
- Bug fixes
- Refactoring
- Test additions/changes

The review will be skipped if **any** commit in your PR contains `[skip-review]`.

### Pull Request Process

1. **Create a feature branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes and commit**

   ```bash
   # Make changes
   make check
   git add .
   git commit -m "feat(scope): description"
   ```

3. **Push to your fork**

   ```bash
   git push origin feature/your-feature-name
   ```

4. **Create a Pull Request**
   - Use a clear, descriptive title
   - Reference any related issues
   - Describe what changed and why
   - Include testing instructions if applicable

5. **Address Review Feedback**
   - Respond to comments
   - Make requested changes
   - Push updates to your branch
   - Request re-review when ready

### PR Review Checklist

Before requesting review, ensure:

- [ ] `make check` passes
- [ ] All tests pass with `make test`
- [ ] New functionality has tests
- [ ] Public APIs have godoc comments
- [ ] Examples added for new features
- [ ] Benchmarks updated with `make bench-save` (if changes affect performance)
- [ ] No unintended files committed (e.g., binaries, editor files)
- [ ] Commit messages follow conventional format

## CI/CD and Automation

### Automated Workflows

When you create a PR, several automated workflows run:

1. **Go Tests** - Runs test suite across multiple Go versions
2. **golangci-lint** - Lints code for issues
3. **Claude Code Review** (optional) - AI-powered code review
   - Skipped if `[skip-review]` in any commit message
   - Provides feedback on code quality, bugs, performance, security

### Workflow Status

Check workflow status:

- In your PR - See status checks at the bottom
- On the Actions tab - https://github.com/erraggy/oastools/actions

### Common CI Issues

**Tests fail on CI but pass locally:**

- Ensure you're testing with race detection: `go test -race`
- Check for timing-dependent tests
- Verify all test files are committed

**Exit code 143 (SIGTERM):**

- This means the test process was killed by the runner
- Common with `go test -race` on GitHub Actions
- Usually indicates tests hung or timed out
- Current mitigations in place:
  - Test timeout: 10 minutes per package
  - Job timeout: 15 minutes total
  - Limited parallelism: `-parallel=4`
  - `GOMAXPROCS=2` to prevent resource exhaustion
- If you see this error intermittently, it's likely a runner resource issue, not your code
- Related: [actions/runner-images#6680](https://github.com/actions/runner-images/issues/6680), [actions/runner-images#7146](https://github.com/actions/runner-images/issues/7146)

**Linter fails:**

- Run `make lint` locally
- Fix reported issues
- Push fixes

**Claude Code Review comments:**

- Review the feedback (visible in PR comments)
- Address legitimate concerns
- Respond to questions
- Push updates if needed

## Key OpenAPI Concepts

### Supported OAS Versions

oastools supports all major OpenAPI Specification versions:

- **OAS 2.0** (Swagger) - [Specification](https://spec.openapis.org/oas/v2.0.html)
- **OAS 3.0.x** (3.0.0 - 3.0.4) - [Specification](https://spec.openapis.org/oas/v3.0.3.html)
- **OAS 3.1.x** (3.1.0 - 3.1.2) - [Specification](https://spec.openapis.org/oas/v3.1.0.html)
- **OAS 3.2.0** - [Specification](https://spec.openapis.org/oas/v3.2.0.html)

All versions use **JSON Schema Draft 2020-12** for schema definitions.

### Version Evolution

Understanding how OAS evolved helps when working with conversion and validation:

**OAS 2.0 → 3.0 Changes:**

- `host`/`basePath`/`schemes` → unified `servers` array
- `definitions`/`parameters`/`responses` → `components.*`
- `consumes` + body param → `requestBody.content`
- `produces` + schema → `responses.*.content`
- Added: callbacks, links, cookie parameters

**OAS 3.0 → 3.1 Changes:**

- Full JSON Schema alignment
- `type` can be array: `["string", "null"]`
- Deprecated `nullable` field
- Added `webhooks` for event-driven APIs
- Added `license.identifier`

### Critical Type Handling

**Be careful with `interface{}` fields:**

Some OAS 3.1+ fields accept multiple types and are defined as `interface{}`:

```go
// schema.Type can be string OR []string
if typeStr, ok := schema.Type.(string); ok {
    // Single type: "string"
} else if typeArr, ok := schema.Type.([]string); ok {
    // Multiple types: ["string", "null"]
}
```

**Always use type assertions - never assume the type!**

### Version-Specific Features

**OAS 2.0 Only:**

- `allowEmptyValue` (removed in 3.0+)
- `collectionFormat` (replaced by `style`/`explode`)

**OAS 3.0+ Only:**

- `requestBody` (replaces body parameters)
- `callbacks` (async operations)
- `links` (operation relationships)
- Cookie parameters (`in: cookie`)
- TRACE HTTP method

**OAS 3.1+ Only:**

- `webhooks` (event subscriptions)
- Type arrays for nullable
- `license.identifier`

When working with conversions, these differences matter!

## Getting Help

### Resources

- **Documentation**: See [CLAUDE.md](https://github.com/erraggy/oastools/blob/main/CLAUDE.md) for technical details
- **Release Process**: See [RELEASES.md](https://github.com/erraggy/oastools/blob/main/RELEASES.md) for release workflow
- **Issues**: [GitHub Issues](https://github.com/erraggy/oastools/issues)
- **Pull Requests**: [GitHub PRs](https://github.com/erraggy/oastools/pulls)

### OpenAPI Specifications

- [OAS 2.0 Specification](https://spec.openapis.org/oas/v2.0.html)
- [OAS 3.0 Specification](https://spec.openapis.org/oas/v3.0.3.html)
- [OAS 3.1 Specification](https://spec.openapis.org/oas/v3.1.0.html)
- [JSON Schema Draft 2020-12](https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html)

### Asking Questions

- Open a [GitHub Issue](https://github.com/erraggy/oastools/issues/new) for bugs or feature requests
- Start a [GitHub Discussion](https://github.com/erraggy/oastools/discussions) for questions
- Check existing issues/discussions first - your question may already be answered

### Before Filing an Issue

1. Search existing issues
2. Provide a minimal reproduction case
3. Include relevant version information:

   ```bash
   oastools --version
   go version
   ```

4. Include error messages and stack traces
5. Describe expected vs actual behavior

## Common Pitfalls

### Type Assertions

**Problem**: Assuming `schema.Type` is always a string

```go
// ❌ Wrong - will panic if Type is []string
typeStr := schema.Type.(string)

// ✅ Correct - safe type assertion
if typeStr, ok := schema.Type.(string); ok {
    // Handle string case
} else if typeArr, ok := schema.Type.([]string); ok {
    // Handle array case
}
```

### Pointer Slices

**Problem**: Creating value slices instead of pointer slices

```go
// ❌ Wrong - OAS3Document.Servers expects []*parser.Server
servers := []parser.Server{{URL: "http://api.example.com"}}

// ✅ Correct - use pointer slice
servers := []*parser.Server{{URL: "http://api.example.com"}}
```

### Document Mutation

**Problem**: Modifying source documents unintentionally

```go
// ❌ Wrong - may mutate original
modified := sourceDoc
modified.Info.Title = "New Title"  // Changes sourceDoc too!

// ✅ Correct - deep copy first
modified := sourceDoc.DeepCopy()
modified.Info.Title = "New Title"  // Safe
```

### Hardcoded Strings

**Problem**: Using string literals instead of constants

```go
// ❌ Wrong
if method == "get" { ... }

// ✅ Correct
if method == httputil.MethodGet { ... }
```

### Missing Tests

**Problem**: Not testing exported functionality

```go
// ❌ Wrong - exported but no tests
func NewParser() *Parser { ... }

// ✅ Correct - exported with tests
func TestNewParser(t *testing.T) { ... }
```

## License

By contributing to oastools, you agree that your contributions will be licensed under the MIT License.

---

**Happy Contributing!** 🎉

We appreciate your time and effort in helping make oastools better. If you have questions or need clarification on anything in this guide, please don't hesitate to ask.
