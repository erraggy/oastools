<a id="top"></a>

# Converter Package Deep Dive

!!! tip "Try it Online"
    No installation required! [Try the converter in your browser →](https://oastools.robnrob.com/convert)

The [`converter`](https://pkg.go.dev/github.com/erraggy/oastools/converter) package provides version conversion for OpenAPI Specification documents, supporting bidirectional conversion between OAS 2.0 and OAS 3.x.

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Conversion Details](#conversion-details)
- [Version-Specific Considerations](#version-specific-considerations)
- [Common Pitfalls and Solutions](#common-pitfalls-and-solutions)
- [Loss of Fidelity](#loss-of-fidelity)
- [Overlay Integration](#overlay-integration)
- [Configuration Reference](#configuration-reference)
- [Best Practices](#best-practices)

---

## Overview

The converter performs best-effort conversion with detailed issue tracking. Features converted include servers, schemas, parameters, security schemes, and request/response bodies. It preserves the input file format (JSON or YAML) for output consistency.

**Supported conversions:**
- OAS 2.0 (Swagger) -> OAS 3.0.x / 3.1.x
- OAS 3.0.x / 3.1.x -> OAS 2.0 (Swagger)

[Back to top](#top)

---

## Key Concepts

### Issue Severity Levels

| Severity | Description |
|----------|-------------|
| Info | Conversion choices and decisions made |
| Warning | Lossy conversions where data may be simplified |
| Critical | Features that cannot be converted |

### Conversion Philosophy

The converter follows these principles:

1. **Best-effort conversion**: Convert as much as possible, track what cannot be converted
2. **Transparency**: Every conversion decision is recorded as an issue
3. **Reversibility awareness**: Some conversions are lossy and cannot be reversed
4. **Version detection**: Automatically detect source version and validate target version

### What Cannot Convert

**OAS 3.x -> OAS 2.0:**
- Webhooks (3.1+ only)
- Callbacks
- Links
- TRACE HTTP method
- Cookie parameters
- Multiple servers (only first is used)
- Content negotiation complexity

**OAS 2.0 -> OAS 3.x:**
- `collectionFormat` (may not map perfectly to `style`/`explode`)
- `allowEmptyValue` (deprecated in 3.x)
- File upload patterns differ significantly

[Back to top](#top)

---

## API Styles

See also: [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/converter#example-package), [Handling issues example](https://pkg.go.dev/github.com/erraggy/oastools/converter#example-package-HandleConversionIssues), [Complex conversion example](https://pkg.go.dev/github.com/erraggy/oastools/converter#example-package-ComplexConversion) on pkg.go.dev

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

### OAS 2.0 to OAS 3.0 Conversion

This is the most common conversion scenario - upgrading legacy Swagger specs:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/erraggy/oastools/converter"
)

func main() {
    // Convert Swagger 2.0 to OpenAPI 3.0.3
    result, err := converter.ConvertWithOptions(
        converter.WithFilePath("swagger.yaml"),
        converter.WithTargetVersion("3.0.3"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Check for issues
    fmt.Printf("Conversion complete: %d info, %d warnings, %d critical\n",
        result.InfoCount, result.WarningCount, result.CriticalCount)

    // Review any warnings or critical issues
    for _, issue := range result.Issues {
        if issue.Severity != "info" {
            fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Location, issue.Message)
        }
    }

    // Write the result
    data, _ := result.Marshal()
    os.WriteFile("openapi.yaml", data, 0644)
}
```

**Example Input (swagger.yaml):**
```yaml
swagger: "2.0"
info:
  title: Pet Store API
  version: "1.0.0"
host: api.petstore.io
basePath: /v1
schemes:
  - https
consumes:
  - application/json
produces:
  - application/json
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: limit
          in: query
          type: integer
          format: int32
      responses:
        200:
          description: A list of pets
          schema:
            type: array
            items:
              $ref: '#/definitions/Pet'
    post:
      operationId: createPet
      parameters:
        - name: body
          in: body
          required: true
          schema:
            $ref: '#/definitions/NewPet'
      responses:
        201:
          description: Pet created
          schema:
            $ref: '#/definitions/Pet'
definitions:
  Pet:
    type: object
    required:
      - id
      - name
    properties:
      id:
        type: integer
        format: int64
      name:
        type: string
      tag:
        type: string
  NewPet:
    type: object
    required:
      - name
    properties:
      name:
        type: string
      tag:
        type: string
```

**Generated Output (openapi.yaml):**
```yaml
openapi: 3.0.3
info:
  title: Pet Store API
  version: "1.0.0"
servers:
  - url: https://api.petstore.io/v1
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
            format: int32
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewPet'
      responses:
        '201':
          description: Pet created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
        tag:
          type: string
    NewPet:
      type: object
      required:
        - name
      properties:
        name:
          type: string
        tag:
          type: string
```

### OAS 3.0 to OAS 3.1 Conversion

Upgrading to take advantage of JSON Schema alignment:

```go
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("openapi-3.0.yaml"),
    converter.WithTargetVersion("3.1.0"),
)
if err != nil {
    log.Fatal(err)
}

// 3.1 enables nullable via type arrays
// nullable: true becomes type: ["string", "null"]
fmt.Printf("Converted to %s\n", result.TargetVersion)
```

**Key Changes in 3.0 -> 3.1:**
- `nullable: true` is converted to type arrays: `type: ["string", "null"]`
- JSON Schema keywords like `unevaluatedProperties` become available
- Webhooks support is added

### OAS 3.x to OAS 2.0 Downgrade

When you need to support older tooling:

```go
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("openapi.yaml"),
    converter.WithTargetVersion("2.0"),
    converter.WithStrictMode(false), // Allow conversion despite critical issues
)
if err != nil {
    log.Fatal(err)
}

// IMPORTANT: Check for critical issues - features that couldn't convert
if result.HasCriticalIssues() {
    fmt.Println("WARNING: Some features could not be converted:")
    for _, issue := range result.Issues {
        if issue.Severity == "critical" {
            fmt.Printf("  - %s: %s\n", issue.Location, issue.Message)
        }
    }
}

data, _ := result.Marshal()
os.WriteFile("swagger.yaml", data, 0644)
```

### Handling Conversion Issues

```go
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("api.yaml"),
    converter.WithTargetVersion("3.0.3"),
    converter.WithIncludeInfo(true), // Include info-level issues for full visibility
)

// Categorize issues by type
var schemaIssues, pathIssues, securityIssues []converter.ConversionIssue

for _, issue := range result.Issues {
    switch {
    case strings.Contains(issue.Location, "schemas"):
        schemaIssues = append(schemaIssues, issue)
    case strings.Contains(issue.Location, "paths"):
        pathIssues = append(pathIssues, issue)
    case strings.Contains(issue.Location, "security"):
        securityIssues = append(securityIssues, issue)
    }
}

fmt.Printf("Schema issues: %d\n", len(schemaIssues))
fmt.Printf("Path issues: %d\n", len(pathIssues))
fmt.Printf("Security issues: %d\n", len(securityIssues))
```

### Batch Conversion

Converting multiple files with consistent settings:

```go
c := converter.New()
c.StrictMode = false
c.IncludeInfo = false // Only warnings and critical

files := []string{"api1.yaml", "api2.yaml", "api3.yaml"}
var totalCritical int

for _, file := range files {
    result, err := c.Convert(file, "3.0.3")
    if err != nil {
        log.Printf("Failed to convert %s: %v", file, err)
        continue
    }

    totalCritical += result.CriticalCount

    // Write output with matching extension
    outFile := strings.TrimSuffix(file, ".yaml") + "-v3.yaml"
    data, _ := result.Marshal()
    os.WriteFile(outFile, data, 0644)

    fmt.Printf("Converted %s: %d warnings, %d critical\n",
        file, result.WarningCount, result.CriticalCount)
}

fmt.Printf("\nTotal critical issues across all files: %d\n", totalCritical)
```

[Back to top](#top)

---

## Conversion Details

### OAS 2.0 -> OAS 3.0

| OAS 2.0 | OAS 3.0 | Notes |
|---------|---------|-------|
| `host`, `basePath`, `schemes` | `servers` array | Combined into URL template |
| `definitions` | `components.schemas` | Reference paths updated |
| `parameters` | `components.parameters` | Reference paths updated |
| `responses` | `components.responses` | Reference paths updated |
| `securityDefinitions` | `components.securitySchemes` | OAuth flows restructured |
| `consumes` + body param | `requestBody.content` | Media types explicit |
| `produces` + schema | `response.content` | Media types explicit |
| `type: file` | Binary string + format | `type: string, format: binary` |
| `collectionFormat` | `style` + `explode` | Mapping varies by format |

**Server URL Construction:**

The converter combines OAS 2.0's separate fields into OAS 3.0's servers array:

```yaml
# OAS 2.0
host: api.example.com
basePath: /v1
schemes:
  - https
  - http

# Converts to OAS 3.0
servers:
  - url: https://api.example.com/v1
  - url: http://api.example.com/v1
```

**Request Body Extraction:**

Body parameters are extracted and converted to requestBody:

```yaml
# OAS 2.0
parameters:
  - name: body
    in: body
    required: true
    schema:
      $ref: '#/definitions/Pet'

# Converts to OAS 3.0
requestBody:
  required: true
  content:
    application/json:  # From consumes
      schema:
        $ref: '#/components/schemas/Pet'
```

### OAS 3.0 -> OAS 2.0

| OAS 3.0 | OAS 2.0 | Notes |
|---------|---------|-------|
| `servers[0]` | `host`, `basePath`, `schemes` | Only first server used |
| `components.schemas` | `definitions` | Reference paths updated |
| `requestBody` | `consumes` + body parameter | Single media type selected |
| `webhooks` | Dropped | Critical issue logged |
| `callbacks` | Dropped | Critical issue logged |
| `links` | Dropped | Critical issue logged |
| `cookie` parameters | Dropped | Critical issue logged |
| TRACE method | Dropped | Critical issue logged |

**Server URL Decomposition:**

```yaml
# OAS 3.0
servers:
  - url: https://api.example.com/v1
  - url: http://staging.example.com/v2  # Ignored with warning

# Converts to OAS 2.0
host: api.example.com
basePath: /v1
schemes:
  - https
```

### OAS 3.0 -> OAS 3.1

| OAS 3.0 | OAS 3.1 | Notes |
|---------|---------|-------|
| `nullable: true` | `type: ["string", "null"]` | Type becomes array |
| `example` | `examples` (preferred) | Can use either |
| N/A | `webhooks` | Now available |
| `exclusiveMinimum: true` | `exclusiveMinimum: <value>` | JSON Schema alignment |

### OAS 3.1 -> OAS 3.0

| OAS 3.1 | OAS 3.0 | Notes |
|---------|---------|-------|
| `type: ["string", "null"]` | `type: string` + `nullable: true` | Array to boolean |
| `webhooks` | Dropped | Critical issue logged |
| `$comment` | Dropped | JSON Schema keyword |
| `unevaluatedProperties` | Dropped | JSON Schema keyword |

[Back to top](#top)

---

## Version-Specific Considerations

### Converting to OAS 3.0.x

**Choose Your Patch Version:**
- `3.0.0` - Initial release, use for maximum compatibility
- `3.0.1` - Clarifications only
- `3.0.2` - More clarifications
- `3.0.3` - **Recommended** - Most stable and widely supported

**Watch For:**
- Security schemes with OAuth2 flows need careful mapping
- Form data and file uploads have different patterns than 2.0
- Global consumes/produces become per-operation content types

### Converting to OAS 3.1.x

**JSON Schema Alignment:**
OAS 3.1 fully aligns with JSON Schema Draft 2020-12. This means:

- `type` can be an array: `type: ["string", "null"]`
- `nullable` is deprecated in favor of type arrays
- New keywords available: `unevaluatedProperties`, `prefixItems`, etc.

**Webhooks:**
OAS 3.1+ introduces webhooks:

```yaml
webhooks:
  newPet:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '200':
          description: Webhook processed
```

**Choose Your Patch Version:**
- `3.1.0` - Initial JSON Schema alignment
- `3.1.1` - Bug fixes and clarifications

### Downgrading to OAS 2.0

**Feature Loss:**
Expect to lose these OAS 3.x features:
- Webhooks (cannot be represented)
- Callbacks (cannot be represented)
- Links (cannot be represented)
- Cookie parameters (not supported)
- Multiple content types per request/response (simplified)

**Best Practices:**
1. Always check `HasCriticalIssues()` after conversion
2. Review all critical issues to understand what was lost
3. Consider if the target tooling truly requires 2.0
4. Document the conversion limitations for API consumers

[Back to top](#top)

---

## Common Pitfalls and Solutions

### Pitfall 1: Ignoring Conversion Issues

**Problem:** Converting without checking the result for issues.

```go
// WRONG: Ignoring issues
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("api.yaml"),
    converter.WithTargetVersion("2.0"),
)
data, _ := result.Marshal()
os.WriteFile("swagger.yaml", data, 0644)
// Webhooks, callbacks, links silently dropped!
```

**Solution:** Always check for issues, especially critical ones:

```go
// CORRECT: Check issues
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("api.yaml"),
    converter.WithTargetVersion("2.0"),
)
if err != nil {
    log.Fatal(err)
}

if result.HasCriticalIssues() {
    log.Printf("WARNING: %d features could not be converted", result.CriticalCount)
    for _, issue := range result.Issues {
        if issue.Severity == "critical" {
            log.Printf("  %s: %s", issue.Location, issue.Message)
        }
    }
}
```

### Pitfall 2: Strict Mode for Downgrades

**Problem:** Using strict mode when downgrading from 3.x to 2.0.

```go
// WRONG: Strict mode fails on any critical issue
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("modern-api.yaml"), // Has webhooks
    converter.WithTargetVersion("2.0"),
    converter.WithStrictMode(true),
)
// Error: conversion has critical issues
```

**Solution:** Disable strict mode for downgrades, handle issues manually:

```go
// CORRECT: Allow conversion, check issues
result, err := converter.ConvertWithOptions(
    converter.WithFilePath("modern-api.yaml"),
    converter.WithTargetVersion("2.0"),
    converter.WithStrictMode(false),
)
if err != nil {
    log.Fatal(err)
}

// Now check what was lost
for _, issue := range result.Issues {
    if issue.Severity == "critical" {
        log.Printf("Feature lost: %s", issue.Message)
    }
}
```

### Pitfall 3: Assuming Reference Paths Are Updated

**Problem:** Assuming only schema refs are updated.

All component references are updated during conversion:

```yaml
# OAS 2.0 refs
$ref: '#/definitions/Pet'
$ref: '#/parameters/LimitParam'
$ref: '#/responses/NotFound'

# After conversion to OAS 3.0
$ref: '#/components/schemas/Pet'
$ref: '#/components/parameters/LimitParam'
$ref: '#/components/responses/NotFound'
```

The converter handles this automatically, but be aware when processing results.

### Pitfall 4: Multiple Content Types

**Problem:** OAS 3.x allows multiple content types per operation; OAS 2.0 doesn't.

```yaml
# OAS 3.0
requestBody:
  content:
    application/json:
      schema: {...}
    application/xml:
      schema: {...}
    text/plain:
      schema: {...}
```

When downgrading to 2.0, only one content type is preserved (typically `application/json`). A warning issue is logged.

**Solution:** Review warnings and ensure the selected content type is appropriate:

```go
for _, issue := range result.Issues {
    if strings.Contains(issue.Message, "content type") {
        log.Printf("Content type selection: %s", issue.Message)
    }
}
```

### Pitfall 5: OAuth Flow Differences

**Problem:** OAuth2 flows have different structures in 2.0 vs 3.0.

```yaml
# OAS 2.0
securityDefinitions:
  oauth2:
    type: oauth2
    flow: accessCode  # Single flow
    authorizationUrl: https://auth.example.com/authorize
    tokenUrl: https://auth.example.com/token
    scopes:
      read: Read access

# OAS 3.0
components:
  securitySchemes:
    oauth2:
      type: oauth2
      flows:  # Multiple flows possible
        authorizationCode:  # Renamed from 'accessCode'
          authorizationUrl: https://auth.example.com/authorize
          tokenUrl: https://auth.example.com/token
          scopes:
            read: Read access
```

The converter handles the flow name mapping (`accessCode` <-> `authorizationCode`, etc.).

[Back to top](#top)

---

## Loss of Fidelity

Understanding what information is lost during conversion is crucial for making informed decisions.

### OAS 3.x -> OAS 2.0 (Significant Loss)

| Feature | Impact | Mitigation |
|---------|--------|------------|
| Webhooks | Complete loss | Document externally or use extensions |
| Callbacks | Complete loss | Document externally |
| Links | Complete loss | Document relationships externally |
| Cookie params | Complete loss | Use header params if possible |
| Multiple servers | Only first used | Document others externally |
| Multiple content types | First used | Ensure JSON is first if preferred |
| TRACE method | Dropped | Use custom extension if needed |

### OAS 2.0 -> OAS 3.0 (Minimal Loss)

| Feature | Impact | Mitigation |
|---------|--------|------------|
| `collectionFormat` | Mapped to style/explode | Verify serialization behavior |
| `allowEmptyValue` | Deprecated in 3.x | Behavior preserved if set |
| File type | Becomes binary string | Functionally equivalent |

### OAS 3.0 <-> OAS 3.1 (Semantic Only)

| Feature | Impact | Mitigation |
|---------|--------|------------|
| `nullable` vs type array | Semantic equivalence | Both work in most tools |
| JSON Schema keywords | Available in 3.1 only | Document requirements |

### Measuring Fidelity Loss

```go
result, _ := converter.ConvertWithOptions(
    converter.WithFilePath("api.yaml"),
    converter.WithTargetVersion("2.0"),
    converter.WithIncludeInfo(true),
)

// Calculate fidelity score
totalFeatures := result.InfoCount + result.WarningCount + result.CriticalCount
if totalFeatures > 0 {
    fidelity := 1.0 - (float64(result.CriticalCount) / float64(totalFeatures))
    fmt.Printf("Conversion fidelity: %.1f%%\n", fidelity*100)
}

// Categorize losses
var losses = map[string]int{}
for _, issue := range result.Issues {
    if issue.Severity == "critical" {
        losses[issue.Location]++
    }
}
fmt.Printf("Features lost by location: %v\n", losses)
```

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

**Use Cases:**

### Pre-Conversion Overlays

Fix issues in the source document before conversion:

```yaml
# fix-v2.yaml
overlay: 1.0.0
info:
  title: Fix OAS 2.0 Issues
actions:
  - target: $.info
    update:
      contact:
        email: api@example.com
  - target: $.paths./legacy-endpoint
    remove: true  # Remove deprecated endpoint before conversion
```

### Post-Conversion Overlays

Add OAS 3.x specific enhancements:

```yaml
# enhance.yaml
overlay: 1.0.0
info:
  title: Add OAS 3.0 Enhancements
actions:
  - target: $.servers
    update:
      - url: https://api.example.com/v3
        description: Production
      - url: https://staging.example.com/v3
        description: Staging
  - target: $.components.schemas.Pet
    update:
      x-oai-display-name: Pet Object
```

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

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `StrictMode` | `bool` | `false` | Return error on critical issues |
| `IncludeInfo` | `bool` | `false` | Include info-level issues in result |

### ConversionResult Fields

| Field | Type | Description |
|-------|------|-------------|
| `Document` | `any` | Converted document (*OAS2Document or *OAS3Document) |
| `TargetVersion` | `string` | Target OAS version string |
| `Issues` | `[]ConversionIssue` | All conversion issues |
| `CriticalCount` | `int` | Number of critical issues |
| `WarningCount` | `int` | Number of warnings |
| `InfoCount` | `int` | Number of info items |
| `SourceFormat` | `SourceFormat` | Original format (JSON/YAML) |

### ConversionIssue Fields

| Field | Type | Description |
|-------|------|-------------|
| `Severity` | `string` | "info", "warning", or "critical" |
| `Location` | `string` | JSON path to affected element |
| `Message` | `string` | Human-readable description |
| `Code` | `string` | Machine-readable issue code |

[Back to top](#top)

---

## Best Practices

1. **Always check issues** - Use `HasCriticalIssues()` and review warnings before using converted documents in production.

2. **Validate after conversion** - The converted document may have structural issues that the converter cannot detect. Run through the validator:
   ```go
   result, _ := converter.ConvertWithOptions(...)
   parseResult := &parser.ParseResult{Document: result.Document, ...}
   valResult, _ := validator.ValidateWithOptions(validator.WithParsed(*parseResult))
   ```

3. **Review critical issues** - Critical issues indicate features that couldn't be converted. Document these for API consumers.

4. **Use overlays for fixes** - Pre/post conversion overlays can address gaps that the converter cannot handle automatically.

5. **Preserve format** - Use `result.Marshal()` to maintain JSON/YAML consistency with the source document.

6. **Test round-trip conversions** - If you need bidirectional compatibility, test converting A->B->A and verify the result.

7. **Document version requirements** - If your API requires 3.1+ features (webhooks, JSON Schema keywords), document this for consumers.

8. **Use appropriate target versions**:
   - For maximum compatibility: `3.0.3` or `2.0`
   - For latest features: `3.1.0` or `3.2.0`
   - For JSON Schema alignment: `3.1.0+`

9. **Handle nullable correctly** - When converting 3.1 -> 3.0, verify that `nullable: true` is set where expected.

10. **Consider tooling compatibility** - Some tools don't support 3.1+ yet. Check your toolchain before upgrading.

[Back to top](#top)

---

## Learn More

For additional examples and complete API documentation:

- [API Reference on pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools/converter) - Complete API documentation with all examples
- [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/converter#example-package) - Convert OAS 2.0 to OAS 3.x
- [Handling issues example](https://pkg.go.dev/github.com/erraggy/oastools/converter#example-package-HandleConversionIssues) - Process conversion issues by severity
- [Complex conversion example](https://pkg.go.dev/github.com/erraggy/oastools/converter#example-package-ComplexConversion) - Advanced conversion scenarios
