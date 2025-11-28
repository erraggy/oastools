# Builder Package Implementation Plan

> **Purpose**: This document outlines the design and implementation plan for the `builder` package,
> which provides programmatic construction of OpenAPI Specification (OAS) documents with automatic
> schema generation from Go types via reflection.

## Overview

The `builder` package enables users to construct OpenAPI Specification documents programmatically in Go.
The key feature is **reflection-based schema generation**: users pass Go types directly to the API, and
the builder automatically generates OpenAPI-compatible JSON schemas in the `components.schemas` section.

### Goals

1. **Reflection-Based Schema Generation**: Automatically convert Go types to OpenAPI-compatible JSON schemas using reflection
2. **Operation Addition**: Provide a fluent API to add API operations with Go types for request/response bodies
3. **Document Finalization**: Combine all accumulated operations and components into a complete OAS document
4. **Type Safety**: Accept Go types directly (e.g., `foo.MyResponse`) rather than manual schema definitions
5. **Consistency**: Follow existing patterns established in `parser`, `converter`, and `joiner` packages

### Non-Goals

- Validation of generated documents (users should use the `validator` package)
- Automatic endpoint discovery or code scanning
- OpenAPI 2.0 (Swagger) support initially - focus on OAS 3.x

---

## Design Components

### 1. Reflection-Based Schema Generation

The core feature of this package is automatic schema generation from Go types via reflection.
When users pass a Go type to the API, the builder inspects the type structure and generates
an OpenAPI-compatible JSON Schema.

#### Core Reflection API

```go
// SchemaFrom generates an OpenAPI schema from a Go type using reflection
// The type is registered in components.schemas with a name derived from the type
func SchemaFrom(v any) *parser.Schema

// SchemaFromType generates a schema from a reflect.Type
func SchemaFromType(t reflect.Type) *parser.Schema

// RegisterType registers a Go type and returns a $ref to it
// The schema is automatically generated via reflection and added to components.schemas
func (b *Builder) RegisterType(v any) *parser.Schema

// RegisterTypeAs registers a Go type with a custom schema name
func (b *Builder) RegisterTypeAs(name string, v any) *parser.Schema
```

#### Type Mapping Rules

Go types are mapped to OpenAPI schemas as follows:

| Go Type | OpenAPI Type | Format | Notes |
|---------|--------------|--------|-------|
| `string` | `string` | - | - |
| `int`, `int32` | `integer` | `int32` | - |
| `int64` | `integer` | `int64` | - |
| `float32` | `number` | `float` | - |
| `float64` | `number` | `double` | - |
| `bool` | `boolean` | - | - |
| `[]T` | `array` | - | items schema from T |
| `map[string]T` | `object` | - | additionalProperties from T |
| `struct` | `object` | - | properties from fields |
| `*T` | schema of T | - | nullable in OAS 3.0, type array in 3.1+ |
| `time.Time` | `string` | `date-time` | - |
| `uuid.UUID` | `string` | `uuid` | - |

#### Struct Tag Support

The builder recognizes struct tags for customizing schema generation:

```go
type User struct {
    ID        int64     `json:"id" oas:"description=Unique user identifier"`
    Name      string    `json:"name" oas:"minLength=1,maxLength=100"`
    Email     string    `json:"email" oas:"format=email"`
    Role      string    `json:"role" oas:"enum=admin|user|guest"`
    Age       int       `json:"age,omitempty" oas:"minimum=0,maximum=150"`
    CreatedAt time.Time `json:"created_at" oas:"readOnly=true"`
    Password  string    `json:"-"` // Excluded from schema (json:"-")
}
```

