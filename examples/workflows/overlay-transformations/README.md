# Overlay Transformations

Demonstrates applying OpenAPI Overlay transformations using the overlay package.

## What You'll Learn

- How to parse and validate overlay documents
- Using JSONPath expressions to target specific nodes
- Preview changes with dry-run mode
- Apply environment-specific customizations

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/overlay-transformations
go run main.go
```

## Expected Output

```
Overlay Transformations Workflow
=================================

Base Spec: base.yaml
Overlay: production.yaml

[1/4] Validating overlay document...
      Overlay is valid
      Title: Production Environment Overlay
      Actions defined: 5

[2/4] Parsing base specification...
      Version: 3.0.3

[3/4] Previewing changes (dry-run)...
      Would apply: 5 action(s)
      Would skip: 0 action(s)

      Changes:
        - update 1 node(s) at $.info
        - update 1 node(s) at $.servers[0]
        - remove 1 node(s) at $.paths['/internal/metrics']
        - remove 1 node(s) at $.paths['/internal/health']
        - update 4 node(s) at $.paths.*.get.responses.200

[4/4] Applying overlay...
      Actions applied: 5
      Actions skipped: 0

--- Transformation Results ---
New Title: Payment API (Production)
Environment: production
Production URL: https://api.payments.example.com/v1
Paths (after removing internal): 2

Remaining paths:
  - /payments
  - /payments/{paymentId}

---
Overlay applied successfully
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates the overlay transformation workflow |
| specs/base.yaml | Base OpenAPI specification with internal endpoints |
| specs/production.yaml | Overlay document for production customizations |

## Key Concepts

### Overlay Document Structure

```yaml
overlay: "1.0.0"
info:
  title: Production Environment Overlay
  version: "1.0.0"
actions:
  - target: <JSONPath expression>
    update: <value>  # or remove: true
```

### JSONPath Expressions

| Expression | Matches |
|------------|---------|
| `$.info` | The info object |
| `$.servers[0]` | First server |
| `$.paths.*` | All path items |
| `$.paths.*.get` | All GET operations |
| `$..description` | All descriptions at any depth |
| `$.paths['/internal/health']` | Path with special characters (bracket notation) |
| `$.paths[?@.x-internal==true]` | Paths with x-internal: true |

> **Note:** Paths containing slashes or special characters must use bracket notation with quotes, e.g., `$.paths['/internal/health']`.

### Action Types

| Action | Purpose |
|--------|---------|
| `update` | Merge values into target nodes |
| `remove: true` | Delete target nodes |

### Use Cases

- **Environment customization** - Different servers, rate limits, descriptions
- **Internal filtering** - Remove internal endpoints for public docs
- **Security additions** - Add authentication requirements
- **Metadata injection** - Add tracking, versioning, or governance info

## Next Steps

- [Overlay Deep Dive](https://erraggy.github.io/oastools/packages/overlay/) - Complete overlay documentation
- [OpenAPI Overlay Spec](https://spec.openapis.org/overlay/v1.0.0.html) - Official specification
- [HTTP Validation](../http-validation/) - Validate requests against the spec

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
