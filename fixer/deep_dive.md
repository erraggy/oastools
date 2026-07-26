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
| `FixTypePathParameterNotRequired` | ✅ Enabled | Sets `required: true` on existing `in: path` parameters that omit it |
| `FixTypeRenamedGenericSchema` | ❌ Disabled | Renames schemas containing URL-unsafe characters |
| `FixTypePrunedUnusedSchema` | ❌ Disabled | Removes unreferenced schema definitions |
| `FixTypePrunedEmptyPath` | ❌ Disabled | Removes paths with no HTTP operations |
| `FixTypeEnumCSVExpanded` | ❌ Disabled | Expands CSV enum strings to typed arrays (e.g., "1,2,3" → [1, 2, 3]) |
| `FixTypeDuplicateOperationId` | ❌ Disabled | Renames duplicate operationId values to ensure uniqueness |
| `FixTypeStubMissingRef` | ❌ Disabled | Creates empty stubs for unresolved `$ref` targets |

**How `FixTypeMissingPathParameter` decides what is missing**

A path template variable counts as declared when it is declared on the
operation or on the containing path item, whether inline or as a `$ref` into the
document's reusable parameter definitions (OAS 2.0's root-level `parameters` or
OAS 3.x's `components.parameters`). References are resolved, including chains of
them, so a shared parameter is never added a second time.

When a `$ref` cannot be resolved, no parameters are added for that operation.
That covers an external file or URL, a reference that dangles or names a
component which is not a parameter, and a cycle or over-long chain. The
reference may already declare the variable, and adding a duplicate name and
location would produce an invalid document.

This is deliberately more conservative than the validator, which reports a
reference naming a non-parameter component as an error. The fixer cannot know
what the author intended by such a reference, so it declines to guess rather
than adding a parameter beside one it does not understand.

**Where `FixTypePathParameterNotRequired` applies**

Every OAS version requires `required` on an `in: path` parameter and permits no
value other than `true`, so this repair involves no judgment and loses nothing.
It runs at the three sites the validator reports the defect:

| Site | Reported path |
|---|---|
| OAS 2.0 root-level `parameters` | `parameters.{name}` |
| OAS 3.x `components.parameters` | `components.parameters.{name}` |
| Path item and operation, both versions | `paths.{path}[.{method}].parameters[{i}]` |

Parameters that are a `$ref` are skipped. The defect belongs to the definition
being referenced, not to the reference — writing `required` beside a `$ref` would
add a sibling the spec does not allow there. Fixing the definition clears the
error for every use site at once.

The defect test itself, `paramutil.NeedsRequiredTrue`, is shared with the
validator, so what `validate` reports is exactly what `fix` repairs. A parity test
validates a defective spec, fixes it, and re-validates to assert no
`required: true` error survives.

**Why are some fixes disabled by default?**

Disabled fixes fall into two categories:

- **Performance-sensitive**: Schema renaming (`FixTypeRenamedGenericSchema`) and pruning (`FixTypePrunedUnusedSchema`, `FixTypePrunedEmptyPath`) walk all references and compute unused schemas, which can significantly slow processing of large specifications.
- **Behavioral impact**: `FixTypeDuplicateOperationId` renames operation IDs that clients and SDK generators may already depend on. `FixTypeStubMissingRef` injects synthetic placeholder content into the document. Both are opt-in to avoid unexpected breakage.

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
| `WithMutableInput(bool)` | Skip defensive copy when caller owns input |
| `WithUserAgent(userAgent string)` | Custom User-Agent for HTTP requests |
| `WithSourceMap(sm *parser.SourceMap)` | Source map for line/column info in fixes |
| `WithOperationIdNamingConfig(config OperationIdNamingConfig)` | Configuration for duplicate operationId renaming |
| `WithStubConfig(config StubConfig)` | Configuration for missing reference stub creation |
| `WithStubResponseDescription(desc string)` | Default description for stubbed responses |

### Fixer Fields

| Field | Type | Description |
|-------|------|-------------|
| `InferTypes` | `bool` | Enable type inference |
| `EnabledFixes` | `[]FixType` | Fix types to apply (empty = all) |
| `UserAgent` | `string` | User-Agent string for HTTP requests |
| `SourceMap` | `*parser.SourceMap` | Source location lookup for fix issues |
| `GenericNamingConfig` | `GenericNamingConfig` | Custom naming rules |
| `OperationIdNamingConfig` | `OperationIdNamingConfig` | Configuration for duplicate operationId renaming |
| `StubConfig` | `StubConfig` | Configuration for missing reference stub creation |
| `DryRun` | `bool` | Preview mode |
| `MutableInput` | `bool` | Skip defensive copy |

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