**Supported `oas` tag options:**
- `description=<text>` - Field description
- `format=<format>` - Override format (email, uri, uuid, date, date-time, etc.)
- `enum=<val1>|<val2>|...` - Enumeration values (pipe-separated)
- `minimum=<n>`, `maximum=<n>` - Numeric constraints
- `minLength=<n>`, `maxLength=<n>` - String length constraints
- `pattern=<regex>` - String pattern
- `minItems=<n>`, `maxItems=<n>` - Array constraints
- `readOnly=true`, `writeOnly=true` - Access modifiers
- `nullable=true` - Explicitly nullable
- `deprecated=true` - Mark as deprecated
- `example=<value>` - Example value (JSON encoded)

#### Required Fields Detection

Required fields are determined by:
1. Non-pointer fields without `omitempty` are required
2. Fields with `oas:"required=true"` are explicitly required
3. Fields with `oas:"required=false"` are explicitly optional

```go
type CreateUserRequest struct {
    Name     string  `json:"name"`              // Required (no omitempty, not a pointer)
    Email    string  `json:"email"`             // Required
    Age      *int    `json:"age,omitempty"`     // Optional (pointer + omitempty)
    Nickname string  `json:"nickname,omitempty"` // Optional (omitempty)
}
```

#### Nested Type Handling

Nested structs and complex types are handled automatically:

```go
type Order struct {
    ID       int64       `json:"id"`
    Customer Customer    `json:"customer"`      // Generates $ref to Customer schema
    Items    []OrderItem `json:"items"`         // Array with $ref to OrderItem
    Metadata map[string]string `json:"metadata"` // additionalProperties: string
}

type Customer struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}

type OrderItem struct {
    ProductID int64   `json:"product_id"`
    Quantity  int     `json:"quantity"`
    Price     float64 `json:"price"`
}

// Using in builder - all nested types are automatically registered
spec.AddOperation(http.MethodPost, "/orders",
    builder.WithResponse(http.StatusOK, Order{}),
)
// This automatically registers: Order, Customer, OrderItem in components.schemas
```

#### Schema Name Generation

Schema names are generated from type information:
- Simple types: Use type name (e.g., `User`, `Order`)
- Generic types: Include type parameters (e.g., `Page[User]`)
- Anonymous structs: Generate unique name or inline

```go
// Name derivation examples
type User struct{}           // → "User"
type foo.Response struct{}   // → "Response" (package prefix stripped)
type Page[T any] struct{}    // → "PageUser" when T=User
```

#### Schema Caching

To prevent duplicate generation and handle circular references:

```go
type schemaCache struct {
    schemas map[reflect.Type]*parser.Schema
    names   map[reflect.Type]string
    inProgress map[reflect.Type]bool // Detect circular refs
}
```

### 2. Operation Addition

Operations are added to paths using a fluent builder API that accepts Go types directly.

#### Core API Design

```go
// Builder is the main entry point for constructing OAS documents
type Builder struct {
    version      parser.OASVersion
    info         *parser.Info
    servers      []*parser.Server
    paths        parser.Paths
    components   *parser.Components
    tags         []*parser.Tag
    security     []parser.SecurityRequirement
    schemaCache  *schemaCache // For reflection-based schema generation
}

// New creates a new Builder for the specified OAS version
func New(version parser.OASVersion) *Builder

// NewWithInfo creates a Builder with pre-configured Info
func NewWithInfo(version parser.OASVersion, info *parser.Info) *Builder
```

#### Operation Configuration

```go
// AddOperation adds an API operation to the specification
// Go types passed to options are automatically converted to schemas via reflection
func (b *Builder) AddOperation(method, path string, opts ...OperationOption) *Builder

// OperationOption configures an operation
type OperationOption func(*operationConfig)

// Operation configuration options
func WithOperationID(id string) OperationOption
func WithSummary(summary string) OperationOption
func WithDescription(desc string) OperationOption
func WithTags(tags ...string) OperationOption
func WithDeprecated(deprecated bool) OperationOption

// Request configuration - accepts Go types directly
func WithRequestBody(contentType string, bodyType any, opts ...RequestBodyOption) OperationOption
func WithParameter(param *parser.Parameter) OperationOption
func WithQueryParam(name string, paramType any, opts ...ParamOption) OperationOption
func WithPathParam(name string, paramType any, opts ...ParamOption) OperationOption
func WithHeaderParam(name string, paramType any, opts ...ParamOption) OperationOption

// Response configuration - accepts Go types directly
func WithResponse(statusCode int, responseType any, opts ...ResponseOption) OperationOption
func WithResponseRef(statusCode int, ref string) OperationOption
func WithDefaultResponse(responseType any, opts ...ResponseOption) OperationOption

// Security configuration
func WithSecurity(requirements ...parser.SecurityRequirement) OperationOption
func WithNoSecurity() OperationOption
```

