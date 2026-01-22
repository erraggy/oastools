# Stub Missing References Fixer

**Date:** 2025-01-21
**Status:** Design Complete
**Author:** Brainstorming session

## Problem Statement

When `go-restful-openapi` generates swagger specs from Go code using `interface{}` types, it may create `$ref` pointers to schema definitions that don't exist. For example:

```json
{
  "definitions": {},
  "paths": {
    "/test": {
      "get": {
        "responses": {
          "200": {
            "schema": { "$ref": "#/definitions/foo.Bar" }
          }
        }
      }
    }
  }
}
```

This causes validation errors:
```
✗ $ref '#/definitions/foo.Bar' does not resolve to a valid component in the document
```

## Solution

Add a new fixer that:
1. Collects all local `$ref` values in the document
2. Identifies refs that point to non-existent definitions
3. Creates minimal stub definitions to make the document valid

## Scope

### Included (Phase 1)
- **Schemas** (`#/definitions/...` for OAS 2.0, `#/components/schemas/...` for OAS 3.x)
  - Stub: `{}` (empty schema = "any value allowed")
- **Responses** (`#/responses/...` for OAS 2.0, `#/components/responses/...` for OAS 3.x)
  - Stub: `{ "description": "<configurable>" }`

### Excluded
- **External refs** (e.g., `./other.yaml#/definitions/Foo`) - only local `#/...` refs
- **Parameters** - require `name` and `in` fields which can't be inferred from ref path
- **RequestBodies** - technically valid but not useful with empty content

## Design

### Fix Type

```go
FixTypeStubMissingRef FixType = "stub-missing-ref"
```

### Configuration

```go
// StubConfig configures how missing reference stubs are created
type StubConfig struct {
    // ResponseDescription is the description text for stub responses.
    // Default: "Auto-generated stub for missing reference"
    ResponseDescription string
}

func DefaultStubConfig() StubConfig {
    return StubConfig{
        ResponseDescription: "Auto-generated stub for missing reference",
    }
}
```

### Functional Options

```go
// WithStubConfig sets the configuration for missing reference stubs.
func WithStubConfig(config StubConfig) Option

// WithStubResponseDescription sets the description for stub responses.
func WithStubResponseDescription(desc string) Option
```

### CLI Flags

```bash
oastools fix --stub-missing-refs api.yaml
oastools fix --stub-missing-refs --stub-response-desc "TODO: document" api.yaml
```

### Default Behavior

- **Opt-in**: Fix is not enabled by default (consistent with other potentially-surprising fixes)
- **Fix ordering**: Runs early in pipeline, before fixes that traverse refs (like prune-unused)

## Implementation

### New File: `fixer/stub_missing_refs.go`

```go
// isLocalRef returns true if the ref is a local document reference
func isLocalRef(ref string) bool {
    return strings.HasPrefix(ref, "#/")
}

// stubMissingRefsOAS2 finds and stubs missing refs in an OAS 2.0 document
func (f *Fixer) stubMissingRefsOAS2(doc *parser.OAS2Document, result *FixResult) {
    collector := NewRefCollector()
    collector.CollectOAS2(doc)

    // Find missing schema refs
    for ref := range collector.RefsByType[RefTypeSchema] {
        if !isLocalRef(ref) {
            continue
        }
        name := ExtractSchemaNameFromRef(ref, parser.OASVersion20)
        if name == "" {
            continue
        }
        if _, exists := doc.Definitions[name]; !exists {
            f.stubSchemaOAS2(doc, name, result)
        }
    }

    // Find missing response refs
    for ref := range collector.RefsByType[RefTypeResponse] {
        if !isLocalRef(ref) {
            continue
        }
        name := extractResponseNameFromRef(ref, parser.OASVersion20)
        if name == "" {
            continue
        }
        if _, exists := doc.Responses[name]; !exists {
            f.stubResponseOAS2(doc, name, result)
        }
    }
}

func (f *Fixer) stubSchemaOAS2(doc *parser.OAS2Document, name string, result *FixResult) {
    if doc.Definitions == nil {
        doc.Definitions = make(map[string]*parser.Schema)
    }

    stub := &parser.Schema{}
    doc.Definitions[name] = stub

    fix := Fix{
        Type:        FixTypeStubMissingRef,
        Path:        fmt.Sprintf("definitions.%s", name),
        Description: fmt.Sprintf("Created stub schema for missing reference #/definitions/%s", name),
        Before:      nil,
        After:       stub,
    }
    f.populateFixLocation(&fix)
    result.Fixes = append(result.Fixes, fix)
    result.FixCount++
}

func (f *Fixer) stubResponseOAS2(doc *parser.OAS2Document, name string, result *FixResult) {
    if doc.Responses == nil {
        doc.Responses = make(map[string]*parser.Response)
    }

    desc := f.StubConfig.ResponseDescription
    if desc == "" {
        desc = DefaultStubConfig().ResponseDescription
    }

    stub := &parser.Response{Description: desc}
    doc.Responses[name] = stub

    fix := Fix{
        Type:        FixTypeStubMissingRef,
        Path:        fmt.Sprintf("responses.%s", name),
        Description: fmt.Sprintf("Created stub response for missing reference #/responses/%s", name),
        Before:      nil,
        After:       stub,
    }
    f.populateFixLocation(&fix)
    result.Fixes = append(result.Fixes, fix)
    result.FixCount++
}
```

