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
- **OAS 3.0.0**: https://spec.openapis.org/oas/v3.0.0.html
- **OAS 3.0.1**: https://spec.openapis.org/oas/v3.0.1.html
- **OAS 3.0.2**: https://spec.openapis.org/oas/v3.0.2.html
- **OAS 3.0.3**: https://spec.openapis.org/oas/v3.0.3.html
- **OAS 3.0.4**: https://spec.openapis.org/oas/v3.0.4.html
- **OAS 3.1.0**: https://spec.openapis.org/oas/v3.1.0.html
- **OAS 3.1.1**: https://spec.openapis.org/oas/v3.1.1.html
- **OAS 3.1.2**: https://spec.openapis.org/oas/v3.1.2.html
- **OAS 3.2.0**: https://spec.openapis.org/oas/v3.2.0.html

All OAS versions utilize the **JSON Schema Specification Draft 2020-12** for schema definitions:
- https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html

## Key OpenAPI Specification Concepts

Understanding the evolution and differences between OAS versions is critical when working with this codebase. This section documents key concepts learned during implementation.

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
Several OAS 3.1+ fields use `interface{}` to support multiple types:
- `schema.Type`: Can be `string` (e.g., `"string"`) or `[]string` (e.g., `["string", "null"]`)
- Always use type assertions when accessing these fields:
  ```go
  if typeStr, ok := schema.Type.(string); ok {
      // Handle string type
  } else if typeArr, ok := schema.Type.([]string); ok {
      // Handle array type
  }
  ```

**Pointer vs Value Types:**
- `OAS3Document.Servers`: Uses `[]*parser.Server` (slice of pointers), not `[]parser.Server`
- When creating server objects, always use `&parser.Server{...}` for pointer semantics
- This pattern applies to other nested structures to avoid unexpected mutations

### Structural Differences Between Versions

**OAS 2.0 Document Structure:**
```yaml
swagger: "2.0"
info: {...}
host: api.example.com
basePath: /v1
schemes: [https]
consumes: [application/json]
produces: [application/json]
paths: {...}
definitions: {...}
parameters: {...}
responses: {...}
securityDefinitions: {...}
```

**OAS 3.x Document Structure:**
```yaml
openapi: 3.0.3
info: {...}
servers:
  - url: https://api.example.com/v1
paths: {...}
components:
  schemas: {...}
  parameters: {...}
  responses: {...}
  securitySchemes: {...}
```

### Version-Specific Features

**OAS 2.0 Only:**
- `allowEmptyValue`: Removed in OAS 3.0+
- `collectionFormat`: Replaced by `style` and `explode` in OAS 3.0+
- Single `host`/`basePath`/`schemes`: Replaced by flexible `servers` array

**OAS 3.0+ Only:**
- `requestBody`: Replaces body parameters and consumes
- `callbacks`: Asynchronous callback definitions
- `links`: Relationships between operations
- Cookie parameters (`in: cookie`)
- `servers` array with variable substitution

**OAS 3.1+ Only:**
- `webhooks`: Event-driven API definitions
- JSON Schema 2020-12 alignment
- `type` as array for nullable types
- `license.identifier` field

**OAS 3.x Method Support:**
- TRACE method is OAS 3.x only (not in OAS 2.0)

### Conversion Challenges and Solutions

**Multiple Servers → Single Host/BasePath:**
- OAS 3.x supports multiple servers; OAS 2.0 supports only one host/basePath/schemes combination
- Solution: Use first server and warn about others
- Parse server URL to extract host, basePath, and scheme components

**Multiple Media Types:**
- OAS 3.x: Each request/response can have multiple media types in `content` object
- OAS 2.0: Single schema with `consumes`/`produces` arrays
- Solution: Extract first media type's schema and collect all media types in consumes/produces

**Security Scheme Conversion:**
- OAS 3.x HTTP schemes (bearer, basic) → OAS 2.0 basic auth only
- OAS 3.x OAuth2 flows (multiple) → OAS 2.0 single flow
- OpenID Connect (OAS 3.x+) → No equivalent in OAS 2.0 (critical issue)

**Parameter Serialization:**
- OAS 2.0 `collectionFormat` → OAS 3.x `style` and `explode`
- No perfect mapping; requires context-specific warnings