#### Key Design: Go Types as Parameters

The central design principle is that **Go types are passed directly** and converted to schemas:

```go
// Instead of manually building schemas:
// ❌ builder.WithResponse(200, builder.ObjectSchema(...))

// Pass Go types directly:
// ✅ builder.WithResponse(200, MyResponse{})

type MyResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
}

spec.AddOperation(http.MethodGet, "/api/status",
    builder.WithResponse(http.StatusOK, MyResponse{}),
)
```

#### Sub-option Types

```go
// RequestBodyOption configures a request body
type RequestBodyOption func(*requestBodyConfig)

func WithRequired(required bool) RequestBodyOption
func WithRequestDescription(desc string) RequestBodyOption
func WithRequestExample(example any) RequestBodyOption

// ResponseOption configures a response
type ResponseOption func(*responseConfig)

func WithResponseDescription(desc string) ResponseOption
func WithResponseExample(example any) ResponseOption
func WithResponseHeader(name string, header *parser.Header) ResponseOption

// ParamOption configures a parameter
type ParamOption func(*paramConfig)

func WithParamDescription(desc string) ParamOption
func WithParamRequired(required bool) ParamOption
func WithParamExample(example any) ParamOption
func WithParamDeprecated(deprecated bool) ParamOption
```

### 3. Document-Level Configuration

```go
// Document-level configuration
func (b *Builder) SetInfo(info *parser.Info) *Builder
func (b *Builder) SetTitle(title string) *Builder
func (b *Builder) SetVersion(version string) *Builder
func (b *Builder) SetDescription(desc string) *Builder

// Server configuration
func (b *Builder) AddServer(url string, opts ...ServerOption) *Builder

type ServerOption func(*serverConfig)
func WithServerDescription(desc string) ServerOption
func WithServerVariable(name, defaultValue string, opts ...ServerVariableOption) ServerOption

// Tags
func (b *Builder) AddTag(name string, opts ...TagOption) *Builder

type TagOption func(*tagConfig)
func WithTagDescription(desc string) TagOption
func WithTagExternalDocs(url, desc string) TagOption

// Global security
func (b *Builder) SetSecurity(requirements ...parser.SecurityRequirement) *Builder
func (b *Builder) AddSecurityScheme(name string, scheme *parser.SecurityScheme) *Builder
```

### 4. Document Finalization

```go
// Build creates the final OAS document
func (b *Builder) Build() (*parser.OAS3Document, error)

// BuildResult creates a ParseResult for compatibility with other packages
func (b *Builder) BuildResult() (*parser.ParseResult, error)

// MarshalYAML returns the document as YAML bytes
func (b *Builder) MarshalYAML() ([]byte, error)

// MarshalJSON returns the document as JSON bytes
func (b *Builder) MarshalJSON() ([]byte, error)

// WriteFile writes the document to a file (format inferred from extension)
func (b *Builder) WriteFile(path string) error
```

---

## Internal Mechanism

### State Management

The Builder maintains internal state for accumulated components and reflection cache:

