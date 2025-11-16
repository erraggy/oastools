# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`oastools` is a Go-based command-line tool for working with OpenAPI Specification (OAS) files. The primary goals are:
- Validating OpenAPI specification files
- Parsing and analyzing OAS documents
- Joining multiple OpenAPI specification documents

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
  - *Future:* **internal/testutil/** - Test fixtures and helpers for unit tests
  - *Future:* **internal/copyutil/** - Generic deep copy utilities

- **testdata/** - Test fixtures including sample OpenAPI specification files

- **doc.go** - Root package documentation for the oastools library

### Design Patterns

- **Public API**: All core packages (parser, validator, joiner, converter) are public and can be imported by external projects
- **Separation of concerns**: Each package has a single, well-defined responsibility
- **CLI structure**: Simple command dispatcher in main.go that delegates to library packages
- **Comprehensive documentation**: Each package includes doc.go for package-level documentation and example_test.go for godoc examples

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

**Before Submitting Code:**

1. Run `make test` to ensure all tests pass
2. Run `make test-coverage` to review coverage report
3. Verify that all new exported functionality has dedicated test cases
4. Check that test names clearly describe what they test

**Never submit a PR with:**
- Untested exported functions
- Untested exported methods
- Untested exported types or their fields
- Tests that only cover the "happy path" without error cases

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
