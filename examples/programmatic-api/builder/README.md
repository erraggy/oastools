# Builder

Demonstrates programmatic OpenAPI specification construction using the builder package, including the new ServerBuilder for creating runnable HTTP servers.

## What You'll Learn

- How to create an OpenAPI spec from scratch using the fluent API
- Defining operations with parameters, request bodies, and responses
- Using Go struct tags for automatic schema generation
- Configuring security schemes
- Building and serializing specifications
- **Creating runnable HTTP servers with ServerBuilder**
- **Testing handlers with the built-in test helpers**

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/programmatic-api/builder
go run main.go
```

## Expected Output

```
Builder Workflow
================

[1/6] Creating OpenAPI 3.2.0 spec builder...
      Base spec created

[2/6] Adding servers...
      Added 2 servers (production + staging)

[3/6] Adding tags...
      Added 'books' tag

[4/6] Configuring security...
      Added API key security scheme (header: X-API-Key)

[5/6] Adding operations...
      ✓ GET /books (listBooks)
      ✓ POST /books (createBook)
      ✓ GET /books/{bookId} (getBook)
      ✓ PUT /books/{bookId} (updateBook)
      ✓ DELETE /books/{bookId} (deleteBook)

[6/6] Building specification...
      Build successful!

--- Specification Summary ---
OpenAPI Version: 3.2.0
Title: Book Store API
Version: 1.0.0
Servers: 2
Tags: 1
Paths: 2
Schemas: 4
Security Schemes: 1
Operations: 5

Generated Schemas:
  - main.Error
  - main.Book
  - main.CreateBookRequest
  - main.UpdateBookRequest

Paths defined:
  - /books
  - /books/{bookId}

=================================

[Bonus] ServerBuilder - Runnable HTTP Server

[1/3] Creating ServerBuilder...
      ServerBuilder created

[2/3] Adding operations and handlers...
      ✓ GET /status with handler
      ✓ POST /messages with handler

[3/3] Building server...
      Server built successfully!

--- Server Summary ---
Handler Type: http.HandlerFunc
Has Spec: true
Has Validator: false

Testing with ServerTest helper:
  GET /status → 200
  Response: {status: "ok", version: "1.0.0"}

Server is ready to run with http.ListenAndServe(":8080", result.Handler)

---
Builder example complete
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates the builder workflow for creating complete APIs |

## Key Concepts

### Creating a Builder

```go
spec := builder.New(parser.OASVersion320).
    SetTitle("My API").
    SetVersion("1.0.0").
    SetDescription("API description")
```

### Adding Servers

```go
spec.AddServer("https://api.example.com/v1",
    builder.WithServerDescription("Production"),
)
```

### Struct Tags for Schema Generation

```go
type Book struct {
    ID    int64  `json:"id" oas:"readOnly=true"`
    Title string `json:"title" oas:"minLength=1,maxLength=200"`
    Genre string `json:"genre" oas:"enum=fiction|non-fiction|sci-fi"`
}
```

| Tag | Description |
|-----|-------------|
| `description=...` | Field description |
| `minLength=N` | Minimum string length |
| `maxLength=N` | Maximum string length |
| `minimum=N` | Minimum numeric value |
| `maximum=N` | Maximum numeric value |
| `pattern=...` | Regex pattern |
| `enum=a\|b\|c` | Enumeration values (pipe-separated) |
| `format=...` | OpenAPI format (email, uri, date-time, etc.) |
| `readOnly=true` | Mark as read-only |
| `writeOnly=true` | Mark as write-only |

### Adding Operations

```go
spec.AddOperation(http.MethodGet, "/books/{bookId}",
    builder.WithOperationID("getBook"),
    builder.WithSummary("Get a book by ID"),
    builder.WithTags("books"),
    builder.WithPathParam("bookId", int64(0),
        builder.WithParamDescription("Book ID"),
    ),
    builder.WithResponse(http.StatusOK, Book{}),
    builder.WithResponse(http.StatusNotFound, Error{}),
)
```

### Request Bodies

```go
spec.AddOperation(http.MethodPost, "/books",
    builder.WithRequestBody("application/json", CreateBookRequest{},
        builder.WithRequired(true),
        builder.WithRequestDescription("Book to create"),
    ),
    builder.WithResponse(http.StatusCreated, Book{}),
)
```