```go
type Builder struct {
    // Configuration
    version    parser.OASVersion
    
    // Document sections
    info       *parser.Info
    servers    []*parser.Server
    paths      parser.Paths
    tags       []*parser.Tag
    security   []parser.SecurityRequirement
    
    // Components (tracked separately for deduplication)
    schemas         map[string]*parser.Schema
    responses       map[string]*parser.Response
    parameters      map[string]*parser.Parameter
    requestBodies   map[string]*parser.RequestBody
    securitySchemes map[string]*parser.SecurityScheme
    
    // Reflection cache for schema generation
    schemaCache     *schemaCache
    
    // Tracking
    operationIDs map[string]bool // Track used operation IDs for uniqueness
    errors       []error          // Accumulated validation errors
}

// schemaCache manages reflection-based schema generation
type schemaCache struct {
    byType     map[reflect.Type]*parser.Schema // Type → Schema
    byName     map[string]reflect.Type         // Name → Type (for conflict detection)
    inProgress map[reflect.Type]bool           // Circular reference detection
}
```

### Reflection-Based Schema Generation Flow

When a Go type is encountered (via `WithResponse`, `WithRequestBody`, etc.):

```go
// generateSchema converts a Go type to an OpenAPI schema
func (b *Builder) generateSchema(v any) *parser.Schema {
    t := reflect.TypeOf(v)
    if t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    
    // 1. Check cache first
    if schema, exists := b.schemaCache.byType[t]; exists {
        return b.refToSchema(t)
    }
    
    // 2. Mark as in-progress (circular reference detection)
    b.schemaCache.inProgress[t] = true
    defer delete(b.schemaCache.inProgress, t)
    
    // 3. Generate schema based on kind
    var schema *parser.Schema
    switch t.Kind() {
    case reflect.Struct:
        schema = b.generateStructSchema(t)
    case reflect.Slice, reflect.Array:
        schema = b.generateArraySchema(t)
    case reflect.Map:
        schema = b.generateMapSchema(t)
    default:
        schema = b.generatePrimitiveSchema(t)
    }
    
    // 4. Register named types in components.schemas
    if shouldRegister(t) {
        name := b.schemaName(t)
        b.schemas[name] = schema
        b.schemaCache.byType[t] = schema
        b.schemaCache.byName[name] = t
        return b.refToSchema(t)
    }
    
    return schema
}

// generateStructSchema reflects on a struct type
func (b *Builder) generateStructSchema(t reflect.Type) *parser.Schema {
    properties := make(map[string]*parser.Schema)
    required := []string{}
    
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        
        // Skip unexported fields
        if !field.IsExported() {
            continue
        }
        
        // Parse json tag for field name
        jsonTag := field.Tag.Get("json")
        if jsonTag == "-" {
            continue // Explicitly excluded
        }
        
        name, opts := parseJSONTag(jsonTag)
        if name == "" {
            name = field.Name
        }
        
        // Generate schema for field type
        fieldSchema := b.generateSchema(reflect.Zero(field.Type).Interface())
        
        // Apply oas tag customizations
        oasTag := field.Tag.Get("oas")
        if oasTag != "" {
            fieldSchema = applyOASTag(fieldSchema, oasTag)
        }
        
        properties[name] = fieldSchema
        
        // Determine if required
        if isRequired(field, opts) {
            required = append(required, name)
        }
    }
    
    return &parser.Schema{
        Type:       "object",
        Properties: properties,
        Required:   required,
    }
}
```

### Circular Reference Handling

Circular references are handled by detecting in-progress types:

```go
type Node struct {
    Value    int    `json:"value"`
    Children []Node `json:"children"` // Self-referencing
}

// Generated schema:
// Node:
//   type: object
//   properties:
//     value:
//       type: integer
//     children:
//       type: array
//       items:
//         $ref: '#/components/schemas/Node'
```

### Operation Building Flow

