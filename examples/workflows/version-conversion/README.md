# Version Conversion

Demonstrates converting between OpenAPI specification versions using the converter package.

## What You'll Learn

- How to convert Swagger 2.0 to OpenAPI 3.0.3
- Understanding conversion issues and their severity levels
- Key structural changes between OAS 2.0 and 3.x

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/version-conversion
go run main.go
```

## Expected Output

```
Version Conversion Workflow
===========================

Input: specs/swagger-v2.yaml

[1/3] Converting OAS 2.0 -> OAS 3.0.3...
      Source: 2.0
      Target: 3.0.3

[2/3] Conversion Issues:
      Critical: 0
      Warnings: 0
      Info: 3

      Details:
        [INFO] (document): host/basePath/schemes converted to servers
        [INFO] (document): definitions converted to components/schemas
        [INFO] (document): securityDefinitions converted to components/securitySchemes

[3/3] Key Conversions Applied:
      - host/basePath/schemes -> servers array
      - definitions -> components/schemas
      - consumes/produces -> requestBody/response content
      - securityDefinitions -> components/securitySchemes
      - body parameters -> requestBody objects

--- Converted Specification (excerpt) ---
openapi: 3.0.3
info:
  title: Legacy User API
  version: 1.0.0
  description: A Swagger 2.0 API to convert to OpenAPI 3.x
servers:
  - url: https://api.example.com/v1
...

---
Conversion completed successfully
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates OAS 2.0 to 3.x conversion workflow |
| specs/swagger-v2.yaml | Swagger 2.0 source specification |

## Key Concepts

### Severity Levels

| Level | Meaning |
|-------|---------|
| CRITICAL | Conversion may have lost essential functionality |
| WARNING | Some features may not work as expected |
| INFO | Informational note about transformation |

### Major Structural Changes

**OAS 2.0 → OAS 3.x:**

| OAS 2.0 | OAS 3.x |
|---------|---------|
| `host`, `basePath`, `schemes` | `servers` array with URL templates |
| `definitions` | `components/schemas` |
| `securityDefinitions` | `components/securitySchemes` |
| `consumes`, `produces` | `requestBody.content`, `response.content` |
| `body` parameter | `requestBody` object |

### Bidirectional Conversion

The converter supports both directions:
- OAS 2.0 → 3.0.x, 3.1.x, 3.2.0
- OAS 3.x → 2.0 (with potential information loss)

## Next Steps

- [Converter Deep Dive](../../../packages/converter/) - Complete converter documentation
- [Breaking Change Detection](../breaking-change-detection/) - Compare API versions
- [Multi-API Merge](../multi-api-merge/) - Merge multiple specifications

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
