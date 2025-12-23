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

### Local Development

```bash
# Preview documentation locally (blocking - Ctrl+C to stop)
make docs-serve

# Or run in background
make docs-start   # Start server at http://127.0.0.1:8000/oastools/
make docs-stop    # Stop background server

# Build static site (outputs to site/)
make docs-build

# Clean generated documentation artifacts
make docs-clean
```

### CI Deployment

Documentation is automatically deployed on every push to `main` via the `.github/workflows/docs.yml` workflow:

1. **Trigger**: Push to `main` branch or manual workflow dispatch
2. **Build**: `scripts/prepare-docs.sh` prepares content, then MkDocs builds the site
3. **Deploy**: `mkdocs gh-deploy --force` pushes to `gh-pages` branch

### Documentation Structure

The documentation system uses [MkDocs](https://www.mkdocs.org/) with the [Material theme](https://squidfunk.github.io/mkdocs-material/).

**Source files:**
- `mkdocs.yml` - MkDocs configuration (navigation, theme, extensions)
- `docs/` - Static documentation files (developer-guide.md, cli-reference.md, etc.)
- `<package>/deep_dive.md` - Package-specific deep dive guides (copied to `docs/packages/`)
- `scripts/prepare-docs.sh` - Prepares documentation by copying and transforming files

**Generated files (gitignored):**
- `site/` - Built static site
- `docs/index.md` - Generated from README.md
- `docs/packages/*.md` - Copied from `<package>/deep_dive.md` files
- `.tmp/` - Temporary files for background server

### Deep Dive Files

Each feature-rich package should have a `deep_dive.md` file in its directory. These are automatically included in the documentation site:

```
parser/deep_dive.md      → docs/packages/parser.md
validator/deep_dive.md   → docs/packages/validator.md
converter/deep_dive.md   → docs/packages/converter.md
...
```

**Deep dive requirements:**
- Comprehensive API coverage with practical examples
- All functional options documented
- Common patterns and best practices
- Integration examples with other packages

### Adding Documentation

When adding new documentation:

1. **For new packages**: Create `<package>/deep_dive.md` - it will be auto-included
2. **For new guides**: Add to `docs/` and update `mkdocs.yml` navigation
3. **Preview locally**: Run `make docs-serve` before committing
4. **Verify links**: Check that internal links work after deployment

### Theme Customization

The `overrides/` directory contains customizations to the MkDocs Material theme using the official [theme extension](https://squidfunk.github.io/mkdocs-material/customization/#extending-the-theme) mechanism.

**Current customizations:**
- `overrides/main.html` - Adds TTL-based cache expiration (1 hour) for GitHub repository facts (stars, forks, version) which MkDocs Material caches indefinitely in sessionStorage

**How it works:**
- `custom_dir: overrides` in `mkdocs.yml` enables theme extension
- Files in `overrides/` mirror the theme structure and override/extend templates
- Template blocks can be overridden by extending `base.html`

**References:**
- [MkDocs Material Customization](https://squidfunk.github.io/mkdocs-material/customization/)
- [Theme Extension Guide](https://squidfunk.github.io/mkdocs-material/customization/#extending-the-theme)
- `overrides/README.md` - Detailed documentation of our customizations

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

**TestCircularReferenceDetection** (`parser/resolver_test.go`):
- This test has a history of timeout issues related to circular reference handling
- **Root causes (both fixed in the same commit):**
  1. **Circular Go pointer creation**: When resolving A -> B -> A circular refs, shallow copying created actual Go pointer chains that caused `yaml.Marshal` to infinite loop. Fixed by deep copying resolved content using `deepCopyJSONValue()`.
  2. **Closure bug in ResolveLocal**: The `ref` variable was modified after the defer was registered (`ref = strings.TrimPrefix(ref, "#")`), but Go closures capture by reference. The defer was clearing the wrong key in the visited map. Fixed by passing `ref` as a parameter to the defer function to capture by value: `defer func(rf string) { r.visited[rf] = false }(ref)`.
- If this test starts failing/hanging again:
  1. Check `parser/resolver.go` - `resolveRefsRecursive` function (deep copy at line ~422)
  2. Check `parser/resolver.go` - `ResolveLocal` function (parameterized defer at line ~92)
  3. Verify resolved content is being deep-copied, not shallow-copied
  4. Verify defer closures capture ref by value (pass as parameter), not by reference
  5. Test with: `go test -timeout=30s github.com/erraggy/oastools/parser -run "TestCircularReferenceDetection"`
- The `hasCircularRefs` flag in RefResolver must prevent yaml.Marshal from being called on circular structures

### Codecov Patch Coverage Requirements

**CRITICAL: Write tests to meet 70% patch coverage BEFORE creating PRs.**

The `.codecov.yml` configuration requires 70% patch coverage on all PRs. This is a blocking requirement—PRs will fail the codecov/patch check if new/modified code doesn't have adequate test coverage.

**Workflow requirement:**
1. Implement the feature or fix
2. Write comprehensive tests for all new code paths
3. Verify coverage locally before pushing
4. Create PR only after coverage requirements are met

The `.codecov.yml` configuration:
- **Project coverage**: auto target with 1% threshold (overall project)
- **Patch coverage**: 70% target with 5% threshold (new/modified lines only)

**When adding new code, ensure:**

1. **All branches are tested** - Functions with multiple `if` statements, nil checks, or `switch` cases need tests that exercise each branch. Defensive nil checks that are unlikely to be hit still count against patch coverage.

2. **Test all code paths** - If a function checks parameters, request bodies, responses, and default responses, add separate tests for each path to ensure coverage.

3. **Check coverage locally before pushing:**
   ```bash
   # Check coverage for a specific package
   go test -coverprofile=cover.out ./package/
   go tool cover -func=cover.out | grep "function_name"

   # For new packages, aim for 70%+ on each file
   go tool cover -func=cover.out | tail -1  # Shows total
   ```

4. **For functions with many nil-check branches**, consider adding targeted unit tests that construct scenarios to hit each branch, rather than relying solely on integration tests.

**Common patch coverage failures:**
- New helper functions with multiple conditional paths
- Error handling branches that require specific error conditions
- Nil checks for optional struct fields that are rarely nil in tests
- Functional options that aren't exercised in tests

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

### Benchmark Reliability and Performance Regression Detection

**CRITICAL: File-based benchmarks are NOT reliable for detecting performance regressions.**

#### The I/O Variance Problem

Investigation in v1.28.1 revealed that saved benchmark files showed apparent regressions of 51-82% compared to earlier versions. However, when running both versions back-to-back on the same machine:

| Benchmark | Saved v1.25.0 | Saved v1.28.1 | Live v1.25.0 | Live HEAD |
|-----------|---------------|---------------|--------------|-----------|
| Parse/SmallOAS3 | 143 µs | 217 µs | **176 µs** | **173 µs** |
| Join/TwoDocs | 103 µs | 188 µs | **185 µs** | **186 µs** |

**Conclusion:** The "regressions" were **not real code changes**—they were artifacts of I/O variance in saved benchmarks. File I/O can vary **+/- 50%** depending on:
- Filesystem caching state
- System load during benchmark capture
- Disk performance and fragmentation
- Background processes

#### Reliable Benchmarks for Regression Detection

**✅ RECOMMENDED: Use I/O-isolated benchmarks for detecting performance regressions:**

| Package | Reliable Benchmark | Description |
|---------|-------------------|-------------|
| parser | `BenchmarkParseCore` | Pre-loads all test files, benchmarks only parsing logic |
| joiner | `BenchmarkJoinParsed` | Pre-parses all documents, benchmarks only joining logic |
| validator | `BenchmarkValidateParsed` | Pre-parsed, benchmarks only validation |
| fixer | `BenchmarkFixParsed` | Pre-parsed, benchmarks only fixing |
| converter | `BenchmarkConvertParsed*` | Pre-parsed, benchmarks only conversion (OAS2ToOAS3, OAS3ToOAS2) |
| differ | `BenchmarkDiff/Parsed` | Pre-parsed sub-benchmark, benchmarks only diffing |

**❌ UNRELIABLE: File-based benchmarks are for informational purposes only:**
- `BenchmarkParse` - Includes file I/O variance
- `BenchmarkJoin` - Includes file I/O variance
- `BenchmarkValidate` - Includes file I/O variance
- `BenchmarkFix` - Includes file I/O variance
- `BenchmarkConvert` - Includes file I/O variance
- `BenchmarkDiff` - Includes file I/O variance

#### Detecting Real Performance Regressions

To check for actual performance regressions:

```bash
# 1. Run ONLY the I/O-isolated benchmarks
go test -bench='ParseCore|JoinParsed|ValidateParsed|FixParsed|ConvertParsed|Diff/Parsed' -benchmem ./...

# 2. Save results with a version tag
go test -bench='ParseCore|JoinParsed|ValidateParsed|FixParsed|ConvertParsed|Diff/Parsed' -benchmem ./... > benchmarks/benchmark-v1.X.Y-core.txt

# 3. Compare with previous version using benchstat
benchstat benchmarks/benchmark-v1.OLD.Y-core.txt benchmarks/benchmark-v1.X.Y-core.txt
```

**What counts as a regression:**
- A statistically significant slowdown (benchstat shows `+X%` with `p < 0.05`)
- Consistent across multiple runs
- Observed in I/O-isolated benchmarks

**What does NOT count as a regression:**
- File-based benchmark variance (even 50%+ changes may be noise)
- One-off measurements without statistical validation
- Changes only in `BenchmarkParse`, `BenchmarkJoin`, etc. (file I/O benchmarks)

#### Key Takeaway

**If you suspect a performance regression, always verify by:**
1. Running the specific benchmark multiple times
2. Using only `*Core`, `*Parsed`, or `*Bytes` benchmarks
3. Comparing with `benchstat` for statistical significance
4. Running both versions back-to-back on the same machine if needed

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

## Development Workflow

For detailed information on the development workflow from commit to pull request to release, see [WORKFLOW.md](WORKFLOW.md).

**CRITICAL: Always use feature branches. NEVER commit directly to main.**

Create a feature branch before making any changes:
```bash
git checkout main
git pull origin main
git checkout -b <type>/<description>  # e.g., feat/add-feature, fix/bug-name, chore/update-docs
```

**Quick reference:**
- **Before committing:** Run `make check`, verify test coverage, update benchmarks if needed
- **Commit format:** Conventional commits (e.g., `feat(parser): add feature`)
- **Local review:** `./scripts/local-code-review.sh branch`
- **Creating PR:** Use `gh pr create` with detailed description
- **Merging PR:** Use `gh pr merge <PR_NUMBER> --squash --admin` (branch protections require `--admin`; auto-delete branch is enabled)
- **Release process:** Tag → CI builds draft → Review → Publish

### Addressing Issues When Found

**Favor fixing issues immediately over deferring them.** When working on a feature or bug fix, if you discover related issues or technical debt:

1. **Add findings to the todo list** - Track what you discovered
2. **Address them in the current work** - Fix dead code, improve coverage, clean up patterns
3. **Only defer if explicitly requested** - The user can ask to defer, but the default is to fix now

This prevents accumulation of technical debt and ensures the codebase improves continuously. Examples of issues to address immediately:
- Dead code or unreachable branches
- Missing test coverage for new code paths
- Inconsistent patterns that should be unified
- Documentation that's out of sync with code

## Agent-Based Development Workflow

**When the user enters Plan Mode for a new feature or bug fix, use the specialized agent workflow:**

### Workflow Overview

1. **Architect Agent** → Plans and breaks down the work
2. **Developer Agents** → Implement the code (can run in parallel for independent packages)
3. **Maintainer Agent** → Reviews code quality, security, and consistency
4. **DevOps-Engineer Agent** → Handles benchmarks, CI/CD verification

### Step-by-Step Process

**Phase 1: Architecture & Planning**
```
1. User enters Plan Mode with feature/bug request
2. Spawn `architect` agent to:
   - Explore the codebase to understand scope
   - Design implementation approach
   - Create detailed specifications for each phase
   - Identify files to create/modify
   - Define test requirements
3. Write plan to the plan file for user approval
```

**Phase 2: Implementation**
```
1. After plan approval, spawn `developer` agents
2. Run independent phases IN PARALLEL when possible:
   - Different packages can be implemented concurrently
   - Dependent phases must be sequential
3. Each developer agent receives:
   - Specific files to create/modify
   - Detailed implementation requirements
   - Test coverage expectations
4. Track progress with TodoWrite tool
```

**Phase 3: Quality Assurance**
```
1. Spawn `maintainer` agent to review:
   - Code quality and consistency
   - Security vulnerabilities
   - Error handling patterns
   - Test coverage
2. Address any issues found
```

**Phase 4: Finalization**
```
1. Spawn `devops-engineer` agent for:
   - Benchmark updates (if performance-related)
   - CI/CD verification
   - Release preparation (if applicable)
2. Run `make check` to verify all tests pass
3. Create PR with comprehensive description
```

### Agent Capabilities

| Agent | Use For | Tools Available |
|-------|---------|-----------------|
| `architect` | Planning features, designing APIs, architectural decisions | Read, Grep, Glob, Bash |
| `developer` | Implementing code, writing tests, fixing bugs | Read, Edit, Write, Grep, Glob, Bash |
| `maintainer` | Code review, security audit, consistency checks | Read, Grep, Glob, Bash |
| `devops-engineer` | Releases, CI/CD, benchmarks, tooling | Read, Bash, Grep, Glob |

### Parallelization Guidelines

**Can run in parallel:**
- Implementation of independent packages (e.g., validator and converter)
- Test file creation alongside implementation
- Documentation updates

**Must run sequentially:**
- Core infrastructure before consuming packages
- Struct changes before methods using those structs
- Phase N before Phase N+1 when there are dependencies

### Example: Feature Implementation

```
User: "Add source map line tracking to all packages"

1. Architect creates plan:
   - Phase 1: Core types (parser/sourcemap.go)
   - Phase 2: Issue struct enhancement (internal/issues)
   - Phase 3: Validator integration
   - Phase 4: Other packages (parallel: converter, differ, fixer, generator)
   - Phase 5: CLI integration

2. Implementation:
   - Developer A: Phase 1 + Phase 2 (sequential - Phase 2 depends on 1)
   - Developer B: Phase 3 (after Phase 1+2 complete)
   - Developer C: Phase 4 packages (parallel, after Phase 1+2)
   - Developer D: Phase 5 (after Phase 3+4)

3. Review:
   - Maintainer reviews all changes
   - DevOps verifies benchmarks and CI
```

## Release Process

For detailed release procedures, see the **PR-to-Release Workflow** section in [WORKFLOW.md](WORKFLOW.md).

**Quick reference:**
1. Update benchmarks per [BENCHMARK_UPDATE_PROCESS.md](BENCHMARK_UPDATE_PROCESS.md)
2. Tag release: `git tag v1.X.Y && git push origin v1.X.Y`
3. Monitor workflow: `gh run watch <RUN_ID>`
4. Verify draft: `gh release view v1.X.Y`
5. Generate release notes (use Claude Code)
6. Publish: `gh release edit v1.X.Y --draft=false`

**Semantic versioning:**
- **PATCH**: Bug fixes, docs, small refactors
- **MINOR**: New features, APIs (backward compatible)
- **MAJOR**: Breaking API changes

For detailed release workflow, see [WORKFLOW.md](WORKFLOW.md).

## Adding a New Package/Feature Checklist

When adding a new package (like `fixer`, `generator`, etc.) or major feature, ensure these are updated:

### Required Updates

1. **Package Implementation**
   - Create package directory with implementation files
   - Add `doc.go` with package documentation
   - Add `example_test.go` with runnable godoc examples
   - Add `deep_dive.md` with comprehensive usage guide (for feature-rich packages)
   - Add comprehensive tests (`*_test.go`)

2. **CLI Integration** (if applicable)
   - Add command in `cmd/oastools/commands/`
   - Register in `cmd/oastools/main.go`
   - Add to CLI help text

3. **Benchmark Tests**
   - Create `*_bench_test.go` with benchmarks for:
     - File-based operations (parse + process)
     - Pre-parsed operations (`*Parsed` methods)
     - Functional options API
   - Follow Go 1.24+ `for b.Loop()` pattern

4. **Documentation Updates**
   - **README.md**:
     - Add to Features section
     - Add to Quick Start examples (CLI and library)
     - Add to Project Structure
     - Update package count in "All X main packages..."
     - Add to Integration Testing pipeline list
     - Add row to Document Processing Performance table
   - **benchmarks.md**:
     - Update package count in overview
     - Add new section in Benchmark Suite
     - Add performance metrics section with tables
     - Update "Running Benchmarks" command lists
     - Update Performance Best Practices
   - **BENCHMARK_UPDATE_PROCESS.md**:
     - Add to core packages list
     - Add benchmark command
     - Add section with formatting instructions
     - Update example table if applicable
   - **CLAUDE.md** (this file):
     - Add to Public API Structure package list
     - Add Key API Features section
   - **AGENTS.md**:
     - Add to Architecture section
   - **docs/developer-guide.md**:
     - Add to Table of Contents
     - Add CLI Usage section
     - Add Library Usage section with examples
     - Add deep dive cross-reference at end of package section
   - **Package deep_dive.md** (for feature-rich packages):
     - Table of contents with section links
     - Comprehensive API coverage with practical examples
     - All functional options documented
     - Common patterns and best practices
     - Integration examples with other packages
     - Back-to-top links for navigation in tall documents

5. **Makefile Updates**
   - Add to `.PHONY` targets list
   - Add `bench-<package>` target
   - Add to `bench` target package list
   - Add to `bench-save` command
   - Add to `bench-baseline` command

6. **Verification**
   - Run `make check` to verify all tests pass
   - Run `make bench-<package>` to verify benchmarks work
   - Verify `make lint` passes
   - Verify package is importable: `go doc github.com/erraggy/oastools/<package>`

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
- `github.com/erraggy/oastools/builder` - Build OpenAPI specifications programmatically

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
- Configuration: `BreakingRules` (BreakingRulesConfig) for customizing breaking change detection
- Presets: `DefaultRules()`, `StrictRules()`, `LenientRules()` for common rule configurations
- Options: `WithBreakingRules(rules)` for custom breaking change policies
- Returns DiffResult with changes categorized by severity (Critical, Error, Warning, Info)

**Fixer Package:**
- Functional options: `fixer.FixWithOptions(fixer.WithFilePath(...), fixer.WithInferTypes(...), ...)`
- Struct-based: `fixer.New()`, `Fixer.Fix()`, `Fixer.FixParsed()`
- Configuration: `InferTypes`, `EnabledFixes`, `GenericNamingConfig`, `DryRun`
- Fix types: `FixTypeMissingPathParameter`, `FixTypeRenamedGenericSchema`, `FixTypePrunedUnusedSchema`, `FixTypePrunedEmptyPath`
- **Default**: Only `FixTypeMissingPathParameter` enabled (expensive fixes are opt-in for performance)
- Generic naming: `GenericNamingUnderscore`, `GenericNamingOf`, `GenericNamingFor`, `GenericNamingFlattened`, `GenericNamingDot`
- Options: `WithGenericNaming(strategy)`, `WithGenericNamingConfig(config)`, `WithEnabledFixes(fixes...)`, `WithDryRun(bool)`
- Returns FixResult with list of applied fixes and fixed document

**HTTP Validator Package:**
- Functional options: `httpvalidator.ValidateRequestWithOptions(req, httpvalidator.WithFilePath(...), httpvalidator.WithStrictMode(...), ...)`
- Struct-based: `httpvalidator.New(parsed)`, `Validator.ValidateRequest(req)`, `Validator.ValidateResponseData(req, statusCode, headers, body)`
- Configuration: `StrictMode`, `IncludeWarnings`
- Request validation: Path params, query params, headers, cookies, request body
- Response validation: Status codes, headers, response body
- Parameter deserialization: All OAS serialization styles (simple, form, matrix, label, deepObject, spaceDelimited, pipeDelimited)
- Schema validation: Type checking, constraints (min/max, pattern, enum), composition (allOf/anyOf/oneOf)
- Returns ValidationResult with Valid flag, Errors, Warnings, and deserialized parameter maps (PathParams, QueryParams, HeaderParams, CookieParams)
- Middleware-friendly: `ValidateResponseData()` accepts captured response parts for middleware use

**Overlay Package:**
- Functional options: `overlay.ApplyWithOptions(overlay.WithSpecFilePath(...), overlay.WithOverlayFilePath(...), ...)`
- Struct-based: `overlay.NewApplier()`, `Applier.Apply()`, `Applier.ApplyParsed()`, `Applier.DryRun()`
- Configuration: `StrictTargets` (fail if any target matches nothing)
- Parsing: `overlay.ParseOverlay()`, `overlay.ParseOverlayFile()`, `overlay.Validate()`
- Preview: `overlay.DryRunWithOptions()` - preview changes without applying
- Returns ApplyResult with ActionsApplied, ActionsSkipped, Changes, Warnings
- JSONPath support: recursive descent (`$..field`), compound filters (`&&`, `||`)
- Integration: Joiner and Converter support pre/post overlays

**Builder Package:**
- Creates OAS documents programmatically with fluent API: `builder.New(version).SetTitle(...).AddOperation(...).BuildOAS3()`
- Supports OAS 2.0 (`BuildOAS2()`) and OAS 3.x (`BuildOAS3()`)
- Automatic schema generation from Go types via reflection
- Schema naming options: `WithSchemaNaming(strategy)`, `WithSchemaNameTemplate(tmpl)`, `WithSchemaNameFunc(fn)`
- Generic naming options: `WithGenericNaming(strategy)`, `WithGenericNamingConfig(config)`
- Built-in schema strategies: `SchemaNamingDefault`, `SchemaNamingPascalCase`, `SchemaNamingCamelCase`, `SchemaNamingSnakeCase`, `SchemaNamingKebabCase`, `SchemaNamingTypeOnly`, `SchemaNamingFullPath`
- Generic strategies: `GenericNamingUnderscore`, `GenericNamingOf`, `GenericNamingFor`, `GenericNamingAngleBrackets`, `GenericNamingFlattened`
- Template functions: `pascal`, `camel`, `snake`, `kebab`, `upper`, `lower`, `title`, `sanitize`, `trimPrefix`, `trimSuffix`, `replace`, `join`
- `RegisterTypeAs` always takes precedence over any naming strategy
- Vendor extensions: `WithOperationExtension`, `WithParamExtension`, `WithResponseExtension`, `WithRequestBodyExtension`
- OAS 2.0 options: `WithConsumes`, `WithProduces`, `WithParamAllowEmptyValue`, `WithParamCollectionFormat`
- Multi-content-type: `WithRequestBodyContentTypes`, `WithResponseContentTypes`

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

// Fix
result, _ := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
    fixer.WithInferTypes(true),
)

// Overlay
result, _ := overlay.ApplyWithOptions(
    overlay.WithSpecFilePath("openapi.yaml"),
    overlay.WithOverlayFilePath("production.yaml"),
)

// HTTP Validator (runtime request validation)
result, _ := httpvalidator.ValidateRequestWithOptions(
    req, // *http.Request
    httpvalidator.WithFilePath("openapi.yaml"),
    httpvalidator.WithStrictMode(true),
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

// Fixer with type inference
f := fixer.New()
f.InferTypes = true
result1, _ := f.Fix("api1.yaml")
result2, _ := f.Fix("api2.yaml")

// Overlay applier with strict mode
a := overlay.NewApplier()
a.StrictTargets = true
result1, _ := a.Apply("api1.yaml", "overlay1.yaml")
result2, _ := a.Apply("api2.yaml", "overlay2.yaml")

// HTTP Validator for multiple requests
parsed, _ := parser.ParseWithOptions(parser.WithFilePath("openapi.yaml"))
hv, _ := httpvalidator.New(parsed)
hv.StrictMode = true
result1, _ := hv.ValidateRequest(req1)
result2, _ := hv.ValidateRequest(req2)
```
