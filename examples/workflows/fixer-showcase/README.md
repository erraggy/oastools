# Fixer Showcase

Demonstrates all available fix types in the oastools fixer package with before/after comparison.

## What You'll Learn

- All available fix types and what each one does
- When to use each fix type
- Using dry-run mode to preview changes
- Applying multiple fixes at once
- Chaining fixes with validation using `ToParseResult()`

## Prerequisites

- Go 1.25+

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
| `FixTypePathParameterNotRequired` | Declared path params missing `required` | `in: path` without `required` -> `required: true` |
| `FixTypePrunedUnusedSchema` | Unreferenced schemas | Orphan schemas -> removed |
| `FixTypeStubMissingRef` | `$ref` to an undefined target | `$ref: '#/components/schemas/Order'` -> stub created |

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
  - Declared path parameters missing required: true
  - Unused/unreferenced schemas
  - $refs pointing at schemas that do not exist

[0/9] Initial Validation
------------------------
  [X] Found 8 validation errors:
    - oas 3.0.3: duplicate operationId 'getPets' at 'paths./pet...
    - oas 3.0.3: invalid parameter 'paths./orders/{orderId}.get...
    - Component name "Response[Pet]" must match ^[a-zA-Z0-9._-]+$
    ... and 5 more

[1/9] Fix: CSV Enums
------------------------
  -> CSV enum values -> proper arrays
  [OK] Applied 1 fix(es):
    - expanded CSV enum string to 5 individual values

[2/9] Fix: Duplicate OperationIds
------------------------
  -> Duplicate IDs -> unique suffixed IDs
  [OK] Applied 1 fix(es):
    - renamed duplicate operationId "getPets" to "getPets2"...

...

[6/9] Fix: Path Parameters Missing required
------------------------
  -> Declared {orderId} param -> required: true
  [OK] Applied 1 fix(es):
    - Set required: true on path parameter 'orderId'

...

[8/9] Fix: Stub Missing Refs
------------------------
  -> $ref to undefined Order -> stub created
  [OK] Applied 1 fix(es):
    - Created stub schema for missing reference #/components/schemas/Order

[9/9] Apply ALL Fixes
------------------------
  Dry-run preview:
    Would apply 9 fixes
    - duplicate-operation-id: 1
    - enum-csv-expanded: 1
    - missing-path-parameter: 2
    - path-parameter-not-required: 1
    - pruned-empty-path: 1
    - pruned-unused-schema: 2
    - renamed-generic-schema: 1
    (stub-missing-ref is absent above: stubbing is skipped in
     dry-run mode, so the applied count below is higher)

  Applying all fixes:
  [OK] Applied 10 total fixes

  Validation after fixes:
  [OK] Spec is now VALID!
  -> Final schema count: 3
  -> Schemas: Order, Pet, Response_Pet_

=======================================
Available Fix Types:
  fixer.FixTypeEnumCSVExpanded            - Convert CSV enums to arrays
  fixer.FixTypeDuplicateOperationId       - Make operation IDs unique
  fixer.FixTypePrunedEmptyPath            - Remove empty path items
  fixer.FixTypeRenamedGenericSchema       - Sanitize generic names
  fixer.FixTypeMissingPathParameter       - Add missing path params
  fixer.FixTypePathParameterNotRequired   - Set required: true on path params
  fixer.FixTypePrunedUnusedSchema         - Remove unreferenced schemas
  fixer.FixTypeStubMissingRef             - Stub out unresolved $refs
```

> The `[0/9]` error list is elided above because the validator does not fix the
> order of its findings; the `[9/9]` fix-type summary is sorted and stable.

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates all fix types individually and combined |
| specs/problematic-api.yaml | OpenAPI spec with all fixable issues |

## Key Concepts

### FixType Constants

Each fix has a corresponding constant:

```go
fixer.FixTypeEnumCSVExpanded            // "enum-csv-expanded"
fixer.FixTypeDuplicateOperationId       // "duplicate-operation-id"
fixer.FixTypePrunedEmptyPath            // "pruned-empty-path"
fixer.FixTypeRenamedGenericSchema       // "renamed-generic-schema"
fixer.FixTypeMissingPathParameter       // "missing-path-parameter"
fixer.FixTypePathParameterNotRequired   // "path-parameter-not-required"
fixer.FixTypePrunedUnusedSchema         // "pruned-unused-schema"
fixer.FixTypeStubMissingRef             // "stub-missing-ref"
```

### Enabling Specific Fixes

By default, only the two path parameter fixes are enabled (`DefaultEnabledFixes()`:
`FixTypeMissingPathParameter` and `FixTypePathParameterNotRequired`) — both are
mechanical repairs the spec leaves no room to interpret. Every other fix type
renames or removes content, so it stays opt-in:

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

`FixTypeStubMissingRef` is the one exception: stubbing runs first in the pipeline
because later passes traverse `$ref`s, and it is skipped under dry-run. A preview
therefore undercounts by the number of stubs a real run would create.

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
