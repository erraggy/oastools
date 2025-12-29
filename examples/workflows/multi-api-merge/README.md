# Multi-API Merge

Demonstrates merging multiple OpenAPI specifications using the joiner package.

## What You'll Learn

- How to merge microservice specifications into a unified API
- Configuring collision resolution strategies
- Using semantic deduplication for shared schemas
- Understanding merge warnings and collisions

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/multi-api-merge
go run main.go
```

## Expected Output

```
Multi-API Merge Workflow
========================

Inputs:
  1. users-api.yaml
  2. orders-api.yaml

[1/4] Configuration:
      Path Strategy: fail-on-paths
      Schema Strategy: accept-left
      Semantic Deduplication: true
      Deduplicate Tags: true
      Merge Arrays: true

[2/4] Joining specifications...
      Result Version: 3.0.3
      Collisions Resolved: 1

[3/4] No warnings

[4/4] Writing merged specification...
      Output: /tmp/merged-api.yaml

--- Merged API Summary ---
Title: Users API
Version: 1.0.0

Servers: 2
  - https://users.example.com/v1
  - https://orders.example.com/v1

Tags: 2
  - users
  - orders

Paths: 4
  - /users
  - /users/{userId}
  - /orders
  - /orders/{orderId}

Schemas: 4
  - User
  - Error
  - Order
  - CreateOrderRequest

---
Merge completed successfully
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates the multi-spec merge workflow |
| specs/users-api.yaml | User management microservice spec |
| specs/orders-api.yaml | Order management microservice spec |

## Key Concepts

### Collision Strategies

| Strategy | Behavior |
|----------|----------|
| `StrategyFailOnPaths` | Error if paths conflict |
| `StrategyFailOnCollision` | Error on any component collision |
| `StrategyAcceptLeft` | Keep the first document's version |
| `StrategyAcceptRight` | Keep the second document's version |

### Semantic Deduplication

When `SemanticDeduplication: true`, the joiner identifies structurally identical schemas across documents and consolidates them. In this example:
- Both APIs define an `Error` schema with identical structure
- The joiner recognizes this and keeps a single `Error` schema

### Merge Arrays

When `MergeArrays: true`:
- `servers` arrays are concatenated
- `tags` are combined (with deduplication if enabled)
- `security` requirements are merged

## Use Cases

- **API Gateway composition** - Combine microservice specs into a gateway API
- **Modular documentation** - Split large APIs into manageable parts
- **Multi-team development** - Each team maintains their own spec

## Next Steps

- [Joiner Deep Dive](../../packages/joiner/) - Complete joiner documentation
- [Breaking Change Detection](../breaking-change-detection/) - Compare merged versions
- [Overlay Transformations](../overlay-transformations/) - Apply post-merge customizations

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
