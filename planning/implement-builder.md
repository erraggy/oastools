# Builder Package Implementation Plan

> **Purpose**: This document outlines the design and implementation plan for the `builder` package,
> which provides programmatic construction of OpenAPI Specification (OAS) documents.

## Overview

The `builder` package enables users to construct OpenAPI Specification documents programmatically in Go.
Instead of manually creating YAML/JSON files, users can define their API operations, schemas, and
components using a fluent Go API, with type safety and compile-time validation.

### Goals

1. **Schema Generation**: Convert Go types to OpenAPI-compatible JSON schemas in the `components.schemas` section
2. **Operation Addition**: Provide a fluent API to add API operations with associated metadata
3. **Document Finalization**: Combine all accumulated operations and components into a complete OAS document
4. **Type Safety**: Leverage Go's type system for compile-time validation where possible
5. **Consistency**: Follow existing patterns established in `parser`, `converter`, and `joiner` packages

### Non-Goals

- Full runtime reflection-based schema generation (complex nested types requiring deep reflection)
- Validation of generated documents (users should use the `validator` package)
- Automatic endpoint discovery or code scanning
- OpenAPI 2.0 (Swagger) support initially - focus on OAS 3.x

---

## Design Components

### 1. Schema Generation

Schema generation converts Go types to OpenAPI Schema objects that can be placed in `components.schemas`.

#### Approach: Explicit Schema Definition with Helpers

Rather than full reflection-based generation, provide helper functions for common schema patterns:

```go
// Schema helpers for common types
func StringSchema() *parser.Schema
func IntSchema() *parser.Schema
func Int64Schema() *parser.Schema
func Float64Schema() *parser.Schema
func BoolSchema() *parser.Schema
func ArraySchema(items *parser.Schema) *parser.Schema
func ObjectSchema(properties map[string]*parser.Schema, required []string) *parser.Schema
func RefSchema(ref string) *parser.Schema

// Format-specific schemas
func DateTimeSchema() *parser.Schema
func DateSchema() *parser.Schema
func EmailSchema() *parser.Schema
func UUIDSchema() *parser.Schema
func URISchema() *parser.Schema

// Schema modifiers (return modified copy)
func (s *SchemaBuilder) WithDescription(desc string) *SchemaBuilder
func (s *SchemaBuilder) WithExample(example any) *SchemaBuilder
func (s *SchemaBuilder) WithEnum(values ...any) *SchemaBuilder
func (s *SchemaBuilder) WithMinimum(min float64) *SchemaBuilder
func (s *SchemaBuilder) WithMaximum(max float64) *SchemaBuilder
func (s *SchemaBuilder) WithMinLength(min int) *SchemaBuilder
func (s *SchemaBuilder) WithMaxLength(max int) *SchemaBuilder
func (s *SchemaBuilder) WithPattern(pattern string) *SchemaBuilder
func (s *SchemaBuilder) WithNullable(nullable bool) *SchemaBuilder
func (s *SchemaBuilder) Build() *parser.Schema
```

#### Schema Registration

Schemas are registered by name and automatically added to `components.schemas`:

```go
// Register a schema by name
builder.RegisterSchema("MyResponse", schema)

// Schemas are automatically referenced using $ref
// When used in operations, they generate: $ref: "#/components/schemas/MyResponse"
```

#### Example: Defining a Response Schema

```go
// Define the schema
userSchema := builder.ObjectSchema(
    map[string]*parser.Schema{
        "id":    builder.Int64Schema().WithDescription("Unique user ID").Build(),
        "name":  builder.StringSchema().WithMinLength(1).WithMaxLength(100).Build(),
        "email": builder.EmailSchema().WithDescription("User email address").Build(),
        "role":  builder.StringSchema().WithEnum("admin", "user", "guest").Build(),
        "createdAt": builder.DateTimeSchema().Build(),
    },
    []string{"id", "name", "email"}, // required fields
)

// Register it
spec.RegisterSchema("User", userSchema)
```

### 2. Operation Addition

Operations are added to paths using a fluent builder API.

#### Core API Design

```go
// Builder is the main entry point for constructing OAS documents
type Builder struct {
    version    parser.OASVersion
    info       *parser.Info
    servers    []*parser.Server
    paths      parser.Paths
    components *parser.Components
    tags       []*parser.Tag
    security   []parser.SecurityRequirement
}

// New creates a new Builder for the specified OAS version
func New(version parser.OASVersion) *Builder

// NewWithInfo creates a Builder with pre-configured Info
func NewWithInfo(version parser.OASVersion, info *parser.Info) *Builder
```

#### Operation Configuration

