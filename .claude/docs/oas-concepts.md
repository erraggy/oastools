# OpenAPI Specification Concepts

## Supported Versions

- **OAS 2.0** (Swagger): https://spec.openapis.org/oas/v2.0.html
- **OAS 3.0.x**: https://spec.openapis.org/oas/v3.0.0.html through v3.0.4
- **OAS 3.1.x**: https://spec.openapis.org/oas/v3.1.0.html through v3.1.2
- **OAS 3.2.0**: https://spec.openapis.org/oas/v3.2.0.html

All OAS versions utilize **JSON Schema Specification Draft 2020-12**: https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html

## OAS Version Evolution

### OAS 2.0 (Swagger) → OAS 3.0
- **Servers**: `host`, `basePath`, and `schemes` → unified `servers` array with URL templates
- **Components**: `definitions`, `parameters`, `responses`, `securityDefinitions` → `components.*`
- **Request Bodies**: `consumes` + body parameter → `requestBody.content` with media types
- **Response Bodies**: `produces` + schema → `responses.*.content` with media types
- **Security**: `securityDefinitions` → `components.securitySchemes` with flows restructuring
- **New Features**: Links, callbacks, and more flexible parameter serialization

### OAS 3.0 → OAS 3.1
- **JSON Schema Alignment**: OAS 3.1 fully aligns with JSON Schema Draft 2020-12
- **Type Arrays**: `type` can be a string or array (e.g., `type: ["string", "null"]`)
- **Nullable Handling**: Deprecated `nullable: true` in favor of `type: ["string", "null"]`
- **Webhooks**: New top-level `webhooks` object for event-driven APIs
- **License**: Added `identifier` field to license object

## Version-Specific Features

**OAS 2.0 Only:**
- `allowEmptyValue`, `collectionFormat`, single `host`/`basePath`/`schemes`

**OAS 3.0+ Only:**
- `requestBody`, `callbacks`, `links`, cookie parameters, `servers` array, TRACE method

**OAS 3.1+ Only:**
- `webhooks`, JSON Schema 2020-12 alignment, `type` as array, `license.identifier`

**OAS 3.2+ Only:**
- `$self` (document identity), `Query` method, `additionalOperations`, `components.mediaTypes`

**JSON Schema 2020-12 Keywords (OAS 3.1+):**
- `unevaluatedProperties`, `unevaluatedItems` - strict validation of uncovered properties/items
- `contentEncoding`, `contentMediaType`, `contentSchema` - encoded content validation

## Critical Type System Considerations

### interface{} Fields
Several OAS 3.1+ fields use `interface{}` to support multiple types. Always use type assertions:
```go
if typeStr, ok := schema.Type.(string); ok {
    // Handle string type
} else if typeArr, ok := schema.Type.([]string); ok {
    // Handle array type
}
```

### Pointer vs Value Types
- `OAS3Document.Servers` uses `[]*parser.Server` (slice of pointers)
- Always use `&parser.Server{...}` for pointer semantics
- This pattern applies to other nested structures to avoid unexpected mutations

## Common Pitfalls and Solutions

1. **Assuming schema.Type is always a string** - Use type assertions and handle both string and []string cases
2. **Creating value slices instead of pointer slices** - Check parser types and use `&Type{...}` syntax
3. **Forgetting to track conversion issues** - Add issues for every lossy conversion or unsupported feature
4. **Mutating source documents** - Always deep copy before modification using generated `DeepCopy()` methods (e.g., `doc.DeepCopy()`). Never use JSON marshal/unmarshal — it loses `interface{}` type distinctions and drops `json:"-"` fields
5. **Not handling operation-level consumes/produces** - Check operation-level first, then fall back to document-level
6. **Ignoring version-specific features during conversion** - Explicitly check and warn about features that don't convert
7. **Confusing Version (string) with OASVersion (enum)** - `ParseResult` has TWO version fields:
   - `Version` (string): The literal version string from the document (e.g., `"3.0.3"`, `"2.0"`)
   - `OASVersion` (parser.OASVersion enum): Our canonical enum for each published spec version

### OASVersion Constants
See `parser/versions.go`:
- `OASVersion20` - OpenAPI 2.0 (Swagger)
- `OASVersion300`, `OASVersion301`, `OASVersion302`, `OASVersion303`, `OASVersion304` - OpenAPI 3.0.x
- `OASVersion310`, `OASVersion311`, `OASVersion312` - OpenAPI 3.1.x
- `OASVersion320` - OpenAPI 3.2.0

### ParseResult in Tests
**ALWAYS set both fields when constructing ParseResult:**
```go
parseResult := parser.ParseResult{
    Version:    "3.0.0",               // String from document
    OASVersion: parser.OASVersion300,  // Our enum - REQUIRED for validation
    Document:   &parser.OAS3Document{...},
}
```
The validator uses `OASVersion` to determine which validation rules to apply. Setting only `Version` will cause "unsupported OAS version: unknown" errors.
