# Feature Request: Operation-Level Consumes/Produces Support

## Summary

Add builder methods to support operation-level `consumes`/`produces` (OAS 2.0) and multi-content-type request bodies/responses (OAS 3.x).

## Background

### Use Case: go-restful Integration

The [go-restful](https://github.com/emicklei/go-restful) framework allows specifying multiple MIME types per route:

```go
ws.Route(ws.POST("/users").
    Consumes("application/json", "application/xml", "text/yaml").
    Produces("application/json", "application/xml").
    To(createUser).
    Reads(User{}).
    Returns(201, "Created", User{}))
```

When generating OpenAPI specs from go-restful routes, we need to preserve **all** content types, not just the first one.

### Current Limitation

The current builder API only supports a single content type per request body or response:

```go
// Only one content type can be specified
builder.WithRequestBody("application/json", User{})
builder.WithResponse(200, User{}, builder.WithResponseContentType("application/json"))
```

### OAS 2.0 vs OAS 3.x Differences

| Version | Representation |
|---------|----------------|
| **OAS 2.0** | Operation has `consumes: []string` and `produces: []string` arrays |
| **OAS 3.x** | Request body has `content: map[string]MediaType`; Responses have `content: map[string]MediaType` |

## Proposed API

### New Operation Options

#### `WithConsumes` (OAS 2.0)

Sets operation-level consumes array. Ignored for OAS 3.x (use request body content types instead).

```go
func WithConsumes(mimeTypes ...string) OperationOption
```

**Usage:**
```go
builder.AddOperation("POST", "/users",
    builder.WithConsumes("application/json", "application/xml"),
    builder.WithRequestBody("application/json", User{}),
)
```

**OAS 2.0 Output:**
```yaml
paths:
  /users:
    post:
      consumes:
        - application/json
        - application/xml
      parameters:
        - in: body
          schema:
            $ref: '#/definitions/User'
```

#### `WithProduces` (OAS 2.0)

Sets operation-level produces array. Ignored for OAS 3.x (use response content types instead).

```go
func WithProduces(mimeTypes ...string) OperationOption
```

**Usage:**
```go
builder.AddOperation("GET", "/users/{id}",
    builder.WithProduces("application/json", "application/xml"),
    builder.WithResponse(200, User{}),
)
```

**OAS 2.0 Output:**
```yaml
paths:
  /users/{id}:
    get:
      produces:
        - application/json
        - application/xml
      responses:
        200:
          schema:
            $ref: '#/definitions/User'
```

### New Request Body Options

#### `WithRequestBodyContentTypes` (OAS 3.x)

Registers the same schema under multiple content types.

```go
func WithRequestBodyContentTypes(contentTypes []string, bodyType any, opts ...RequestBodyOption) OperationOption
```

**Usage:**
```go
builder.AddOperation("POST", "/users",
    builder.WithRequestBodyContentTypes(
        []string{"application/json", "application/xml"},
        User{},
    ),
)
```

**OAS 3.x Output:**
```yaml
paths:
  /users:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/User'
          application/xml:
            schema:
              $ref: '#/components/schemas/User'
```

### New Response Options

#### `WithResponseContentTypes` (OAS 3.x)

Registers the same response schema under multiple content types.

```go
func WithResponseContentTypes(statusCode int, contentTypes []string, responseType any, opts ...ResponseOption) OperationOption
```

**Usage:**
```go
builder.AddOperation("GET", "/users/{id}",
    builder.WithResponseContentTypes(
        200,
        []string{"application/json", "application/xml"},
        User{},
    ),
)
```

**OAS 3.x Output:**
```yaml
paths:
  /users/{id}:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
            application/xml:
              schema:
                $ref: '#/components/schemas/User'
```

## Alternative: Unified API

Instead of separate methods, provide a unified approach that works across OAS versions:

```go
// Single method that handles both OAS 2.0 and 3.x appropriately
builder.AddOperation("POST", "/users",
    builder.WithMediaTypes(
        builder.Consumes("application/json", "application/xml"),
        builder.Produces("application/json", "application/xml"),
    ),
    builder.WithRequestBody("application/json", User{}), // Primary schema
    builder.WithResponse(200, User{}),
)
```

**Behavior:**
- **OAS 2.0**: Sets `consumes` and `produces` arrays on the operation
- **OAS 3.x**: Expands request body and response content to include all specified media types

## Implementation Notes

### OAS 2.0 Implementation

Add fields to the internal operation state and serialize to `parser.OAS2Operation`:

```go
type operationState struct {
    // ... existing fields ...
    consumes []string
    produces []string
}

func (b *Builder) buildOAS2Operation(state *operationState) *parser.Operation {
    op := &parser.Operation{
        // ... existing fields ...
        Consumes: state.consumes,
        Produces: state.produces,
    }
    return op
}
```

### OAS 3.x Implementation

When building request body or responses, iterate over content types:

```go
func (b *Builder) buildOAS3RequestBody(state *operationState) *parser.RequestBody {
    content := make(map[string]*parser.MediaType)
    for _, ct := range state.requestBodyContentTypes {
        content[ct] = &parser.MediaType{
            Schema: state.requestBodySchema,
        }
    }
    return &parser.RequestBody{Content: content}
}
```

### Backward Compatibility

- Existing `WithRequestBody(contentType, bodyType)` continues to work unchanged
- New methods are additive and optional
- When both old and new methods are used, new methods take precedence

## Test Cases

### OAS 2.0 Tests

```go
func TestWithConsumesProduces_OAS2(t *testing.T) {
    b := builder.New(parser.OASVersion20)
    b.AddOperation("POST", "/test",
        builder.WithConsumes("application/json", "application/xml"),
        builder.WithProduces("application/json"),
        builder.WithRequestBody("application/json", TestBody{}),
        builder.WithResponse(200, TestResponse{}),
    )
    doc, _ := b.BuildOAS2()

    op := doc.Paths["/test"].Post
    assert.Equal(t, []string{"application/json", "application/xml"}, op.Consumes)
    assert.Equal(t, []string{"application/json"}, op.Produces)
}
```

### OAS 3.x Tests

```go
func TestWithRequestBodyContentTypes_OAS3(t *testing.T) {
    b := builder.New(parser.OASVersion310)
    b.AddOperation("POST", "/test",
        builder.WithRequestBodyContentTypes(
            []string{"application/json", "application/xml"},
            TestBody{},
        ),
    )
    doc, _ := b.BuildOAS3()

    content := doc.Paths["/test"].Post.RequestBody.Content
    assert.Contains(t, content, "application/json")
    assert.Contains(t, content, "application/xml")
}
```

## Priority

**High** - This is a common requirement for frameworks that support content negotiation (go-restful, Echo, Gin, etc.).

## Related Issues

- go-restful-openapi limitation: "Consumes/Produces at operation level not implemented"
