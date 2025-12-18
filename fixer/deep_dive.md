<a id="top"></a>

# Fixer Package Deep Dive

The fixer package provides automatic fixes for common OpenAPI Specification validation errors, supporting both OAS 2.0 and OAS 3.x documents.

## Table of Contents

- [Overview](#overview)
- [Fix Types](#fix-types)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Generic Naming Strategies](#generic-naming-strategies)
- [Configuration Reference](#configuration-reference)
- [Best Practices](#best-practices)

---

## Overview

The fixer analyzes OAS documents and applies fixes for issues that would cause validation failures. It preserves the input file format (JSON or YAML) for output consistency.

**Common use cases:**
- Add missing path parameters automatically
- Rename schemas with invalid characters (e.g., `Response[User]`)
- Remove unused schema definitions
- Clean up empty path items

[Back to top](#top)

---

## Fix Types

| Fix Type | Default | Description |
|----------|---------|-------------|
| `FixTypeMissingPathParameter` | ✅ Enabled | Adds Parameter objects for undeclared path template variables |
| `FixTypeRenamedGenericSchema` | ❌ Disabled | Renames schemas containing URL-unsafe characters |
| `FixTypePrunedUnusedSchema` | ❌ Disabled | Removes unreferenced schema definitions |
| `FixTypePrunedEmptyPath` | ❌ Disabled | Removes paths with no HTTP operations |

**Why are some fixes disabled by default?**

Schema renaming and pruning involve expensive operations (walking all references, computing unused schemas) that can significantly slow down processing of large specifications. Enable them explicitly when needed.

[Back to top](#top)

---

## API Styles

### Functional Options (Recommended)

```go
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Applied %d fixes\n", result.FixCount)
```

### Struct-Based (Reusable)

```go
f := fixer.New()
f.InferTypes = true

result1, _ := f.Fix("api1.yaml")
result2, _ := f.Fix("api2.yaml")
```

### Enable Specific Fixes

```go
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("api.yaml"),
    fixer.WithEnabledFixes(
        fixer.FixTypeMissingPathParameter,
        fixer.FixTypeRenamedGenericSchema,
        fixer.FixTypePrunedUnusedSchema,
    ),
)
```

### Enable ALL Fixes

```go
f := fixer.New()
f.EnabledFixes = []fixer.FixType{} // Empty slice enables all
result, _ := f.Fix("api.yaml")
```

[Back to top](#top)

---

## Practical Examples

### Basic Fixing

```go
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
)
if err != nil {
    log.Fatal(err)
}
for _, fix := range result.Fixes {
    fmt.Printf("Fixed: %s at %s\n", fix.Type, fix.Path)
}
```

### Type Inference

When enabled, the fixer infers parameter types from naming conventions:

| Pattern | Inferred Type |
|---------|---------------|
| `*id`, `*Id`, `*ID` | `integer` |
| `*uuid`, `*guid` | `string` (format: uuid) |
| Everything else | `string` |

```go
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
    fixer.WithInferTypes(true),
)
```

### Dry-Run Mode

Preview fixes without applying them:

```go
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
    fixer.WithDryRun(true),
)
fmt.Printf("Would apply %d fixes\n", result.FixCount)
// result.Document is unchanged
```

### Generic Schema Renaming

```go
result, err := fixer.FixWithOptions(
    fixer.WithFilePath("api.yaml"),
    fixer.WithEnabledFixes(fixer.FixTypeRenamedGenericSchema),
    fixer.WithGenericNaming(fixer.GenericNamingOf),
)
// Response[User] → ResponseOfUser
```

[Back to top](#top)

---

## Generic Naming Strategies

When fixing invalid schema names like `Response[User]`:

| Strategy | Result |
|----------|--------|
| `GenericNamingUnderscore` | `Response_User_` |
| `GenericNamingOf` | `ResponseOfUser` |
| `GenericNamingFor` | `ResponseForUser` |
| `GenericNamingFlattened` | `ResponseUser` |
| `GenericNamingDot` | `Response.User` |

Configure with `WithGenericNaming()` or `WithGenericNamingConfig()`.

[Back to top](#top)

---

## Configuration Reference

### Functional Options

| Option | Description |
|--------|-------------|
| `WithFilePath(path)` | Path to specification file |
| `WithParsed(result)` | Pre-parsed ParseResult |
| `WithInferTypes(bool)` | Infer parameter types from names |
| `WithEnabledFixes(fixes...)` | Specific fix types to enable |
| `WithGenericNaming(strategy)` | Naming strategy for generic schemas |
| `WithGenericNamingConfig(cfg)` | Custom naming configuration |
| `WithDryRun(bool)` | Preview without applying |

### Fixer Fields

| Field | Type | Description |
|-------|------|-------------|
| `InferTypes` | `bool` | Enable type inference |
| `EnabledFixes` | `[]FixType` | Fix types to apply (empty = all) |
| `GenericNamingConfig` | `*GenericNamingConfig` | Custom naming rules |
| `DryRun` | `bool` | Preview mode |

### FixResult Fields

| Field | Type | Description |
|-------|------|-------------|
| `Document` | `any` | Fixed document |
| `Fixes` | `[]Fix` | Applied fixes with details |
| `FixCount` | `int` | Total fixes applied |
| `SourceFormat` | `SourceFormat` | Preserved format |

[Back to top](#top)

---

## Best Practices

1. **Start with defaults** - `FixTypeMissingPathParameter` handles the most common issue
2. **Enable expensive fixes only when needed** - Schema pruning/renaming can be slow on large specs
3. **Use dry-run in CI** - Verify what would change before applying
4. **Validate after fixing** - Ensure the fixed document is valid
5. **Pipeline usage** - `oastools fix api.yaml | oastools validate -q -`

[Back to top](#top)
