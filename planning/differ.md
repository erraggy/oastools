# Differ Package Implementation Plan

## Overview
Implement a new `differ` package and `diff` command to compare OpenAPI specifications with two operational modes:
1. **Simple mode**: Report semantic differences between documents
2. **Breaking changes mode**: Categorize changes and identify breaking API changes

## Architecture

### Package Structure
- **differ/** - Public package for OAS comparison
  - `differ.go` - Core API (Differ struct, Diff/DiffParsed methods)
  - `simple.go` - Simple semantic diff implementation
  - `breaking.go` - Breaking change detection logic
  - `doc.go` - Package documentation
  - `example_test.go` - Runnable examples
  - `differ_test.go` - Comprehensive tests
  - `differ_bench_test.go` - Benchmark tests

### API Design

**Public Types:**
```go
type Differ struct {
    Mode DiffMode
    IncludeInfo bool
    UserAgent string
}

type DiffMode int
const (
    ModeSimple DiffMode = iota      // Report semantic differences
    ModeBreaking                     // Detect breaking changes
)

type DiffResult struct {
    SourceVersion string
    SourceOASVersion parser.OASVersion
    TargetVersion string
    TargetOASVersion parser.OASVersion
    Changes []Change
    BreakingCount int
    WarningCount int
    InfoCount int
    HasBreakingChanges bool
}

type Change struct {
    Path string
    Type ChangeType
    Category ChangeCategory
    Severity severity.Severity
    OldValue any
    NewValue any
    Message string
}

type ChangeType string
const (
    ChangeTypeAdded ChangeType = "added"
    ChangeTypeRemoved = "removed"
    ChangeTypeModified = "modified"
)

type ChangeCategory string
const (
    CategoryEndpoint ChangeCategory = "endpoint"
    CategoryOperation = "operation"
    CategoryParameter = "parameter"
    CategoryRequestBody = "request_body"
    CategoryResponse = "response"
    CategorySchema = "schema"
    CategorySecurity = "security"
    CategoryServer = "server"
    CategoryInfo = "info"
)
```

**Public Functions:**
- `New()` - Create Differ with defaults
- `Diff(sourcePath, targetPath)` - Compare two files
- `DiffParsed(source, target)` - Compare parsed results

### Change Detection Logic

**Breaking Changes (Severity: Critical/Error):**
- Removing endpoints, operations, or required parameters
- Changing parameter types or constraints (stricter)
- Removing response status codes (success codes)
- Making optional parameters required
- Changing authentication/security requirements
- Removing properties from request/response schemas
- Changing property types incompatibly

**Warnings:**
- Adding required parameters/properties
- Deprecating operations
- Changing descriptions significantly
- Modifying example values
- Changing server URLs

**Info:**
- Adding new endpoints/operations
- Adding optional parameters
- Relaxing constraints
- Documentation updates

## Breaking Change Detection Categories

Based on research from oasdiff and OpenAPI best practices:

### Endpoint Changes
- **Breaking**: Remove endpoint, remove HTTP method
- **Info**: Add endpoint, add HTTP method

### Parameter Changes
- **Breaking**: Remove required parameter, change parameter type, change parameter location, make optional parameter required, add enum constraint, reduce max length, increase min length
- **Warning**: Change parameter name, change parameter description
- **Info**: Add optional parameter, remove enum constraint, increase max length, decrease min length

### Request Body Changes
- **Breaking**: Remove required property, change property type, make optional property required, add new required property, change content-type
- **Warning**: Change property description
- **Info**: Add optional property, relax constraints

### Response Changes
- **Breaking**: Remove success status code (2xx), change response type, remove required property from response
- **Warning**: Add new error status code, change response description
- **Info**: Add optional property to response, add new success status code

### Schema Changes
- **Breaking**: Remove required property, change property type, change from oneOf/anyOf to specific type, add required property
- **Warning**: Change property format
- **Info**: Add optional property, relax validation rules

### Security Changes
- **Breaking**: Add new required security scheme, remove security scheme, change security type
- **Warning**: Change security scopes
- **Info**: Add optional security scheme

### Server Changes
- **Warning**: Change server URL, remove server
- **Info**: Add server

## CLI Integration

Add `diff` command to `cmd/oastools/main.go`:
```bash
# Simple mode - show all semantic differences
oastools diff <source> <target>

# Breaking changes mode - categorize and identify breaking changes
oastools diff <source> <target> --breaking

# Include informational changes
oastools diff <source> <target> --breaking --include-info
```

Output format:
- Text output with symbols: ✗ (breaking), ⚠ (warning), ℹ (info)
- Grouped by category (endpoints, operations, parameters, etc.)
- Summary at the end with counts

## Testing Strategy
- Unit tests for each change detection category
- Integration tests with real OAS examples
- Test both OAS 2.0 and 3.x comparisons
- Test cross-version comparisons (2.0 vs 3.x)
- Test edge cases (empty specs, identical specs, completely different specs)
- Benchmark performance with large specs

## Implementation Steps
1. Create differ package structure and core types
2. Implement simple mode (semantic diff)
3. Implement breaking change detection logic
4. Add comprehensive tests for all change categories
5. Integrate CLI command
6. Add documentation (doc.go) and examples (example_test.go)
7. Update root documentation and README
8. Run make check to ensure quality
