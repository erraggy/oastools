# Public API Filter

Demonstrates extracting public-facing API endpoints by filtering out internal, admin, and deprecated paths using the walker's SkipChildren action.

## What You'll Learn

- Using SkipChildren for subtree filtering to exclude entire path branches
- Maintaining context across handler calls (currentPath pattern)
- Building filtered subsets of API documents
- Combining multiple filtering criteria (path prefixes + deprecation status)

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/walker/public-api-filter
go run main.go
```

## Expected Output

```
Public API Extraction Report
============================

Included Paths (4):
  /pets
  /pets/{petId}
  /users
  /users/{userId}

Public Operations (5):
  GET    /pets                - listPets: List all pets
  POST   /pets                - createPet: Create a new pet
  GET    /pets/{petId}        - getPetById: Get pet by ID
  GET    /users               - listUsers: List users
  GET    /users/{userId}      - getUserById: Get user by ID

Filtered Out:
  Internal/Admin paths skipped (5):
    - /_admin/config
    - /admin/users
    - /admin/users/{userId}
    - /internal/health
    - /internal/metrics

  Deprecated operations skipped (1):
    - DELETE /pets/{petId}
```

## Files

| File | Purpose |
|------|---------|
| main.go | Filters API paths and operations using walker handlers |
| go.mod | Module definition with local replace directive |
| specs/full-api.yaml | Sample API with public, internal, admin, and deprecated endpoints |

## Key Concepts

### SkipChildren vs Continue

The walker provides flow control through returned actions:

```go
// Continue - process this node and all its children
return walker.Continue

// SkipChildren - process this node but skip all descendants
return walker.SkipChildren
```

When a PathHandler returns SkipChildren, none of the operations, parameters, or responses under that path are visited. This is more efficient than checking each operation individually.

### Maintaining State Between Handlers

Handlers use closure variables to share context:

```go
var currentPath string
var skipCurrentPath bool

walker.Walk(parseResult,
    walker.WithPathHandler(func(pathTemplate string, pathItem *parser.PathItem, path string) walker.Action {
        if isInternalPath(pathTemplate) {
            skipCurrentPath = true
            return walker.SkipChildren
        }
        currentPath = pathTemplate
        skipCurrentPath = false
        return walker.Continue
    }),

    walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
        // Access currentPath from the enclosing scope
        if skipCurrentPath {
            return walker.SkipChildren
        }
        // Use currentPath to associate operation with its path
        ...
    }),
)
```

This pattern enables operations to know which path they belong to.

### Filter Criteria Composition

Multiple filtering rules can be combined:

1. **Path-based filtering**: Skip entire path subtrees with SkipChildren
2. **Operation-level filtering**: Check individual operations for deprecation
3. **Cascading filters**: Path filter runs first, operation filter only sees non-filtered paths

## Use Cases

- **Public Documentation**: Generate docs only for customer-facing endpoints
- **Partner API Specs**: Create filtered specs for external partners
- **API Exposure Control**: Audit which endpoints are publicly accessible
- **SDK Generation**: Generate client code only for public operations
- **Security Review**: Identify and verify internal-only endpoints

## Next Steps

- [Walker Deep Dive](../../../walker/deep_dive.md) - Complete walker documentation
- [API Statistics](../api-statistics/) - Collect statistics across the full API
- [Reference Collector](../reference-collector/) - Track schema definitions and references

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