### Query Parameters with Constraints

```go
builder.WithQueryParam("limit", int32(0),
    builder.WithParamDescription("Maximum items"),
    builder.WithParamMinimum(1),
    builder.WithParamMaximum(100),
    builder.WithParamDefault(20),
)
```

### Security Configuration

```go
spec.AddAPIKeySecurityScheme(
    "api_key",      // scheme name
    "header",       // location: header, query, or cookie
    "X-API-Key",    // header/parameter name
    "Description",
).SetSecurity(builder.SecurityRequirement("api_key"))
```

### Building the Spec

```go
// Type-safe build (OAS 3.x)
doc, err := spec.BuildOAS3()
if err != nil {
    log.Fatal(err)
}

// Generic build (any version)
generic, err := spec.Build()
```

### Advanced Features

| Feature | Method |
|---------|--------|
| OAuth2 schemes | `AddOAuth2SecurityScheme()` |
| Bearer auth | `AddHTTPSecurityScheme("bearer", "bearer", "JWT", "desc")` |
| Schema naming | `builder.WithSchemaNaming(builder.SchemaNamingPascalCase)` |
| Custom templates | `builder.WithSchemaNameTemplate("API{{pascal .Type}}")` |
| Deduplication | `builder.WithSemanticDeduplication(true)` |
| From existing doc | `builder.FromDocument(existingDoc)` |

---

## ServerBuilder - Runnable HTTP Servers

ServerBuilder extends Builder to create production-ready HTTP servers with automatic routing and optional request validation.

### Creating a ServerBuilder

```go
srv := builder.NewServerBuilder(parser.OASVersion320, builder.WithoutValidation()).
    SetTitle("My API").
    SetVersion("1.0.0")
```

### Registering Handlers

```go
// Add operation (same API as Builder)
srv.AddOperation(http.MethodGet, "/status",
    builder.WithOperationID("getStatus"),
    builder.WithResponse(http.StatusOK, StatusResponse{}),
)

// Register handler for the operation
srv.Handle(http.MethodGet, "/status", func(ctx context.Context, req *builder.Request) builder.Response {
    return builder.JSON(http.StatusOK, StatusResponse{Status: "ok"})
})
```

### Response Helpers

```go
// JSON response
return builder.JSON(http.StatusOK, data)

// Error response
return builder.Error(http.StatusBadRequest, "invalid input")

// No content (204)
return builder.NoContent()

// Redirect
return builder.Redirect(http.StatusFound, "/new-location")

// Custom response with headers
return builder.NewResponse(http.StatusOK).
    Header("X-Custom", "value").
    JSON(data)
```

### Building and Running

```go
result, err := srv.BuildServer()
if err != nil {
    log.Fatal(err)
}

// result.Handler is a standard http.Handler
http.ListenAndServe(":8080", result.Handler)

// result.Spec contains the generated OpenAPI document
// result.Validator is present if validation is enabled
```

### Request Validation

```go
// Enable request validation
srv := builder.NewServerBuilder(parser.OASVersion320,
    builder.WithValidationConfig(builder.ValidationConfig{
        IncludeRequestValidation: true,
        StrictMode:               false,
    }),
)
```

### Testing Handlers

```go
result := srv.MustBuildServer()
test := builder.NewServerTest(result)

// GET with JSON response
var status StatusResponse
rec, err := test.GetJSON("/status", &status)

// POST with JSON body
var created Book
rec, err := test.PostJSON("/books", CreateBookRequest{Title: "Test"}, &created)
```

### Adding Middleware

```go
srv.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Request-ID", uuid.New().String())
        next.ServeHTTP(w, r)
    })
})
```

---

## Use Cases

- **Code-first API development** - Define your API in Go, generate the spec
- **Dynamic spec generation** - Build specs based on runtime configuration
- **Test fixtures** - Create synthetic specs for testing
- **API composition** - Programmatically combine API elements
- **Rapid prototyping** - Use ServerBuilder to create working APIs quickly
- **Spec-validated servers** - Enable request validation for contract-first development

## Next Steps

- [Builder Deep Dive](https://erraggy.github.io/oastools/packages/builder/) - Complete documentation
- [HTTPValidator](../../workflows/http-validation/) - Validate requests against your spec
- [Code Generation](../../petstore/) - Generate server code from specs

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
