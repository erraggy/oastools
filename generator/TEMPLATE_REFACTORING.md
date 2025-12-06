# Generator Package Template Refactoring

## Overview

This document outlines the refactoring of the `generator` package from string-based code generation (using `bytes.Buffer.WriteString()`) to Go's `text/template` package.

## Current Status

### Infrastructure ✅ Complete

The following infrastructure is in place and ready for use:

#### Files Created
- `templates.go` - Template embedding via `go:embed` and execution logic
- `template_data.go` - Data structures for template inputs
- `templates/` directory structure with template files:
  - `common/header.go.tmpl` - Package header and imports
  - `types/` - Type generation templates:
    - `struct.go.tmpl` - Struct type definitions
    - `enum.go.tmpl` - Enum types
    - `alias.go.tmpl` - Type aliases
    - `allof.go.tmpl` - AllOf composition
    - `oneof.go.tmpl` - OneOf union types
    - `unmarshal.go.tmpl` - Discriminator unmarshaling
  - `client/` - Client generation templates:
    - `client.go.tmpl` - Client struct, constructor, options
    - `method.go.tmpl` - Client methods
  - `server/` - Server generation templates:
    - `interface.go.tmpl` - ServerInterface and request types

### Current Implementation Status

#### oas3_generator.go
- Status: **Unchanged** - uses WriteString approach
- Methods to refactor:
  - `generateTypes()` (lines 33-108)
  - `generateSchemaType()` (lines 117-247)
  - `generateClient()` (lines 587-717)
  - `generateClientMethod()` (lines 722-850)
  - `generateServer()` (lines 998-1130)
  - `generateServerMethodSignature()` (lines 1133-1150)
  - `generateRequestType()` (lines 1153-1231)

#### oas2_generator.go
- Status: **Unchanged** - uses WriteString approach
- Similar methods to oas3_generator.go

## Refactoring Strategy

### Phase 1: Build Template Data Functions (In Progress)

Create builder functions that convert parser types to template data structures:

```go
// Example builder function
func buildStructData(typeName string, schema *parser.Schema) StructData {
    // Convert schema properties to field data
    // Handle comments, types, tags
    // Return structured data ready for templates
}
```

**Key Principle**: Move all generation logic to Go code. Templates only handle formatting.

### Phase 2: Update Generation Methods

Refactor existing generation methods to:
1. Collect necessary data from parser types
2. Build template data structures
3. Execute templates
4. Return formatted bytes

Example transformation:

**Before:**
```go
func (cg *oas3CodeGenerator) generateTypes() error {
    var buf bytes.Buffer
    buf.WriteString("// Code generated...\n")
    buf.WriteString(fmt.Sprintf("package %s\n", cg.result.PackageName))
    // 50+ more WriteString calls
    formatted, _ := format.Source(buf.Bytes())
}
```

**After:**
```go
func (cg *oas3CodeGenerator) generateTypes() error {
    data := cg.buildTypesTemplateData() // Build structured data
    formatted, err := executeTemplate("types.go.tmpl", data)
    // Handle result
}
```

### Phase 3: Incremental Refactoring

Refactor one method at a time:
1. `generateTypes()` - simpler, good starting point
2. `generateSchemaType()` - more complex logic
3. `generateClient()` - large method
4. `generateServer()` - large method

### Phase 4: Testing

All existing tests must pass without modification:
- `generator_test.go`
- `types_test.go`
- `client_test.go`
- `server_test.go`

Output should be byte-for-byte identical.

## Template Data Structures

### TypesFileData
```go
type TypesFileData struct {
    Header  HeaderData      // Package name and imports
    Types   []TypeDefinition // All type definitions
}
```

### ClientFileData
```go
type ClientFileData struct {
    Header            HeaderData          // Package name and imports
    DefaultUserAgent  string             // Default User-Agent string
    Methods           []ClientMethodData // Client methods
    ParamsStructs     []ParamsStructData // Query param structs
}
```

### ServerFileData
```go
type ServerFileData struct {
    Header        HeaderData          // Package name and imports
    Methods       []ServerMethodData  // Interface methods
    RequestTypes  []RequestTypeData   // Request structs
}
```

## Design Principles

### 1. Templates Are Dumb Formatters
- Templates only handle formatting and output layout
- All complex logic stays in Go code
- Data passed to templates should be fully resolved

### 2. Small, Composable Templates
- Each template is ~20-50 lines
- Easy to understand and modify
- Can be tested independently

### 3. Deterministic Output
- All collections are sorted before template execution
- No randomization or non-deterministic behavior
- Output is byte-for-byte reproducible

### 4. Backward Compatibility
- Refactoring is a drop-in replacement
- No API changes
- Existing tests pass without modification

## Next Steps

1. **Create builder functions** for each template data structure
2. **Refactor `generateTypes()`** as the first complete example
3. **Verify output** is identical to original
4. **Refactor remaining methods** incrementally
5. **Run full test suite** to ensure correctness
6. **Update documentation** with final architecture

## Files Involved

### Core Generation Files
- `/Users/robbie/code/oastools/generator/oas3_generator.go`
- `/Users/robbie/code/oastools/generator/oas2_generator.go`

### New Template Infrastructure
- `/Users/robbie/code/oastools/generator/templates.go`
- `/Users/robbie/code/oastools/generator/template_data.go`
- `/Users/robbie/code/oastools/generator/templates/`

### Tests
- `/Users/robbie/code/oastools/generator/generator_test.go`
- `/Users/robbie/code/oastools/generator/types_test.go`
- `/Users/robbie/code/oastools/generator/client_test.go`
- `/Users/robbie/code/oastools/generator/server_test.go`

## Maintenance Notes

When adding new features to code generation:
1. Add data to appropriate template data structure
2. Update corresponding template file
3. Add builder logic in generator methods
4. Test that output is as expected

Templates should NOT be modified to handle complex logic - extend the Go code instead.