### Best Practices for OAS Document Manipulation

**Deep Copying Documents:**
```go
// Use JSON marshal/unmarshal for deep copy to avoid mutations
data, _ := json.Marshal(srcDocument)
var dstDocument parser.OAS3Document
json.Unmarshal(data, &dstDocument)
// Restore fields that don't round-trip through JSON
dstDocument.OASVersion = srcDocument.OASVersion
```

**Type Assertions:**
```go
// Always check interface{} fields before using
if typeStr, ok := schema.Type.(string); ok {
    converted.Type = typeStr
}
```

**Issue Tracking:**
- Use severity levels: Info (informational), Warning (lossy), Critical (data loss)
- Provide context with each issue to help users understand impact
- Track JSON path for precise issue location (e.g., `paths./pets.get.parameters[0]`)

**Version Detection:**
```go
// Use parser.ParseVersion for robust version detection
version, ok := parser.ParseVersion("3.0.3")
if !ok {
    // Handle invalid version
}
```

### Common Pitfalls and Solutions

**Pitfall 1: Assuming schema.Type is always a string**
- Solution: Use type assertions and handle both string and []string cases

**Pitfall 2: Creating value slices instead of pointer slices**
- Solution: Check parser types (e.g., `[]*parser.Server`) and use `&Type{...}` syntax

**Pitfall 3: Forgetting to track conversion issues**
- Solution: Add issues for every lossy conversion or unsupported feature

**Pitfall 4: Mutating source documents**
- Solution: Always deep copy before modification

**Pitfall 5: Not handling operation-level consumes/produces**
- Solution: Check operation-level first, then fall back to document-level

**Pitfall 6: Ignoring version-specific features during conversion**
- Solution: Explicitly check and warn about features that don't convert (webhooks, callbacks, links, etc.)

## Development Commands

### Recommended Workflow

After making changes to Go source files, run:
```bash
make check
```
This will run all quality checks (tidy, fmt, lint, test) and show git status to address all issues at once.

### Building and Running
```bash
# Build the binary (output: bin/oastools)
make build

# Install to $GOPATH/bin
make install

# Run the binary directly
./bin/oastools [command]
```

### Testing
```bash
# Run all tests with race detection and coverage
# Note: If gotestsum is installed, it will be used automatically for better output formatting
make test

# Generate and view HTML coverage report
make test-coverage
```

### Code Quality
```bash
# Format all Go code
make fmt

# Run go vet
make vet

# Run golangci-lint (requires golangci-lint to be installed)
make lint
```

### Dependency Management
```bash
# Download and tidy dependencies
make deps
```

### Cleanup
```bash
# Remove build artifacts and coverage reports
make clean
```

## Architecture

### Directory Structure

