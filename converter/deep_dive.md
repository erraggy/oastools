<a id="top"></a>

# Converter Package Deep Dive

The converter package provides version conversion for OpenAPI Specification documents, supporting bidirectional conversion between OAS 2.0 and OAS 3.x.

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Conversion Details](#conversion-details)
- [Overlay Integration](#overlay-integration)
- [Configuration Reference](#configuration-reference)
- [Best Practices](#best-practices)

---

## Overview

The converter performs best-effort conversion with detailed issue tracking. Features converted include servers, schemas, parameters, security schemes, and request/response bodies. It preserves the input file format (JSON or YAML) for output consistency.

**Supported conversions:**
- OAS 2.0 (Swagger) → OAS 3.0.x / 3.1.x
- OAS 3.0.x / 3.1.x → OAS 2.0 (Swagger)

[Back to top](#top)

---

## Key Concepts

### Issue Severity Levels

| Severity | Description |
|----------|-------------|
| 🔵 **Info** | Conversion choices and decisions made |
| 🟡 **Warning** | Lossy conversions where data may be simplified |
| 🔴 **Critical** | Features that cannot be converted |

### What Cannot Convert

**OAS 3.x → OAS 2.0:**
- Webhooks (3.1+ only)
- Callbacks
- Links
- TRACE HTTP method
- Cookie parameters

**OAS 2.0 → OAS 3.x:**
- `collectionFormat` (may not map perfectly)
- `allowEmptyValue` (deprecated in 3.x)

[Back to top](#top)

---

## API Styles

### Functional Options (Recommended)

```go
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)
if err != nil {
    log.Fatal(err)
}
```

### Struct-Based (Reusable)

```go
c := converter.New()
c.StrictMode = false

result1, _ := c.Convert("api1.yaml", "3.0.3")
result2, _ := c.Convert("api2.yaml", "3.0.3")
```

### Pre-Parsed Documents

```go
parseResult, _ := parser.ParseWithOptions(parser.WithFilePath("swagger.yaml"))

result, _ := converter.ConvertWithOptions(
    converter.WithParsed(*parseResult),
    converter.WithTargetVersion("3.0.3"),
)
```

[Back to top](#top)

---

## Practical Examples

### Basic Conversion

```go
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Converted to OAS %s\n", result.TargetVersion)
```

### Handling Conversion Issues

```go
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("api.yaml"),
    converter.WithTargetVersion("3.0.3"),
)

if result.HasCriticalIssues() {
    fmt.Printf("🔴 %d critical issues\n", result.CriticalCount)
}
if result.HasWarnings() {
    fmt.Printf("🟡 %d warnings\n", result.WarningCount)
}

for _, issue := range result.Issues {
    fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Location, issue.Message)
}
```

### Writing Output

```go
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
)

// Write preserving original format
data, _ := result.Marshal() // Uses result.SourceFormat
os.WriteFile("openapi.yaml", data, 0644)
```

[Back to top](#top)

---

## Conversion Details

### OAS 2.0 → OAS 3.0

| OAS 2.0 | OAS 3.0 |
|---------|---------|
| `host`, `basePath`, `schemes` | `servers` array |
| `definitions` | `components.schemas` |
| `parameters` | `components.parameters` |
| `responses` | `components.responses` |
| `securityDefinitions` | `components.securitySchemes` |
| `consumes` + body param | `requestBody.content` |
| `produces` + schema | `response.content` |

### OAS 3.0 → OAS 2.0

| OAS 3.0 | OAS 2.0 |
|---------|---------|
| `servers[0]` | `host`, `basePath`, `schemes` |
| `components.schemas` | `definitions` |
| `requestBody` | `consumes` + body parameter |
| `webhooks` | ❌ Dropped (critical issue) |
| `callbacks` | ❌ Dropped (critical issue) |
| `links` | ❌ Dropped (critical issue) |

[Back to top](#top)

---

## Overlay Integration

Apply transformations before or after conversion:

```go
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("swagger.yaml"),
    converter.WithTargetVersion("3.0.3"),
    converter.WithPreConversionOverlayFile("fix-v2.yaml"),   // Fix v2-specific issues
    converter.WithPostConversionOverlayFile("enhance.yaml"), // Add v3 extensions
)
```

**Use cases:**
- **Pre-conversion:** Normalize or fix the source document before conversion
- **Post-conversion:** Add version-specific extensions to the result

[Back to top](#top)

---

## Configuration Reference

### Functional Options

| Option | Description |
|--------|-------------|
| `WithFilePath(path)` | Path to specification file |
| `WithParsed(result)` | Pre-parsed ParseResult |
| `WithTargetVersion(v)` | Target OAS version (e.g., "3.0.3", "2.0") |
| `WithStrictMode(bool)` | Fail on critical issues |
| `WithIncludeInfo(bool)` | Include info-level issues |
| `WithPreConversionOverlayFile(path)` | Overlay to apply before conversion |
| `WithPostConversionOverlayFile(path)` | Overlay to apply after conversion |

### Converter Fields

| Field | Type | Description |
|-------|------|-------------|
| `StrictMode` | `bool` | Return error on critical issues |
| `IncludeInfo` | `bool` | Include info-level issues in result |

### ConversionResult Fields

| Field | Type | Description |
|-------|------|-------------|
| `Document` | `any` | Converted document |
| `TargetVersion` | `string` | Target OAS version |
| `Issues` | `[]ConversionIssue` | All conversion issues |
| `CriticalCount` | `int` | Number of critical issues |
| `WarningCount` | `int` | Number of warnings |
| `InfoCount` | `int` | Number of info items |
| `SourceFormat` | `SourceFormat` | Preserved format |

[Back to top](#top)

---

## Best Practices

1. **Always check issues** - Use `HasCriticalIssues()` and review warnings
2. **Validate after conversion** - The converted document may have structural issues
3. **Review critical issues** - They indicate features that couldn't be converted
4. **Use overlays for fixes** - Pre/post overlays can address conversion gaps
5. **Preserve format** - Use `result.Marshal()` to maintain JSON/YAML consistency

[Back to top](#top)