Similar `stubMissingRefsOAS3` implementation for OAS 3.x documents.

### Files to Modify

| File | Change |
|------|--------|
| `fixer/fixer.go` | Add `FixTypeStubMissingRef` constant, `StubConfig` field |
| `fixer/stub_missing_refs.go` | **New** - Core implementation |
| `fixer/stub_missing_refs_test.go` | **New** - Tests |
| `fixer/oas2.go` | Call `stubMissingRefsOAS2()` in fix pipeline |
| `fixer/oas3.go` | Call `stubMissingRefsOAS3()` in fix pipeline |
| `fixer/pipeline.go` | Add `StubConfig` to fixConfig, wire up options |
| `fixer/doc.go` | Document new fix type |
| `cmd/oastools/fix.go` | Add CLI flags |

## Testing

### Core Acceptance Test

```go
func TestStubMissingRef_FixesValidationError(t *testing.T) {
    // 1. Document with missing ref
    input := `{"swagger":"2.0","info":{"title":"Test","version":"1.0"},
               "paths":{"/test":{"get":{"responses":{"200":{
                 "description":"ok","schema":{"$ref":"#/definitions/foo.Bar"}}}}}},
               "definitions":{}}`

    // 2. Validate - should FAIL with specific error
    v := validator.New()
    result, _ := v.Validate(input)
    require.False(t, result.Valid)
    require.Len(t, result.Errors, 1)
    require.Contains(t, result.Errors[0].Error(),
        "$ref '#/definitions/foo.Bar' does not resolve")

    // 3. Fix
    fixed, err := fixer.FixWithOptions(
        fixer.WithParsed(...),
        fixer.WithEnabledFixes(fixer.FixTypeStubMissingRef),
    )
    require.NoError(t, err)
    require.Equal(t, 1, fixed.FixCount)
    require.Equal(t, fixer.FixTypeStubMissingRef, fixed.Fixes[0].Type)

    // 4. Validate again - should PASS
    result2, _ := v.ValidateParsed(*fixed.ToParseResult())
    require.True(t, result2.Valid, "should pass after stubbing")
}
```

### Additional Test Cases

| Test Name | Scenario |
|-----------|----------|
| `TestStubMissingSchema_OAS2` | Missing `#/definitions/Foo` gets stubbed |
| `TestStubMissingSchema_OAS3` | Missing `#/components/schemas/Foo` gets stubbed |
| `TestStubMissingResponse_OAS2` | Missing `#/responses/NotFound` gets stubbed |
| `TestStubMissingResponse_OAS3` | Missing `#/components/responses/NotFound` gets stubbed |
| `TestStubMissing_MultipleRefs` | Multiple missing refs all get stubbed |
| `TestStubMissing_ExistingNotTouched` | Existing definitions are not modified |
| `TestStubMissing_ExternalRefIgnored` | `./other.yaml#/definitions/X` is skipped |
| `TestStubMissing_NilMapsInitialized` | `Definitions: nil` gets initialized before stub |
| `TestStubMissing_CustomResponseDesc` | Custom description appears in stub |
| `TestStubMissing_DisabledByDefault` | Fix doesn't run unless explicitly enabled |
| `TestStubMissing_DryRun` | Dry run reports fixes without modifying doc |

## Usage Examples

### CLI

```bash
# Basic usage
oastools fix --stub-missing-refs api.yaml

# With custom response description
oastools fix --stub-missing-refs --stub-response-desc "TODO: Document this response" api.yaml

# Combine with other fixes
oastools fix --stub-missing-refs --prune-unused --rename-generics api.yaml

# Pipeline: fix then validate
oastools fix --stub-missing-refs api.yaml | oastools validate -
```

### Programmatic

```go
// Basic usage
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("api.yaml"),
    fixer.WithEnabledFixes(fixer.FixTypeStubMissingRef),
)

// With custom config
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("api.yaml"),
    fixer.WithEnabledFixes(fixer.FixTypeStubMissingRef),
    fixer.WithStubResponseDescription("TODO: Document this response"),
)

// Enable multiple fixes
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("api.yaml"),
    fixer.WithEnabledFixes(
        fixer.FixTypeMissingPathParameter,
        fixer.FixTypeStubMissingRef,
        fixer.FixTypePrunedUnusedSchema,
    ),
)
```

## Future Considerations

### Potential Extensions
- Support for stubbing other OAS 3.x component types (Headers, Examples, Links, Callbacks)
- Option to add `x-stub: true` extension to mark auto-generated stubs
- Option to use schema name as `title` field for documentation

### Not Planned
- External ref resolution (out of scope - would require file system access)
- Parameter stubbing (required fields can't be inferred)
- RequestBody stubbing (valid but not useful)
