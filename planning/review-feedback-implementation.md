# Review Feedback Implementation Plan

**Branch:** `refactor/review-feedback`
**Created:** November 26, 2025
**Based on:** Full code review in `planning/full-review.md`

## Overview

This document outlines the implementation plan to address high-priority feedback from the comprehensive code review. We focus on three main areas:

1. **CLI Argument Parsing** - Migrate to Go stdlib `flag` package
2. **JSON Marshaling Code Duplication** - Reduce boilerplate in `*_json.go` files
3. **Documentation Gaps** - Address minor documentation improvements

## Priority 1: CLI Argument Parsing Refactor

### Current State

**Problem:** Manual flag parsing in `cmd/oastools/main.go` is verbose and error-prone:

```go
for i := 0; i < len(args); i++ {
    switch args[i] {
    case "--strict":
        strict = true
    case "--warnings":
        includeWarnings = true
    case "--help", "-h":
        showHelp = true
    default:
        if strings.HasPrefix(args[i], "-") {
            return fmt.Errorf("unknown flag: %s", args[i])
        }
        // positional argument handling
    }
}
```

**Issues:**
- No automatic help generation
- Manual error handling for unknown flags
- Complex logic for mixing flags and positional arguments
- Difficult to maintain as commands grow
- No support for flag abbreviations
- No type safety for flag values

### Proposed Solution: Use Go stdlib `flag` package

The Go stdlib `flag` package provides:
- Automatic help text generation
- Type-safe flag parsing
- Built-in error handling
- Support for multiple flag types (bool, string, int, duration, etc.)
- Automatic unknown flag detection

### Implementation Plan

#### Step 1: Audit Current CLI Commands

Document all existing commands and their flags:

**Commands:**
- `parse` - Parse and output OpenAPI document structure
- `validate` - Validate OpenAPI document against specification
- `join` - Join multiple OpenAPI documents
- `convert` - Convert between OAS versions
- `diff` - Compare two OpenAPI documents

**Common Flags:**
- Global: `--help`, `-h`, `--version`, `-v`

**Per-Command Flags:**

`parse`:
- `--resolve-refs` (bool) - Resolve external references
- `--validate-structure` (bool) - Validate document structure during parse
- FILE (positional) - Path to OpenAPI file

`validate`:
- `--strict` (bool) - Enable strict validation mode
- `--warnings` (bool) - Include warning-level issues
- FILE (positional) - Path to OpenAPI file

`join`:
- `--output`, `-o` (string) - Output file path
- `--strategy` (string) - Collision strategy (accept-left, accept-right, fail, fail-on-paths)
- `--tag-strategy` (string) - Tag collision strategy
- `--path-strategy` (string) - Path collision strategy
- FILES (positional, multiple) - Paths to OpenAPI files

`convert`:
- `--to` (string, required) - Target OAS version (2.0, 3.0.3, 3.1.0)
- `--output`, `-o` (string) - Output file path
- `--strict` (bool) - Strict conversion mode
- `--include-info` (bool) - Include conversion info messages
- FILE (positional) - Path to OpenAPI file

`diff`:
- `--mode` (string) - Diff mode (simple, breaking)
- `--min-severity` (string) - Minimum severity to display (info, warning, error, critical)
- SOURCE (positional) - Source OpenAPI file
- TARGET (positional) - Target OpenAPI file

#### Step 2: Design FlagSet Architecture

**Approach:** Create a `FlagSet` per command

```go
type CommandConfig struct {
    Name        string
    Description string
    Setup       func(*flag.FlagSet) interface{} // Returns command-specific config struct
    Execute     func(config interface{}, args []string) error
}
```

**Benefits:**
- Each command has isolated flag parsing
- Clear separation of concerns
- Easy to test individual commands
- Help text is command-specific

#### Step 3: Create Command Structs

Define structs for each command's configuration:

```go
// Parse command flags
type parseFlags struct {
    resolveRefs      bool
    validateStructure bool
}

// Validate command flags
type validateFlags struct {
    strict           bool
    includeWarnings  bool
}

// Join command flags
type joinFlags struct {
    output          string
    strategy        string
    tagStrategy     string
    pathStrategy    string
}

// Convert command flags
type convertFlags struct {
    targetVersion   string
    output          string
    strict          bool
    includeInfo     bool
}

// Diff command flags
type diffFlags struct {
    mode            string
    minSeverity     string
}
```

