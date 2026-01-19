# Schema Deduplication

Demonstrates schema deduplication strategies when merging OpenAPI specifications using the joiner package.

## What You'll Learn

- How to identify structurally identical schemas across documents
- Using `deduplicate-equivalent` strategy for same-named schema collisions
- Using `semantic-deduplication` for different-named equivalent schemas
- Understanding when each approach applies

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/schema-deduplication
go run main.go
```

## Expected Output

```
Schema Deduplication Strategies
================================

Scenario: Both APIs have error schemas with IDENTICAL structure
  - users-api.yaml: UserError {code, message, details}
  - products-api.yaml: ProductError {code, message, details}

These are structurally equivalent but have different names.

[1/3] Baseline: No deduplication
-----------------------------------------
  Result: Success
  Schemas in merged doc: [Product ProductError User UserError]

  Note: Both UserError and ProductError exist in output.
  This is wasteful since they're structurally identical!

[2/3] Strategy: deduplicate-equivalent
-----------------------------------------
  This strategy handles SAME-named collisions.
  When two schemas named 'Error' collide:
    - If structurally equivalent -> keep one
    - If different -> fail

  Configuration:
    SchemaStrategy: deduplicate
    EquivalenceMode: deep

  Use case: When teams independently define the same schema
  with the same name - common with shared types like Error.

[3/3] Strategy: semantic-deduplication
-----------------------------------------
  Result: Success
  Schemas in merged doc: [Product ProductError User]

  UserError was deduplicated to ProductError
     (ProductError < UserError alphabetically)

  Warnings:
    - semantic deduplication: consolidated 1 duplicate schema(s)

  Configuration:
    SemanticDeduplication: true

  The joiner identified that UserError = ProductError
  and consolidated them. All $refs are automatically rewritten.

=========================================
Key Takeaway:
  - deduplicate-equivalent: Merges SAME-named schemas if equivalent
  - semantic-deduplication: Finds DIFFERENT-named equivalent schemas
                            and consolidates to canonical name
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates the schema deduplication workflow |
| specs/users-api.yaml | User service spec with `User` and `UserError` schemas |
| specs/products-api.yaml | Product service spec with `Product` and `ProductError` schemas |

## Key Concepts

### Deduplication Strategies

| Strategy | When to Use |
|----------|-------------|
| `deduplicate-equivalent` | Same-named schemas colliding (e.g., both have `Error`) |
| `semantic-deduplication` | Different-named but identical schemas (e.g., `UserError` vs `ProductError`) |

### How `deduplicate-equivalent` Works

This is a **collision strategy** (`SchemaStrategy`) that triggers when two schemas have the **same name**:

1. When `UserAPI` and `ProductAPI` both define `Error`
2. The joiner compares their structure using `EquivalenceMode` (shallow or deep)
3. If equivalent -> keeps one, discards the duplicate
4. If different -> fails with collision error

```go
config.SchemaStrategy = joiner.StrategyDeduplicateEquivalent
config.EquivalenceMode = "deep"
```

### How Semantic Deduplication Works

This is a **post-merge optimization** that finds schemas with **different names** but identical structure:

1. After merging, scans all schemas
2. Groups schemas by structural equivalence
3. Selects canonical name (alphabetically first)
4. Removes duplicates and rewrites all `$ref` pointers

```go
config.SemanticDeduplication = true
```

### Equivalence Modes

| Mode | Comparison Depth |
|------|------------------|
| `none` | Disabled (no equivalence checking) |
| `shallow` | Top-level properties only |
| `deep` | Full recursive comparison including nested schemas |

## Use Cases

- **Shared error schemas** - Multiple microservices define identical error responses
- **Common data types** - Teams independently define equivalent Address, Money, or Timestamp schemas
- **API consolidation** - Merging specs that evolved separately but converged on structure
- **Spec cleanup** - Reducing schema bloat in merged specifications

## Next Steps

- [Multi-API Merge](../multi-api-merge/) - Basic multi-spec merging workflow
- [Breaking Change Detection](../breaking-change-detection/) - Compare merged versions
- [Joiner Deep Dive](../../../packages/joiner/) - Complete joiner documentation

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