```go
// AddOperation implementation flow
func (b *Builder) AddOperation(method, path string, opts ...OperationOption) *Builder {
    // 1. Create operation config with defaults
    cfg := &operationConfig{
        responses: make(map[string]*parser.Response),
    }
    
    // 2. Apply all options
    for _, opt := range opts {
        opt(cfg)
    }
    
    // 3. Validate operation ID uniqueness
    if cfg.operationID != "" {
        if b.operationIDs[cfg.operationID] {
            b.errors = append(b.errors, fmt.Errorf("duplicate operation ID: %s", cfg.operationID))
        }
        b.operationIDs[cfg.operationID] = true
    }
    
    // 4. Build Operation struct
    op := &parser.Operation{
        OperationID: cfg.operationID,
        Summary:     cfg.summary,
        Description: cfg.description,
        Tags:        cfg.tags,
        Parameters:  cfg.parameters,
        RequestBody: cfg.requestBody,
        Responses:   b.buildResponses(cfg.responses),
        Security:    cfg.security,
        Deprecated:  cfg.deprecated,
    }
    
    // 5. Get or create PathItem
    pathItem := b.getOrCreatePathItem(path)
    
    // 6. Assign operation to method
    b.setOperation(pathItem, method, op)
    
    return b
}
```

### Validation During Build

The `Build()` method performs final validation:

```go
func (b *Builder) Build() (*parser.OAS3Document, error) {
    // Check accumulated errors
    if len(b.errors) > 0 {
        return nil, fmt.Errorf("builder has %d error(s): %v", len(b.errors), b.errors[0])
    }
    
    // Validate required fields
    if b.info == nil {
        return nil, fmt.Errorf("info is required")
    }
    if b.info.Title == "" {
        return nil, fmt.Errorf("info.title is required")
    }
    if b.info.Version == "" {
        return nil, fmt.Errorf("info.version is required")
    }
    
    // Build components
    components := &parser.Components{}
    if len(b.schemas) > 0 {
        components.Schemas = b.schemas
    }
    if len(b.responses) > 0 {
        components.Responses = b.responses
    }
    // ... other component types
    
    // Create document
    doc := &parser.OAS3Document{
        OpenAPI:    b.version.String(),
        OASVersion: b.version,
        Info:       b.info,
        Servers:    b.servers,
        Paths:      b.paths,
        Components: components,
        Tags:       b.tags,
        Security:   b.security,
    }
    
    return doc, nil
}
```

---

## Example Usage

### Complete API Definition Example (Reflection-Based)

```go
package main

import (
    "log"
    "net/http"
    "time"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

// Define your Go types - these will be reflected into OpenAPI schemas
type Pet struct {
    ID        int64     `json:"id" oas:"description=Unique pet identifier"`
    Name      string    `json:"name" oas:"minLength=1,description=Pet name"`
    Tag       string    `json:"tag,omitempty" oas:"description=Optional tag"`
    CreatedAt time.Time `json:"created_at" oas:"readOnly=true"`
}

type Error struct {
    Code    int32  `json:"code" oas:"description=Error code"`
    Message string `json:"message" oas:"description=Error message"`
}

type PetList struct {
    Items []Pet `json:"items"`
    Total int   `json:"total"`
}

type CreatePetRequest struct {
    Name string `json:"name" oas:"minLength=1"`
    Tag  string `json:"tag,omitempty"`
}

func main() {
    // Create a new builder for OAS 3.2.0
    spec := builder.New(parser.OASVersion320).
        SetTitle("Pet Store API").
        SetVersion("1.0.0").
        SetDescription("A sample Pet Store API")
    
    // Add server
    spec.AddServer("https://api.petstore.example.com/v1",
        builder.WithServerDescription("Production server"),
    )
    
    // Add operations - Go types are automatically converted to schemas
    spec.AddOperation(http.MethodGet, "/pets",
        builder.WithOperationID("listPets"),
        builder.WithSummary("List all pets"),
        builder.WithTags("pets"),
        builder.WithQueryParam("limit", int32(0),  // Pass Go type for reflection
            builder.WithParamDescription("Maximum number of pets to return"),
        ),
        builder.WithResponse(http.StatusOK, PetList{},  // Reflect PetList struct
            builder.WithResponseDescription("A list of pets"),
        ),
        builder.WithResponse(http.StatusInternalServerError, Error{},
            builder.WithResponseDescription("Unexpected error"),
        ),
    )
    
    spec.AddOperation(http.MethodPost, "/pets",
        builder.WithOperationID("createPet"),
        builder.WithSummary("Create a pet"),
        builder.WithTags("pets"),
        builder.WithRequestBody("application/json", CreatePetRequest{},
            builder.WithRequired(true),
        ),
        builder.WithResponse(http.StatusCreated, Pet{},
            builder.WithResponseDescription("Created pet"),
        ),
    )
    
    spec.AddOperation(http.MethodGet, "/pets/{petId}",
        builder.WithOperationID("getPet"),
        builder.WithSummary("Get a pet by ID"),
        builder.WithTags("pets"),
        builder.WithPathParam("petId", int64(0),  // Reflect int64 type
            builder.WithParamDescription("The ID of the pet to retrieve"),
            builder.WithParamRequired(true),
        ),
        builder.WithResponse(http.StatusOK, Pet{},
            builder.WithResponseDescription("The requested pet"),
        ),
        builder.WithResponse(http.StatusNotFound, Error{},
            builder.WithResponseDescription("Pet not found"),
        ),
    )
    
    // Build and write - all schemas auto-registered in components.schemas
    if err := spec.WriteFile("petstore.yaml"); err != nil {
        log.Fatal(err)
    }
}
```