#### Step 4: Implement Flag Parsing Functions

Create setup functions for each command:

```go
func setupParseFlags(fs *flag.FlagSet) *parseFlags {
    flags := &parseFlags{}
    fs.BoolVar(&flags.resolveRefs, "resolve-refs", false, "resolve external $ref references")
    fs.BoolVar(&flags.validateStructure, "validate-structure", false, "validate document structure during parsing")
    return flags
}

func setupValidateFlags(fs *flag.FlagSet) *validateFlags {
    flags := &validateFlags{}
    fs.BoolVar(&flags.strict, "strict", false, "enable strict validation mode")
    fs.BoolVar(&flags.includeWarnings, "warnings", false, "include warning-level validation issues")
    return flags
}

// ... similar for other commands
```

#### Step 5: Refactor Command Handlers

Update each command handler to use the new flag-based approach:

```go
func handleParse(args []string) error {
    fs := flag.NewFlagSet("parse", flag.ContinueOnError)
    flags := setupParseFlags(fs)

    if err := fs.Parse(args); err != nil {
        return err
    }

    if fs.NArg() != 1 {
        return fmt.Errorf("usage: oastools parse [flags] FILE")
    }

    filePath := fs.Arg(0)

    // Use flags.resolveRefs, flags.validateStructure, etc.
    result, err := parser.ParseWithOptions(
        parser.WithFilePath(filePath),
        parser.WithResolveRefs(flags.resolveRefs),
        parser.WithValidateStructure(flags.validateStructure),
    )

    // ... rest of implementation
}
```

#### Step 6: Update Help Text

Leverage `flag` package's automatic help generation:

```go
func printHelp() {
    fmt.Println("oastools - OpenAPI Specification Tools")
    fmt.Println()
    fmt.Println("Usage:")
    fmt.Println("  oastools <command> [flags] [arguments]")
    fmt.Println()
    fmt.Println("Commands:")
    fmt.Println("  parse      Parse and output OpenAPI document structure")
    fmt.Println("  validate   Validate OpenAPI document")
    fmt.Println("  join       Join multiple OpenAPI documents")
    fmt.Println("  convert    Convert between OAS versions")
    fmt.Println("  diff       Compare two OpenAPI documents")
    fmt.Println()
    fmt.Println("Use 'oastools <command> --help' for command-specific help.")
}

func printCommandHelp(command string) {
    fs := flag.NewFlagSet(command, flag.ContinueOnError)

    switch command {
    case "parse":
        setupParseFlags(fs)
        fmt.Printf("Usage: oastools parse [flags] FILE\n\n")
        fmt.Println("Parse and output OpenAPI document structure")
        fmt.Println("\nFlags:")
        fs.PrintDefaults()
    case "validate":
        setupValidateFlags(fs)
        fmt.Printf("Usage: oastools validate [flags] FILE\n\n")
        fmt.Println("Validate OpenAPI document against specification")
        fmt.Println("\nFlags:")
        fs.PrintDefaults()
    // ... similar for other commands
    }
}
```

#### Step 7: Testing Strategy

**Unit Tests:**
- Test flag parsing for each command
- Test error handling for invalid flags
- Test positional argument handling
- Test help text generation

**Integration Tests:**
- Test full command execution with various flag combinations
- Test flag validation (e.g., required flags, valid enum values)

```go
func TestParseFlags(t *testing.T) {
    tests := []struct {
        name        string
        args        []string
        wantResolve bool
        wantValidate bool
        wantErr     bool
    }{
        {
            name: "no flags",
            args: []string{"openapi.yaml"},
            wantResolve: false,
            wantValidate: false,
        },
        {
            name: "resolve refs",
            args: []string{"--resolve-refs", "openapi.yaml"},
            wantResolve: true,
            wantValidate: false,
        },
        {
            name: "both flags",
            args: []string{"--resolve-refs", "--validate-structure", "openapi.yaml"},
            wantResolve: true,
            wantValidate: true,
        },
        {
            name: "unknown flag",
            args: []string{"--unknown", "openapi.yaml"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            fs := flag.NewFlagSet("parse", flag.ContinueOnError)
            flags := setupParseFlags(fs)

            err := fs.Parse(tt.args)
            if (err != nil) != tt.wantErr {
                t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
            }

            if !tt.wantErr {
                if flags.resolveRefs != tt.wantResolve {
                    t.Errorf("resolveRefs = %v, want %v", flags.resolveRefs, tt.wantResolve)
                }
                if flags.validateStructure != tt.wantValidate {
                    t.Errorf("validateStructure = %v, want %v", flags.validateStructure, tt.wantValidate)
                }
            }
        })
    }
}
```

