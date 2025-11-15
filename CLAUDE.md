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

- **testdata/** - Test fixtures including sample OpenAPI specification files

- **doc.go** - Root package documentation for the oastools library

### Design Patterns

- **Public API**: All core packages (parser, validator, joiner) are public and can be imported by external projects
- **Separation of concerns**: Each package has a single, well-defined responsibility
- **CLI structure**: Simple command dispatcher in main.go that delegates to library packages
- **Comprehensive documentation**: Each package includes doc.go for package-level documentation and example_test.go for godoc examples

### Extension Points

When adding new commands:
1. Add the command case to the switch statement in `cmd/oastools/main.go`
2. Create corresponding logic in the appropriate public package (parser, validator, or joiner)
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

// Struct methods
func TestParserParse(t *testing.T) { ... }
func TestValidatorValidate(t *testing.T) { ... }
func TestJoinerJoin(t *testing.T) { ... }
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

As of v1.3.0, all core packages are public and can be imported:

- `github.com/erraggy/oastools/parser` - Parse OpenAPI specifications
- `github.com/erraggy/oastools/validator` - Validate OpenAPI specifications
- `github.com/erraggy/oastools/joiner` - Join multiple OpenAPI specifications

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
