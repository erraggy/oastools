# Schema Renaming

Demonstrates schema renaming strategies for resolving component collisions while preserving both schemas.

## What You'll Learn

- How to preserve both conflicting schemas using rename strategies
- Using `rename-left` vs `rename-right` strategies
- Customizing renamed schema names with Go templates
- Applying namespace prefixes for consistent naming conventions

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/schema-renaming
go run main.go
```

## Expected Output

```
Schema Renaming Strategies
==========================

Scenario: Both APIs legitimately need different 'Account' schemas
  - billing-api.yaml: Account {accountId, balance, creditLimit, paymentTerms}
  - crm-api.yaml: Account {accountId, companyName, industry, employeeCount}

Unlike accept-left/right, we need BOTH schemas preserved!

[1/4] Strategy: rename-right
---------------------------------------------
  Result: Success
  Schemas: [Account Account_crm_api Contact Invoice]

  How it works:
    - billing-api's Account -> Account (kept original name)
    - crm-api's Account -> Account_crm_api (renamed)

  All $refs in crm-api paths now point to Account_crm_api

  Schema properties comparison:
    Account_crm_api: [accountId annualRevenue companyName employeeCount industry]
    Account: [accountId balance creditLimit lastPaymentDate paymentTerms]

[2/4] Strategy: rename-left
---------------------------------------------
  Result: Success
  Schemas: [Account Account_billing_api Contact Invoice]

  How it works:
    - billing-api's Account -> Account_billing_api (renamed)
    - crm-api's Account -> Account (kept original name)

  All $refs in billing-api paths now point to Account_billing_api

[3/4] Custom rename template
---------------------------------------------
  Result: Success
  Schemas: [Account Contact CrmApiAccount Invoice]

  Template: {{.Source | pascalCase}}{{.Name}}

[4/4] Namespace prefixes
---------------------------------------------
  Result: Success
  Schemas: [Account CRM_Account Contact Invoice]

  Configuration:
    NamespacePrefix: billing-api.yaml -> Billing
                     crm-api.yaml -> CRM
    AlwaysApplyPrefix: false (only on collision)

===============================================
Key Takeaway: Rename strategies preserve BOTH schemas.
The joiner automatically rewrites all $ref pointers!
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates all schema renaming strategies |
| specs/billing-api.yaml | Billing API with Account schema (balance, creditLimit) |
| specs/crm-api.yaml | CRM API with Account schema (companyName, industry) |

## Key Concepts

### rename-left vs rename-right

Unlike `accept-left` and `accept-right` which discard one schema, the rename strategies **preserve both schemas** by renaming the colliding one:

| Strategy | Left Schema | Right Schema |
|----------|-------------|--------------|
| `rename-right` | Keeps original name | Renamed using template |
| `rename-left` | Renamed using template | Keeps original name |

The joiner automatically rewrites all `$ref` pointers to use the new schema name.

### RenameTemplate Syntax

The `RenameTemplate` config option uses Go text/template syntax:

```go
config.RenameTemplate = "{{.Source | pascalCase}}{{.Name}}"
```

**Available Variables:**

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.Name}}` | Original schema name | `Account` |
| `{{.Source}}` | Source filename (no extension) | `crm_api` |
| `{{.Index}}` | Document index (0-based) | `1` |

**Available Functions:**

| Function | Output | Example Input -> Output |
|----------|--------|-------------------------|
| `pascalCase` | PascalCase | `crm_api` -> `CrmApi` |
| `camelCase` | camelCase | `crm_api` -> `crmApi` |
| `snakeCase` | snake_case | `CrmApi` -> `crm_api` |
| `kebabCase` | kebab-case | `CrmApi` -> `crm-api` |

**Default Template:** `{{.Name}}_{{.Source}}`

### NamespacePrefix Configuration

For explicit control over renamed schema names, use `NamespacePrefix`:

```go
config.NamespacePrefix = map[string]string{
    "/path/to/billing-api.yaml": "Billing",
    "/path/to/crm-api.yaml":     "CRM",
}
```

When a collision occurs, the schema from the mapped source gets the prefix: `Account` -> `CRM_Account`.

**Note:** Keys must be the full file paths as passed to the joiner, not just basenames.

### AlwaysApplyPrefix Option

| Value | Behavior |
|-------|----------|
| `false` (default) | Only apply prefix on collision |
| `true` | Prefix ALL schemas from mapped sources |

Setting `AlwaysApplyPrefix = true` is useful for:
- Consistent naming across large merges
- Avoiding future collisions when APIs evolve
- Clear provenance of schemas in the merged output

## Use Cases

- **Merging domain APIs** - When different domains legitimately use the same type name (e.g., billing Account vs CRM Account)
- **API versioning** - Preserving both v1 and v2 schemas in a unified spec
- **Multi-tenant APIs** - Combining tenant-specific schemas with unique prefixes
- **Code generation** - Ensuring generated types don't have name conflicts

## Next Steps

- [Multi-API Merge](../multi-api-merge/) - Basic merge workflow with collision handling
- [Collision Resolution](../collision-resolution/) - Compare accept vs rename strategies
- [Schema Deduplication](../schema-deduplication/) - Consolidate identical schemas

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