### Benefits of This Approach

1. **Maintainability:** Adding new flags is straightforward
2. **Consistency:** All commands use the same pattern
3. **Error Handling:** Automatic unknown flag detection
4. **Documentation:** Help text is auto-generated and consistent
5. **Testing:** Easy to test flag parsing independently
6. **Type Safety:** Compile-time checking for flag types
7. **Zero Dependencies:** Uses only Go stdlib

### Migration Path

1. Implement new flag-based parsing alongside existing code
2. Test thoroughly with existing test cases
3. Switch over in a single commit
4. Remove old manual parsing code

### Estimated Effort

- **Setup infrastructure:** 2-3 hours
- **Migrate all commands:** 3-4 hours
- **Write tests:** 2-3 hours
- **Documentation updates:** 1 hour
- **Total:** ~8-11 hours

## Priority 2: JSON Marshaling Code Duplication

### Current State

**Problem:** The `*_json.go` files contain repetitive boilerplate for custom JSON marshaling:

Files affected:
- `parser/common_json.go` (~300 lines)
- `parser/oas2_json.go` (~400 lines)
- `parser/oas3_json.go` (~800 lines)
- `parser/schema_json.go` (~200 lines)

**Pattern Example:**

```go
func (s *Schema) MarshalJSON() ([]byte, error) {
    temp := make(map[string]any)

    if s.Type != nil {
        temp["type"] = s.Type
    }
    if s.Format != "" {
        temp["format"] = s.Format
    }
    // ... 20+ more fields

    for k, v := range s.Extra {
        temp[k] = v
    }

    return json.Marshal(temp)
}

func (s *Schema) UnmarshalJSON(data []byte) error {
    var temp map[string]any
    if err := json.Unmarshal(data, &temp); err != nil {
        return err
    }

    if v, ok := temp["type"]; ok {
        s.Type = v
        delete(temp, "type")
    }
    if v, ok := temp["format"].(string); ok {
        s.Format = v
        delete(temp, "format")
    }
    // ... 20+ more fields

    if len(temp) > 0 {
        s.Extra = temp
    }

    return nil
}
```

**Issues:**
- ~1700 lines of repetitive code across 4 files
- Manual field mapping is error-prone
- Adding new fields requires updating both marshal and unmarshal
- Type assertions are verbose and repeated
- No compile-time checking that all fields are handled

### Proposed Solution: Helper Functions

Rather than code generation (which adds build complexity), create helper functions to reduce boilerplate while maintaining flexibility.

### Implementation Plan

#### Step 1: Analyze Common Patterns

**Marshal Pattern:**
```go
temp := make(map[string]any)
// Set known fields
// Merge Extra map
return json.Marshal(temp)
```

**Unmarshal Pattern:**
```go
var temp map[string]any
json.Unmarshal(data, &temp)
// Extract known fields
// Store remaining in Extra
```

#### Step 2: Create Helper Package

Create `parser/internal/jsonhelpers/` (internal to parser package):