- **cmd/oastools/** - CLI entry point with command routing and user interface
  - `main.go` contains the command dispatcher and usage information

- **parser/** - Public parsing library for OpenAPI specifications
  - Logic for parsing YAML/JSON OAS files into Go structures
  - External reference resolution and version detection
  - Package documentation in `doc.go` and examples in `example_test.go`

- **validator/** - Public validation library for OpenAPI specifications
  - Logic for validating OpenAPI specifications against the spec schema
  - Structural, format, and semantic validation
  - Package documentation in `doc.go` and examples in `example_test.go`

- **joiner/** - Public joining library for OpenAPI specifications
  - Logic for joining multiple OpenAPI specification files
  - Flexible collision resolution strategies
  - Package documentation in `doc.go` and examples in `example_test.go`

- **converter/** - Public conversion library for OpenAPI specifications
  - Logic for converting between OAS versions (2.0 ↔ 3.x)
  - Best-effort conversion with transparent issue tracking
  - Package documentation in `doc.go` and examples in `example_test.go`

- **differ/** - Public diffing library for OpenAPI specifications
  - Logic for comparing OpenAPI specification files
  - Simple semantic diff and breaking change detection
  - Categorizes changes by severity (Critical, Error, Warning, Info)
  - Package documentation in `doc.go` and examples in `example_test.go`

- **internal/** - Internal packages with shared utilities (not part of public API)
  - **internal/httputil/** - HTTP-related validation constants and utilities
    - HTTP status code validation and RFC 9110 standards
    - HTTP method constants and media type validation
  - **internal/severity/** - Severity level type for issue reporting
    - Unified severity levels across validator and converter
    - SeverityError, SeverityWarning, SeverityInfo, SeverityCritical
  - **internal/issues/** - Unified issue type for validation and conversion errors
    - Consolidated Issue struct with all necessary fields
    - Supports both validation (SpecRef) and conversion (Context) use cases
  - **internal/testutil/** - Test fixtures and helpers for unit tests
  - *Future:* **internal/copyutil/** - Generic deep copy utilities

- **testdata/** - Test fixtures including sample OpenAPI specification files

- **doc.go** - Root package documentation for the oastools library

### Design Patterns

- **Public API**: All core packages (parser, validator, joiner, converter) are public and can be imported by external projects
- **Separation of concerns**: Each package has a single, well-defined responsibility
- **CLI structure**: Simple command dispatcher in main.go that delegates to library packages
- **Comprehensive documentation**: Each package includes doc.go for package-level documentation and example_test.go for godoc examples
- **Format preservation**: Input file format (JSON or YAML) is automatically detected and preserved throughout the conversion and joining pipelines

### Format Preservation

**IMPORTANT: The parser, converter, and joiner automatically preserve the input file format (JSON or YAML).**

When converting or joining OpenAPI specifications, oastools maintains format consistency:

**Implementation:**

1. **Parser** (`parser/parser.go`):
   - Adds `SourceFormat` field to `ParseResult` to track the detected format
   - Detects format from file extension (`.json`, `.yaml`, `.yml`)
   - For readers/bytes, detects format from content (JSON starts with `{` or `[`)
   - Format detection functions: `detectFormatFromPath()`, `detectFormatFromContent()`

2. **Converter** (`converter/converter.go`):
   - Adds `SourceFormat` field to `ConversionResult` copied from `ParseResult`
   - CLI (`cmd/oastools/main.go`) uses `result.SourceFormat` to choose marshaler:
     - `parser.SourceFormatJSON` → `json.MarshalIndent()` with 2-space indentation
     - `parser.SourceFormatYAML` → `yaml.Marshal()`

3. **Joiner** (`joiner/joiner.go`):
   - Adds `SourceFormat` field to `JoinResult` copied from first document's `ParseResult`
   - `WriteResult()` method uses `result.SourceFormat` to choose marshaler:
     - `parser.SourceFormatJSON` → `marshalJSON()` helper (uses `json.MarshalIndent()`)
     - `parser.SourceFormatYAML` → `yaml.Marshal()`

**Test Coverage:**

- `converter/converter_test.go`: `TestJSONFormatPreservation`, `TestYAMLFormatPreservation`
- `joiner/joiner_test.go`: `TestJSONFormatPreservation`, `TestYAMLFormatPreservation`
- Test files: `testdata/minimal-oas2.json`, `testdata/minimal-oas3.json`

**Format Detection Logic:**

```go
// File extension detection
func detectFormatFromPath(path string) SourceFormat {
    ext := filepath.Ext(path)
    switch ext {
    case ".json":
        return SourceFormatJSON
    case ".yaml", ".yml":
        return SourceFormatYAML
    default:
        return SourceFormatUnknown
    }
}

// Content-based detection (for readers/bytes)
func detectFormatFromContent(data []byte) SourceFormat {
    // Trim leading whitespace, check first character
    // JSON objects/arrays start with '{' or '['
    // Otherwise assume YAML
}
```

**Key Design Decisions:**

1. **First file wins for joiner**: When joining multiple files, the format of the first file determines the output format
2. **Default to YAML on unknown**: If format cannot be determined, default to YAML (most common for OpenAPI specs)
3. **Consistent indentation**: JSON output uses 2-space indentation to match common formatting standards

### Constant Usage

**IMPORTANT: Use package-level constants instead of string literals to maintain consistency and enable single-point-of-change updates.**

When constants exist for frequently-used values, always use them instead of embedding string literals throughout the code:

**HTTP Methods:**
- Use `internal/httputil` constants for HTTP methods: `httputil.MethodGet`, `httputil.MethodPost`, etc.
- These are defined in lowercase (`"get"`, `"post"`, etc.) to match OpenAPI specification usage
- Example: `parser.GetOAS2Operations()` uses `httputil.MethodGet` instead of `"get"`

**HTTP Status Codes:**
- Use validation functions `httputil.ValidateStatusCode()` for code validation
- Use `httputil.StandardHTTPStatusCodes` map for checking RFC 9110 standard codes

**Severity Levels:**
- Use `severity.SeverityError`, `severity.SeverityWarning`, etc. from `internal/severity` package
- Don't hardcode severity values in individual packages

**Benefits of this approach:**
1. **Single source of truth**: Changes to a value only need to be made in one place
2. **Type safety**: Reduces risk of typos in string literals
3. **Maintainability**: Clear intent and easier to find all usages
4. **Consistency**: Ensures the same value is used everywhere

When adding new utilities or extracting duplicated code, always expose the constant values through package-level exports rather than hiding them inside functions.

### Extension Points

When adding new commands:
1. Add the command case to the switch statement in `cmd/oastools/main.go`
2. Create corresponding logic in the appropriate public package (parser, validator, joiner, or converter)
3. Update the `printUsage()` function to document the new command
4. Add test files in the same package as the implementation
5. Update package documentation in `doc.go` if adding new public APIs
6. Add examples to `example_test.go` for new functionality

When adding new public APIs:
1. Ensure all exported types and functions have godoc comments
2. Update the package-level `doc.go` with usage examples
3. Add runnable examples to `example_test.go`
4. Update the root `doc.go` if the change affects the overall library usage

### Testing Strategy

- Unit tests live alongside implementation files (e.g., `validator.go` → `validator_test.go`)
- Integration tests should use fixtures from `testdata/`
- Run tests with race detection enabled to catch concurrency issues
- Aim for high test coverage, especially for validation, parsing, and joining logic

### Test Coverage Requirements

**CRITICAL: All exported functionality MUST have comprehensive test coverage.**

When adding or modifying exported functionality, you MUST include test coverage for:

1. **Exported Functions** - All package-level functions and methods
   - Test all exported convenience functions (e.g., `parser.Parse()`, `validator.Validate()`, `joiner.Join()`)
   - Test all struct methods (e.g., `Parser.Parse()`, `Validator.ValidateParsed()`, `Joiner.JoinParsed()`)
   - Include both success and error cases
   - Test with various input combinations and edge cases

2. **Exported Types** - All public structs, interfaces, and type aliases
   - Test struct initialization and default values
   - Test all exported fields and their behavior
   - Test type conversions and assertions

3. **Exported Constants and Variables**
   - Test that constants have expected values
   - Test exported variables and their initialization

**Test Coverage Guidelines:**

- **Positive Cases**: Test that functionality works correctly with valid inputs
- **Negative Cases**: Test error handling with invalid inputs, missing files, malformed data
- **Edge Cases**: Test boundary conditions, empty inputs, nil values, large inputs
- **Integration**: Test how components work together (e.g., parse then validate, parse then join)
- **Documentation**: Use descriptive test names that clearly explain what is being tested

**Example Test Naming Pattern:**
```go
// Package-level convenience functions
func TestParseConvenience(t *testing.T) { ... }
func TestValidateConvenience(t *testing.T) { ... }
func TestJoinConvenience(t *testing.T) { ... }
func TestConvertConvenience(t *testing.T) { ... }

// Struct methods
func TestParserParse(t *testing.T) { ... }
func TestValidatorValidate(t *testing.T) { ... }
func TestJoinerJoin(t *testing.T) { ... }
func TestConverterConvert(t *testing.T) { ... }
```

### Benchmark Test Requirements

**CRITICAL: Use the Go 1.24+ `for b.Loop()` pattern for all benchmarks.**

When writing benchmark tests, you MUST follow these requirements:

1. **Use `for b.Loop()` Pattern** (Go 1.24+)
   - **ALWAYS** use `for b.Loop()` instead of `for i := 0; i < b.N; i++`
   - **DO NOT** call `b.ReportAllocs()` manually (handled automatically by `b.Loop()`)
   - This is the modern, consistent pattern used across all packages

   **Correct Pattern**:
   ```go
   func BenchmarkDiffParsed(b *testing.B) {
       // Setup (parsing, creating instances, etc.)
       source, _ := parser.Parse("file.yaml", false, true)
       target, _ := parser.Parse("file2.yaml", false, true)

       for b.Loop() {  // ✅ Correct - modern Go 1.24+ pattern
           _, err := DiffParsed(*source, *target)
           if err != nil {
               b.Fatal(err)
           }
       }
   }
   ```

   **Incorrect Pattern**:
   ```go
   func BenchmarkDiffParsed(b *testing.B) {
       source, _ := parser.Parse("file.yaml", false, true)
       target, _ := parser.Parse("file2.yaml", false, true)

       b.ReportAllocs()  // ❌ Wrong - redundant with b.Loop()
       for i := 0; i < b.N; i++ {  // ❌ Wrong - old pattern
           _, err := DiffParsed(*source, *target)
           if err != nil {
               b.Fatal(err)
           }
       }
   }
   ```

2. **No Manual Timer Resets**
   - **DO NOT** call `b.ResetTimer()` for trivial setup (e.g., creating a Differ instance)
   - `b.Loop()` handles timing automatically
   - Only include expensive setup (like parsing) outside the loop

3. **Package-Level Comment**
   - Add this comment at the top of each benchmark file:
   ```go
   // Note on b.Fatalf usage in benchmarks:
   // Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
   // These operations (parse, diff, validate, etc.) should never fail with valid test fixtures.
   // If they do fail, it indicates a bug that should halt the benchmark immediately.
   ```

4. **Consistency Across Packages**
   - All benchmark files (`parser`, `validator`, `converter`, `joiner`, `differ`) use the same pattern
   - This ensures maintainability and consistency with Go evolution

**Why This Matters**:
- **Consistency**: All existing benchmarks use `for b.Loop()`
- **Best Practice**: Modern Go 1.24+ approach
- **Automatic Handling**: `b.Loop()` manages allocations reporting and timing
- **Future Compatibility**: Aligns with Go's benchmark evolution

## Submitting changes

**Before Submitting Code:**

1. Run `make check` to ensure all code is formatted and all lints/tests pass
2. Run `make test-coverage` to review coverage report
3. Verify that all new exported functionality has dedicated test cases
4. Check that test names clearly describe what they test

**Never submit a PR with:**
- Untested exported functions
- Untested exported methods
- Untested exported types or their fields
- Tests that only cover the "happy path" without error cases

**When changes are ready for commit:**
- Always use a conventional commit message for the first line and keep it within 72 characters
- The rest of the commit message should be simply formatted with maximum width (columns) of 100, and should summarize just the basic reasoning why and the changes included.
- Following the commit, and after the human has pushed it to origin, be prepared to create a PR using `gh` that uses the same first line of the commit message as the PR's title, and well formatted markdown content that lays out the reasoning as well as the changes made and any useful context. Here is where the details are to go.
- The actual commit will be a much briefer version of the messaging used to cover the changes

### Retrieving PR Comments and Review Feedback

**Getting comments from Pull Requests:**

To retrieve all comments from a PR (including automated workflow comments from claude-review, github-actions, etc.):

```bash
gh pr view <PR_NUMBER> --comments
```

**Important notes:**
- This command displays **all comments** on the PR in chronological order
- Comments from github-actions bot (including claude-review workflow) appear with `author: github-actions[bot]`
- The output includes the full comment text, making it easy to read review feedback
- This is the **reliable method** for getting PR comments - other approaches (like using `gh api`) may not work consistently

**Example usage:**
```bash
# Get all comments from PR #25
gh pr view 25 --comments

# The output will show each comment with metadata:
# author: github-actions[bot]
# association: none
# edited: false
# status: none
# --
# [Full comment text here]
# --
```

**When to use this:**
- After creating a PR and waiting for automated review workflows (claude-review typically takes 2+ minutes)
- To check feedback before merging a PR
- To retrieve specific review recommendations or issues that need addressing
- To understand what changes are needed based on automated code review

### Creating a new release

**Release Infrastructure:**

This repository uses **GoReleaser** with **GitHub Actions** to automate binary builds and Homebrew publishing.
The recommended workflow is:

1. Create a GitHub Release with detailed notes using `gh release create`
2. GoReleaser automatically triggers, adds binaries to your release, and publishes to Homebrew

This approach maintains full control over release notes while automating the build and distribution process.

**Prerequisites:**

1. Ensure you are on the `main` branch
2. Ensure your local `main` is up-to-date with `origin/main`
3. Verify all tests pass: `make check`
4. Update benchmark results: Follow the process in [BENCHMARK_UPDATE_PROCESS.md](BENCHMARK_UPDATE_PROCESS.md)
5. Review merged PRs since last release to understand changes

**Semantic Versioning (SemVer) Rules:**

The version tag follows the format `vMAJOR.MINOR.PATCH` (e.g., `v1.7.0`):

- **PATCH** (increment 3rd number): Bug fixes, documentation updates, minor changes without new features
  - Example: `v1.6.0` → `v1.6.1`
  - Use when: Fixing bugs, updating docs, small refactors with no API changes

- **MINOR** (increment 2nd number): New features, larger implementation changes, performance improvements
  - Example: `v1.6.0` → `v1.7.0`
  - Use when: Adding new functionality, significant optimizations, new public APIs (backward compatible)

- **MAJOR** (increment 1st number): Breaking changes to public APIs
  - Example: `v1.6.0` → `v2.0.0`
  - Use when: Removing/changing public APIs, incompatible changes
  - Note: Major version bumps require careful planning and are extremely rare

**GitHub Personal Access Token (PAT) Setup:**

**REQUIRED:** The automated release workflow needs a Personal Access Token to push to the Homebrew tap repository.

The default `GITHUB_TOKEN` provided by GitHub Actions only has permissions for the current repository and
**cannot** push to the separate `homebrew-oastools` repository. You must create a PAT with `repo` scope.

**For Automated Releases (GitHub Actions) - REQUIRED SETUP:**

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Set a descriptive name (e.g., "GoReleaser - oastools Homebrew Publishing")
4. Select scopes: **`repo`** (full control of private repositories)
5. Click "Generate token" and **copy the token immediately** (you won't be able to see it again)
6. Go to the oastools repository → Settings → Secrets and variables → Actions
7. Click "New repository secret"
8. Name: `HOMEBREW_TAP_TOKEN`
9. Value: Paste the token you copied
10. Click "Add secret"

The `.github/workflows/release.yml` file is already configured to use `${{ secrets.HOMEBREW_TAP_TOKEN }}`.

**Verify PAT Setup:**

Before creating your first release, verify the PAT secret is configured:
```bash
gh secret list --repo erraggy/oastools
```

You should see `HOMEBREW_TAP_TOKEN` in the list.

**For Local Releases:**

If you prefer to run releases locally (not recommended), you'll need to use the same PAT:

1. Create a PAT as described above (or use the same one)
2. Export it in your shell:
   ```bash
   export GITHUB_TOKEN="ghp_your_token_here"
   ```
3. Use the `make release-test` target to test locally before running a real release

**IMPORTANT:** Never commit the PAT to git. Keep it secure in GitHub Secrets or your local environment only.

**Release Make Targets:**

The following make targets are available to help with testing:

```bash
# Test the GoReleaser configuration locally (no publishing)
make release-test

# Clean up local release artifacts
make release-clean
```

**Pre-Release Checklist:**

Before creating your first release, ensure:
- [ ] `HOMEBREW_TAP_TOKEN` secret is created and added to repository secrets
- [ ] Secret verified with: `gh secret list --repo erraggy/oastools`
- [ ] Local test successful: `make release-test`
- [ ] Commit author email in `.goreleaser.yaml` matches a verified email in your GitHub account
- [ ] All changes committed and pushed to `main`
- [ ] All tests pass: `make check`

**Release Process (Recommended Workflow):**

This workflow creates the GitHub Release first with detailed notes, then GoReleaser adds binaries and publishes to Homebrew.

1. **Determine the version number** based on the changes included (see SemVer rules above)

2. **Test the release configuration locally** (optional but recommended):
   ```bash
   make release-test
   ```
   This runs GoReleaser in snapshot mode to verify everything builds correctly without publishing.

3. **Create the GitHub Release with detailed notes**:
   ```bash
   gh release create v1.7.1 \
     --title "v1.7.1 - Brief summary within 72 chars" \
     --notes "$(cat <<'EOF'
   ## Summary

   High-level overview of what this release delivers.

   ## What's New

   - Feature 1: Description
   - Feature 2: Description
   - Performance: Improvements achieved

   ## Changes

   - Change 1
   - Change 2

   ## Technical Details

   Additional context, benchmark results, migration notes, etc.

   ## Related PRs

   - #17 - PR title
   - #18 - PR title

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

   **Important:** The `gh release create` command automatically:
   - Creates the version tag (e.g., `v1.7.1`)
   - Creates the GitHub Release with your detailed notes
   - Triggers the GitHub Actions workflow

4. **Monitor the automated build**:
   The GitHub Actions workflow will automatically:
   - Build binaries for all platforms (Linux, macOS, Windows)
   - Add binary archives to your GitHub Release
   - Publish the Homebrew formula to `homebrew-oastools`

   Monitor progress at:
   - Workflow: https://github.com/erraggy/oastools/actions
   - Release: https://github.com/erraggy/oastools/releases
   - Homebrew tap: https://github.com/erraggy/homebrew-oastools

5. **Verify the release**:
   - Check that binaries were added to the GitHub Release
   - Verify Homebrew formula was updated in `homebrew-oastools`
   - Test Homebrew installation (optional):
     ```bash
     brew tap erraggy/oastools
     brew install oastools
     oastools --version
     ```

**After Release:**

- Verify the release appears correctly on GitHub
- Test Homebrew installation: `brew tap erraggy/oastools && brew install oastools`
- Announce the release (if applicable)
- Update project documentation if needed

**Troubleshooting:**

**Issue: GoReleaser can't push to homebrew-oastools**
- Verify the `homebrew-oastools` repository exists and is public
- Check that the GitHub token has `repo` scope
- Ensure the commit author email in `.goreleaser.yaml` matches a verified email in your GitHub account

**Issue: Build fails for certain platforms**
- Review the GitHub Actions logs for specific error messages
- Check if any dependencies require CGO (we've set `CGO_ENABLED=0`)
- Test locally with `make release-test` to reproduce the issue

**Issue: Homebrew formula doesn't work**
- Verify the formula in `homebrew-oastools` repository
- Test installation in a clean environment
- Check binary architecture matches the user's system

## Go Module

- Module path: `github.com/erraggy/oastools`
- Minimum Go version: 1.24

## Public API Structure

As of v1.3.0, all core packages are public and can be imported. Planned for v1.5.0, the converter package will be added:

- `github.com/erraggy/oastools/parser` - Parse OpenAPI specifications
- `github.com/erraggy/oastools/validator` - Validate OpenAPI specifications
- `github.com/erraggy/oastools/joiner` - Join multiple OpenAPI specifications
- `github.com/erraggy/oastools/converter` - Convert between OpenAPI specification versions (planned for v1.5.0)

Each package includes:
- `doc.go` - Comprehensive package-level documentation
- `example_test.go` - Runnable examples for godoc
- Full godoc comments on all exported types and functions

### API Design Philosophy

The oastools library provides **two complementary API styles**:

1. **Package-level convenience functions** - For simple, one-off operations
2. **Struct-based API** - For reusable instances with configuration

**When to use convenience functions:**
- Simple scripts or one-time operations
- Prototyping and quick testing
- Code examples and documentation
- Default configuration is sufficient

**When to use struct-based API:**
- Processing multiple files with the same configuration
- Need to reuse the same parser/validator/joiner instance
- Advanced configuration requirements
- Performance-critical scenarios where instance reuse matters

### Key API Features

**Parser Package:**

Package-level convenience functions:
- `parser.Parse(specPath, resolveRefs, validateStructure)` - Parse a file with options
- `parser.ParseReader(r, resolveRefs, validateStructure)` - Parse from io.Reader
- `parser.ParseBytes(data, resolveRefs, validateStructure)` - Parse from bytes

Struct-based API:
- `parser.New()` - Create a Parser instance with default settings
- `Parser.Parse(specPath)` - Parse a file using instance configuration
- `Parser.ParseReader(r)` - Parse from io.Reader using instance configuration
- `Parser.ParseBytes(data)` - Parse from bytes using instance configuration

Notes:
- `parser.ParseResult` includes a `SourcePath` field that tracks the document's source:
  - For `Parse(path)`: contains the actual file path
  - For `ParseReader(r)`: set to `"ParseReader.yaml"`
  - For `ParseBytes(data)`: set to `"ParseBytes.yaml"`
- ParseResult is treated as immutable after creation

**Validator Package:**

Package-level convenience functions:
- `validator.Validate(specPath, includeWarnings, strictMode)` - Validate a file with options
- `validator.ValidateParsed(parseResult, includeWarnings, strictMode)` - Validate an already-parsed result

Struct-based API:
- `validator.New()` - Create a Validator instance with default settings
- `Validator.Validate(specPath)` - Parse and validate a file
- `Validator.ValidateParsed(parseResult)` - Validate an already-parsed ParseResult
  - Useful when you need to parse once and validate multiple times
  - Enables efficient workflows when combining parser with validator

**Joiner Package:**

Package-level convenience functions:
- `joiner.Join(specPaths, config)` - Join files with configuration
- `joiner.JoinParsed(parsedDocs, config)` - Join already-parsed documents

Struct-based API:
- `joiner.New(config)` - Create a Joiner instance with configuration
- `Joiner.Join(specPaths)` - Parse and join multiple files
- `Joiner.JoinParsed(parsedDocs)` - Join already-parsed ParseResult documents
  - Efficient when documents are already parsed
  - Enables advanced workflows where parsing and joining are separated
  - All input documents must be pre-validated (Errors slice must be empty)
- `Joiner.WriteResult(result, outputPath)` - Write joined result to file

**Converter Package (planned for v1.5.0):**

Package-level convenience functions:
- `converter.Convert(specPath, targetVersion)` - Convert a file to target OAS version
- `converter.ConvertParsed(parseResult, targetVersion)` - Convert an already-parsed result

Struct-based API:
- `converter.New()` - Create a Converter instance with default settings
- `Converter.Convert(specPath, targetVersion)` - Parse and convert a file
- `Converter.ConvertParsed(parseResult, targetVersion)` - Convert an already-parsed ParseResult
  - Efficient when document is already parsed
  - Enables workflows where parsing and conversion are separated
  - Returns ConversionResult with severity-tracked issues (Info, Warning, Critical)

Configuration:
- `StrictMode bool` - Fail on any issues (even warnings)
- `IncludeInfo bool` - Include informational messages in results

### Usage Examples

**Quick parsing with convenience function:**
```go
result, err := parser.Parse("openapi.yaml", false, true)
if err != nil {
    log.Fatal(err)
}
```

**Reusable parser instance:**
```go
p := parser.New()
p.ResolveRefs = false
p.ValidateStructure = true

result1, _ := p.Parse("api1.yaml")
result2, _ := p.Parse("api2.yaml")
result3, _ := p.Parse("api3.yaml")
```

**Quick validation with convenience function:**
```go
result, err := validator.Validate("openapi.yaml", true, false)
if err != nil {
    log.Fatal(err)
}
if !result.Valid {
    // Handle errors
}
```

**Reusable validator instance:**
```go
v := validator.New()
v.IncludeWarnings = true
v.StrictMode = false

result1, _ := v.Validate("api1.yaml")
result2, _ := v.Validate("api2.yaml")
```

**Quick join with convenience function:**
```go
config := joiner.DefaultConfig()
config.PathStrategy = joiner.StrategyAcceptLeft

result, err := joiner.Join([]string{"base.yaml", "ext.yaml"}, config)
if err != nil {
    log.Fatal(err)
}
```

**Reusable joiner instance:**
```go
config := joiner.DefaultConfig()
config.SchemaStrategy = joiner.StrategyAcceptLeft

j := joiner.New(config)
result1, _ := j.Join([]string{"api1-base.yaml", "api1-ext.yaml"})
result2, _ := j.Join([]string{"api2-base.yaml", "api2-ext.yaml"})
```

**Quick conversion with convenience function:**
```go
result, err := converter.Convert("swagger.yaml", "3.0.3")
if err != nil {
    log.Fatal(err)
}
if result.HasCriticalIssues() {
    fmt.Printf("Conversion completed with %d critical issue(s)\n", result.CriticalCount)
}
```

**Reusable converter instance:**
```go
c := converter.New()
c.StrictMode = false
c.IncludeInfo = true

result1, _ := c.Convert("swagger-v1.yaml", "3.0.3")
result2, _ := c.Convert("swagger-v2.yaml", "3.0.3")
```