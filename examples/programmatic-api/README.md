# Programmatic API Examples

This directory contains examples demonstrating how to construct OpenAPI specifications programmatically using Go code.

## Available Examples

| Example | Package | Description | Time |
|---------|---------|-------------|------|
| [builder](builder/) | builder | Fluent API for constructing specs + ServerBuilder for runnable servers | 5 min |

## Quick Start

```bash
cd examples/programmatic-api/builder
go run main.go
```

## Why Programmatic API?

Instead of writing YAML/JSON by hand, the builder package lets you:

- **Type-safe construction** - Compile-time checks catch errors
- **Code reuse** - Share schemas and patterns across APIs
- **Dynamic generation** - Build specs from runtime configuration
- **IDE support** - Autocomplete, documentation, refactoring

## Builder Overview

The [builder](builder/) example demonstrates:

### Specification Construction

```go
spec := builder.New(parser.OASVersion320).
    SetTitle("My API").
    SetVersion("1.0.0").
    AddServer("https://api.example.com/v1")

spec.AddOperation(http.MethodGet, "/users/{id}",
    builder.WithPathParam("id", int64(0)),
    builder.WithResponse(http.StatusOK, User{}),
)

doc, err := spec.BuildOAS3()
```

### Struct-Based Schema Generation

```go
type User struct {
    ID    int64  `json:"id" oas:"readOnly=true"`
    Email string `json:"email" oas:"format=email"`
    Role  string `json:"role" oas:"enum=admin|user|guest"`
}
```

The `oas` struct tag provides full control over schema properties.

### ServerBuilder (New!)

Create runnable HTTP servers from your spec:

```go
srv := builder.NewServerBuilder(parser.OASVersion320).
    SetTitle("Quick API").
    SetVersion("1.0.0")

srv.AddOperation(http.MethodGet, "/status",
    builder.WithResponse(http.StatusOK, StatusResponse{}),
)

srv.Handle(http.MethodGet, "/status", func(ctx context.Context, req *builder.Request) builder.Response {
    return builder.JSON(http.StatusOK, StatusResponse{Status: "ok"})
})

result := srv.MustBuildServer()
http.ListenAndServe(":8080", result.Handler)
```

## Use Cases

| Use Case | Approach |
|----------|----------|
| Code-first API development | Define types → generate spec |
| Dynamic spec generation | Build from config/database |
| Test fixtures | Create synthetic specs |
| API composition | Combine specs programmatically |
| Rapid prototyping | ServerBuilder for working APIs |
| Contract-first validation | ServerBuilder with validation enabled |

## Features Demonstrated

### Builder Package

- `builder.New()` - Create new specification
- `.SetTitle()`, `.SetVersion()`, `.SetDescription()` - Metadata
- `.AddServer()` - Server definitions
- `.AddTag()` - Operation grouping
- `.AddOperation()` - Define endpoints
- `.AddAPIKeySecurityScheme()` - Security configuration
- `.BuildOAS3()` - Type-safe build

### Operation Options

- `WithPathParam()`, `WithQueryParam()` - Parameters
- `WithRequestBody()` - Request body schemas
- `WithResponse()` - Response definitions
- `WithParamMinimum()`, `WithParamMaximum()` - Constraints
- `WithParamEnum()` - Enumeration values

### ServerBuilder Package

- `NewServerBuilder()` - Create server builder
- `.Handle()` - Register handlers
- `.Use()` - Add middleware
- `.BuildServer()` - Build handler
- `builder.JSON()`, `builder.Error()` - Response helpers
- `NewServerTest()` - Testing utilities

## Next Steps

- [Builder Deep Dive](../../packages/builder/) - Complete documentation
- [Workflow Examples](../workflows/) - Common API transformation patterns
- [HTTP Validation](../workflows/http-validation/) - Validate requests against specs

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