```go
package jsonhelpers

import (
    "encoding/json"
    "reflect"
)

// FieldMapper handles mapping between struct fields and JSON
type FieldMapper struct {
    knownFields map[string]bool
}

// NewFieldMapper creates a new field mapper
func NewFieldMapper(fields ...string) *FieldMapper {
    fm := &FieldMapper{
        knownFields: make(map[string]bool, len(fields)),
    }
    for _, field := range fields {
        fm.knownFields[field] = true
    }
    return fm
}

// IsKnown returns true if the field is a known field
func (fm *FieldMapper) IsKnown(field string) bool {
    return fm.knownFields[field]
}

// MarshalWithExtras marshals a struct while preserving extension fields
func MarshalWithExtras(base map[string]any, extras map[string]any) ([]byte, error) {
    if extras != nil {
        for k, v := range extras {
            base[k] = v
        }
    }
    return json.Marshal(base)
}

// UnmarshalWithExtras unmarshals JSON and extracts known fields
func UnmarshalWithExtras(data []byte, fm *FieldMapper, setter func(key string, value any)) (map[string]any, error) {
    var temp map[string]any
    if err := json.Unmarshal(data, &temp); err != nil {
        return nil, err
    }

    extras := make(map[string]any)
    for k, v := range temp {
        if fm.IsKnown(k) {
            setter(k, v)
        } else {
            extras[k] = v
        }
    }

    if len(extras) == 0 {
        return nil, nil
    }
    return extras, nil
}

// SetString safely sets a string field from map value
func SetString(target *string, value any) bool {
    if s, ok := value.(string); ok {
        *target = s
        return true
    }
    return false
}

// SetBool safely sets a bool field from map value
func SetBool(target *bool, value any) bool {
    if b, ok := value.(bool); ok {
        *target = b
        return true
    }
    return false
}

// SetInt safely sets an int field from map value
func SetInt(target *int, value any) bool {
    if f, ok := value.(float64); ok {
        *target = int(f)
        return true
    }
    return false
}

// SetStringSlice safely sets a []string field from map value
func SetStringSlice(target *[]string, value any) bool {
    if arr, ok := value.([]any); ok {
        result := make([]string, 0, len(arr))
        for _, item := range arr {
            if s, ok := item.(string); ok {
                result = append(result, s)
            }
        }
        *target = result
        return true
    }
    return false
}

// SetStringMap safely sets a map[string]string field from map value
func SetStringMap(target *map[string]string, value any) bool {
    if m, ok := value.(map[string]any); ok {
        result := make(map[string]string, len(m))
        for k, v := range m {
            if s, ok := v.(string); ok {
                result[k] = s
            }
        }
        *target = result
        return true
    }
    return false
}
```

#### Step 3: Refactor Schema JSON Handling

**Before:**
```go
func (s *Schema) MarshalJSON() ([]byte, error) {
    temp := make(map[string]any)

    if s.Type != nil {
        temp["type"] = s.Type
    }
    if s.Format != "" {
        temp["format"] = s.Format
    }
    if s.Title != "" {
        temp["title"] = s.Title
    }
    if s.Description != "" {
        temp["description"] = s.Description
    }
    if s.Default != nil {
        temp["default"] = s.Default
    }
    // ... 15+ more fields

    for k, v := range s.Extra {
        temp[k] = v
    }

    return json.Marshal(temp)
}
```

**After:**
```go
var schemaFields = jsonhelpers.NewFieldMapper(
    "type", "format", "title", "description", "default",
    "nullable", "readOnly", "writeOnly", "deprecated",
    "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum",
    "minLength", "maxLength", "pattern",
    "minItems", "maxItems", "uniqueItems",
    "minProperties", "maxProperties", "required",
    "enum", "multipleOf",
    "items", "properties", "additionalProperties",
    "allOf", "oneOf", "anyOf", "not",
    "discriminator", "xml", "externalDocs", "example",
)

func (s *Schema) MarshalJSON() ([]byte, error) {
    base := make(map[string]any)

    if s.Type != nil {
        base["type"] = s.Type
    }
    if s.Format != "" {
        base["format"] = s.Format
    }
    if s.Title != "" {
        base["title"] = s.Title
    }
    if s.Description != "" {
        base["description"] = s.Description
    }
    if s.Default != nil {
        base["default"] = s.Default
    }
    // ... still need field assignments, but merge is cleaner

    return jsonhelpers.MarshalWithExtras(base, s.Extra)
}

func (s *Schema) UnmarshalJSON(data []byte) error {
    s.Extra = nil // Reset

    extras, err := jsonhelpers.UnmarshalWithExtras(data, schemaFields, func(key string, value any) {
        switch key {
        case "type":
            s.Type = value
        case "format":
            jsonhelpers.SetString(&s.Format, value)
        case "title":
            jsonhelpers.SetString(&s.Title, value)
        case "description":
            jsonhelpers.SetString(&s.Description, value)
        case "default":
            s.Default = value
        // ... more cases
        }
    })

    if err != nil {
        return err
    }

    s.Extra = extras
    return nil
}
```

**Analysis:** This approach reduces boilerplate moderately (~20-30%) but maintains clarity. The real win is in consistency and type-safe helpers.

#### Step 4: Alternative Approach - Reflection-Based Helper

