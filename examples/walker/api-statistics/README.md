# API Statistics

Demonstrates collecting API statistics in a single pass using multiple walker handlers.

## What You'll Learn

- Using multiple handlers to collect statistics in one traversal
- Type-safe access to Info, Operation, Schema, Parameter, and Tag nodes
- Building statistics using closure state that captures data across handlers
- Understanding walker traversal order (document root to nested children)
- Handling `schema.Type` as both string and array (OAS 3.1 compatibility)

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/walker/api-statistics
go run main.go
```

## Expected Output

```
API Statistics Report
=====================

API: Petstore API v1.0.0

Operations (3 total):
  GET:     2
  POST:    1

Schemas by Type (20 total):
  array:     1
  integer:   3
  object:    3
  string:    6

Parameters by Location:
  path:      1
  query:     1

Tags:
  - pets
```

## Files

| File | Purpose |
|------|---------|
| main.go | Collects API statistics using multiple walker handlers |
| go.mod | Module definition with local replace directive |

## Key Concepts

### Handler Registration Pattern

The walker uses the functional options pattern for handler registration:

```go
walker.Walk(parseResult,
    walker.WithInfoHandler(func(wc *walker.WalkContext, info *parser.Info) walker.Action { ... }),
    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action { ... }),
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action { ... }),
)
```

Each handler receives a `WalkContext` (with JSON path via `wc.JSONPath`) and the typed node.

### Traversal Order

The walker visits nodes in document order (OAS 3.x):
1. Document root (OAS2Document or OAS3Document)
2. Info object
3. ExternalDocs (root level)
4. Servers
5. Paths and operations (parameters, request body, responses, callbacks)
6. Webhooks (OAS 3.1+)
7. Components (schemas, parameters, etc.)
8. Tags

This allows collecting statistics that depend on earlier nodes.

### Closure State

Handlers share state through closures:

```go
stats := &APIStats{...}  // Shared state

walker.Walk(parseResult,
    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
        stats.TotalOperations++  // Update shared state
        return walker.Continue
    }),
)
```

This pattern enables collecting statistics from multiple node types in a single traversal.

## Use Cases

- **API Governance**: Verify endpoint naming conventions and coverage
- **Documentation Generation**: Extract API metadata for docs
- **Complexity Analysis**: Measure API size and schema complexity
- **Migration Planning**: Inventory operations before version upgrades

## Next Steps

- [Walker Deep Dive](../../../walker/deep_dive.md) - Complete walker documentation
- [Reference Collector](../reference-collector/) - Track schema definitions and references
- [API Documentation](../api-documentation/) - Generate API documentation with endpoint details

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