```go
// AddOperation adds an API operation to the specification
func (b *Builder) AddOperation(method, path string, opts ...OperationOption) *Builder

// OperationOption configures an operation
type OperationOption func(*operationConfig)

// Operation configuration options
func WithOperationID(id string) OperationOption
func WithSummary(summary string) OperationOption
func WithDescription(desc string) OperationOption
func WithTags(tags ...string) OperationOption
func WithDeprecated(deprecated bool) OperationOption

// Request configuration
func WithRequestBody(contentType string, schema *parser.Schema, opts ...RequestBodyOption) OperationOption
func WithParameter(param *parser.Parameter) OperationOption
func WithQueryParam(name string, schema *parser.Schema, opts ...ParamOption) OperationOption
func WithPathParam(name string, schema *parser.Schema, opts ...ParamOption) OperationOption
func WithHeaderParam(name string, schema *parser.Schema, opts ...ParamOption) OperationOption

// Response configuration
func WithResponse(statusCode int, schema *parser.Schema, opts ...ResponseOption) OperationOption
func WithResponseRef(statusCode int, ref string) OperationOption
func WithDefaultResponse(schema *parser.Schema, opts ...ResponseOption) OperationOption

// Security configuration
func WithSecurity(requirements ...parser.SecurityRequirement) OperationOption
func WithNoSecurity() OperationOption
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

The Builder maintains internal state for accumulated components:

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
    
    // Tracking
    operationIDs map[string]bool // Track used operation IDs for uniqueness
    errors       []error          // Accumulated validation errors
}
```

### Schema Reference Management

When schemas are used in operations, the builder automatically:
1. Registers the schema in `components.schemas` (if not already registered)
2. Returns a `$ref` reference to the schema

```go
// Internal: Convert schema usage to reference
func (b *Builder) schemaRef(name string, schema *parser.Schema) *parser.Schema {
    // Register schema if not already present
    if _, exists := b.schemas[name]; !exists {
        b.schemas[name] = schema
    }
    
    // Return a reference schema
    return &parser.Schema{
        Ref: "#/components/schemas/" + name,
    }
}
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

### Complete API Definition Example

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

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
    
    // Define schemas
    petSchema := builder.ObjectSchema(
        map[string]*parser.Schema{
            "id":   builder.Int64Schema().Build(),
            "name": builder.StringSchema().WithMinLength(1).Build(),
            "tag":  builder.StringSchema().Build(),
        },
        []string{"id", "name"},
    )
    spec.RegisterSchema("Pet", petSchema)
    
    errorSchema := builder.ObjectSchema(
        map[string]*parser.Schema{
            "code":    builder.Int32Schema().Build(),
            "message": builder.StringSchema().Build(),
        },
        []string{"code", "message"},
    )
    spec.RegisterSchema("Error", errorSchema)
    
    // Add operations
    spec.AddOperation(http.MethodGet, "/pets",
        builder.WithOperationID("listPets"),
        builder.WithSummary("List all pets"),
        builder.WithTags("pets"),
        builder.WithQueryParam("limit", 
            builder.Int32Schema().WithMaximum(100).Build(),
            builder.WithParamDescription("Maximum number of pets to return"),
        ),
        builder.WithResponse(http.StatusOK,
            builder.ArraySchema(builder.RefSchema("#/components/schemas/Pet")),
            builder.WithResponseDescription("A list of pets"),
        ),
        builder.WithResponse(http.StatusInternalServerError,
            builder.RefSchema("#/components/schemas/Error"),
            builder.WithResponseDescription("Unexpected error"),
        ),
    )
    
    spec.AddOperation(http.MethodPost, "/pets",
        builder.WithOperationID("createPet"),
        builder.WithSummary("Create a pet"),
        builder.WithTags("pets"),
        builder.WithRequestBody("application/json",
            builder.RefSchema("#/components/schemas/Pet"),
            builder.WithRequired(true),
        ),
        builder.WithResponse(http.StatusCreated,
            builder.RefSchema("#/components/schemas/Pet"),
            builder.WithResponseDescription("Created pet"),
        ),
    )
    
    spec.AddOperation(http.MethodGet, "/pets/{petId}",
        builder.WithOperationID("getPet"),
        builder.WithSummary("Get a pet by ID"),
        builder.WithTags("pets"),
        builder.WithPathParam("petId",
            builder.Int64Schema().Build(),
            builder.WithParamDescription("The ID of the pet to retrieve"),
            builder.WithParamRequired(true),
        ),
        builder.WithResponse(http.StatusOK,
            builder.RefSchema("#/components/schemas/Pet"),
            builder.WithResponseDescription("The requested pet"),
        ),
        builder.WithResponse(http.StatusNotFound,
            builder.RefSchema("#/components/schemas/Error"),
            builder.WithResponseDescription("Pet not found"),
        ),
    )
    
    // Build and write
    if err := spec.WriteFile("petstore.yaml"); err != nil {
        log.Fatal(err)
    }
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

// Add new operation
spec.AddOperation(http.MethodPost, "/new-endpoint",
    builder.WithOperationID("newOperation"),
    builder.WithResponse(http.StatusOK, builder.StringSchema().Build()),
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
├── schema.go           # Schema helper functions
├── operation.go        # Operation options and building
├── server.go           # Server configuration
├── response.go         # Response configuration
├── parameter.go        # Parameter configuration  
├── security.go         # Security scheme configuration
├── builder_test.go     # Core builder tests
├── schema_test.go      # Schema helper tests
├── operation_test.go   # Operation builder tests
├── example_test.go     # Godoc examples
└── builder_bench_test.go # Performance benchmarks
```

