# Breaking Change Detection

Demonstrates comparing API versions and detecting breaking changes using the differ package.

## What You'll Learn

- How to compare two API versions programmatically
- Understanding severity levels for API changes
- Using diff results for CI/CD pipeline gates
- Interpreting change categories (endpoint, parameter, schema)

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/breaking-change-detection
go run main.go
```

## Expected Output

```
Breaking Change Detection Workflow
===================================

Comparing:
  Source (old): v1.yaml
  Target (new): v2.yaml

[1/3] Analyzing changes...
      Source Version: 3.0.3
      Target Version: 3.0.3
      Total Changes: 12

[2/3] Change Summary:
      Breaking (Error+Critical): 5
      Warnings: 3
      Info: 4

[3/3] Detailed Changes:

      parameter:
        [ERROR] required changed from false to true
             at document.paths./products.get.parameters[category:query].required

      schema:
        [ERROR] maximum constraint changed
             at document.paths./products.get.parameters[limit:query].schema.maximum
        [ERROR] schema type changed
             at document.paths./products/{productId}.get.parameters[productId:path].schema.type
        [ERROR] required field "sku" added
             at document.components.schemas.Product.required[sku]
        [WARNING] property "inStock" removed
             at document.components.schemas.Product.properties.inStock
        [WARNING] property "sku" added
             at document.components.schemas.Product.properties.sku
        [INFO] schema "Review" added
             at document.components.schemas.Review

      response:
        [WARNING] response code 404 removed
             at document.paths./products/{productId}.get.responses[404]

      operation:
        [ERROR] operation delete removed
             at document.paths./products/{productId}.delete

      endpoint:
        [INFO] endpoint "/products/{productId}/reviews" added
             at document.paths./products/{productId}/reviews

      info:
        [INFO] API version changed from "1.0.0" to "2.0.0"
             at document.info.version
        [INFO] description changed
             at document.info.description

---
BREAKING CHANGES DETECTED: 5

Recommendations:
  - Consider incrementing major version
  - Update API documentation
  - Notify API consumers
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates the breaking change detection workflow |
| specs/v1.yaml | Original API version |
| specs/v2.yaml | Updated API version with breaking changes |

## Key Concepts

### Severity Levels

| Level | Meaning | CI Action |
|-------|---------|-----------|
| CRITICAL | API consumers **will** break | Block deployment |
| ERROR | API consumers **likely** to break | Block deployment |
| WARNING | API consumers **may** be affected | Review required |
| INFO | Non-breaking additions | Safe to deploy |

### Breaking Changes in v2

| Change | Category | Severity |
|--------|----------|----------|
| DELETE endpoint removed | endpoint | CRITICAL |
| Parameter made required | parameter | ERROR |
| Parameter type changed | parameter | ERROR |
| Required field added | schema | ERROR |
| Field removed | schema | ERROR |
| Maximum reduced | parameter | WARNING |

### Non-Breaking Changes

| Change | Category | Severity |
|--------|----------|----------|
| New endpoint added | endpoint | INFO |
| New schema added | schema | INFO |

### CI/CD Integration

Use the exit code for pipeline gates:
- Exit 0: No breaking changes, safe to deploy
- Exit 1: Breaking changes detected, block deployment

```yaml
# Example GitHub Actions step
- name: Check for breaking changes
  run: |
    go run examples/workflows/breaking-change-detection/main.go
```

## Next Steps

- [Differ Deep Dive](../../packages/differ/) - Complete differ documentation
- [Breaking Changes Guide](../../breaking-changes/) - Detailed explanation
- [Multi-API Merge](../multi-api-merge/) - Merge API versions

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
