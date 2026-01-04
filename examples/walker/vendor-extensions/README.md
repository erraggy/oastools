# Vendor Extensions

Demonstrates adding vendor extensions (x-*) to an OpenAPI specification for downstream tooling integration.

## What You'll Learn

- In-place mutation via pointer receivers (handlers receive `*parser.Schema`, `*parser.Operation`, etc.)
- Using the `Extra` map to add vendor extensions to any OAS node
- Conditional mutation based on node properties (HTTP method, path prefix, deprecation status)
- Using `SkipChildren` to exclude deprecated operations from enhancement
- Outputting the modified document as YAML

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/walker/vendor-extensions
go run main.go
```

## Expected Output

The output shows the full modified YAML specification. Key excerpts demonstrating the added extensions:

**Schema with processing metadata:**
```yaml
Pet:
  type: object
  x-processed: true
  x-processed-at: "2024-01-15T10:30:00Z"
  properties:
    id:
      type: integer
      x-processed: true
      x-processed-at: "2024-01-15T10:30:00Z"
```

**Operation with rate limiting:**
```yaml
get:
  summary: List all pets
  operationId: listPets
  x-rate-limit: 100
  x-cache-ttl: 60
```

**Modification summary:**
```
Modification Summary
--------------------
Schemas processed:     20
Operations enhanced:   3 (with rate limits)
Operations skipped:    0 (deprecated)
Paths marked internal: 0
```

## Files

| File | Purpose |
|------|---------|
| main.go | Adds vendor extensions using walker mutation handlers |
| go.mod | Module definition with local replace directive |

## Key Concepts

### Mutation Support

Walker handlers receive pointers to nodes, enabling in-place mutation:

```go
walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
    if schema.Extra == nil {
        schema.Extra = make(map[string]any)
    }
    schema.Extra["x-processed"] = true
    return walker.Continue
})
```

The original document is modified directly - no need to return a modified copy.

### Extra Map for Vendor Extensions

All OAS nodes have an `Extra` field (`map[string]any`) that captures vendor extensions:

```go
if op.Extra == nil {
    op.Extra = make(map[string]any)
}
op.Extra["x-rate-limit"] = 100
op.Extra["x-cache-ttl"] = 60
```

When marshaled, these become `x-rate-limit: 100` and `x-cache-ttl: 60` in the output.

### SkipChildren for Conditional Processing

Use `SkipChildren` to exclude entire subtrees from processing:

```go
walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
    if op.Deprecated {
        return walker.SkipChildren  // Don't process children of deprecated operations
    }
    // ... enhance non-deprecated operations
    return walker.Continue
})
```

## Use Cases

- **API Gateway Configuration**: Add rate limiting, caching, and routing extensions
- **Documentation Generation**: Mark internal/external APIs, add custom metadata
- **Code Generation**: Add generator hints (x-go-type, x-nullable, etc.)
- **Validation Enhancement**: Add custom validation rules (x-pattern, x-min-length)
- **Deprecation Workflow**: Tag deprecated operations with migration paths

## Next Steps

- [Walker Deep Dive](../../../walker/deep_dive.md) - Complete walker documentation
- [API Statistics](../api-statistics/) - Collect statistics using multiple handlers
- [Reference Collector](../reference-collector/) - Track schema definitions and references

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