**Generated `components.schemas`:**
```yaml
components:
  schemas:
    Pet:
      type: object
      required: [id, name, created_at]
      properties:
        id:
          type: integer
          format: int64
          description: Unique pet identifier
        name:
          type: string
          minLength: 1
          description: Pet name
        tag:
          type: string
          description: Optional tag
        created_at:
          type: string
          format: date-time
          readOnly: true
    Error:
      type: object
      required: [code, message]
      properties:
        code:
          type: integer
          format: int32
          description: Error code
        message:
          type: string
          description: Error message
    PetList:
      type: object
      required: [items, total]
      properties:
        items:
          type: array
          items:
            $ref: '#/components/schemas/Pet'
        total:
          type: integer
    CreatePetRequest:
      type: object
      required: [name]
      properties:
        name:
          type: string
          minLength: 1
        tag:
          type: string
```

### Matching the Original API Design

The API matches the originally specified pattern:

```go
// Original requirement:
mySpec, err := builder.New(parser.OASVersion320).
    AddOperation(http.MethodGet, "/foo/bar/v1", builder.WithResponse(http.StatusOK, foo.MyResponse{}))

// foo.MyResponse is a Go type that gets reflected into a schema
type MyResponse struct {
    Success bool   `json:"success"`
    Data    string `json:"data"`
}
```

### Using with Validator

```go
// Build the spec
doc, err := spec.Build()
if err != nil {
    log.Fatal(err)
}

// Convert to ParseResult for validation
result := spec.BuildResult()

// Validate with the validator package
valResult, err := validator.ValidateWithOptions(
    validator.WithParsed(*result),
    validator.WithIncludeWarnings(true),
)
if err != nil {
    log.Fatal(err)
}
if !valResult.Valid {
    for _, issue := range valResult.Errors {
        log.Printf("Validation issue: %s", issue)
    }
}
```

### Integration with Existing Documents

```go
// Parse existing document
existing, _ := parser.ParseWithOptions(
    parser.WithFilePath("existing-api.yaml"),
)

// Create builder from existing document
spec := builder.FromDocument(existing.Document.(*parser.OAS3Document))

// Add new operation using Go types
type NewEndpointResponse struct {
    Status string `json:"status"`
}

spec.AddOperation(http.MethodPost, "/new-endpoint",
    builder.WithOperationID("newOperation"),
    builder.WithResponse(http.StatusOK, NewEndpointResponse{}),
)

// Build updated document
updated, _ := spec.Build()
```

---

## File Structure

