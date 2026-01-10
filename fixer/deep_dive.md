<a id="top"></a>

# Fixer Package Deep Dive

!!! tip "Try it Online"
    No installation required! [Try the fixer in your browser →](https://oastools.robnrob.com/fix)

The [`fixer`](https://pkg.go.dev/github.com/erraggy/oastools/fixer) package provides automatic fixes for common OpenAPI Specification validation errors, supporting both OAS 2.0 and OAS 3.x documents.

## Table of Contents

- [Overview](#overview)
- [Fix Types](#fix-types)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Generic Naming Strategies](#generic-naming-strategies)
- [Configuration Reference](#configuration-reference)
- [Package Chaining](#package-chaining)
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
| `FixTypeEnumCSVExpanded` | ❌ Disabled | Expands CSV enum strings to typed arrays (e.g., "1,2,3" → [1, 2, 3]) |

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

See also: [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-package), [Functional options](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-FixWithOptions) on pkg.go.dev

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

See also: [Type inference example](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-Fixer_InferTypes) on pkg.go.dev

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

See also: [Dry-run example](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-WithDryRun) on pkg.go.dev

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

See also: [Generic naming example](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-FixWithOptions-GenericNaming) on pkg.go.dev

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

See also: [Naming config example](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-WithGenericNamingConfig), [Strategy example](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-GenericNamingStrategy) on pkg.go.dev

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
| `ToParseResult()` | `*parser.ParseResult` | Converts result for package chaining |

[Back to top](#top)

---

## Package Chaining

The `ToParseResult()` method enables seamless chaining with other oastools packages by converting `FixResult` to a `parser.ParseResult`:

```go
// Fix then validate
fixResult, err := fixer.FixWithOptions(
    fixer.WithFilePath("openapi.yaml"),
    fixer.WithInferTypes(true),
)
if err != nil {
    log.Fatal(err)
}

// Chain to validator
v := validator.New()
valResult, _ := v.ValidateParsed(*fixResult.ToParseResult())
fmt.Printf("Valid: %v\n", valResult.Valid)

// Or chain to converter
c := converter.New()
convResult, _ := c.ConvertParsed(*fixResult.ToParseResult(), "3.1.0")
```

This enables workflows like: `parse → fix → validate → convert → join`

[Back to top](#top)

---

## Best Practices

1. **Start with defaults** - `FixTypeMissingPathParameter` handles the most common issue
2. **Enable expensive fixes only when needed** - Schema pruning/renaming can be slow on large specs
3. **Use dry-run in CI** - Verify what would change before applying
4. **Validate after fixing** - Ensure the fixed document is valid
5. **Pipeline usage** - `oastools fix api.yaml | oastools validate -q -`

[Back to top](#top)

---

## Learn More

For additional examples and complete API documentation:

- 📦 [API Reference on pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools/fixer) - Complete API documentation with all examples
- 🔧 [Selective fixes example](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-WithEnabledFixes) - Enable specific fix types
- 🗑️ [Prune unused schemas](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-package-PruneUnusedSchemas) - Remove unreferenced definitions
- 📁 [Prune empty paths](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-package-PruneEmptyPaths) - Clean up empty path items
- ✅ [Check results example](https://pkg.go.dev/github.com/erraggy/oastools/fixer#example-FixResult_HasFixes) - Inspect applied fixes
