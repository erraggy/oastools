# Implementation Plan: Explicit Parameter Type and Format Support in oastools Builder

## Executive Summary

This implementation plan addresses a gap identified while integrating oastools with go-restful-openapi (PR#125). Currently, the oastools builder package automatically infers OpenAPI `type` and `format` from Go types via reflection. However, there are legitimate use cases where developers need explicit control over these values, particularly when integrating with frameworks like go-restful that expose separate `DataType` and `DataFormat` fields.

The plan introduces `WithParamType()`, `WithParamFormat()`, and `WithParamSchema()` options to the builder package, following established patterns already used for request bodies (`WithRequestBodyRawSchema`).

## Problem Statement

### Current Behavior

The oastools builder automatically maps Go types to OpenAPI types and formats:

| Go Type | OpenAPI Type | OpenAPI Format |
|---------|--------------|----------------|
| `string` | string | - |
| `int`, `int32` | integer | int32 |
| `int64` | integer | int64 |
| `float32` | number | float |
| `float64` | number | double |
| `time.Time` | string | date-time |

When defining a parameter, developers pass a Go value and the type/format are inferred:

```go
// Type: integer, Format: int64 (inferred from int64)
builder.WithQueryParam("user_id", int64(0))

// Type: string, Format: (empty) (inferred from string)
builder.WithQueryParam("uuid", "")
```

### The Gap

There is no way to explicitly set the format for a parameter. For example:

```go
// DESIRED: Type: string, Format: uuid
// ACTUAL: Type: string, Format: (empty)
builder.WithQueryParam("uuid", "")

// DESIRED: Type: string, Format: email  
// ACTUAL: Type: string, Format: (empty)
builder.WithQueryParam("email", "")

// DESIRED: Type: string, Format: date
// ACTUAL: Type: string, Format: date-time (if using time.Time)
builder.WithQueryParam("birth_date", time.Time{})
```

### Impact on go-restful-openapi Integration

The go-restful framework's `ParameterData` struct has explicit fields:

```go
type ParameterData struct {
    Name, Description, DataType, DataFormat string
    // ... other fields
}
```

Developers using go-restful can set these explicitly:

```go
ws.QueryParameter("user_id", "User identifier").
    DataType("string").
    DataFormat("uuid")
```

The legacy `BuildSwagger` function in go-restful-openapi correctly maps `p.Format = param.DataFormat`. However, when using oastools builder via `BuildOAS2` or `BuildOAS3`, there's no way to pass through explicit `DataFormat` values because oastools lacks `WithParamFormat()` and `WithParamType()` options.

The current workaround documented in PR#125 is to use `PostBuildOAS2Handler`/`PostBuildOAS3Handler` to modify the document after building, which is cumbersome and error-prone.

## Proposed Solution

### New Parameter Options

Add three new `ParamOption` functions that follow the established patterns in the builder package.

#### 1. `WithParamType(typeName string) ParamOption`

Explicitly sets the OpenAPI type for a parameter, overriding the type inferred from reflection.

```go
// Force type to "string" even if a different Go type is passed
builder.WithQueryParam("data", []byte{},
    builder.WithParamType("string"),
    builder.WithParamFormat("byte"),
)
```

#### 2. `WithParamFormat(format string) ParamOption`

Explicitly sets the OpenAPI format for a parameter, overriding the format inferred from reflection.

```go
// Set format to "uuid" for a string parameter
builder.WithQueryParam("user_id", "",
    builder.WithParamFormat("uuid"),
)

// Set format to "email" for a string parameter
builder.WithQueryParam("email", "",
    builder.WithParamFormat("email"),
)

// Set format to "date" instead of "date-time" for a time field
builder.WithQueryParam("birth_date", time.Time{},
    builder.WithParamFormat("date"),
)
```

#### 3. `WithParamSchema(schema *parser.Schema) ParamOption`

Provides a pre-built schema for full control, similar to `WithRequestBodyRawSchema`.

```go
// Full schema control for complex cases
builder.WithQueryParam("complex_param", nil,
    builder.WithParamSchema(&parser.Schema{
        Type:   "array",
        Items:  &parser.Schema{Type: "string", Format: "uuid"},
        MinItems: ptr(1),
        MaxItems: ptr(10),
    }),
)
```

### Implementation Details

#### File: `builder/parameter.go`

Add fields to `paramConfig`:

```go
type paramConfig struct {
    // ... existing fields ...
    
    // Type/Format override fields
    typeOverride   string         // Explicit type override (e.g., "string", "integer")
    formatOverride string         // Explicit format override (e.g., "uuid", "email", "date")
    schemaOverride *parser.Schema // Complete schema override (takes precedence)
}
```

Add option functions:

```go
// WithParamType sets an explicit OpenAPI type for the parameter.
// This overrides the type that would be inferred from the Go type.
// 
// Valid types per OpenAPI specification: "string", "integer", "number", 
// "boolean", "array", "object".
//
// Example:
//
//     builder.WithQueryParam("data", []byte{},
//         builder.WithParamType("string"),
//         builder.WithParamFormat("byte"),
//     )
func WithParamType(typeName string) ParamOption {
    return func(cfg *paramConfig) {
        cfg.typeOverride = typeName
    }
}

// WithParamFormat sets an explicit OpenAPI format for the parameter.
// This overrides the format that would be inferred from the Go type.
//
// Common formats include: "int32", "int64", "float", "double", "byte", 
// "binary", "date", "date-time", "password", "email", "uri", "uuid", 
// "hostname", "ipv4", "ipv6".
//
// Example:
//
//     builder.WithQueryParam("user_id", "",
//         builder.WithParamFormat("uuid"),
//     )
func WithParamFormat(format string) ParamOption {
    return func(cfg *paramConfig) {
        cfg.formatOverride = format
    }
}

// WithParamSchema sets a complete schema for the parameter.
// This takes precedence over type/format inference and the 
// WithParamType/WithParamFormat options.
//
// Use this for complex schemas that cannot be easily represented 
// with Go types (e.g., oneOf, arrays with specific item constraints).
//
// Example:
//
//     builder.WithQueryParam("ids", nil,
//         builder.WithParamSchema(&parser.Schema{
//             Type:  "array",
//             Items: &parser.Schema{Type: "string", Format: "uuid"},
//         }),
//     )
func WithParamSchema(schema *parser.Schema) ParamOption {
    return func(cfg *paramConfig) {
        cfg.schemaOverride = schema
    }
}
```

#### File: `builder/operation.go`

Modify the parameter building logic in `AddOperation` to apply overrides:

```go
func (b *Builder) buildParameter(pb *parameterBuilder) *parser.Parameter {
    param := pb.param
    cfg := pb.config
    
    // Determine schema
    var schema *parser.Schema
    
    if cfg != nil && cfg.schemaOverride != nil {
        // Full schema override takes highest precedence
        schema = cfg.schemaOverride
    } else {
        // Generate schema from Go type via reflection
        schema = b.generateSchema(pb.pType)
        
        // Apply type/format overrides if specified
        if cfg != nil {
            if cfg.typeOverride != "" {
                schema = copySchema(schema)
                schema.Type = cfg.typeOverride
            }
            if cfg.formatOverride != "" {
                if schema == copySchema(schema) {
                    schema = copySchema(schema)
                }
                schema.Format = cfg.formatOverride
            }
        }
    }
    
    // Apply constraints to schema
    if cfg != nil && hasParamConstraints(cfg) {
        schema = applyParamConstraintsToSchema(schema, cfg)
    }
    
    // Set schema based on OAS version
    if b.version == parser.OASVersion20 {
        // OAS 2.0: type/format are top-level fields
        param.Type = schema.Type
        param.Format = schema.Format
        applyParamConstraintsToParam(param, cfg)
    } else {
        // OAS 3.x: schema is nested
        param.Schema = schema
    }
    
    return param
}
```

### Go-Restful-OpenAPI Integration

With these new options, the `oastools_builder.go` in go-restful-openapi can be updated to properly map `DataFormat`:

```go
func mapParameter(param restful.ParameterData, config Config) builder.OperationOption {
    var opts []builder.ParamOption
    
    if param.Description != "" {
        opts = append(opts, builder.WithParamDescription(param.Description))
    }
    if param.Required {
        opts = append(opts, builder.WithParamRequired(true))
    }
    
    // Map explicit DataType if different from inferred type
    if param.DataType != "" {
        oasType := mapRestfulTypeToOAS(param.DataType)
        opts = append(opts, builder.WithParamType(oasType))
    }
    
    // Map explicit DataFormat
    if param.DataFormat != "" {
        opts = append(opts, builder.WithParamFormat(param.DataFormat))
    }
    
    // ... rest of parameter mapping
}
```

## Additional Improvements Identified

During analysis of PR#125, several additional improvements were identified that would enhance the go-restful-openapi integration.

### 1. Style and Explode Support

OpenAPI 3.x parameters support `style` and `explode` fields for serialization control. The builder should expose these:

```go
// WithParamStyle sets the serialization style for the parameter.
// Valid styles depend on parameter location:
// - query: "form" (default), "spaceDelimited", "pipeDelimited", "deepObject"
// - path: "simple" (default), "label", "matrix"
// - header: "simple" (default)
// - cookie: "form" (default)
func WithParamStyle(style string) ParamOption {
    return func(cfg *paramConfig) {
        cfg.style = style
    }
}

// WithParamExplode sets whether arrays and objects should be exploded.
// When true, each value is a separate query parameter.
// Default depends on style: true for "form", false for others.
func WithParamExplode(explode bool) ParamOption {
    return func(cfg *paramConfig) {
        cfg.explode = &explode
    }
}
```

### 2. AllowReserved Support

For query parameters, OpenAPI 3.x supports `allowReserved` to permit RFC3986 reserved characters:

```go
// WithParamAllowReserved sets whether reserved characters are allowed.
// Only applicable to query parameters in OAS 3.x.
// When true, characters :/?#[]@!$&'()*+,;= are not percent-encoded.
func WithParamAllowReserved(allow bool) ParamOption {
    return func(cfg *paramConfig) {
        cfg.allowReserved = allow
    }
}
```

### 3. Content Media Type for Complex Parameters

For complex query parameters with JSON encoding, OAS 3.x uses `content` instead of `schema`:

```go
// WithParamContent sets the parameter's content with a media type.
// Use this instead of schema when the parameter requires a specific
// media type encoding (e.g., JSON-encoded query parameter).
//
// Example:
//
//     builder.WithQueryParam("filter", nil,
//         builder.WithParamContent("application/json", FilterSchema{}),
//     )
func WithParamContent(mediaType string, bodyType any) ParamOption {
    return func(cfg *paramConfig) {
        cfg.contentType = mediaType
        cfg.contentSchema = bodyType
    }
}
```

### 4. SchemaFormatHandler Equivalent

go-restful-openapi's legacy API has `Config.SchemaFormatHandler` which allows custom type-to-format mapping:

```go
type MapSchemaFormatFunc func(typeName string) string
```

Consider adding a builder option for this pattern:

```go
// WithTypeFormatMapping registers custom Go type to OpenAPI format mappings.
// This is applied globally during schema generation.
//
// Example:
//
//     spec := builder.New(parser.OASVersion320,
//         builder.WithTypeFormatMapping(func(typeName string) string {
//             switch typeName {
//             case "uuid.UUID":
//                 return "uuid"
//             case "decimal.Decimal":
//                 return "decimal"
//             default:
//                 return ""
//             }
//         }),
//     )
func WithTypeFormatMapping(fn func(typeName string) string) BuilderOption {
    return func(cfg *builderConfig) {
        cfg.typeFormatMapper = fn
    }
}
```

## Implementation Phases

### Phase 1: Core Type/Format Override (Priority: High)

**Scope**: `WithParamType()`, `WithParamFormat()`, `WithParamSchema()`

**Files Modified**:
- `builder/parameter.go` - Add config fields and option functions
- `builder/operation.go` - Modify parameter building logic
- `builder/parameter_test.go` - Unit tests for new options
- `builder/integration_oas2_test.go` - Integration tests for OAS 2.0
- `builder/integration_oas3_test.go` - Integration tests for OAS 3.x

**Estimated Effort**: 2-3 hours

**Acceptance Criteria**:
- `WithParamType("string")` overrides inferred type
- `WithParamFormat("uuid")` overrides inferred format
- `WithParamSchema(schema)` provides complete schema control
- Works correctly for both OAS 2.0 and OAS 3.x
- All existing tests continue to pass
- New tests cover positive, negative, and edge cases

### Phase 2: Serialization Options (Priority: Medium)

**Scope**: `WithParamStyle()`, `WithParamExplode()`, `WithParamAllowReserved()`

**Files Modified**:
- `builder/parameter.go` - Add config fields and option functions
- `builder/operation.go` - Apply serialization options
- `builder/parameter_test.go` - Unit tests

**Estimated Effort**: 1-2 hours

**Acceptance Criteria**:
- Style options are applied for OAS 3.x only
- Proper defaults based on OpenAPI specification
- Validation errors for invalid style/location combinations

### Phase 3: Advanced Features (Priority: Low)

**Scope**: `WithParamContent()`, `WithTypeFormatMapping()`

**Files Modified**:
- `builder/parameter.go` - Add content support
- `builder/options.go` - Add type format mapping builder option
- `builder/reflect.go` - Apply type format mapping during reflection

**Estimated Effort**: 2-3 hours

**Acceptance Criteria**:
- Content-based parameters work for complex JSON query params
- Global type format mapping affects all schema generation

## Test Plan

### Unit Tests

```go
func TestWithParamFormat(t *testing.T) {
    cfg := &paramConfig{}
    WithParamFormat("uuid")(cfg)
    assert.Equal(t, "uuid", cfg.formatOverride)
}

func TestWithParamType(t *testing.T) {
    cfg := &paramConfig{}
    WithParamType("string")(cfg)
    assert.Equal(t, "string", cfg.typeOverride)
}

func TestWithParamSchema(t *testing.T) {
    schema := &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}}
    cfg := &paramConfig{}
    WithParamSchema(schema)(cfg)
    assert.Same(t, schema, cfg.schemaOverride)
}
```

### Integration Tests (OAS 3.x)

```go
func TestBuilder_ExplicitFormat_OAS3(t *testing.T) {
    spec := New(parser.OASVersion320).
        SetTitle("Test API").
        SetVersion("1.0.0").
        AddOperation(http.MethodGet, "/users/{user_id}",
            WithPathParam("user_id", "",
                WithParamFormat("uuid"),
                WithParamDescription("User UUID"),
            ),
            WithQueryParam("email", "",
                WithParamFormat("email"),
            ),
            WithQueryParam("birth_date", "",
                WithParamFormat("date"),
            ),
            WithResponse(http.StatusOK, struct{}{}),
        )

    doc, err := spec.BuildOAS3()
    require.NoError(t, err)

    // Verify path parameter
    params := doc.Paths["/users/{user_id}"].Get.Parameters
    userIdParam := findParam(params, "user_id")
    require.NotNil(t, userIdParam.Schema)
    assert.Equal(t, "string", userIdParam.Schema.Type)
    assert.Equal(t, "uuid", userIdParam.Schema.Format)

    // Verify query parameters
    emailParam := findParam(params, "email")
    assert.Equal(t, "email", emailParam.Schema.Format)

    birthDateParam := findParam(params, "birth_date")
    assert.Equal(t, "date", birthDateParam.Schema.Format)
}

func TestBuilder_ExplicitType_OAS3(t *testing.T) {
    spec := New(parser.OASVersion320).
        SetTitle("Test API").
        SetVersion("1.0.0").
        AddOperation(http.MethodPost, "/upload",
            WithQueryParam("data", []byte{},
                WithParamType("string"),
                WithParamFormat("byte"),
            ),
            WithResponse(http.StatusOK, struct{}{}),
        )

    doc, err := spec.BuildOAS3()
    require.NoError(t, err)

    dataParam := findParam(doc.Paths["/upload"].Post.Parameters, "data")
    require.NotNil(t, dataParam.Schema)
    assert.Equal(t, "string", dataParam.Schema.Type)
    assert.Equal(t, "byte", dataParam.Schema.Format)
}

func TestBuilder_SchemaOverride_OAS3(t *testing.T) {
    schema := &parser.Schema{
        Type:  "array",
        Items: &parser.Schema{Type: "string", Format: "uuid"},
        MinItems: intPtr(1),
        MaxItems: intPtr(10),
    }

    spec := New(parser.OASVersion320).
        SetTitle("Test API").
        SetVersion("1.0.0").
        AddOperation(http.MethodGet, "/items",
            WithQueryParam("ids", nil,
                WithParamSchema(schema),
            ),
            WithResponse(http.StatusOK, struct{}{}),
        )

    doc, err := spec.BuildOAS3()
    require.NoError(t, err)

    idsParam := findParam(doc.Paths["/items"].Get.Parameters, "ids")
    require.NotNil(t, idsParam.Schema)
    assert.Equal(t, "array", idsParam.Schema.Type)
    assert.Equal(t, "string", idsParam.Schema.Items.(*parser.Schema).Type)
    assert.Equal(t, "uuid", idsParam.Schema.Items.(*parser.Schema).Format)
    assert.Equal(t, 1, *idsParam.Schema.MinItems)
    assert.Equal(t, 10, *idsParam.Schema.MaxItems)
}
```

### Integration Tests (OAS 2.0)

```go
func TestBuilder_ExplicitFormat_OAS2(t *testing.T) {
    spec := New(parser.OASVersion20).
        SetTitle("Test API").
        SetVersion("1.0.0").
        AddOperation(http.MethodGet, "/users/{user_id}",
            WithPathParam("user_id", "",
                WithParamFormat("uuid"),
            ),
            WithQueryParam("email", "",
                WithParamFormat("email"),
            ),
            WithResponse(http.StatusOK, struct{}{}),
        )

    doc, err := spec.BuildOAS2()
    require.NoError(t, err)

    params := doc.Paths["/users/{user_id}"].Get.Parameters
    
    // OAS 2.0: type/format are top-level parameter fields
    userIdParam := findParam(params, "user_id")
    assert.Equal(t, "string", userIdParam.Type)
    assert.Equal(t, "uuid", userIdParam.Format)

    emailParam := findParam(params, "email")
    assert.Equal(t, "string", emailParam.Type)
    assert.Equal(t, "email", emailParam.Format)
}
```

### Precedence Tests

```go
func TestBuilder_SchemaTakesPrecedenceOverTypeFormat(t *testing.T) {
    // schemaOverride should take precedence over typeOverride and formatOverride
    schema := &parser.Schema{Type: "number", Format: "decimal"}

    spec := New(parser.OASVersion320).
        SetTitle("Test API").
        SetVersion("1.0.0").
        AddOperation(http.MethodGet, "/test",
            WithQueryParam("amount", int64(0),
                WithParamType("string"),      // Should be ignored
                WithParamFormat("currency"),  // Should be ignored
                WithParamSchema(schema),      // Should win
            ),
            WithResponse(http.StatusOK, struct{}{}),
        )

    doc, err := spec.BuildOAS3()
    require.NoError(t, err)

    amountParam := findParam(doc.Paths["/test"].Get.Parameters, "amount")
    assert.Equal(t, "number", amountParam.Schema.Type)
    assert.Equal(t, "decimal", amountParam.Schema.Format)
}

func TestBuilder_FormatWithoutTypeUsesInferredType(t *testing.T) {
    // formatOverride alone should preserve inferred type
    spec := New(parser.OASVersion320).
        SetTitle("Test API").
        SetVersion("1.0.0").
        AddOperation(http.MethodGet, "/test",
            WithQueryParam("id", "",  // Inferred type: string
                WithParamFormat("uuid"),
            ),
            WithResponse(http.StatusOK, struct{}{}),
        )

    doc, err := spec.BuildOAS3()
    require.NoError(t, err)

    idParam := findParam(doc.Paths["/test"].Get.Parameters, "id")
    assert.Equal(t, "string", idParam.Schema.Type)  // Preserved from inference
    assert.Equal(t, "uuid", idParam.Schema.Format)  // From override
}
```

## Documentation Updates

### `builder/doc.go`

Add documentation for the new options:

```go
// # Explicit Parameter Types and Formats
//
// While the builder automatically infers OpenAPI types and formats from Go types,
// you can explicitly override these when needed:
//
//     // Override format only (type inferred from Go type)
//     builder.WithQueryParam("user_id", "",
//         builder.WithParamFormat("uuid"),
//     )
//
//     // Override both type and format
//     builder.WithQueryParam("data", []byte{},
//         builder.WithParamType("string"),
//         builder.WithParamFormat("byte"),
//     )
//
//     // Full schema control
//     builder.WithQueryParam("ids", nil,
//         builder.WithParamSchema(&parser.Schema{
//             Type:  "array",
//             Items: &parser.Schema{Type: "string", Format: "uuid"},
//         }),
//     )
//
// Precedence rules:
//   1. WithParamSchema takes highest precedence
//   2. WithParamType overrides inferred type
//   3. WithParamFormat overrides inferred format
//   4. Constraints are applied after type/format resolution
```

### `builder/deep_dive.md`

Add a section on explicit type/format control:

```markdown
## Explicit Parameter Types and Formats

The builder infers OpenAPI types and formats from Go types, but you can 
override these when integrating with other frameworks or when you need 
specific format values like "uuid" or "email".

### Format Override

Use `WithParamFormat` to set a specific format while keeping the inferred type:

| Use Case | Code | Result |
|----------|------|--------|
| UUID identifier | `WithParamFormat("uuid")` | `type: string, format: uuid` |
| Email address | `WithParamFormat("email")` | `type: string, format: email` |
| Date only | `WithParamFormat("date")` | `type: string, format: date` |
| URI | `WithParamFormat("uri")` | `type: string, format: uri` |

### Type Override

Use `WithParamType` when you need to change the inferred type:

| Use Case | Code | Result |
|----------|------|--------|
| Base64 data | `WithParamType("string"), WithParamFormat("byte")` | `type: string, format: byte` |
| Binary data | `WithParamType("string"), WithParamFormat("binary")` | `type: string, format: binary` |

### Full Schema Control

For complex cases, use `WithParamSchema` with a complete schema:

```go
WithParamSchema(&parser.Schema{
    Type:     "array",
    Items:    &parser.Schema{Type: "string", Format: "uuid"},
    MinItems: intPtr(1),
    MaxItems: intPtr(100),
})
```
```

## Backward Compatibility

These changes are fully backward compatible:

1. **Existing code continues to work unchanged** - The default behavior (type/format inference) remains when no override options are provided.

2. **No breaking changes to existing APIs** - New options are additive and optional.

3. **Consistent with existing patterns** - The new options follow the same functional options pattern used throughout the builder package (`WithParamDescription`, `WithParamRequired`, etc.).

## Conclusion

This implementation plan provides a clear path to adding explicit type and format control to the oastools builder package. The changes are minimal, focused, and follow established patterns. Phase 1 (core type/format override) addresses the immediate need for go-restful-openapi integration, while Phases 2 and 3 provide optional enhancements for more advanced use cases.

The implementation maintains full backward compatibility and requires no changes to existing user code unless they want to use the new explicit override capabilities.