---

## Implementation Phases

### Phase 1: Core Foundation
- [ ] Create `builder/doc.go` with package documentation
- [ ] Implement `Builder` struct and `New()` function
- [ ] Implement basic schema helpers (string, int, bool, array, object)
- [ ] Implement `AddOperation()` with basic options
- [ ] Implement `Build()` to create `*parser.OAS3Document`
- [ ] Add unit tests for core functionality

### Phase 2: Schema Enhancement
- [ ] Add format-specific schemas (date-time, email, uuid, uri)
- [ ] Add `SchemaBuilder` with modifiers (description, example, enum, etc.)
- [ ] Implement `RegisterSchema()` for component registration
- [ ] Add automatic `$ref` generation for registered schemas
- [ ] Add tests for all schema helpers

### Phase 3: Operation Options
- [ ] Implement all `OperationOption` types
- [ ] Implement `WithRequestBody()` and `RequestBodyOption` types
- [ ] Implement `WithResponse()` and `ResponseOption` types
- [ ] Implement parameter options (query, path, header)
- [ ] Add operation ID uniqueness validation
- [ ] Add tests for all operation options

### Phase 4: Document Configuration
- [ ] Implement server configuration (`AddServer()`, `ServerOption`)
- [ ] Implement tag configuration (`AddTag()`, `TagOption`)
- [ ] Implement security configuration
- [ ] Implement `WriteFile()` with format detection
- [ ] Add `MarshalYAML()` and `MarshalJSON()`

### Phase 5: Integration
- [ ] Implement `BuildResult()` for validator compatibility
- [ ] Implement `FromDocument()` to create builder from existing doc
- [ ] Add comprehensive integration tests
- [ ] Add benchmark tests
- [ ] Add godoc examples

### Phase 6: Documentation & Polish
- [ ] Update README.md with builder package description
- [ ] Add example_test.go with comprehensive examples
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

### Schema Helpers
| Function | Description |
|----------|-------------|
| `StringSchema()` | Create string schema |
| `IntSchema()` / `Int32Schema()` / `Int64Schema()` | Create integer schemas |
| `Float32Schema()` / `Float64Schema()` | Create number schemas |
| `BoolSchema()` | Create boolean schema |
| `ArraySchema(items)` | Create array schema |
| `ObjectSchema(props, required)` | Create object schema |
| `RefSchema(ref)` | Create reference schema |
| `DateTimeSchema()` | Create date-time formatted string |
| `EmailSchema()` | Create email formatted string |
| `UUIDSchema()` | Create UUID formatted string |

### Operation Options
| Option | Description |
|--------|-------------|
| `WithOperationID(id)` | Set operation ID |
| `WithSummary(s)` | Set operation summary |
| `WithDescription(d)` | Set operation description |
| `WithTags(tags...)` | Set operation tags |
| `WithResponse(code, schema, opts...)` | Add response |
| `WithRequestBody(ct, schema, opts...)` | Set request body |
| `WithQueryParam(name, schema, opts...)` | Add query parameter |
| `WithPathParam(name, schema, opts...)` | Add path parameter |
| `WithHeaderParam(name, schema, opts...)` | Add header parameter |
| `WithSecurity(reqs...)` | Set operation security |
| `WithDeprecated(bool)` | Mark as deprecated |

### Document Methods
| Method | Description |
|--------|-------------|
| `AddOperation(method, path, opts...)` | Add API operation |
| `RegisterSchema(name, schema)` | Register component schema |
| `AddServer(url, opts...)` | Add server |
| `AddTag(name, opts...)` | Add tag |
| `SetInfo(info)` | Set document info |
| `Build()` | Create OAS3Document |
| `BuildResult()` | Create ParseResult |
| `WriteFile(path)` | Write to file |

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

1. **Reflection Support**: Should we add optional reflection-based schema generation for complex Go types?
   - Pro: Easier for users with existing Go structs
   - Con: Adds complexity, less explicit control

2. **Versioning**: Should the builder support OAS 2.0 (Swagger)?
   - Recommendation: Start with OAS 3.x only, add 2.0 later if needed

3. **Validation Level**: How much validation should `Build()` perform?
   - Recommendation: Basic structural validation only; recommend users use validator package

4. **Immutability**: Should Builder methods return a new Builder (immutable) or modify in place?
   - Recommendation: Modify in place (like other packages), document as not thread-safe

5. **Component Auto-Registration**: Should schemas be auto-registered when used in operations?
   - Recommendation: Require explicit `RegisterSchema()` for clarity

---

## References

- OpenAPI Specification 3.2.0: https://spec.openapis.org/oas/v3.2.0.html
- JSON Schema Draft 2020-12: https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html
- Existing parser types: `github.com/erraggy/oastools/parser`
- Functional options pattern: https://go.dev/blog/using-go-modules
