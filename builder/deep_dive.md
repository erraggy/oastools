<a id="top"></a>

# Builder Package Deep Dive

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Building Runnable HTTP Servers](#building-runnable-http-servers)
- [Configuration Reference](#configuration-reference)
- [Best Practices](#best-practices)

---

The [`builder`](https://pkg.go.dev/github.com/erraggy/oastools/builder) package enables programmatic construction of OpenAPI Specification documents using a fluent Go API. Instead of writing YAML or JSON by hand, you define your API specification in Go code with automatic reflection-based schema generation from your Go types.

## Overview

The builder transforms Go types into OpenAPI schemas automatically. When you pass a Go struct to define a response body or parameter, the builder inspects the type via reflection and generates the appropriate JSON Schema representation. This approach keeps your API specification synchronized with your actual data types, reducing drift between documentation and implementation.

The builder supports both OAS 2.0 (Swagger) and OAS 3.x (3.0.0 through 3.2.0), with automatic adjustment of `$ref` paths and component locations based on the target version.

## Key Concepts

### Reflection-Based Schema Generation

The core feature of the builder is automatic schema generation from Go types. Rather than manually defining JSON Schema, you pass Go values and the builder introspects their structure.

**Type Mappings:**

| Go Type | OpenAPI Type | Format |
|---------|--------------|--------|
| `string` | string | - |
| `int`, `int32` | integer | int32 |
| `int64` | integer | int64 |
| `float32` | number | float |
| `float64` | number | double |
| `bool` | boolean | - |
| `[]T` | array | items from T |
| `map[string]T` | object | additionalProperties from T |
| `struct` | object | properties from fields |
| `*T` | schema of T | nullable |
| `time.Time` | string | date-time |

Nested structures are recursively processed, and named types are registered as reusable schemas in `components/schemas` (OAS 3.x) or `definitions` (OAS 2.0).

### Schema Naming

By default, schemas are named using the Go convention of `package.TypeName`. For example, a `User` struct in the `models` package becomes the schema `models.User`. This naming ensures uniqueness when multiple packages define types with the same name.

The builder provides extensible naming strategies for cases where you need different conventions, such as PascalCase for JSON Schema compatibility or custom templates for specific naming requirements.

### Version-Aware Reference Paths

The builder automatically adjusts `$ref` paths based on the OAS version. When you register a type, references are generated correctly for the target version.

**OAS 3.x references:**

```yaml
$ref: "#/components/schemas/models.User"
$ref: "#/components/parameters/LimitParam"
$ref: "#/components/responses/ErrorResponse"
```

**OAS 2.0 references:**

```yaml
$ref: "#/definitions/models.User"
$ref: "#/parameters/LimitParam"
$ref: "#/responses/ErrorResponse"
```

[↑ Back to top](#top)

## API Styles

### Fluent Builder API

The builder uses method chaining for a fluent construction experience:

```go
spec := builder.New(parser.OASVersion320).
    SetTitle("My API").
    SetVersion("1.0.0").
    SetDescription("A comprehensive API example")

spec.AddOperation(http.MethodGet, "/users",
    builder.WithOperationID("listUsers"),
    builder.WithResponse(http.StatusOK, []User{}),
)

doc, err := spec.BuildOAS3()
```

### Functional Options for Operations

Operations accept functional options that configure various aspects:

```go
spec.AddOperation(http.MethodPost, "/users",
    builder.WithOperationID("createUser"),
    builder.WithSummary("Create a new user"),
    builder.WithDescription("Creates a user with the provided details"),
    builder.WithTags("users", "admin"),
    builder.WithRequestBody("application/json", CreateUserRequest{}),
    builder.WithResponse(http.StatusCreated, User{}),
    builder.WithResponse(http.StatusBadRequest, ErrorResponse{}),
)
```

[↑ Back to top](#top)

## Practical Examples

### Building a Simple API Specification

See also: [Complete API example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-CompleteAPI) on pkg.go.dev

The most straightforward use case constructs an API from Go types:

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

// Define your domain types
type User struct {
    ID        int64  `json:"id"`
    Name      string `json:"name"`
    Email     string `json:"email" oas:"format=email"`
    CreatedAt string `json:"created_at" oas:"format=date-time"`
}

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email" oas:"format=email"`
}

type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

func main() {
    // Create a new builder for OAS 3.2.0
    spec := builder.New(parser.OASVersion320).
        SetTitle("User Management API").
        SetVersion("1.0.0").
        SetDescription("API for managing user accounts")
    
    // Add a server
    spec.AddServer("https://api.example.com/v1",
        builder.WithServerDescription("Production server"))
    spec.AddServer("https://staging-api.example.com/v1",
        builder.WithServerDescription("Staging server"))
    
    // Define operations using Go types
    spec.AddOperation(http.MethodGet, "/users",
        builder.WithOperationID("listUsers"),
        builder.WithSummary("List all users"),
        builder.WithTags("users"),
        builder.WithQueryParam("limit", int(0), builder.WithParamDescription("Maximum results")),
        builder.WithQueryParam("offset", int(0), builder.WithParamDescription("Pagination offset")),
        builder.WithResponse(http.StatusOK, []User{}),
    )
    
    spec.AddOperation(http.MethodPost, "/users",
        builder.WithOperationID("createUser"),
        builder.WithSummary("Create a new user"),
        builder.WithTags("users"),
        builder.WithRequestBody("application/json", CreateUserRequest{}),
        builder.WithResponse(http.StatusCreated, User{}),
        builder.WithResponse(http.StatusBadRequest, ErrorResponse{}),
    )
    
    spec.AddOperation(http.MethodGet, "/users/{userId}",
        builder.WithOperationID("getUser"),
        builder.WithSummary("Get user by ID"),
        builder.WithTags("users"),
        builder.WithPathParam("userId", int64(0)),
        builder.WithResponse(http.StatusOK, User{}),
        builder.WithResponse(http.StatusNotFound, ErrorResponse{}),
    )
    
    // Build the OAS 3 document
    doc, err := spec.BuildOAS3()
    if err != nil {
        log.Fatal(err)
    }
    
    // Write to file
    if err := spec.WriteFile("openapi.yaml"); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Generated API spec with %d paths\n", len(doc.Paths))
    fmt.Printf("Registered schemas: %d\n", len(doc.Components.Schemas))
}
```

**Generated Output (openapi.yaml):**

```yaml
openapi: 3.2.0
info:
  title: User Management API
  version: 1.0.0
  description: API for managing user accounts
servers:
  - url: https://api.example.com/v1
    description: Production server
  - url: https://staging-api.example.com/v1
    description: Staging server
paths:
  /users:
    get:
      operationId: listUsers
      summary: List all users
      tags:
        - users
      parameters:
        - name: limit
          in: query
          description: Maximum results
          schema:
            type: integer
            format: int32
        - name: offset
          in: query
          description: Pagination offset
          schema:
            type: integer
            format: int32
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/main.User'
    post:
      operationId: createUser
      summary: Create a new user
      tags:
        - users
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/main.CreateUserRequest'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/main.User'
        '400':
          description: Bad Request
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/main.ErrorResponse'
  /users/{userId}:
    get:
      operationId: getUser
      summary: Get user by ID
      tags:
        - users
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: integer
            format: int64
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/main.User'
        '404':
          description: Not Found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/main.ErrorResponse'
components:
  schemas:
    main.User:
      type: object
      required:
        - id
        - name
        - email
        - created_at
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
        email:
          type: string
          format: email
        created_at:
          type: string
          format: date-time
    main.CreateUserRequest:
      type: object
      required:
        - name
        - email
      properties:
        name:
          type: string
        email:
          type: string
          format: email
    main.ErrorResponse:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: integer
          format: int32
        message:
          type: string
```

### Using OAS Tags for Schema Customization

The `oas` struct tag provides fine-grained control over generated schemas:

```go
package main

import (
    "net/http"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

type Product struct {
    // Basic field with format
    ID int64 `json:"id" oas:"format=int64"`
    
    // String with constraints
    Name string `json:"name" oas:"minLength=1,maxLength=100"`
    
    // String with pattern validation
    SKU string `json:"sku" oas:"pattern=^[A-Z]{3}-[0-9]{6}$"`
    
    // Number with range constraints
    Price float64 `json:"price" oas:"minimum=0,maximum=999999.99"`
    
    // Integer with constraints
    Quantity int `json:"quantity" oas:"minimum=0,maximum=10000"`
    
    // Enum field
    Status string `json:"status" oas:"enum=draft|active|archived"`
    
    // Deprecated field
    LegacyCode string `json:"legacy_code,omitempty" oas:"deprecated=true"`
    
    // Field with example
    Description string `json:"description" oas:"example=A high-quality product"`
    
    // Field with custom title
    Category string `json:"category" oas:"title=Product Category"`
    
    // Read-only field (not accepted in requests)
    CreatedAt string `json:"created_at" oas:"readOnly=true,format=date-time"`
    
    // Write-only field (not included in responses)
    AdminNotes string `json:"admin_notes,omitempty" oas:"writeOnly=true"`
}

func main() {
    spec := builder.New(parser.OASVersion320).
        SetTitle("Product API").
        SetVersion("1.0.0")
    
    spec.AddOperation(http.MethodPost, "/products",
        builder.WithOperationID("createProduct"),
        builder.WithRequestBody("application/json", Product{}),
        builder.WithResponse(http.StatusCreated, Product{}),
    )
    
    doc, _ := spec.BuildOAS3()
    // Schema will include all OAS tag constraints
}
```

**Supported OAS Tag Options:**

| Tag | Description | Example |
|-----|-------------|---------|
| `format` | String/number format | `format=email`, `format=int64` |
| `minimum` | Minimum numeric value | `minimum=0` |
| `maximum` | Maximum numeric value | `maximum=100` |
| `exclusiveMinimum` | Exclusive minimum | `exclusiveMinimum=0` |
| `exclusiveMaximum` | Exclusive maximum | `exclusiveMaximum=100` |
| `minLength` | Minimum string length | `minLength=1` |
| `maxLength` | Maximum string length | `maxLength=255` |
| `pattern` | Regex pattern | `pattern=^[A-Z]+$` |
| `minItems` | Minimum array items | `minItems=1` |
| `maxItems` | Maximum array items | `maxItems=100` |
| `uniqueItems` | Array uniqueness | `uniqueItems=true` |
| `enum` | Enumeration values | `enum=a\|b\|c` |
| `deprecated` | Mark as deprecated | `deprecated=true` |
| `readOnly` | Read-only field | `readOnly=true` |
| `writeOnly` | Write-only field | `writeOnly=true` |
| `nullable` | Nullable field | `nullable=true` |
| `title` | Schema title | `title=User Name` |
| `example` | Example value | `example=john@example.com` |

[↑ Back to top](#top)

### Custom Schema Naming Strategies

See also: [PascalCase example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-SchemaNamingPascalCase), [Template example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-SchemaNamingTemplate), [Custom function example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-SchemaNamingCustomFunc) on pkg.go.dev

When the default `package.TypeName` naming doesn't fit your needs, use custom naming strategies:

```go
package main

import (
    "net/http"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

type User struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}

func main() {
    // PascalCase naming: "ModelsUser" instead of "models.User"
    spec := builder.New(parser.OASVersion320,
        builder.WithSchemaNaming(builder.SchemaNamingPascalCase),
    ).SetTitle("API").SetVersion("1.0.0")
    
    spec.AddOperation(http.MethodGet, "/users",
        builder.WithOperationID("listUsers"),
        builder.WithResponse(http.StatusOK, []User{}),
    )
    
    // Schema will be named "MainUser" instead of "main.User"
    doc, _ := spec.BuildOAS3()
}
```

**Available Naming Strategies:**

| Strategy | Example | Use Case |
|----------|---------|----------|
| `SchemaNamingDefault` | `models.User` | Standard Go-style naming |
| `SchemaNamingPascalCase` | `ModelsUser` | JSON Schema compatibility |
| `SchemaNamingCamelCase` | `modelsUser` | JavaScript conventions |
| `SchemaNamingSnakeCase` | `models_user` | Database-style naming |
| `SchemaNamingKebabCase` | `models-user` | URL-friendly naming |
| `SchemaNamingTypeOnly` | `User` | When package uniqueness isn't needed |
| `SchemaNamingFullPath` | `github.com_org_models_User` | Full disambiguation |

**Custom Templates:**

For complete control, use Go text templates:

```go
// Custom separator
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNameTemplate(`{{.Package}}+{{.Type}}`),
)
// Result: "models+User"

// Uppercase with underscore
spec = builder.New(parser.OASVersion320,
    builder.WithSchemaNameTemplate(`{{upper .Package}}_{{upper .Type}}`),
)
// Result: "MODELS_USER"
```

**Available Template Functions:** `pascal`, `camel`, `snake`, `kebab`, `upper`, `lower`, `title`, `sanitize`, `trimPrefix`, `trimSuffix`, `replace`, `join`

**Custom Naming Function:**

For maximum flexibility, provide a custom function:

```go
spec := builder.New(parser.OASVersion320,
    builder.WithSchemaNameFunc(func(ctx builder.SchemaNameContext) string {
        // Custom logic based on package, type, or other metadata
        if ctx.Package == "internal" {
            return "Internal" + ctx.Type
        }
        return ctx.Type
    }),
)
```

[↑ Back to top](#top)

### Modifying Existing Documents

See also: [FromDocument example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-FromDocument) on pkg.go.dev

The builder can extend existing OAS documents rather than creating from scratch:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

type HealthResponse struct {
    Status string `json:"status"`
}

func main() {
    // Parse existing document
    parseResult, err := parser.ParseWithOptions(
        parser.WithFilePath("existing-api.yaml"),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    existingDoc, ok := parseResult.OAS3Document()
    if !ok {
        log.Fatal("Expected OAS 3 document")
    }
    
    // Create builder from existing document
    spec := builder.FromDocument(existingDoc)
    
    // Add new endpoints
    spec.AddOperation(http.MethodGet, "/health",
        builder.WithOperationID("healthCheck"),
        builder.WithResponse(http.StatusOK, HealthResponse{}),
    )
    
    spec.AddOperation(http.MethodGet, "/ready",
        builder.WithOperationID("readinessCheck"),
        builder.WithResponse(http.StatusOK, HealthResponse{}),
    )
    
    // Build updated document
    doc, err := spec.BuildOAS3()
    if err != nil {
        log.Fatal(err)
    }
    
    // Original paths are preserved, new paths added
    log.Printf("Total paths: %d", len(doc.Paths))
}
```

### Building OAS 2.0 (Swagger) Documents

The same API works for Swagger 2.0 with automatic schema placement:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

type Pet struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}

func main() {
    // Specify OAS 2.0
    spec := builder.New(parser.OASVersion20).
        SetTitle("Pet Store").
        SetVersion("1.0.0")
    
    spec.AddOperation(http.MethodGet, "/pets",
        builder.WithOperationID("listPets"),
        builder.WithResponse(http.StatusOK, []Pet{}),
    )
    
    // BuildOAS2 returns *parser.OAS2Document
    doc, err := spec.BuildOAS2()
    if err != nil {
        log.Fatal(err)
    }
    
    // Schemas are in definitions for OAS 2.0
    log.Printf("Definitions: %d", len(doc.Definitions))
    
    // Refs use #/definitions/ path
    // $ref: "#/definitions/main.Pet"
}
```

### Adding Security Schemes

See also: [Security example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-WithSecurity) on pkg.go.dev

Define authentication methods for your API:

```go
package main

import (
    "net/http"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

func main() {
    spec := builder.New(parser.OASVersion320).
        SetTitle("Secure API").
        SetVersion("1.0.0")
    
    // Add API key security scheme
    spec.AddSecurityScheme("apiKey", &parser.SecurityScheme{
        Type: "apiKey",
        In:   "header",
        Name: "X-API-Key",
    })
    
    // Add Bearer token security scheme
    spec.AddSecurityScheme("bearerAuth", &parser.SecurityScheme{
        Type:   "http",
        Scheme: "bearer",
    })
    
    // Add OAuth2 security scheme
    spec.AddSecurityScheme("oauth2", &parser.SecurityScheme{
        Type: "oauth2",
        Flows: &parser.OAuthFlows{
            ClientCredentials: &parser.OAuthFlow{
                TokenURL: "https://auth.example.com/token",
                Scopes: map[string]string{
                    "read":  "Read access",
                    "write": "Write access",
                },
            },
        },
    })
    
    // Apply security to specific operation
    spec.AddOperation(http.MethodGet, "/protected",
        builder.WithOperationID("protectedEndpoint"),
        builder.WithSecurity([]string{"bearerAuth"}),
        builder.WithResponse(http.StatusOK, struct{}{}),
    )
    
    // Or apply global security
    spec.SetSecurity(
        parser.SecurityRequirement{"apiKey": []string{}},
    )
    
    doc, _ := spec.BuildOAS3()
}
```

[↑ Back to top](#top)

### Webhooks (OAS 3.1+)

For OAS 3.1 and later, add webhook definitions:

```go
package main

import (
    "net/http"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

type UserEvent struct {
    EventType string `json:"event_type"`
    UserID    int64  `json:"user_id"`
    Timestamp string `json:"timestamp" oas:"format=date-time"`
}

func main() {
    // Use OAS 3.1.0 for webhook support
    spec := builder.New(parser.OASVersion310).
        SetTitle("Webhook API").
        SetVersion("1.0.0")
    
    // Add webhook
    spec.AddWebhook("userCreated", http.MethodPost,
        builder.WithRequestBody("application/json", UserEvent{}),
        builder.WithResponse(http.StatusOK, struct{}{}),
    )
    
    spec.AddWebhook("userDeleted", http.MethodPost,
        builder.WithRequestBody("application/json", UserEvent{}),
        builder.WithResponse(http.StatusOK, struct{}{}),
    )
    
    doc, _ := spec.BuildOAS3()
    // doc.Webhooks will contain the webhook definitions
}
```

### Semantic Schema Deduplication

See also: [Deduplication example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-SemanticDeduplication) on pkg.go.dev

When building complex APIs, you might create equivalent schemas through different paths. Enable deduplication to consolidate them:

```go
package main

import (
    "net/http"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

// These two types are structurally identical
type Address struct {
    Street string `json:"street"`
    City   string `json:"city"`
    Zip    string `json:"zip"`
}

type ShippingAddress struct {
    Street string `json:"street"`
    City   string `json:"city"`
    Zip    string `json:"zip"`
}

func main() {
    spec := builder.New(parser.OASVersion320,
        builder.WithSemanticDeduplication(true),
    ).SetTitle("API").SetVersion("1.0.0")
    
    // Both types get registered
    spec.AddOperation(http.MethodPost, "/users",
        builder.WithOperationID("createUser"),
        builder.WithRequestBody("application/json", struct {
            HomeAddress Address `json:"home_address"`
        }{}),
        builder.WithResponse(http.StatusOK, struct{}{}),
    )
    
    spec.AddOperation(http.MethodPost, "/orders",
        builder.WithOperationID("createOrder"),
        builder.WithRequestBody("application/json", struct {
            ShipTo ShippingAddress `json:"ship_to"`
        }{}),
        builder.WithResponse(http.StatusOK, struct{}{}),
    )
    
    doc, _ := spec.BuildOAS3()
    
    // With deduplication enabled, Address and ShippingAddress
    // are consolidated into a single schema (Address, alphabetically first)
    // All references are rewritten automatically
}
```

### Registering Types Explicitly

When you need to register a schema without using it in an operation:

```go
package main

import (
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

type Pagination struct {
    Page     int `json:"page"`
    PageSize int `json:"page_size"`
    Total    int `json:"total"`
}

func main() {
    spec := builder.New(parser.OASVersion320).
        SetTitle("API").
        SetVersion("1.0.0")
    
    // Register type explicitly
    spec.RegisterType(Pagination{})
    
    // Or register with a custom name
    spec.RegisterTypeAs("PaginationInfo", Pagination{})
    
    doc, _ := spec.BuildOAS3()
    // Both schemas are available in components/schemas
}
```

### Integration with Validator

Validate built documents before using them:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
    "github.com/erraggy/oastools/validator"
)

type User struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}

func main() {
    spec := builder.New(parser.OASVersion320).
        SetTitle("API").
        SetVersion("1.0.0")
    
    spec.AddOperation(http.MethodGet, "/users",
        builder.WithOperationID("listUsers"),
        builder.WithResponse(http.StatusOK, []User{}),
    )
    
    // Build a ParseResult for validation
    parseResult, err := spec.BuildResult()
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate the built document
    valResult, err := validator.ValidateWithOptions(
        validator.WithParsed(*parseResult),
        validator.WithIncludeWarnings(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    if !valResult.Valid {
        log.Printf("Validation errors: %d", valResult.ErrorCount)
        for _, e := range valResult.Errors {
            log.Printf("  %s: %s", e.Path, e.Message)
        }
    } else {
        log.Println("Document is valid")
    }
}
```

### Building Runnable HTTP Servers

See also: [ServerBuilder example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-ServerBuilder), [CRUD example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-ServerBuilderCRUD), [Testing example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-ServerBuilderTesting) on pkg.go.dev

The `ServerBuilder` extends `Builder` to create a complete HTTP server from your OpenAPI specification. Instead of just generating the spec, you can define handlers and build a production-ready `http.Handler`:

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

type Pet struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}

func main() {
    // Create a ServerBuilder (extends Builder with server capabilities)
    srv := builder.NewServerBuilder(parser.OASVersion320).
        SetTitle("Pet Store API").
        SetVersion("1.0.0")

    // Define operations (same API as Builder)
    srv.AddOperation(http.MethodGet, "/pets",
        builder.WithOperationID("listPets"),
        builder.WithResponse(http.StatusOK, []Pet{}),
    )

    srv.AddOperation(http.MethodPost, "/pets",
        builder.WithOperationID("createPet"),
        builder.WithRequestBody("application/json", Pet{}),
        builder.WithResponse(http.StatusCreated, Pet{}),
    )

    srv.AddOperation(http.MethodGet, "/pets/{petId}",
        builder.WithOperationID("getPet"),
        builder.WithPathParam("petId", int64(0)),
        builder.WithResponse(http.StatusOK, Pet{}),
    )

    // Register handlers for each operation using method + path
    srv.Handle(http.MethodGet, "/pets", func(_ context.Context, _ *builder.Request) builder.Response {
        pets := []Pet{{ID: 1, Name: "Fluffy"}, {ID: 2, Name: "Buddy"}}
        return builder.JSON(http.StatusOK, pets)
    })

    srv.Handle(http.MethodPost, "/pets", func(_ context.Context, req *builder.Request) builder.Response {
        // req.Body contains the parsed request body
        return builder.JSON(http.StatusCreated, req.Body)
    })

    srv.Handle(http.MethodGet, "/pets/{petId}", func(_ context.Context, req *builder.Request) builder.Response {
        // req.PathParams contains typed path parameters
        petID := req.PathParams["petId"]
        return builder.JSON(http.StatusOK, Pet{ID: petID.(int64), Name: "Fluffy"})
    })

    // Build the server
    result, err := srv.BuildServer()
    if err != nil {
        log.Fatal(err)
    }

    // result.Handler is a standard http.Handler
    // result.Spec contains the generated OpenAPI document
    // result.Validator is available if validation was enabled

    log.Println("Starting server on :8080")
    log.Fatal(http.ListenAndServe(":8080", result.Handler))
}
```

**Response Helpers:**

The ServerBuilder provides convenience functions for common response patterns:

```go
// JSON response
builder.JSON(http.StatusOK, data)

// Error response with message
builder.Error(http.StatusBadRequest, "invalid request")

// Error with additional details
builder.ErrorWithDetails(http.StatusBadRequest, "validation failed", map[string]any{
    "field": "email",
    "error": "invalid format",
})

// 204 No Content
builder.NoContent()

// Redirect
builder.Redirect(http.StatusFound, "/new-location")

// Stream binary data
builder.Stream(http.StatusOK, "application/octet-stream", reader)

// Complex responses with ResponseBuilder
builder.NewResponse(http.StatusOK).
    Header("X-Request-ID", "12345").
    Header("Cache-Control", "no-store").
    JSON(data)
```

**Middleware:**

Add middleware for cross-cutting concerns:

```go
srv := builder.NewServerBuilder(parser.OASVersion320)

// Add logging middleware
srv.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
})

// Add CORS middleware
srv.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        next.ServeHTTP(w, r)
    })
})
```

Middleware is applied in order: first added = outermost (executes first on request).

**Request Validation:**

Enable automatic request validation against the OpenAPI specification:

```go
srv := builder.NewServerBuilder(parser.OASVersion320,
    builder.WithValidationConfig(builder.ValidationConfig{
        IncludeRequestValidation: true,  // Validate incoming requests
        StrictMode:               false, // Treat warnings as errors
    }),
)
```

Invalid requests are rejected before reaching your handlers, ensuring your handlers receive only valid data.

**Testing Utilities:**

The package provides testing helpers for writing handler tests without starting a real server:

```go
func TestListPets(t *testing.T) {
    srv := builder.NewServerBuilder(parser.OASVersion320, builder.WithoutValidation())

    srv.AddOperation(http.MethodGet, "/pets",
        builder.WithOperationID("listPets"),
        builder.WithResponse(http.StatusOK, []Pet{}),
    )

    srv.Handle(http.MethodGet, "/pets", listPetsHandler)

    result := srv.MustBuildServer()
    test := builder.NewServerTest(result)

    // GetJSON makes a GET request and decodes the JSON response
    var pets []Pet
    rec, err := test.GetJSON("/pets", &pets)
    if err != nil {
        t.Fatal(err)
    }

    if rec.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", rec.Code)
    }

    if len(pets) == 0 {
        t.Error("Expected pets")
    }
}
```

Available test methods:

- `GetJSON(path, target)` - GET with JSON response
- `PostJSON(path, body, target)` - POST JSON with response
- `PutJSON(path, body, target)` - PUT JSON with response
- `Delete(path)` - DELETE request
- `Request(method, path)` - Create a TestRequest for custom requests

**Converting Builder to ServerBuilder:**

If you have an existing `Builder`, convert it to a `ServerBuilder`:

```go
// Create specification with Builder
spec := builder.New(parser.OASVersion320).
    SetTitle("My API").
    SetVersion("1.0.0")