For even more DRY code, consider a reflection-based approach:

```go
package jsonhelpers

import (
    "encoding/json"
    "reflect"
    "strings"
)

// MarshalWithExtensions marshals a struct, preserving extension fields
func MarshalWithExtensions(v any, extras map[string]any) ([]byte, error) {
    // Marshal the struct normally
    baseData, err := json.Marshal(v)
    if err != nil {
        return nil, err
    }

    if extras == nil || len(extras) == 0 {
        return baseData, nil
    }

    // Unmarshal to map, merge extras, re-marshal
    var base map[string]any
    if err := json.Unmarshal(baseData, &base); err != nil {
        return nil, err
    }

    for k, v := range extras {
        base[k] = v
    }

    return json.Marshal(base)
}

// UnmarshalWithExtensions unmarshals JSON, extracting known fields
func UnmarshalWithExtensions(data []byte, v any) (map[string]any, error) {
    // Unmarshal to map
    var temp map[string]any
    if err := json.Unmarshal(data, &temp); err != nil {
        return nil, err
    }

    // Marshal back to JSON and unmarshal into struct
    // This populates known fields
    cleanData, err := json.Marshal(temp)
    if err != nil {
        return nil, err
    }

    if err := json.Unmarshal(cleanData, v); err != nil {
        return nil, err
    }

    // Determine which fields are known
    knownFields := getStructJSONFields(v)

    // Extract unknown fields (extensions)
    extras := make(map[string]any)
    for k, val := range temp {
        if !knownFields[k] {
            extras[k] = val
        }
    }

    if len(extras) == 0 {
        return nil, nil
    }
    return extras, nil
}

// getStructJSONFields returns a map of JSON field names from struct tags
func getStructJSONFields(v any) map[string]bool {
    fields := make(map[string]bool)

    val := reflect.ValueOf(v)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }

    if val.Kind() != reflect.Struct {
        return fields
    }

    typ := val.Type()
    for i := 0; i < typ.NumField(); i++ {
        field := typ.Field(i)

        // Skip unexported fields
        if !field.IsExported() {
            continue
        }

        // Get JSON tag
        jsonTag := field.Tag.Get("json")
        if jsonTag == "" || jsonTag == "-" {
            continue
        }

        // Extract field name (before comma)
        parts := strings.Split(jsonTag, ",")
        fieldName := parts[0]

        fields[fieldName] = true
    }

    return fields
}
```

**Usage:**
```go
func (s *Schema) MarshalJSON() ([]byte, error) {
    type Alias Schema // Prevent recursion
    return jsonhelpers.MarshalWithExtensions((*Alias)(s), s.Extra)
}

func (s *Schema) UnmarshalJSON(data []byte) error {
    type Alias Schema // Prevent recursion
    s.Extra = nil

    extras, err := jsonhelpers.UnmarshalWithExtensions(data, (*Alias)(s))
    if err != nil {
        return err
    }

    s.Extra = extras
    return nil
}
```

**Benefits:**
- Reduces ~200 lines to ~10 lines per type
- Automatically handles all fields based on struct tags
- Compile-time safe (uses existing struct definitions)
- No manual field enumeration

**Tradeoffs:**
- Slight performance overhead (double marshal/unmarshal)
- Less explicit (uses reflection)
- Harder to debug for new contributors

#### Step 5: Benchmark Both Approaches

Create benchmarks to measure performance impact:

```go
func BenchmarkSchemaUnmarshalManual(b *testing.B) {
    data := []byte(`{"type":"object","properties":{"id":{"type":"integer"}}}`)
    for b.Loop() {
        var s Schema
        if err := json.Unmarshal(data, &s); err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkSchemaUnmarshalReflection(b *testing.B) {
    data := []byte(`{"type":"object","properties":{"id":{"type":"integer"}}}`)
    for b.Loop() {
        var s Schema
        if err := json.Unmarshal(data, &s); err != nil {
            b.Fatal(err)
        }
    }
}
```

Run benchmarks and compare:
```bash
make bench-parser
```

**Decision criteria:**
- If performance difference < 10%: Use reflection approach (more DRY)
- If performance difference > 10%: Use helper functions approach (balanced)

#### Step 6: Implementation Strategy

**Phase 1: Create helpers package**
- Implement both approaches
- Write comprehensive tests
- Run benchmarks

