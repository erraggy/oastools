<a id="top"></a>

# Overlay Package Deep Dive

The overlay package implements the [OpenAPI Overlay Specification v1.0.0](https://github.com/OAI/Overlay-Specification), providing a standardized mechanism for augmenting OpenAPI documents through targeted transformations.

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [JSONPath Reference](#jsonpath-reference)
- [Configuration Reference](#configuration-reference)
- [Best Practices](#best-practices)

---

## Overview

Overlays use JSONPath expressions to select specific locations in an OpenAPI document and apply updates or removals. This enables environment-specific customizations, removing internal endpoints for public APIs, or batch-updating descriptions across an entire specification.

**Common use cases:**
- Remove internal/deprecated paths for public documentation
- Add environment-specific server URLs
- Update descriptions or metadata in bulk
- Add vendor extensions across multiple operations

[Back to top](#top)

---

## Key Concepts

### Overlay Document Structure

An overlay document contains:

```yaml
overlay: 1.0.0
info:
  title: Production Customizations
  version: 1.0.0
extends: https://example.com/openapi.yaml  # Optional
actions:
  - target: $.info
    update:
      x-environment: production
  - target: $.paths[?@.x-internal==true]
    remove: true
```

### Action Types

| Type | Description |
|------|-------------|
| **Update** | Merges content into matched nodes. Objects are recursively merged; arrays are appended. |
| **Remove** | Deletes matched nodes from their parent container. Takes precedence if both specified. |

### Dry-Run Mode

Preview changes without modifying the document:

```go
result, _ := overlay.DryRunWithOptions(
    overlay.WithSpecFilePath("openapi.yaml"),
    overlay.WithOverlayFilePath("changes.yaml"),
)
for _, change := range result.Changes {
    fmt.Printf("Would %s %d nodes at %s\n",
        change.Operation, change.MatchCount, change.Target)
}
```

[Back to top](#top)

---

## API Styles

### Functional Options (Recommended)

```go
result, err := overlay.ApplyWithOptions(
    overlay.WithSpecFilePath("openapi.yaml"),
    overlay.WithOverlayFilePath("changes.yaml"),
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Applied %d changes\n", result.ActionsApplied)
```

### Struct-Based (Reusable)

```go
a := overlay.NewApplier()
a.StrictTargets = true  // Fail if any target matches nothing

result1, _ := a.Apply("api1.yaml", "overlay1.yaml")
result2, _ := a.Apply("api2.yaml", "overlay2.yaml")
```

### Pre-Parsed Documents

For performance when processing multiple overlays:

```go
parseResult, _ := parser.ParseWithOptions(parser.WithFilePath("openapi.yaml"))
overlayDoc, _ := overlay.ParseOverlayFile("changes.yaml")

result, _ := overlay.ApplyWithOptions(
    overlay.WithSpecParsed(*parseResult),
    overlay.WithOverlayParsed(overlayDoc),
)
```

[Back to top](#top)

---

## Practical Examples

### Remove Internal Endpoints

```go
o := &overlay.Overlay{
    Version: "1.0.0",
    Info:    overlay.Info{Title: "Remove Internal", Version: "1.0.0"},
    Actions: []overlay.Action{
        {
            Target: "$.paths[?@.x-internal==true]",
            Remove: true,
        },
    },
}
```

### Update All Descriptions (Recursive Descent)

```go
// Find and update ALL descriptions at any depth
o := &overlay.Overlay{
    Version: "1.0.0",
    Info:    overlay.Info{Title: "Update Descriptions", Version: "1.0.0"},
    Actions: []overlay.Action{
        {
            Target: "$..description",
            Update: "Updated by overlay",
        },
    },
}
```

### Compound Filters

```go
// Match paths that are BOTH deprecated AND internal
o := &overlay.Overlay{
    Version: "1.0.0",
    Info:    overlay.Info{Title: "Filter Test", Version: "1.0.0"},
    Actions: []overlay.Action{
        {
            Target: "$.paths[?@.deprecated==true && @.x-internal==true]",
            Update: map[string]any{"x-removal-scheduled": "2025-01-01"},
        },
    },
}
```

### Validation Before Application

```go
o, _ := overlay.ParseOverlayFile("changes.yaml")
if errs := overlay.Validate(o); len(errs) > 0 {
    for _, err := range errs {
        fmt.Printf("Validation error: %s\n", err.Message)
    }
}
```

[Back to top](#top)

---

## JSONPath Reference

| Expression | Description | Example |
|------------|-------------|---------|
| `$.field` | Root field access | `$.info`, `$.paths` |
| `$.paths['/users']` | Bracket notation | Access path by key |
| `$.paths.*` | Wildcard | All paths |
| `$.servers[0]` | Array index | First server |
| `$.servers[-1]` | Negative index | Last server |
| `$..field` | Recursive descent | Find field at any depth |
| `[?@.key==value]` | Simple filter | Match by property |
| `[?@.a==true && @.b==false]` | Compound AND | Multiple conditions |
| `[?@.a==true \|\| @.b==true]` | Compound OR | Either condition |

[Back to top](#top)

---

## Configuration Reference

### Functional Options

| Option | Description |
|--------|-------------|
| `WithSpecFilePath(path)` | Path to OpenAPI specification file |
| `WithSpecParsed(result)` | Pre-parsed ParseResult |
| `WithOverlayFilePath(path)` | Path to overlay file |
| `WithOverlayParsed(o)` | Pre-parsed Overlay struct |
| `WithStrictTargets(bool)` | Fail if any target matches nothing |

### Applier Fields

| Field | Type | Description |
|-------|------|-------------|
| `StrictTargets` | `bool` | When true, returns error if any action's target matches zero nodes |

### Result Fields

| Field | Type | Description |
|-------|------|-------------|
| `ActionsApplied` | `int` | Number of actions that matched and modified nodes |
| `ActionsSkipped` | `int` | Number of actions with no matching targets |
| `Changes` | `[]Change` | Details of each change (for dry-run) |
| `Warnings` | `[]string` | Non-fatal warnings during application |
| `Document` | `any` | The modified document |

[Back to top](#top)

---

## Best Practices

1. **Use dry-run first** - Preview changes before applying to production specs
2. **Validate overlays** - Call `overlay.Validate()` before application
3. **Order actions carefully** - Actions are applied in order; earlier actions affect later ones
4. **Use StrictTargets in CI** - Catch typos in JSONPath expressions
5. **Combine with converter** - Use `WithPreConversionOverlayFile` and `WithPostConversionOverlayFile` for version migrations

[Back to top](#top)