spec.AddOperation(http.MethodGet, "/status",
    builder.WithOperationID("getStatus"),
    builder.WithResponse(http.StatusOK, StatusResponse{}),
)

// Convert to ServerBuilder to add handlers
srv := builder.FromBuilder(spec, builder.WithoutValidation())

srv.Handle(http.MethodGet, "/status", statusHandler)

result := srv.MustBuildServer()
```

**Configuration Options:**

| Option | Description |
|--------|-------------|
| `WithoutValidation()` | Disable request validation |
| `WithValidationConfig(cfg)` | Configure validation behavior |
| `WithRecovery()` | Enable panic recovery middleware |
| `WithRequestLogging(fn)` | Enable request logging with a logger function |
| `WithErrorHandler(fn)` | Custom error handler |
| `WithNotFoundHandler(h)` | Custom 404 handler |
| `WithMethodNotAllowedHandler(h)` | Custom 405 handler |
| `WithRouter(strategy)` | Set the routing strategy |
| `WithStdlibRouter()` | Use net/http with PathMatcherSet for routing (default) |

[↑ Back to top](#top)

## Configuration Reference

### Builder Options

| Option | Description |
|--------|-------------|
| `WithSchemaNaming(strategy)` | Set built-in naming strategy |
| `WithSchemaNameTemplate(string)` | Custom Go template for naming |
| `WithSchemaNameFunc(func)` | Custom naming function |
| `WithSemanticDeduplication(bool)` | Enable schema consolidation |

### Operation Options

| Option | Description |
|--------|-------------|
| `WithOperationID(string)` | Set operation identifier |
| `WithSummary(string)` | Brief operation description |
| `WithDescription(string)` | Detailed description |
| `WithTags(...string)` | Categorization tags |
| `WithDeprecated(bool)` | Mark as deprecated |

### Parameter Options

| Option | Description |
|--------|-------------|
| `WithPathParam(name, type)` | Path parameter |
| `WithQueryParam(name, type, ...opts)` | Query parameter |
| `WithHeaderParam(name, type, ...opts)` | Header parameter |
| `WithCookieParam(name, type, ...opts)` | Cookie parameter |
| `WithParamDescription(string)` | Parameter description |
| `WithParamRequired(bool)` | Required flag |
| `WithParamExtension(key, value)` | Add vendor extension (x-*) |
| `WithParamAllowEmptyValue(bool)` | Allow empty values (OAS 2.0) |
| `WithParamCollectionFormat(string)` | Array serialization: csv, ssv, tsv, pipes, multi (OAS 2.0) |
| `WithParamType(string)` | Override inferred OpenAPI type |
| `WithParamFormat(string)` | Override inferred OpenAPI format |
| `WithParamSchema(*parser.Schema)` | Full schema override (highest precedence) |

### Body and Response Options

| Option | Description |
|--------|-------------|
| `WithRequestBody(mediaType, type)` | Request body with schema |
| `WithRequestBodyContentTypes([]string, type)` | Request body with multiple content types |
| `WithRequestBodyExtension(key, value)` | Add vendor extension (x-*) |
| `WithResponse(status, type)` | Response with schema |
| `WithResponseContentTypes(status, []string, type)` | Response with multiple content types |
| `WithResponseDescription(string)` | Response description |
| `WithResponseExtension(key, value)` | Add vendor extension (x-*) |

### Operation Options

| Option | Description |
|--------|-------------|
| `WithOperationExtension(key, value)` | Add vendor extension (x-*) |
| `WithConsumes(...string)` | Operation consumes MIME types (OAS 2.0) |
| `WithProduces(...string)` | Operation produces MIME types (OAS 2.0) |

### Security Options

| Option | Description |
|--------|-------------|
| `WithSecurity([]string)` | Operation-level security |
| `AddSecurityScheme(name, scheme)` | Register security scheme |
| `SetSecurity(requirements...)` | Document-level security |

### Tag Options

| Option | Description |
|--------|-------------|
| `WithTagDescription(desc)` | Set tag description |
| `WithTagExternalDocs(url, desc)` | Set external documentation URL and description for a tag |

### Server Options

| Option | Description |
|--------|-------------|
| `WithServerDescription(desc)` | Set server description |
| `WithServerVariable(name, default, ...opts)` | Add a variable to the server |

### Server Variable Options

| Option | Description |
|--------|-------------|
| `WithServerVariableEnum(values...)` | Set allowed values for a server variable |
| `WithServerVariableDescription(desc)` | Set description for a server variable |

### Response Detail Options

These `ResponseOption` values configure individual responses when passed to `WithResponse` or `WithResponseRawSchema`:

| Option | Description |
|--------|-------------|
| `WithResponseContentType(contentType)` | Set response content type (default: application/json) |
| `WithResponseExample(example)` | Set response example value |
| `WithResponseHeader(name, header)` | Add a header to the response |

### Response-Level Operation Options

These `OperationOption` values add or configure responses at the operation level:

| Option | Description |
|--------|-------------|
| `WithResponseRawSchema(statusCode, contentType, schema, ...opts)` | Add response with pre-built schema |
| `WithResponseRef(statusCode, ref)` | Add a response `$ref` reference |
| `WithDefaultResponse(responseType, ...opts)` | Set the default response for the operation |

### Request Body Detail Options

These options configure request bodies at a finer level than `WithRequestBody`.

| Option | Description |
|--------|-------------|
| `WithRequestBodyRawSchema(contentType, schema, ...opts)` | Request body with pre-built schema |
| `WithRequestDescription(desc)` | Set request body description |
| `WithRequestExample(example)` | Set request body example value |
| `WithRequired(required)` | Set whether request body is required |

### Advanced Operation Options

These options provide lower-level control over operations, including pre-built parameters and handler registration.

| Option | Description |
|--------|-------------|
| `WithHandler(handler)` | Register a typed handler for the operation (ServerBuilder only) |
| `WithHandlerFunc(handler)` | Register an `http.HandlerFunc` handler (ServerBuilder only) |
| `WithNoSecurity()` | Explicitly mark operation as requiring no security |
| `WithParameter(param)` | Add a pre-built `*parser.Parameter` to the operation |
| `WithParameterRef(ref)` | Add a parameter `$ref` reference |
| `WithFileParam(name, ...opts)` | Add a file upload parameter (OAS 2.0: formData; OAS 3.x: multipart) |
| `WithFormParam(name, type, ...opts)` | Add a form parameter (OAS 2.0: formData; OAS 3.x: request body) |

### Parameter Validation Options

These `ParamOption` values set JSON Schema validation constraints on parameters.

| Option | Description |
|--------|-------------|
| `WithParamDefault(value)` | Set default value |
| `WithParamDeprecated(bool)` | Mark parameter as deprecated |
| `WithParamEnum(values...)` | Set allowed values |
| `WithParamExample(example)` | Set example value |
| `WithParamMinimum(min)` | Minimum numeric value |
| `WithParamMaximum(max)` | Maximum numeric value |
| `WithParamExclusiveMinimum(bool)` | Exclusive minimum bound |
| `WithParamExclusiveMaximum(bool)` | Exclusive maximum bound |
| `WithParamMinLength(min)` | Minimum string length |
| `WithParamMaxLength(max)` | Maximum string length |
| `WithParamPattern(pattern)` | Regex pattern for string validation |
| `WithParamMinItems(min)` | Minimum array items |
| `WithParamMaxItems(max)` | Maximum array items |
| `WithParamMultipleOf(value)` | Numeric multiple-of constraint |
| `WithParamUniqueItems(bool)` | Require unique array items |

### Schema Field Processing

| Option | Description |
|--------|-------------|
| `WithSchemaFieldProcessor(fn)` | Custom `BuilderOption` for processing struct field tags during schema generation |

### Generic Naming Options

These `BuilderOption` values configure how Go generic types are represented in schema names.

| Option | Description |
|--------|-------------|
| `WithGenericNaming(strategy)` | Set generic type naming strategy |
| `WithGenericNamingConfig(config)` | Fine-grained control over generic naming |
| `WithGenericSeparator(sep)` | Separator for generic type parameters (default: `_`) |
| `WithGenericParamSeparator(sep)` | Separator between multiple type parameters |
| `WithGenericIncludePackage(bool)` | Include package names in generic type parameters |
| `WithGenericApplyBaseCasing(bool)` | Apply base naming strategy to type parameters |

[↑ Back to top](#top)

## Best Practices

**Use Go types as your source of truth.** When your API types are Go structs, the builder keeps your specification synchronized with your implementation.

**Leverage OAS tags for constraints.** The `oas` struct tag lets you express validation rules directly on your types, keeping schema details close to the data definition.

**Choose a consistent naming strategy** and stick with it across your API. This makes the generated specification predictable and easier to consume.

**Validate built documents** before publishing or using them. The validator catches issues that might not be apparent during construction.

**Use semantic deduplication** when building from multiple modules that might define equivalent types independently.

**Consider OAS 3.1+ for new APIs** to take advantage of features like webhooks and full JSON Schema compatibility.

**Use `BuildResult()` for integration** with other oastools packages, providing a bridge from the builder to validation, conversion, or code generation workflows.

---

## Learn More

For additional examples and complete API documentation:

- 📦 [API Reference on pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools/builder) - Complete API documentation with all examples
- 🔧 [Parameters example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-WithParameters) - Path, query, and header parameters
- 📤 [File upload example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-WithFileUpload) - Multipart file uploads
- 📝 [Form parameters example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-WithFormParameters) - Form data handling
- 🔒 [Parameter constraints example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-WithParameterConstraints) - Validation rules
- 🖥️ [ServerBuilder example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-ServerBuilder) - Building HTTP servers
- 🧪 [Server testing example](https://pkg.go.dev/github.com/erraggy/oastools/builder#example-package-ServerBuilderTesting) - Testing server handlers