```
builder/
├── doc.go              # Package documentation
├── builder.go          # Main Builder type and New functions
├── reflect.go          # Reflection-based schema generation
├── reflect_cache.go    # Schema caching for reflection
├── tags.go             # Struct tag parsing (json, oas)
├── operation.go        # Operation options and building
├── server.go           # Server configuration
├── response.go         # Response configuration
├── parameter.go        # Parameter configuration  
├── security.go         # Security scheme configuration
├── builder_test.go     # Core builder tests
├── reflect_test.go     # Reflection schema tests
├── tags_test.go        # Tag parsing tests
├── operation_test.go   # Operation builder tests
├── example_test.go     # Godoc examples
└── builder_bench_test.go # Performance benchmarks
```

---

## Implementation Phases

### Phase 1: Core Reflection Engine
- [ ] Create `builder/doc.go` with package documentation
- [ ] Implement `Builder` struct and `New()` function
- [ ] Implement core reflection engine (`reflect.go`)
  - [ ] Primitive type mapping (string, int, float, bool)
  - [ ] Struct reflection with property generation
  - [ ] Slice/array handling
  - [ ] Map handling with additionalProperties
  - [ ] Pointer handling (nullable)
- [ ] Implement schema caching (`reflect_cache.go`)
- [ ] Add unit tests for basic type reflection

### Phase 2: Tag Parsing & Customization
- [ ] Implement `json` tag parsing for field names and omitempty
- [ ] Implement `oas` tag parsing (`tags.go`)
  - [ ] description, format, enum
  - [ ] minimum, maximum, minLength, maxLength
  - [ ] pattern, minItems, maxItems
  - [ ] readOnly, writeOnly, nullable, deprecated
- [ ] Implement required field detection
- [ ] Add tests for all tag options

### Phase 3: Operation Integration
- [ ] Implement `AddOperation()` with reflection-based options
- [ ] Implement `WithResponse(statusCode, goType)` 
- [ ] Implement `WithRequestBody(contentType, goType)`
- [ ] Implement parameter options with type reflection
- [ ] Automatic schema registration in components.schemas
- [ ] Add operation ID uniqueness validation
- [ ] Add tests for operation building

### Phase 4: Advanced Reflection
- [ ] Handle circular references
- [ ] Handle embedded structs
- [ ] Handle interface{} / any types
- [ ] Handle special types (time.Time, uuid.UUID)
- [ ] Schema name conflict detection
- [ ] Generic type support (Go 1.18+)

### Phase 5: Document Configuration
- [ ] Implement server configuration (`AddServer()`, `ServerOption`)
- [ ] Implement tag configuration (`AddTag()`, `TagOption`)
- [ ] Implement security configuration
- [ ] Implement `WriteFile()` with format detection
- [ ] Add `MarshalYAML()` and `MarshalJSON()`
- [ ] Implement `Build()` to create `*parser.OAS3Document`

### Phase 6: Integration & Polish
- [ ] Implement `BuildResult()` for validator compatibility
- [ ] Implement `FromDocument()` to create builder from existing doc
- [ ] Add comprehensive integration tests
- [ ] Add benchmark tests for reflection performance
- [ ] Add godoc examples
- [ ] Update README.md with builder package description
- [ ] Performance optimization if needed
- [ ] Final documentation review

---

## API Reference Summary

### Builder Creation
| Function | Description |
|----------|-------------|
| `New(version)` | Create new builder for OAS version |
| `NewWithInfo(version, info)` | Create builder with pre-configured info |
| `FromDocument(doc)` | Create builder from existing document |

### Reflection-Based Schema Generation
| Function | Description |
|----------|-------------|
| `SchemaFrom(v any)` | Generate schema from Go value via reflection |
| `SchemaFromType(t reflect.Type)` | Generate schema from reflect.Type |
| `RegisterType(v any)` | Register Go type and return $ref |
| `RegisterTypeAs(name, v any)` | Register Go type with custom name |