**Phase 2: Refactor one file**
- Start with `parser/schema_json.go` (smallest)
- Verify all tests pass
- Verify benchmarks are acceptable

**Phase 3: Refactor remaining files**
- `parser/common_json.go`
- `parser/oas2_json.go`
- `parser/oas3_json.go`

**Phase 4: Cleanup**
- Remove dead code
- Update documentation
- Final benchmark comparison

### Expected Outcomes

**Code Reduction:**
- Manual: 1700 lines → ~1200 lines (~30% reduction)
- Reflection: 1700 lines → ~200 lines (~88% reduction)

**Maintainability:**
- Centralized extension handling logic
- Consistent patterns across all types
- Easier to add new fields

**Risk Mitigation:**
- Comprehensive test coverage ensures no regressions
- Benchmark validation ensures performance is acceptable
- Phased rollout allows early detection of issues

### Estimated Effort

- **Research and prototype:** 2-3 hours
- **Implement helpers:** 2-3 hours
- **Benchmark and decide:** 1-2 hours
- **Refactor files:** 4-6 hours
- **Testing and validation:** 2-3 hours
- **Total:** ~11-17 hours

## Priority 3: Documentation Gaps

### Current State

**Problem Areas Identified in Review:**

1. **API Documentation:** Need more examples for complex scenarios
2. **Breaking Change Semantics:** No reference document for differ package
3. **ParseResult Mutability:** Comment says "immutable" but not enforced
4. **External Reference Handling:** Joiner limitation not well-documented

### Proposed Improvements

#### 1. API Documentation Examples

**Location:** Package godoc comments and `example_test.go` files

**Add examples for:**

**Converter Package:**
```go
// Example_complexConversion demonstrates converting a complex OAS 2.0 document
// with OAuth2 flows, custom security schemes, and polymorphic schemas to OAS 3.0.
func Example_complexConversion() {
    // Load OAS 2.0 document with complex features
    result, err := converter.ConvertWithOptions(
        converter.WithFilePath("testdata/complex-swagger.yaml"),
        converter.WithTargetVersion("3.0.3"),
        converter.WithStrictMode(false),
        converter.WithIncludeInfo(true),
    )

    if err != nil {
        log.Fatal(err)
    }

    // Review conversion issues
    for _, issue := range result.Issues {
        fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Path, issue.Message)
    }

    // OAuth2 flows are restructured in OAS 3.0
    // Security schemes are moved to components.securitySchemes
    // discriminator property is added to polymorphic schemas
}
```

**Validator Package:**
```go
// Example_customValidation demonstrates how to use the validator with
// custom options for best practice warnings.
func Example_customValidation() {
    v := validator.New()
    v.IncludeWarnings = true
    v.StrictMode = true

    result, err := v.Validate("openapi.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Separate errors from warnings
    errors := []issues.Issue{}
    warnings := []issues.Issue{}

    for _, issue := range result.Errors {
        if issue.Severity == severity.SeverityError {
            errors = append(errors, issue)
        } else if issue.Severity == severity.SeverityWarning {
            warnings = append(warnings, issue)
        }
    }

    fmt.Printf("Errors: %d, Warnings: %d\n", len(errors), len(warnings))
}
```

**Differ Package:**
```go
// Example_breakingChanges demonstrates how to detect breaking changes
// between two API versions and interpret the results.
func Example_breakingChanges() {
    d := differ.New()
    result, err := d.DiffWithOptions(
        differ.WithSourceFilePath("api-v1.yaml"),
        differ.WithTargetFilePath("api-v2.yaml"),
        differ.WithMode(differ.ModeBreaking),
    )

    if err != nil {
        log.Fatal(err)
    }

    // Critical changes: API consumers WILL break
    criticalChanges := result.GetChangesBySeverity(severity.SeverityCritical)
    fmt.Printf("Breaking changes: %d\n", len(criticalChanges))

    for _, change := range criticalChanges {
        fmt.Printf("- %s: %s\n", change.Path, change.Message)
    }

    // Example output:
    // - /paths/users/{id}/delete: Operation removed
    // - /paths/users/parameters/id: Required parameter removed
}
```

#### 2. Breaking Change Reference Document

**Location:** `docs/breaking-changes.md`

