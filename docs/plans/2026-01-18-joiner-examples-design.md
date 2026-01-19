# Joiner Collision Strategy Examples Design

> Date: 2026-01-18
> Status: Draft

## Overview

Expand the examples directory to cover joiner collision strategies identified from integration tests. Three new themed examples grouped by concept.

## Gap Analysis

Integration tests cover 8 joiner strategies, but examples only show basic merging:

| Strategy | In Integration Tests | In Examples |
|----------|---------------------|-------------|
| accept-left | ✅ | ✅ (multi-api-merge) |
| accept-right | ✅ | ❌ |
| fail-on-collision | ✅ | ❌ |
| fail-on-paths | ✅ | ✅ (multi-api-merge) |
| rename-left | ✅ | ❌ |
| rename-right | ✅ | ❌ |
| deduplicate-equivalent | ✅ | ❌ |
| semantic-deduplication | ✅ | ✅ (multi-api-merge) |

## New Examples

### 1. collision-resolution/

**Purpose:** Demonstrate what happens when schemas with the same name but different structures collide.

**Location:** `examples/workflows/collision-resolution/`

**Strategies covered:**
- `fail-on-collision` (default) - Shows the error message
- `accept-left` - Keeps first document's schema
- `accept-right` - Keeps second document's schema

**Spec scenario:**
- `payments-api.yaml`: `Transaction` with `{amount, currency, paymentMethod}`
- `orders-api.yaml`: `Transaction` with `{orderId, items, total}`

**Teaching goal:** Help users understand that accept-left/right silently drops one definition.

### 2. schema-deduplication/

**Purpose:** Demonstrate consolidating structurally identical schemas with different names.

**Location:** `examples/workflows/schema-deduplication/`

**Strategies covered:**
- `deduplicate-equivalent` - Merges same-named equivalent schemas
- `SemanticDeduplication` - Cross-document dedup for different-named equivalents

**Spec scenario:**
- `users-api.yaml`: `UserError` with `{code, message, details}`
- `products-api.yaml`: `ProductError` with `{code, message, details}` (identical structure)

**Teaching goal:** Show when each deduplication approach applies.

### 3. schema-renaming/

**Purpose:** Demonstrate preserving both conflicting schemas by renaming.

**Location:** `examples/workflows/schema-renaming/`

**Strategies covered:**
- `rename-right` - Keep left's name, rename right
- `rename-left` - Keep right's name, rename left
- `RenameTemplate` - Custom naming patterns
- `NamespacePrefix` - Prefix-based naming

**Spec scenario:**
- `billing-api.yaml`: `Account` with `{accountId, balance, creditLimit, paymentTerms}`
- `crm-api.yaml`: `Account` with `{accountId, companyName, contacts, industry}`

**Teaching goal:** When you need both conflicting schemas, renaming preserves both.

## File Structure

```
examples/workflows/
├── collision-resolution/
│   ├── README.md
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   └── specs/
│       ├── payments-api.yaml
│       └── orders-api.yaml
├── schema-deduplication/
│   ├── README.md
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   └── specs/
│       ├── users-api.yaml
│       └── products-api.yaml
└── schema-renaming/
    ├── README.md
    ├── go.mod
    ├── go.sum
    ├── main.go
    └── specs/
        ├── billing-api.yaml
        └── crm-api.yaml
```

## Implementation Plan

### Phase 1: Create Examples (Parallel)

Three parallel developer agents, one per example:

**Agent 1: collision-resolution/**
1. Create directory structure
2. Write specs with conflicting `Transaction` schemas
3. Write main.go demonstrating all three strategies
4. Write README.md with expected output
5. Create go.mod and run `go mod tidy`
6. Verify with `go run main.go`

**Agent 2: schema-deduplication/**
1. Create directory structure
2. Write specs with equivalent `UserError`/`ProductError` schemas
3. Write main.go demonstrating both dedup approaches
4. Write README.md with expected output
5. Create go.mod and run `go mod tidy`
6. Verify with `go run main.go`

**Agent 3: schema-renaming/**
1. Create directory structure
2. Write specs with legitimately different `Account` schemas
3. Write main.go demonstrating rename strategies and templates
4. Write README.md with expected output
5. Create go.mod and run `go mod tidy`
6. Verify with `go run main.go`

### Phase 2: Documentation Updates

After all agents complete:
1. Update `examples/README.md` - Add new examples to tables and feature matrix
2. Update `examples/workflows/README.md` - Add new workflow entries

### Phase 3: Verification

1. Run `make check` to ensure no issues
2. Verify all examples run successfully
3. Review for consistency with existing example style

## Example Code Patterns

### main.go Structure (collision-resolution)

```go
func main() {
    // Load specs
    paymentsPath := findSpecPath("specs/payments-api.yaml")
    ordersPath := findSpecPath("specs/orders-api.yaml")

    fmt.Println("Collision Resolution Strategies")
    fmt.Println("================================")

    // Demo 1: fail-on-collision (default)
    fmt.Println("\n[1/3] Strategy: fail-on-collision (default)")
    demonstrateFailOnCollision(paymentsPath, ordersPath)

    // Demo 2: accept-left
    fmt.Println("\n[2/3] Strategy: accept-left")
    demonstrateAcceptLeft(paymentsPath, ordersPath)

    // Demo 3: accept-right
    fmt.Println("\n[3/3] Strategy: accept-right")
    demonstrateAcceptRight(paymentsPath, ordersPath)
}
```

### README.md Structure

Each README follows the established pattern:
- What You'll Learn
- Prerequisites
- Quick Start
- Expected Output
- Key Concepts (strategy-specific)
- Use Cases
- Next Steps

## Success Criteria

1. All three examples run without errors
2. Output clearly demonstrates each strategy's behavior
3. READMEs explain when to use each strategy
4. Code follows existing example conventions
5. `make check` passes
6. Feature matrix in examples/README.md updated
