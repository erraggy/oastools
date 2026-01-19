# Collision Resolution

Demonstrates how to handle schema collisions when merging OpenAPI specifications using the joiner package.

## What You'll Learn

- How schema collisions occur when merging APIs
- Using `fail-on-collision` to detect conflicts early
- Using `accept-left` to keep the first document's schema
- Using `accept-right` to keep the second document's schema
- Understanding the data loss implications of each strategy

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/collision-resolution
go run main.go
```

## Expected Output

```
Collision Resolution Strategies
================================

Scenario: Both APIs define a 'Transaction' schema with different structures
  - payments-api.yaml: Transaction for payments (amount, currency, paymentMethod)
  - orders-api.yaml: Transaction for orders (orderId, items, total)

[1/3] Strategy: fail-on-collision (default)
---------------------------------------------
  Result: Error (as expected)
  Message: joiner: collision in components.schemas: 'Transaction'
    First defined in:  .../payments-api.yaml at components.schemas.Transaction
    Also defined in:   .../orders-api.yaml at components.schemas.Transaction
    Strategy: fail (set --schema-strategy to 'accept-left' or 'accept-right' to resolve)

  This is the safest default - it forces you to explicitly
  choose how to handle the conflict.

[2/3] Strategy: accept-left
---------------------------------------------
  Result: Success
  Collisions resolved: 1
  Transaction schema kept: payments-api (left)
  Properties: [amount currency id paymentMethod processedAt]
  Warnings:
    - components.schemas 'Transaction' kept from first document: source .../orders-api.yaml

  The orders-api Transaction schema was DROPPED.
  Any code expecting orderId/items/total will break!

[3/3] Strategy: accept-right
---------------------------------------------
  Result: Success
  Collisions resolved: 1
  Transaction schema kept: orders-api (right)
  Properties: [createdAt id items orderId total]
  Warnings:
    - components.schemas 'Transaction' overwritten: source .../orders-api.yaml

  The payments-api Transaction schema was DROPPED.
  Any code expecting amount/currency/paymentMethod will break!

===============================================
Key Takeaway: accept-left/right silently drops one schema.
If you need BOTH schemas, use rename-left/right instead.
See: examples/workflows/schema-renaming/
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates the three collision resolution strategies |
| specs/payments-api.yaml | Payment processing API with Transaction schema |
| specs/orders-api.yaml | Order management API with a different Transaction schema |

## Key Concepts

### Collision Strategies

| Strategy | Behavior | Use Case |
|----------|----------|----------|
| `StrategyFailOnCollision` | Returns an error when schemas collide | CI pipelines, explicit conflict resolution |
| `StrategyAcceptLeft` | Keeps the first document's schema | When the first API is authoritative |
| `StrategyAcceptRight` | Keeps the second document's schema | When newer APIs should override older ones |
| `StrategyRenameLeft` | Keeps both, renames first schema | Preserve all schemas (see schema-renaming example) |
| `StrategyRenameRight` | Keeps both, renames second schema | Preserve all schemas (see schema-renaming example) |

### When to Use Each Strategy

**fail-on-collision (Recommended Default)**
- Best for CI/CD pipelines where collisions should block merges
- Forces explicit decisions about how to resolve conflicts
- Prevents accidental data loss

**accept-left**
- When the first API is the "source of truth"
- Merging secondary/supplementary APIs into a primary API
- Legacy API takes precedence over newer additions

**accept-right**
- When newer APIs should override older definitions
- Progressive migration scenarios
- Last-write-wins semantics

## The Problem with Accept Strategies

Both `accept-left` and `accept-right` silently drop one schema. This can cause:

1. **Runtime errors** - Code generated against the dropped schema won't work
2. **Silent data loss** - Requests/responses may fail validation
3. **Confusion** - Different teams may have different expectations

## Better Alternative: Rename Strategies

If you need BOTH schemas preserved, use `StrategyRenameLeft` or `StrategyRenameRight`:

```go
config := joiner.DefaultConfig()
config.SchemaStrategy = joiner.StrategyRenameRight  // Keeps both schemas
```

This produces:
- `Transaction` (from payments-api)
- `Transaction_orders-api` (renamed from orders-api)

See the [schema-renaming example](../schema-renaming/) for details.

## Next Steps

- [Schema Renaming](../schema-renaming/) - Keep both schemas with automatic renaming
- [Multi-API Merge](../multi-api-merge/) - Complete merge workflow with semantic deduplication
- [Breaking Change Detection](../breaking-change-detection/) - Detect when merges cause breaking changes

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
