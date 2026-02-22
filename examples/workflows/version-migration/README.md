# Version Migration

Demonstrates OAS 3.1/3.2 version conversions and handling lossy downgrades using the converter package.

## What You'll Learn

- OAS version differences (3.0 vs 3.1 vs 3.2)
- Safe upgrades vs lossy downgrades
- How to detect and handle feature loss
- Validating converted specifications

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/version-migration
go run main.go
```

## Expected Output

```
Version Migration: OAS 3.0, 3.1, and 3.2
=========================================

OAS Version Features:
  - 3.0.x: Stable, widely supported
  - 3.1.x: JSON Schema 2020-12, webhooks, type arrays
  - 3.2.x: Latest features and refinements

[1/4] Upgrade: OAS 3.0 -> 3.1
--------------------------------
  Source: classic-api-30.yaml (OAS 3.0.3)
  Target: OAS 3.1.0

  [ok] Converted to OAS 3.1.0
  Conversion issues: 0 critical, 1 warnings, 1 info

  Validating converted spec:
    [ok] Valid!

[2/4] Upgrade: OAS 3.0 -> 3.2 (latest)
--------------------------------
  Source: classic-api-30.yaml (OAS 3.0.3)
  Target: OAS 3.2.0

  [ok] Converted to OAS 3.2.0
  ...

[3/4] Downgrade: OAS 3.1 -> 3.0 (potentially lossy)
--------------------------------
  Source: modern-api-31.yaml (OAS 3.1.0)
  Target: OAS 3.0.3

  Source has 1 webhook(s)
  Source uses JSON Schema dialect: https://json-schema.org/draft/202...

  [ok] Converted to OAS 3.0.3
  [ok] Preserved: 1 webhook(s)
  ...

[4/4] Downgrade: OAS 3.1 -> 2.0 (lossy!)
--------------------------------
  Source: modern-api-31.yaml (OAS 3.1.0)
  Target: OAS 2.0

  Source has 1 webhook(s)
  Source uses JSON Schema dialect: https://json-schema.org/draft/2020-12...

  [ok] Converted to OAS 2.0
  Conversion issues: 1 critical, 0 warnings, 1 info
  Critical issues (features lost):
      [!] webhooks: Webhooks are OAS 3.1+ only and cannot be conver...
  [!] LOST: 1 webhook(s) (not supported in OAS 2.0)
  [!] LOST: components structure (converted to definitions)
  ...
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates version upgrades and lossy downgrades |
| specs/modern-api-31.yaml | OAS 3.1 spec with webhooks, type arrays, JSON Schema 2020-12 |
| specs/classic-api-30.yaml | OAS 3.0.3 spec for upgrade demonstrations |

## Version Comparison

| Feature | OAS 2.0 | OAS 3.0 | OAS 3.1 | OAS 3.2 |
|---------|---------|---------|---------|---------|
| Webhooks | - | - | Yes | Yes |
| Type arrays (`["string", "null"]`) | - | - | Yes | Yes |
| JSON Schema 2020-12 | - | - | Yes | Yes |
| `nullable` keyword | - | Yes | Deprecated | Deprecated |
| `jsonSchemaDialect` | - | - | Yes | Yes |
| `info.summary` | - | - | Yes | Yes |
| `license.identifier` | - | - | Yes | Yes |
| `prefixItems` | - | - | Yes | Yes |
| `contains` | - | - | Yes | Yes |
| `unevaluatedProperties` | - | - | Yes | Yes |
| `contentMediaType` | - | - | Yes | Yes |
| Links & Callbacks | - | Yes | Yes | Yes |
| Components structure | - | Yes | Yes | Yes |
| `$self` document identity | - | - | - | Yes |

## Conversion Paths

### Safe Upgrades

Upgrades preserve all features and may enable new capabilities:

```
3.0.x -> 3.1.x  (gains webhooks, type arrays, JSON Schema keywords)
3.0.x -> 3.2.x  (gains all 3.1 features plus 3.2 additions)
3.1.x -> 3.2.x  (minor refinements)
2.0   -> 3.x    (restructures document, enables modern features)
```

### Lossy Downgrades

Downgrades may lose features that don't exist in older versions:

| Conversion | Features Lost |
|------------|---------------|
| 3.1 -> 3.0 | Type arrays become single types, some JSON Schema keywords |
| 3.1 -> 2.0 | Webhooks, components structure, links, callbacks |
| 3.0 -> 2.0 | Links, callbacks, components structure, servers array |

## Key Concepts

### ConversionResult.Issues

The converter tracks all issues by severity:

| Severity | Meaning | Example |
|----------|---------|---------|
| `Critical` | Feature cannot be converted (data loss) | Webhooks dropped in 2.0 downgrade |
| `Warning` | Best-effort transformation applied | Multiple servers reduced to one |
| `Info` | Informational note | Version string updated |

### Checking for Feature Loss

```go
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("modern-api.yaml"),
    converter.WithTargetVersion("2.0"),
)

// Check for critical issues (lossy conversions)
if result.HasCriticalIssues() {
    for _, issue := range result.Issues {
        if issue.Severity == converter.SeverityCritical {
            fmt.Printf("Feature lost: %s - %s\n", issue.Path, issue.Message)
        }
    }
}
```

### Always Validate After Conversion

```go
// Convert
convResult, _ := converter.ConvertWithOptions(...)

// Validate the result
v := validator.New()
valResult, _ := v.ValidateParsed(*convResult.ToParseResult())

if !valResult.Valid {
    // Handle validation errors
}
```

## Use Cases

### Tool Compatibility

Convert modern specs to older versions for tools that don't support 3.1+:

```go
// Convert 3.1 to 3.0 for older code generators
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("modern-api.yaml"),
    converter.WithTargetVersion("3.0.3"),
)
```

### Legacy System Support

Generate OAS 2.0 for systems that only support Swagger:

```go
// Note: Check for critical issues - webhooks will be lost!
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("api.yaml"),
    converter.WithTargetVersion("2.0"),
)

if result.HasCriticalIssues() {
    log.Println("Warning: Some features were lost in conversion")
}
```

### Spec Modernization

Upgrade older specs to gain new features:

```go
// Upgrade to 3.1 for webhooks and JSON Schema 2020-12
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("legacy-api.yaml"),
    converter.WithTargetVersion("3.1.0"),
)
```

## Next Steps

- [Version Conversion](../version-conversion/) - OAS 2.0 to 3.0 conversion basics
- [Breaking Change Detection](../breaking-change-detection/) - Compare API versions
- [Validate and Fix](../validate-and-fix/) - Auto-fix validation errors

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
