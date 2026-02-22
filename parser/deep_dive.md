<a id="top"></a>

# Parser Package Deep Dive

The [`parser`](https://pkg.go.dev/github.com/erraggy/oastools/parser) package provides parsing for OpenAPI Specification documents, supporting OAS 2.0 through OAS 3.2.0 in YAML and JSON formats.

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [OAS 3.2.0 Features](#oas-320-features)
- [JSON Schema 2020-12 Support](#json-schema-2020-12-support)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Configuration Reference](#configuration-reference)
- [Document Type Helpers](#document-type-helpers)
- [Version-Agnostic Access (DocumentAccessor)](#version-agnostic-access-documentaccessor)
- [Order-Preserving Marshaling](#order-preserving-marshaling)
- [Best Practices](#best-practices)

---

## Overview

The parser can load specifications from local files or remote URLs, resolve external references (`$ref`), validate structure, and preserve unknown fields for forward compatibility. It automatically detects the file format (JSON/YAML) and OAS version.

**Key capabilities:**

- Parse files, URLs, readers, or byte slices
- Resolve local and external `$ref` references
- Detect and handle circular references safely
- Enforce configurable resource limits
- Preserve source format for downstream tools

[Back to top](#top)

---

## Key Concepts

### Format Detection

The parser automatically detects format from:

1. File extension (`.json`, `.yaml`, `.yml`)
2. Content inspection (JSON starts with `{` or `[`)
3. Defaults to YAML if unknown

### Reference Resolution

External `$ref` values are resolved when `ResolveRefs` is enabled:

| Reference Type | Example | Security |
|---------------|---------|----------|
| Local | `#/components/schemas/User` | Always allowed |
| File | `./common.yaml#/schemas/Error` | Path traversal protected |
| HTTP/HTTPS | `https://example.com/schemas.yaml` | Opt-in via `WithResolveHTTPRefs` |

### Circular Reference Handling

When circular references are detected:

- The `$ref` node is left unresolved (preserves the `"$ref"` key)
- A warning is added to `result.Warnings`
- The document remains valid for most operations

Detection triggers:

- A `$ref` points to an ancestor in the current resolution path
- Resolution depth exceeds `MaxRefDepth` (default: 100)

### Resource Limits

| Limit | Default | Description |
|-------|---------|-------------|
| `MaxRefDepth` | 100 | Maximum nested `$ref` resolution depth |
| `MaxCachedDocuments` | 100 | Maximum external documents to cache |
| `MaxFileSize` | 10MB | Maximum file size for external references |

[Back to top](#top)

---

## OAS 3.2.0 Features

OAS 3.2.0 introduces several new capabilities that the parser fully supports:

### Document Identity ($self)

The `$self` field provides a canonical URL for the document:

```go
result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
doc, _ := result.OAS3Document()

if doc.Self != "" {
    fmt.Printf("Document identity: %s\n", doc.Self)
}
```

### Additional HTTP Methods (additionalOperations)

Custom HTTP methods beyond the standard set can be defined via `additionalOperations`:

```go
pathItem := doc.Paths["/resource"]
for method, op := range pathItem.AdditionalOperations {
    fmt.Printf("Custom method %s: %s\n", method, op.OperationID)
}

// Use GetOperations to get all operations including custom methods
allOps := parser.GetOperations(pathItem, parser.OASVersion320)
```

### Reusable Media Types (components/mediaTypes)

Media type definitions can be defined once and referenced:

```go
if doc.Components != nil && doc.Components.MediaTypes != nil {
    for name, mediaType := range doc.Components.MediaTypes {
        fmt.Printf("Media type %s: %v\n", name, mediaType.Schema)
    }
}
```

### QUERY Method

OAS 3.2.0 adds native support for the QUERY HTTP method:

```go
if pathItem.Query != nil {
    fmt.Printf("QUERY operation: %s\n", pathItem.Query.OperationID)
}
```

[Back to top](#top)

---

## JSON Schema 2020-12 Support

The parser supports all JSON Schema Draft 2020-12 keywords used in OAS 3.1+:

### Content Keywords

For schemas representing encoded content:

| Keyword | Type | Description |
|---------|------|-------------|
| `contentEncoding` | `string` | Encoding (e.g., "base64", "base32") |
| `contentMediaType` | `string` | Media type of decoded content |
| `contentSchema` | `*Schema` | Schema for decoded content |

```go
schema := doc.Components.Schemas["EncodedData"]
if schema.ContentEncoding != "" {
    fmt.Printf("Encoding: %s, Media type: %s\n",
        schema.ContentEncoding, schema.ContentMediaType)
}
```

### Unevaluated Keywords

For strict validation of object and array schemas:

| Keyword | Type | Description |
|---------|------|-------------|
| `unevaluatedProperties` | `any` | `*Schema`, `bool`, or `map[string]any` for uncovered properties |
| `unevaluatedItems` | `any` | `*Schema`, `bool`, or `map[string]any` for uncovered array items |

```go
schema := doc.Components.Schemas["StrictObject"]
switch v := schema.UnevaluatedProperties.(type) {
case *parser.Schema:
    // Typed schema - most common after parsing
    fmt.Printf("Unevaluated properties must match: %s\n", v.Ref)
case bool:
    // Boolean value - false disallows, true allows any
    fmt.Printf("Unevaluated properties allowed: %v\n", v)
case map[string]any:
    // Raw map - when schema wasn't typed during parsing
    if ref, ok := v["$ref"].(string); ok {
        fmt.Printf("Raw ref to: %s\n", ref)
    }
default:
    // nil or unexpected type
    fmt.Println("No unevaluatedProperties constraint")
}
```

### Array Index References

JSON Pointer references now support array indices per RFC 6901:

```yaml
# Example: Reference the first parameter's schema
$ref: '#/paths/~1users/get/parameters/0/schema'
```

The resolver handles:

- Valid indices: `0`, `1`, `2`, etc.
- Out-of-bounds errors with descriptive messages
- Non-numeric index errors

[Back to top](#top)

---

## API Styles

See also: [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package), [Functional options example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-FunctionalOptions), [Reusable parser example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-ReusableParser) on pkg.go.dev

### Functional Options (Recommended)

```go
result, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithValidateStructure(true),
    parser.WithResolveRefs(true),
)
if err != nil {
    log.Fatal(err)
}
```

### Struct-Based (Reusable)

```go
p := parser.New()
p.ResolveRefs = false
p.ValidateStructure = true

result1, _ := p.Parse("api1.yaml")
result2, _ := p.Parse("api2.yaml")
```

### Alternative Input Sources

```go
// From URL
result, _ := parser.ParseWithOptions(
    parser.WithFilePath("https://example.com/api/openapi.yaml"),
)

// From reader
result, _ := p.ParseReader(reader, "config.yaml")

// From bytes
result, _ := p.ParseBytes(data, "inline.yaml")
```

[Back to top](#top)

---

## Practical Examples

### Basic File Parsing

```go
result, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
)
if err != nil {
    log.Fatal(err)
}
if len(result.Errors) > 0 {
    fmt.Printf("Parse errors: %d\n", len(result.Errors))
}
fmt.Printf("Parsed %s v%s\n", result.Version, result.OASVersion)
```

### HTTP Reference Resolution

See also: [HTTP refs example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-ParseWithHTTPRefs), [Parse from URL example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-ParseFromURL) on pkg.go.dev

```go
result, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithResolveHTTPRefs(true),      // Enable HTTP refs
    parser.WithInsecureSkipVerify(true),   // For self-signed certs
)
```

### Custom Resource Limits

```go
result, err := parser.ParseWithOptions(
    parser.WithFilePath("large-api.yaml"),
    parser.WithMaxRefDepth(50),
    parser.WithMaxCachedDocuments(200),
    parser.WithMaxFileSize(20*1024*1024), // 20MB
)
```

### Safe Document Mutation with DeepCopy

See also: [DeepCopy example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-DeepCopy) on pkg.go.dev

```go
result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))

// Get the typed document
doc, _ := result.OAS3Document()

// Deep copy before mutation
docCopy := doc.DeepCopy()
docCopy.Info.Title = "Modified API"

// Original unchanged
fmt.Println(doc.Info.Title) // Original title
```

[Back to top](#top)

---

## Configuration Reference

### Functional Options

| Option | Description |
|--------|-------------|
| `WithFilePath(path)` | File path or URL to parse |
| `WithBytes(data []byte)` | Parse from byte slice |
| `WithReader(r io.Reader)` | Parse from an io.Reader |
| `WithResolveRefs(bool)` | Enable `$ref` resolution (default: true) |
| `WithResolveHTTPRefs(bool)` | Enable HTTP/HTTPS ref resolution (default: false) |
| `WithValidateStructure(bool)` | Validate document structure during parsing |
| `WithInsecureSkipVerify(bool)` | Skip TLS verification for HTTPS refs |
| `WithSourceMap(enabled bool)` | Enable source map tracking for line/column info |
| `WithPreserveOrder(enabled bool)` | Preserve original field ordering from source |
| `WithUserAgent(ua string)` | Custom User-Agent for HTTP requests |
| `WithHTTPClient(client *http.Client)` | Custom HTTP client for remote refs |
| `WithMaxRefDepth(n)` | Max nested ref depth (default: 100) |
| `WithMaxCachedDocuments(n)` | Max cached external docs (default: 100) |
| `WithMaxFileSize(n)` | Max file size in bytes (default: 10MB) |
| `WithMaxInputSize(size int)` | Max input size in bytes |
| `WithSourceName(name string)` | Override source name for bytes/reader input |

### ParseResult Fields

| Field | Type | Description |
|-------|------|-------------|
| `Document` | `any` | Parsed document (OAS2Document or OAS3Document) |
| `Version` | `string` | Raw version string from document |
| `OASVersion` | `OASVersion` | Parsed version constant |
| `SourceFormat` | `SourceFormat` | Detected format (JSON or YAML) |
| `SourcePath` | `string` | Original file path |
| `Errors` | `[]error` | Parse errors |
| `Warnings` | `[]string` | Non-fatal warnings |

[Back to top](#top)

---

## Document Type Helpers

See also: [Document type helpers example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-DocumentTypeHelpers) on pkg.go.dev

ParseResult provides convenient methods for version checking and type assertion:

```go
result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))

// Version checking
if result.IsOAS2() {
    fmt.Println("This is a Swagger 2.0 document")
}
if result.IsOAS3() {
    fmt.Println("This is an OAS 3.x document")
}

// Safe type assertion
if doc, ok := result.OAS3Document(); ok {
    fmt.Printf("API: %s v%s\n", doc.Info.Title, doc.Info.Version)
}
if doc, ok := result.OAS2Document(); ok {
    fmt.Printf("Swagger: %s v%s\n", doc.Info.Title, doc.Info.Version)
}
```

[Back to top](#top)

---

## Version-Agnostic Access (DocumentAccessor)

See also: [DocumentAccessor example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-DocumentAccessor) on pkg.go.dev

For code that needs to work uniformly across both OAS 2.0 and 3.x documents without type switches, use the `DocumentAccessor` interface:

```go
result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
if accessor := result.AsAccessor(); accessor != nil {
    // These methods work identically for both versions
    for path := range accessor.GetPaths() {
        fmt.Println("Path:", path)
    }

    // GetSchemas() abstracts the difference:
    // - OAS 2.0: returns doc.Definitions
    // - OAS 3.x: returns doc.Components.Schemas
    for name := range accessor.GetSchemas() {
        fmt.Println("Schema:", name)
    }

    // Get the $ref prefix for schema references
    fmt.Println("Prefix:", accessor.SchemaRefPrefix())
}
```

### DocumentAccessor Methods

| Method | OAS 2.0 Source | OAS 3.x Source |
|--------|---------------|----------------|
| `GetInfo()` | `doc.Info` | `doc.Info` |
| `GetPaths()` | `doc.Paths` | `doc.Paths` |
| `GetSchemas()` | `doc.Definitions` | `doc.Components.Schemas` |
| `GetSecuritySchemes()` | `doc.SecurityDefinitions` | `doc.Components.SecuritySchemes` |
| `GetParameters()` | `doc.Parameters` | `doc.Components.Parameters` |
| `GetResponses()` | `doc.Responses` | `doc.Components.Responses` |
| `SchemaRefPrefix()` | `#/definitions/` | `#/components/schemas/` |

[Back to top](#top)

---

## Order-Preserving Marshaling

The parser can preserve original field ordering from source documents, enabling deterministic output for hash-based caching and diff-friendly serialization.

### Why It Matters

1. **Hash stability**: When caching parsed specs by content hash, roundtrip through parse-then-marshal should produce identical output. Without preserved order, map iteration order causes non-deterministic output.

2. **Diff-friendly**: Editing and re-serializing specs should minimize diffs. Alphabetical reordering of all keys makes diffs noisy and hard to review.

3. **Human readability**: Authors typically place important fields like `openapi`, `info`, and `paths` at the top. Preserving this order maintains the document's logical structure.

### How It Works

When `WithPreserveOrder(true)` is enabled:

1. **Source tree storage**: The parser stores the original `yaml.Node` tree alongside the typed document
2. **Key order extraction**: During marshal, keys are extracted from source nodes in original order
3. **Extra key handling**: Keys added during processing (not in source) are sorted alphabetically and appended
4. **Performance**: O(n) with hash-based indexing for child node lookup

### When to Use It

| Use Case | Recommendation |
|----------|----------------|
| Hash-based caching | Enable - ensures roundtrip identity |
| CI pipelines comparing output | Enable - deterministic output |
| Version control of specs | Enable - cleaner diffs |
| One-off validation | Disable - lower memory overhead |
| Programmatic construction | N/A - no source order available |

### Code Examples

**Parsing with order preservation:**

```go
result, err := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithPreserveOrder(true),
)
if err != nil {
    log.Fatal(err)
}

// Check if order information is available
if result.HasPreservedOrder() {
    fmt.Println("Order was preserved from source")
}
```

**JSON output with preserved order:**

```go
// Compact JSON
jsonBytes, err := result.MarshalOrderedJSON()

// Indented JSON
jsonIndented, err := result.MarshalOrderedJSONIndent("", "  ")
```

**YAML output with preserved order:**

```go
yamlBytes, err := result.MarshalOrderedYAML()
```

### Fallback Behavior

When `PreserveOrder` is not enabled (or for programmatically constructed documents), the ordered marshal methods fall back to standard marshaling:

- **JSON**: Uses `encoding/json` which sorts map keys alphabetically
- **YAML**: Uses `go.yaml.in/yaml/v4` which also sorts keys alphabetically

This ensures deterministic output in all cases, just without preserving the original order.

### Memory Overhead

Enabling `PreserveOrder` stores an additional `*yaml.Node` tree in the `ParseResult`. For typical API specs:

| Spec Size | Approximate Overhead |
|-----------|---------------------|
| Small (<1KB) | ~2-5KB |
| Medium (10-50KB) | ~20-100KB |
| Large (>100KB) | ~200KB+ |

For most use cases, this overhead is negligible compared to the benefits of deterministic output.

### Limitations

- Only works when parsing from source (file, bytes, reader)
- Not available for documents constructed programmatically via the builder package
- Source node structure must match parsed document structure for correct ordering

[Back to top](#top)

---

## Best Practices

1. **Parse once, use many** - Cache ParseResult for operations like validate, convert, diff
2. **Use pre-parsed methods** - `ValidateParsed()`, `ConvertParsed()`, etc. are 9-150x faster
3. **Check warnings for circular refs** - They indicate unresolved references
4. **Enable HTTP refs carefully** - Only for trusted sources; use `WithInsecureSkipVerify` sparingly
5. **Use DeepCopy for mutations** - Never modify the original parsed document

[Back to top](#top)

---

## Learn More

For additional examples and complete API documentation:

- 📦 [API Reference on pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools/parser) - Complete API documentation with all examples
- 🔧 [Functional options example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-FunctionalOptions) - Configure parsing with options
- 🌐 [HTTP refs example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-ParseWithHTTPRefs) - Resolve external HTTP references
- 📋 [DeepCopy example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-DeepCopy) - Safe document mutation
- 🔍 [Type helpers example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-DocumentTypeHelpers) - Version checking and type assertions
- 🔀 [DocumentAccessor example](https://pkg.go.dev/github.com/erraggy/oastools/parser#example-package-DocumentAccessor) - Version-agnostic document access