### Operation Options (Accept Go Types)
| Option | Description |
|--------|-------------|
| `WithOperationID(id)` | Set operation ID |
| `WithSummary(s)` | Set operation summary |
| `WithDescription(d)` | Set operation description |
| `WithTags(tags...)` | Set operation tags |
| `WithResponse(code, goType, opts...)` | Add response (reflects goType) |
| `WithRequestBody(ct, goType, opts...)` | Set request body (reflects goType) |
| `WithQueryParam(name, goType, opts...)` | Add query parameter |
| `WithPathParam(name, goType, opts...)` | Add path parameter |
| `WithHeaderParam(name, goType, opts...)` | Add header parameter |
| `WithSecurity(reqs...)` | Set operation security |
| `WithDeprecated(bool)` | Mark as deprecated |

### Document Methods
| Method | Description |
|--------|-------------|
| `AddOperation(method, path, opts...)` | Add API operation |
| `RegisterType(goType)` | Register Go type as schema |
| `AddServer(url, opts...)` | Add server |
| `AddTag(name, opts...)` | Add tag |
| `SetInfo(info)` | Set document info |
| `Build()` | Create OAS3Document |
| `BuildResult()` | Create ParseResult |
| `WriteFile(path)` | Write to file |

### Struct Tag Reference
| Tag | Description |
|-----|-------------|
| `json:"name"` | JSON field name |
| `json:"name,omitempty"` | Optional field |
| `json:"-"` | Exclude from schema |
| `oas:"description=..."` | Field description |
| `oas:"format=..."` | Override format |
| `oas:"enum=a\|b\|c"` | Enumeration values |
| `oas:"minimum=N"` | Minimum value |
| `oas:"maximum=N"` | Maximum value |
| `oas:"minLength=N"` | Minimum string length |
| `oas:"maxLength=N"` | Maximum string length |
| `oas:"pattern=..."` | Regex pattern |
| `oas:"readOnly=true"` | Read-only field |
| `oas:"writeOnly=true"` | Write-only field |
| `oas:"nullable=true"` | Nullable field |
| `oas:"deprecated=true"` | Deprecated field |

---

## Compatibility Considerations

### Parser Package Integration
- Use `parser.OASVersion` constants for version specification
- Use `parser.Schema`, `parser.Operation`, etc. for all types
- Return `*parser.OAS3Document` from `Build()`
- Return `*parser.ParseResult` from `BuildResult()`

### Validator Package Integration
- `BuildResult()` creates a `ParseResult` that can be validated
- Follow same patterns as joiner/converter for validation flow

### HTTP Method Constants
- Use `httputil.Method*` constants internally
- Accept `http.Method*` from stdlib for user-facing API

### Error Handling
- Follow existing patterns from converter/joiner
- Accumulate errors during building
- Return all errors in `Build()` if validation fails

---

## Open Questions

1. **Anonymous Structs**: How should anonymous/inline structs be handled?
   - Recommendation: Inline them in the schema, don't register as components

2. **Versioning**: Should the builder support OAS 2.0 (Swagger)?
   - Recommendation: Start with OAS 3.x only, add 2.0 later if needed

3. **Validation Level**: How much validation should `Build()` perform?
   - Recommendation: Basic structural validation only; recommend users use validator package

4. **Immutability**: Should Builder methods return a new Builder (immutable) or modify in place?
   - Recommendation: Modify in place (like other packages), document as not thread-safe

5. **Custom Type Handlers**: Should users be able to register custom reflection handlers for specific types?
   - Recommendation: Support via `RegisterTypeHandler(t reflect.Type, handler func() *parser.Schema)`

6. **Performance**: Should schema generation be cached per-type or per-builder?
   - Recommendation: Per-builder cache with option to share cache across builders

---

## References

- OpenAPI Specification 3.2.0: https://spec.openapis.org/oas/v3.2.0.html
- JSON Schema Draft 2020-12: https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html
- Go reflect package: https://pkg.go.dev/reflect
- Existing parser types: `github.com/erraggy/oastools/parser`
- Functional options pattern: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