**Structure:**
```markdown
# Breaking Change Semantics

## Overview

The differ package classifies changes by severity based on their impact on API consumers.

## Severity Levels

### Critical (API consumers WILL break)
- Removed endpoints
- Removed operations (GET, POST, etc.)
- Removed required request parameters
- Changed parameter location (query → header)
- Removed required request body fields
- Changed response status codes (removing success codes)

### Error (API consumers MAY break)
- Changed parameter/property types (string → integer)
- Added new required parameters/fields
- Removed optional fields that were commonly used
- Changed enum values (restricted set)
- Removed response content types

### Warning (API consumers SHOULD be aware)
- Added new optional parameters
- Deprecated operations or parameters
- Changed descriptions that affect semantics
- Added new enum values (expanded set)
- Changed examples

### Info (Non-breaking changes)
- Added new optional fields
- Relaxed constraints (maxLength increased)
- Added new endpoints
- Added new operations to existing paths
- Improved documentation

## Version Compatibility

### Semantic Versioning Guidance

Based on severity:
- **Critical/Error changes:** MAJOR version bump (1.0.0 → 2.0.0)
- **Warning changes:** MINOR version bump (1.0.0 → 1.1.0)
- **Info changes:** PATCH version bump (1.0.0 → 1.0.1)

## Examples

[Detailed examples of each change type]
```

**Implementation:**
1. Create the document with comprehensive examples
2. Link from differ package godoc
3. Reference in CLI help text for `diff` command

#### 3. ParseResult Immutability Documentation

**Location:** `parser/parser.go`

**Current:**
```go
// ParseResult contains the parsed OpenAPI document and metadata.
// The result should be treated as immutable after parsing.
type ParseResult struct {
    // ...
}
```

**Improved:**
```go
// ParseResult contains the parsed OpenAPI document and metadata.
//
// Immutability: While Go does not enforce immutability, callers should treat
// ParseResult as read-only after parsing. Modifying the returned document
// may lead to unexpected behavior if the document is cached or shared.
//
// For document modification use cases:
//  - Converter package: Use for version conversion
//  - Joiner package: Use for merging documents
//  - Manual modification: Create a deep copy first (json.Marshal + Unmarshal)
//
// Thread-safety: ParseResult is safe for concurrent reads after parsing,
// but concurrent modification is not supported.
type ParseResult struct {
    // ...
}
```

**Add helper function:**
```go
// Copy creates a deep copy of the ParseResult.
// This is useful when you need to modify the document without affecting the original.
//
// Example:
//  original, _ := parser.Parse("openapi.yaml")
//  modified := original.Copy()
//  // Safe to modify modified.OAS3.Paths, etc.
func (pr *ParseResult) Copy() (*ParseResult, error) {
    data, err := json.Marshal(pr)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal for copy: %w", err)
    }

    var copy ParseResult
    if err := json.Unmarshal(data, &copy); err != nil {
        return nil, fmt.Errorf("failed to unmarshal copy: %w", err)
    }

    return &copy, nil
}
```

#### 4. External Reference Handling in Joiner

**Location:** `joiner/joiner.go` and `joiner/doc.go`

**Current state:** Limitation mentioned in comments but not in user-facing docs.

**Improved package doc:**
```go
// Package joiner provides functionality for joining multiple OpenAPI
// Specification documents into a single unified document.
//
// # External References
//
// The joiner preserves external $ref values but does NOT resolve or merge
// them. This is intentional to avoid ambiguity and maintain document structure.
//
// If your documents contain external references, you have two options:
//
// 1. Resolve references before joining:
//    parser.ParseWithOptions(parser.WithResolveRefs(true))
//
// 2. Keep external references and resolve after joining:
//    joiner.Join(files) → parser.ParseWithOptions(parser.WithResolveRefs(true))
//
// Example with external references:
//  // Document 1: base.yaml
//  paths:
//    /users:
//      get:
//        responses:
//          200:
//            schema:
//              $ref: "./schemas/user.yaml#/User"
//
//  // Document 2: extension.yaml
//  paths:
//    /posts:
//      get:
//        responses:
//          200:
//            schema:
//              $ref: "./schemas/post.yaml#/Post"
//
//  // After joining, both $ref values are preserved
//  // You can then resolve them with parser.WithResolveRefs(true)
package joiner
```

