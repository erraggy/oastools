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

### Unknown Fields Differ by Format

Every parser struct carries an `Extra` map tagged ``yaml:",inline" json:"-"``, and
the two halves of that tag do not do the same job:

| Format | Unknown field | Result |
|---|---|---|
| YAML | any key the struct does not model | **preserved** — the inline map captures every unknown key |
| JSON | `x-` extension | **preserved** — repopulated by `ExtractExtensions` |
| JSON | anything else | **dropped** — `ExtractExtensions` keeps only `x-` prefixed keys |

So a field oastools does not model survives a YAML round trip and is lost on a
JSON one. YAML preserving it is incidental rather than designed: the inline map is
simply indiscriminate.

This matters when a document uses a specification field newer than the parser. All
OAS 2.0 through 3.2.0 fixed fields are modeled, so a conformant document is not
affected today — but a field added by a future OAS revision will round trip in
YAML and disappear from JSON until it is modeled here. If you are relying on
oastools to round trip a document containing fields it does not model, use YAML.

Note that `Extra` holds only what the struct does not model; a field with a Go
field of its own round trips identically in both formats.

### Discriminator Dialects

OAS 2.0 spells a Schema Object's discriminator as a bare string naming the
property; OAS 3.0+ uses an object with `propertyName` and an optional `mapping`.
Both decode into `*Discriminator`, and `StringForm` records which spelling the
document used:

```go
pet := doc.Definitions["Pet"]          // OAS 2.0: discriminator: petType
pet.Discriminator.PropertyName          // "petType"
pet.Discriminator.StringForm            // true
```

Marshaling reproduces the form it was given, so a 2.0 document is not silently
rewritten into the 3.x object form on a round trip. `Mapping` and any `x-*`
extensions have no representation in the string form and are dropped when
`StringForm` is set.

The parser accepts either dialect regardless of version, because a Schema Object
is decoded before the document version is known to it. Rejecting the form that is
wrong for the version is the validator's job, and re-spelling it for a target
version is the converter's.

### Callback References

A `callbacks` entry is either a [Callback Object](https://spec.openapis.org/oas/v3.2.0.html#callback-object)
or a [Reference Object](https://spec.openapis.org/oas/v3.2.0.html#reference-object).
Both positions that hold one are typed `Map[string, Callback Object | Reference Object]`:
[Operation](https://spec.openapis.org/oas/v3.2.0.html#operation-callbacks) and
[Components](https://spec.openapis.org/oas/v3.2.0.html#components-callbacks).
The [JSON Schema](https://spec.openapis.org/oas/3.2/schema/2025-09-17) tells the
two apart by the presence of a `$ref` key (`callbacks-or-reference` is
`if: {required: [$ref]}, then: reference, else: callbacks`), and declares nine
such unions in all.

Eight of the nine are objects with fixed field names, so each is modeled here as a
struct with a `Ref` field. The Callback Object is the exception: it is an open map
keyed by user-authored runtime expressions, so `Callback` is a map type, and a map
type has no field to hold `$ref`.

The reference form therefore lands on a parallel field, on both `Operation` and
`Components`:

```go
op.Callbacks     // map[string]*Callback:  the Callback Object form
op.CallbackRefs  // map[string]*Reference: the Reference Object form
```

```yaml
callbacks:
  onPetCreated:                                      # → Callbacks
    '{$request.query.callbackUrl}': { post: ... }
  onPetShipped:                                      # → CallbackRefs
    $ref: '#/components/callbacks/shipped'
```

A name appears in one map or the other, never both. Marshaling merges them back
into the single `callbacks` object, so a document round trips with the reference
verbatim; a value assembled in Go that puts one name in both maps is a marshaling
error rather than a silent choice between the forms.

**Reading only `Callbacks` misses the referenced ones.** Anything counting
callbacks, or collecting the components a document depends on, has to read both.
The `walker` package inherits the split: it reports the reference form to its ref
handler as a `callback` node type rather than to its callback handler, so a
traversal registering only the callback handler has the same gap. See that
package's deep dive for the handler pairing.

Enabling `ResolveRefs` replaces a resolvable callback reference with what it
points at, the way it does for every other reference form, and `CallbackRefs` is
then left holding only the references that could not be resolved.

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

The parser models the full set of fixed fields OAS 3.2.0 added over 3.1.1. The
sections below cover the ones with behavior worth explaining; the complete list is
in `parser/doc.go`. Fields added in v1.59.0 include Tag `summary`/`parent`/`kind`,
Server `name`, Response `summary`, the Media Type and Encoding sequential-media-type
fields, Example `dataValue`/`serializedValue`, the Security Scheme and OAuth device
authorization fields, Discriminator `defaultMapping`, XML `nodeType`, and Parameter
`in: "querystring"`.

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

### Boolean Schemas

A schema may be written as a bare boolean rather than an object. `true` accepts
every instance and `false` accepts none, and both are legal anywhere a Schema
Object is expected:

```yaml
components:
  schemas:
    Anything: true
    Nothing: false
    Unconstrained: {}       # an object schema with no keywords, NOT `true`
```

`Schema` is a struct, so the boolean is recorded on a field of its own and read
back through a method rather than by comparing against a value:

```go
value, isBool := doc.Components.Schemas["Anything"].IsBool()
// value == true, isBool == true

value, isBool = doc.Components.Schemas["Unconstrained"].IsBool()
// value == false, isBool == false
```

The two returns are independent, and conflating them is the mistake to avoid:
`(false, false)` is an ordinary object schema, while `(false, true)` is the
boolean schema `false`, which accepts nothing. Construct one with
`NewBoolSchema(true)`.

An empty object and `true` are different schemas, and marshaling reproduces the
form it was given so a round trip does not silently swap one for the other. Every
other field is meaningless alongside the boolean form and is dropped when it is
set, since a boolean schema has no keywords by definition.

The form is valid only for OAS 3.1+, which adopt the 2020-12 dialect. The parser
accepts it at any version, because a Schema Object is decoded before the document
version is known to it; the validator rejects it for 3.0 and 2.0.

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
| `unevaluatedProperties` | `any` | `*Schema` or `bool` for uncovered properties |
| `unevaluatedItems` | `any` | `*Schema` or `bool` for uncovered array items |

Every decode path promotes these to `*Schema`, so a parsed document never leaves
a raw `map[string]any` here — only a hand-constructed one can. Two cases suffice:

```go
schema := doc.Components.Schemas["StrictObject"]
switch v := schema.UnevaluatedProperties.(type) {
case *parser.Schema:
    // Typed schema
    fmt.Printf("Unevaluated properties must match: %s\n", v.Ref)
case bool:
    // Boolean value - false disallows, true allows any
    fmt.Printf("Unevaluated properties allowed: %v\n", v)
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
