# Fixer Showcase

Demonstrates all available fix types in the oastools fixer package with before/after comparison.

## What You'll Learn

- All available fix types and what each one does
- When to use each fix type
- Using dry-run mode to preview changes
- Applying multiple fixes at once
- Chaining fixes with validation using `ToParseResult()`

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/fixer-showcase
go run main.go
```

## Fix Types

| Fix Type | What It Fixes | Example |
|----------|---------------|---------|
| `FixTypeEnumCSVExpanded` | CSV enum values | `enum: ["1,2,3"]` -> `enum: [1, 2, 3]` |
| `FixTypeDuplicateOperationId` | Duplicate operation IDs | `getPets` -> `getPets`, `getPets2` |
| `FixTypePrunedEmptyPath` | Empty path items | `/empty: {}` -> removed |
| `FixTypeRenamedGenericSchema` | Generic schema names | `Response[Pet]` -> `Response_Pet_` |
| `FixTypeMissingPathParameter` | Missing path params | `/{petId}` without param -> param added |
| `FixTypePrunedUnusedSchema` | Unreferenced schemas | Orphan schemas -> removed |

## Expected Output

```
Fixer Showcase: All Available Fix Types
=======================================

This spec intentionally contains common issues:
  - CSV enum values (should be array)
  - Duplicate operationIds
  - Empty path items
  - Generic schema names like Response[Pet]
  - Missing path parameter definitions
  - Unused/unreferenced schemas

[0/7] Initial Validation
------------------------
  [X] Found 4 validation errors:
    - oas 3.0.3: duplicate operationId 'getPets' at 'paths./pet...
    - Path template references parameter '{petId}' but it is no...
    - Path template references parameter '{petId}' but it is no...
    - Duplicate operationId 'getPets' (first seen at paths./pet...

[1/7] Fix: CSV Enums
------------------------
  -> CSV enum values -> proper arrays
  [OK] Applied 1 fix(es):
    - expanded CSV enum string to 5 individual values

[2/7] Fix: Duplicate OperationIds
------------------------
  -> Duplicate IDs -> unique suffixed IDs
  [OK] Applied 1 fix(es):
    - renamed duplicate operationId "getPets" to "getPets2"...

...

[7/7] Apply ALL Fixes
------------------------
  Dry-run preview:
    Would apply 8 fixes
    - pruned-unused-schema: 2
    - pruned-empty-path: 1
    - missing-path-parameter: 2
    - duplicate-operation-id: 1
    - renamed-generic-schema: 1
    - enum-csv-expanded: 1

  Applying all fixes:
  [OK] Applied 8 total fixes

  Validation after fixes:
  [OK] Spec is now VALID!
  -> Final schema count: 2
  -> Schemas: Pet, Response_Pet_

=======================================
Available Fix Types:
  fixer.FixTypeEnumCSVExpanded       - Convert CSV enums to arrays
  fixer.FixTypeDuplicateOperationId  - Make operation IDs unique
  fixer.FixTypePrunedEmptyPath       - Remove empty path items
  fixer.FixTypeRenamedGenericSchema  - Sanitize generic names
  fixer.FixTypeMissingPathParameter  - Add missing path params
  fixer.FixTypePrunedUnusedSchema    - Remove unreferenced schemas
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates all fix types individually and combined |
| specs/problematic-api.yaml | OpenAPI spec with all fixable issues |

## Key Concepts

### FixType Constants

Each fix has a corresponding constant:

```go
fixer.FixTypeEnumCSVExpanded       // "enum-csv-expanded"
fixer.FixTypeDuplicateOperationId  // "duplicate-operation-id"
fixer.FixTypePrunedEmptyPath       // "pruned-empty-path"
fixer.FixTypeRenamedGenericSchema  // "renamed-generic-schema"
fixer.FixTypeMissingPathParameter  // "missing-path-parameter"
fixer.FixTypePrunedUnusedSchema    // "pruned-unused-schema"
```

### Enabling Specific Fixes

By default, only `FixTypeMissingPathParameter` is enabled. To enable others:

```go
f := fixer.New()
f.EnabledFixes = []fixer.FixType{
    fixer.FixTypeMissingPathParameter,
    fixer.FixTypePrunedUnusedSchema,
    fixer.FixTypeRenamedGenericSchema,
}
result, err := f.FixParsed(*parsed)
```

Or using functional options:

```go
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("spec.yaml"),
    fixer.WithEnabledFixes(
        fixer.FixTypeMissingPathParameter,
        fixer.FixTypePrunedUnusedSchema,
    ),
)
```

### Dry-Run Mode

Preview changes without modifying the document:

```go
preview, err := fixer.FixWithOptions(
    fixer.WithParsed(*parsed),
    fixer.WithDryRun(true),
)
fmt.Printf("Would apply %d fixes\n", preview.FixCount)
```

### Chaining with ToParseResult()

Convert fix results for use with other packages:

```go
// Fix
fixResult, _ := fixer.FixWithOptions(
    fixer.WithFilePath("api.yaml"),
    fixer.WithEnabledFixes(...),
)

// Validate the fixed result
v := validator.New()
validation, _ := v.ValidateParsed(*fixResult.ToParseResult())
```

## Use Cases

### CI/CD Pre-commit

Automatically fix specs before committing:

```bash
oastools fix --all spec.yaml -o spec.yaml
```

### Spec Cleanup

Remove unused schemas and fix naming issues:

```go
f := fixer.New()
f.EnabledFixes = []fixer.FixType{
    fixer.FixTypePrunedUnusedSchema,
    fixer.FixTypeRenamedGenericSchema,
}
```

### Legacy Spec Migration

Fix issues from older generators (e.g., go-restful-openapi CSV enums):

```go
f := fixer.New()
f.EnabledFixes = []fixer.FixType{
    fixer.FixTypeEnumCSVExpanded,
    fixer.FixTypeDuplicateOperationId,
}
```

## Next Steps

- [Validate and Fix](../validate-and-fix/) - Simpler validate-fix-validate workflow
- [Version Conversion](../version-conversion/) - Convert between OAS versions
- [Fixer Package Docs](../../../packages/fixer/) - Complete API documentation

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