**Add to JoinResult:**
```go
// JoinResult contains the result of joining multiple OpenAPI documents.
type JoinResult struct {
    // Document is the unified OpenAPI document
    Document interface{}

    // Version is the detected OpenAPI version
    Version string

    // Format is the detected format (json or yaml)
    Format string

    // Collisions contains any collision warnings generated during the join
    Collisions []string

    // ExternalRefsCount tracks the number of external $ref values preserved.
    // If > 0, consider resolving references with parser.WithResolveRefs(true)
    // after joining.
    ExternalRefsCount int
}
```

#### 5. Documentation Testing

Add tests to ensure examples compile and run:

```go
// Run all example tests
func TestExamples(t *testing.T) {
    // This ensures all Example* functions compile and run without errors
}
```

### Implementation Checklist

- [ ] Add complex conversion example to `converter/example_test.go`
- [ ] Add custom validation example to `validator/example_test.go`
- [ ] Add breaking changes example to `differ/example_test.go`
- [ ] Create `docs/breaking-changes.md` with comprehensive reference
- [ ] Update `parser.ParseResult` documentation for immutability
- [ ] Add `ParseResult.Copy()` method with example
- [ ] Update `joiner` package documentation for external references
- [ ] Add `ExternalRefsCount` field to `JoinResult`
- [ ] Add example tests for all new examples
- [ ] Update README.md to link to new documentation

### Estimated Effort

- **Write new examples:** 2-3 hours
- **Breaking change reference doc:** 2-3 hours
- **Parser immutability improvements:** 1 hour
- **Joiner external ref documentation:** 1 hour
- **Testing and validation:** 1-2 hours
- **Total:** ~7-10 hours

## Implementation Order

**Phase 1: CLI Refactor** (Highest Impact, Standalone)
- Can be done independently
- Immediate maintainability improvement
- Clear success criteria (all commands work with new flags)

**Phase 2: Documentation** (Low Risk, High Value)
- Can be done in parallel with CLI work
- No risk of breaking changes
- Improves user experience immediately

**Phase 3: JSON Marshaling** (Performance Sensitive)
- Requires careful benchmarking
- Higher risk of subtle bugs
- Most time-intensive

## Success Criteria

### CLI Refactor
- [ ] All commands use `flag` package
- [ ] All existing tests pass
- [ ] New flag parsing tests added
- [ ] Help text is consistent and auto-generated
- [ ] No regression in functionality

### JSON Marshaling
- [ ] Code reduction of at least 30% (manual) or 80% (reflection)
- [ ] All existing tests pass
- [ ] Performance degradation < 10%
- [ ] All extension fields (x-*) are preserved

### Documentation
- [ ] At least 3 new complex examples added
- [ ] Breaking change reference document complete
- [ ] Parser immutability clearly documented
- [ ] Joiner external reference handling documented
- [ ] All examples tested and working

## Risk Mitigation

### CLI Refactor Risks
- **Risk:** Breaking command-line compatibility
- **Mitigation:** Comprehensive integration tests before/after

### JSON Marshaling Risks
- **Risk:** Extension fields not preserved correctly
- **Mitigation:** Extensive tests for x-* fields in all types
- **Risk:** Performance regression
- **Mitigation:** Benchmark comparison, rollback if > 10% degradation

### Documentation Risks
- **Risk:** Examples become outdated
- **Mitigation:** Test examples in CI, version-pin examples

## Timeline Estimate

- **Phase 1 (CLI):** 8-11 hours → 2-3 days part-time
- **Phase 2 (Docs):** 7-10 hours → 2 days part-time
- **Phase 3 (JSON):** 11-17 hours → 3-4 days part-time

**Total:** ~26-38 hours → 7-10 days part-time

## Next Steps

1. Review this plan with stakeholders
2. Set up feature branch: `refactor/review-feedback`
3. Begin with Phase 1 (CLI refactor)
4. Commit after each phase completion
5. Open PR when all phases complete

## Open Questions

1. **CLI flags:** Should we support short flags (e.g., `-o` for `--output`)?
2. **JSON helpers:** Prefer manual helpers or reflection approach?
3. **Documentation:** Should breaking change docs live in repo or separate wiki?
4. **Versioning:** Do these changes warrant a minor version bump (v1.8.0)?

## References

- Original review: `planning/full-review.md`
- Go flag package: https://pkg.go.dev/flag
- JSON marshaling patterns: https://go.dev/blog/json
