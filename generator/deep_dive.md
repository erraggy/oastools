<a id="top"></a>

# Generator Package Deep Dive

## Table of Contents

- [Overview](#overview)
- [Generation Modes](#generation-modes)
- [Server Extensions](#server-extensions)
- [Key Features](#key-features)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Source Map Integration](#source-map-integration)
- [Configuration Reference](#configuration-reference)
- [GenerateResult Structure](#generateresult-structure)
- [Best Practices](#best-practices)

---

The [`generator`](https://pkg.go.dev/github.com/erraggy/oastools/generator) package creates idiomatic Go code from OpenAPI Specification documents. It produces type-safe API clients, server interface stubs, and model types with comprehensive support for authentication, file splitting for large APIs, and extensible code generation.

## Overview

Code generation transforms your OpenAPI specification into production-ready Go code. Rather than hand-writing HTTP clients or server handlers, the generator produces type-safe code that matches your API contract exactly. The generated code emphasizes Go idioms, proper error handling, and clean interfaces that integrate naturally with existing Go projects.

The generator supports OAS 2.0 through OAS 3.2.0, automatically adapting its output to handle version-specific features like requestBody (OAS 3.x) versus body parameters (OAS 2.0).

[↑ Back to top](#top)

## Generation Modes

The generator supports three complementary modes that can be combined:

**Client Mode** generates an HTTP client with methods for each API operation. Each method has strongly-typed parameters and return types, handles request/response serialization, and supports authentication via ClientOptions.

**Server Mode** generates interface definitions that your server code must implement. This provides a clean contract between your API specification and your implementation, ensuring type safety at compile time.

**Types Mode** generates Go structs from your schema definitions. This mode is automatically enabled when generating clients or servers, but can be used standalone when you only need the model types.

[↑ Back to top](#top)

## Server Extensions

When generating server code (`--server` or `WithServer(true)`), additional extensions provide a complete server framework with runtime validation, request binding, routing, and testing support.

### Available Server Extensions

| Flag / Option | Generated File | Description |
|---------------|----------------|-------------|
| `--server-responses` / `WithServerResponses(true)` | `server_responses.go` | Typed response writers and error helpers |
| `--server-binder` / `WithServerBinder(true)` | `server_binder.go` | Type-safe parameter binding from HTTP requests |
| `--server-middleware` / `WithServerMiddleware(true)` | `server_middleware.go` | Validation middleware using httpvalidator |
| `--server-router` / `WithServerRouter("stdlib")` | `server_router.go` | HTTP router with path matching and handler dispatch |
| `--server-stubs` / `WithServerStubs(true)` | `server_stubs.go` | Configurable stub implementations for testing |
| `--server-all` / `WithServerAll()` | All above | Enable all server extensions |

### Response Helpers (`server_responses.go`)

Response helpers provide type-safe response writing with per-operation response types:

```go
// Generated for each operation
type ListPetsResponse struct {
    statusCode int
    body       any
}

func (ListPetsResponse) Status200(pets []Pet) ListPetsResponse { ... }
func (ListPetsResponse) StatusDefault(err Error) ListPetsResponse { ... }
func (r ListPetsResponse) WriteTo(w http.ResponseWriter) error { ... }

// Common helpers
func WriteJSON(w http.ResponseWriter, statusCode int, body any) { ... }
func WriteError(w http.ResponseWriter, statusCode int, message string) { ... }
func WriteNoContent(w http.ResponseWriter) { ... }
```

**Usage:**
```go
func (s *MyServer) ListPets(ctx context.Context, req *ListPetsRequest) (ListPetsResponse, error) {
    pets, err := s.db.GetPets(req.Limit)
    if err != nil {
        return ListPetsResponse{}.StatusDefault(Error{Message: err.Error()}), nil
    }
    return ListPetsResponse{}.Status200(pets), nil
}
```

### Request Binder (`server_binder.go`)

The request binder extracts and validates parameters from HTTP requests, converting them to typed request structs:

```go
type RequestBinder struct {
    validator *httpvalidator.Validator
}

func NewRequestBinder(parsed *parser.ParseResult) (*RequestBinder, error) { ... }
func NewRequestBinderFromValidator(v *httpvalidator.Validator) *RequestBinder { ... }

// Per-operation binding methods
func (b *RequestBinder) BindListPetsRequest(r *http.Request) (*ListPetsRequest, *BindingError) { ... }
func (b *RequestBinder) BindCreatePetRequest(r *http.Request) (*CreatePetRequest, *BindingError) { ... }
```

**Usage:**
```go
binder, _ := NewRequestBinder(parsed)

http.HandleFunc("/pets", func(w http.ResponseWriter, r *http.Request) {
    req, bindErr := binder.BindListPetsRequest(r)
    if bindErr != nil {
        WriteError(w, 400, bindErr.Error())
        return
    }
    // req is now a typed *ListPetsRequest with validated parameters
})
```

### Validation Middleware (`server_middleware.go`)

Validation middleware integrates with `httpvalidator` for request/response validation:

```go
type ValidationConfig struct {
    IncludeRequestValidation  bool
    IncludeResponseValidation bool
    StrictMode                bool
    OnValidationError         func(w http.ResponseWriter, r *http.Request, result *httpvalidator.RequestValidationResult)
}

func DefaultValidationConfig() ValidationConfig { ... }
func ValidationMiddleware(parsed *parser.ParseResult) func(http.Handler) http.Handler { ... }
func ValidationMiddlewareWithConfig(parsed *parser.ParseResult, cfg ValidationConfig) func(http.Handler) http.Handler { ... }
```

**Usage:**
```go
parsed, _ := parser.ParseWithOptions(parser.WithFilePath("openapi.yaml"))

// Wrap your handler with validation
handler := ValidationMiddleware(parsed)(myHandler)

// Or with custom configuration
cfg := DefaultValidationConfig()
cfg.StrictMode = true
cfg.OnValidationError = func(w http.ResponseWriter, r *http.Request, result *httpvalidator.RequestValidationResult) {
    // result.Errors contains detailed validation errors
    WriteError(w, 422, "validation failed")
}
handler = ValidationMiddlewareWithConfig(parsed, cfg)(myHandler)
```

### Router (`server_router.go`)

The router dispatches HTTP requests to your `ServerInterface` implementation:

```go
type ServerRouter struct {
    server     ServerInterface
    validator  *httpvalidator.Validator
    middleware []func(http.Handler) http.Handler
}

func NewServerRouter(server ServerInterface, parsed *parser.ParseResult, opts ...RouterOption) (*ServerRouter, error) { ... }
func (r *ServerRouter) Handler() http.Handler { ... }
func (r *ServerRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) { ... }

// Router options
func WithMiddleware(mw ...func(http.Handler) http.Handler) RouterOption { ... }

// Path parameter helper
func PathParam(r *http.Request, name string) string { ... }
```

**Usage:**
```go
server := NewMyPetStoreServer()
parsed, _ := parser.ParseWithOptions(parser.WithFilePath("openapi.yaml"))

router, err := NewServerRouter(server, parsed,
    WithMiddleware(loggingMiddleware),
    WithMiddleware(ValidationMiddleware(parsed)),
    WithErrorHandler(func(r *http.Request, err error) {
        // Log errors server-side without exposing to clients
        log.Printf("Handler error: %s %s: %v", r.Method, r.URL.Path, err)
    }),
)
if err != nil {
    log.Fatal(err)
}

http.ListenAndServe(":8080", router)
```

**Error Handling:** The router returns a generic "internal server error" message to clients to prevent information disclosure. Use `WithErrorHandler` to log the actual error for debugging.

### Chi Router (`server_router_chi.go`)

For projects using [chi](https://github.com/go-chi/chi), enable the chi router with `--server-router chi`:

```bash
oastools generate server -i openapi.yaml -o petstore --server-router chi
```

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

server := NewMyPetStoreServer()

// Create chi router with built-in path parameter extraction
router := NewChiRouter(server,
    WithMiddleware(middleware.Logger),
    WithMiddleware(middleware.Recoverer),
    WithErrorHandler(func(r *http.Request, err error) {
        log.Printf("Handler error: %s %s: %v", r.Method, r.URL.Path, err)
    }),
)

http.ListenAndServe(":8080", router)
```

**Key differences from stdlib router:**
- Chi handles path parameter extraction natively via `chi.URLParam()`
- No need for the `PathParam()` helper function
- Compatible with chi's ecosystem of middleware
- `NewChiRouter()` returns `chi.Router` directly (no error return)

### Stub Server (`server_stubs.go`)

The stub server is a configurable mock implementation for testing:

```go
type StubServer struct {
    ListPetsFunc   func(ctx context.Context, req *ListPetsRequest) (ListPetsResponse, error)
    CreatePetFunc  func(ctx context.Context, req *CreatePetRequest) (CreatePetResponse, error)
    ShowPetByIdFunc func(ctx context.Context, req *ShowPetByIdRequest) (ShowPetByIdResponse, error)
}

func NewStubServer() *StubServer { ... }
func NewStubServerWithOptions(opts ...StubServerOption) *StubServer { ... }
func (s *StubServer) Reset() { ... }

// Per-operation options
func WithListPets(fn func(ctx context.Context, req *ListPetsRequest) (ListPetsResponse, error)) StubServerOption { ... }
```

**Usage in tests:**
```go
func TestPetAPI(t *testing.T) {
    stub := NewStubServerWithOptions(
        WithListPets(func(ctx context.Context, req *ListPetsRequest) (ListPetsResponse, error) {
            return ListPetsResponse{}.Status200([]Pet{{ID: 1, Name: "Fluffy"}}), nil
        }),
    )

    parsed, _ := parser.ParseWithOptions(parser.WithFilePath("openapi.yaml"))
    router, _ := NewServerRouter(stub, parsed)

    req := httptest.NewRequest("GET", "/pets", nil)
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, 200, rec.Code)
}
```

### Full Server Generation Example

Generate a complete server with all extensions:

```go
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("petstore.yaml"),
    generator.WithPackageName("petstore"),
    generator.WithServer(true),
    generator.WithServerAll(),  // Enable all extensions
)
if err != nil {
    log.Fatal(err)
}

// Generated files:
// - types.go           (schema types)
// - server.go          (ServerInterface, request types, UnimplementedServer)
// - server_responses.go (response types and helpers)
// - server_binder.go   (request binding)
// - server_middleware.go (validation middleware)
// - server_router.go   (HTTP routing)
// - server_stubs.go    (test stubs)

if err := result.WriteFiles("./generated/petstore"); err != nil {
    log.Fatal(err)
}
```

**CLI equivalent:**
```bash
oastools generate --server --server-all -o ./generated/petstore -p petstore petstore.yaml
```

[↑ Back to top](#top)

## Key Features

### Security Helpers

See also: [Security helpers example](https://pkg.go.dev/github.com/erraggy/oastools/generator#example-package-WithSecurityHelpers) on pkg.go.dev

The generator automatically creates authentication helpers based on your security schemes. These helpers are generated as `ClientOption` functions that configure the client for specific authentication methods.

**Supported Security Types:**

| Security Type | Generated Helper |
|--------------|------------------|
| API Key (header) | `With{Name}APIKey(key string)` |
| API Key (query) | `With{Name}APIKeyQuery(key string)` |
| API Key (cookie) | `With{Name}APIKeyCookie(key string)` |
| HTTP Basic | `With{Name}BasicAuth(username, password string)` |
| HTTP Bearer | `With{Name}BearerToken(token string)` |
| OAuth2 | `With{Name}OAuth2Token(token string)` |
| OpenID Connect | `With{Name}Token(token string)` |

### File Splitting for Large APIs

See also: [File splitting example](https://pkg.go.dev/github.com/erraggy/oastools/generator#example-package-WithFileSplitting) on pkg.go.dev

Large API specifications like Microsoft Graph or Stripe can produce thousands of lines of generated code. The generator automatically splits output across multiple files based on configurable thresholds and grouping strategies.

### Advanced Security Features

See also: [OAuth2 flows example](https://pkg.go.dev/github.com/erraggy/oastools/generator#example-package-WithOAuth2Flows) on pkg.go.dev

Beyond basic authentication helpers, the generator supports advanced security scenarios:

**OAuth2 Token Flows** generates helpers for token acquisition, refresh, and authorization code exchange.

**Credential Management** generates a `CredentialProvider` interface with built-in implementations for memory storage, environment variables, and credential chains.

**Security Enforcement** generates server-side middleware for validating security requirements on incoming requests.

**OpenID Connect Discovery** generates clients for OIDC `.well-known` endpoint discovery and auto-configuration.

[↑ Back to top](#top)

## API Styles

### Functional Options API

Best for one-off generation with inline configuration:

```go
result, err := generator.GenerateWithOptions(
    generator.WithFilePath("openapi.yaml"),
    generator.WithPackageName("petstore"),
    generator.WithClient(true),
    generator.WithServer(true),
)
```

### Struct-Based API

Best for multiple generation operations with consistent configuration:

```go
g := generator.New()
g.PackageName = "api"
g.GenerateClient = true
g.GenerateServer = true
g.UsePointers = true
g.IncludeValidation = true

result1, _ := g.Generate("users-api.yaml")
result2, _ := g.Generate("orders-api.yaml")
```

[↑ Back to top](#top)

## Practical Examples

See also: [Basic example](https://pkg.go.dev/github.com/erraggy/oastools/generator#example-package), [Client and server](https://pkg.go.dev/github.com/erraggy/oastools/generator#example-package-ClientAndServer) on pkg.go.dev

### Generating a Basic Client

The simplest use case generates a client library from an OpenAPI specification:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/generator"
)

func main() {
    result, err := generator.GenerateWithOptions(
        generator.WithFilePath("petstore.yaml"),
        generator.WithPackageName("petstore"),
        generator.WithClient(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Write generated files to directory
    if err := result.WriteFiles("./generated/petstore"); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Generated %d files\n", len(result.Files))
    for _, file := range result.Files {
        fmt.Printf("  %s (%d lines)\n", file.Name, file.LineCount)
    }
}
```

**Example Input (petstore.yaml):**
```yaml
openapi: 3.0.3
info:
  title: Pet Store API
  version: 1.0.0
servers:
  - url: https://api.petstore.example.com/v1
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewPet'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
            format: int64
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
        tag:
          type: string
    NewPet:
      type: object
      required:
        - name
      properties:
        name:
          type: string
        tag:
          type: string
```

**Generated Output Structure:**
```
generated/petstore/
├── client.go          # HTTP client implementation
├── types.go           # Pet, NewPet structs
├── security_helpers.go # Authentication helpers (if security schemes exist)
└── README.md          # Usage documentation
```

**Generated Client Usage:**
```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "myproject/generated/petstore"
)

func main() {
    // Create client with base URL
    client := petstore.NewClient("https://api.petstore.example.com/v1")
    
    // List pets with optional limit
    limit := 10
    pets, err := client.ListPets(context.Background(), &limit)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, pet := range pets {
        fmt.Printf("Pet: %s (ID: %d)\n", pet.Name, pet.ID)
    }
    
    // Create a new pet
    newPet := &petstore.NewPet{
        Name: "Fluffy",
        Tag:  stringPtr("cat"),
    }
    
    created, err := client.CreatePet(context.Background(), newPet)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Created pet with ID: %d\n", created.ID)
    
    // Get specific pet
    pet, err := client.GetPet(context.Background(), created.ID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Retrieved: %s\n", pet.Name)
}

func stringPtr(s string) *string { return &s }
```

### Generating Server Interface

Generate server interfaces for framework-agnostic implementation:

```go
package main

import (
    "log"
    
    "github.com/erraggy/oastools/generator"
)

func main() {
    result, err := generator.GenerateWithOptions(
        generator.WithFilePath("petstore.yaml"),
        generator.WithPackageName("petstore"),
        generator.WithServer(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    if err := result.WriteFiles("./generated/petstore"); err != nil {
        log.Fatal(err)
    }
}
```

**Generated Server Interface:**
```go
// Code generated by oastools. DO NOT EDIT.

package petstore

import "context"

// PetStoreServer defines the interface for the Pet Store API.
// Implement this interface to create your server.
type PetStoreServer interface {
    // ListPets returns a list of pets.
    ListPets(ctx context.Context, req *ListPetsRequest) (*ListPetsResponse, error)
    
    // CreatePet creates a new pet.
    CreatePet(ctx context.Context, req *CreatePetRequest) (*CreatePetResponse, error)
    
    // GetPet returns a specific pet by ID.
    GetPet(ctx context.Context, req *GetPetRequest) (*GetPetResponse, error)
}

// ListPetsRequest contains the parameters for ListPets.
type ListPetsRequest struct {
    Limit *int `json:"limit,omitempty"`
}

// ListPetsResponse contains the response for ListPets.
type ListPetsResponse struct {
    Pets []Pet `json:"pets"`
}

// CreatePetRequest contains the parameters for CreatePet.
type CreatePetRequest struct {
    Body NewPet `json:"body"`
}

// CreatePetResponse contains the response for CreatePet.
type CreatePetResponse struct {
    Pet Pet `json:"pet"`
}

// GetPetRequest contains the parameters for GetPet.
type GetPetRequest struct {
    PetID int64 `json:"petId"`
}

// GetPetResponse contains the response for GetPet.
type GetPetResponse struct {
    Pet Pet `json:"pet"`
}
```

**Implementing the Server:**
```go
package main

import (
    "context"
    "sync"
    "sync/atomic"
    
    "myproject/generated/petstore"
)

// PetStoreImpl implements the petstore.PetStoreServer interface.
type PetStoreImpl struct {
    mu     sync.RWMutex
    pets   map[int64]*petstore.Pet
    nextID int64
}

func NewPetStoreImpl() *PetStoreImpl {
    return &PetStoreImpl{
        pets: make(map[int64]*petstore.Pet),
    }
}

func (s *PetStoreImpl) ListPets(ctx context.Context, req *petstore.ListPetsRequest) (*petstore.ListPetsResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    pets := make([]petstore.Pet, 0, len(s.pets))
    for _, pet := range s.pets {
        pets = append(pets, *pet)
        if req.Limit != nil && len(pets) >= *req.Limit {
            break
        }
    }
    
    return &petstore.ListPetsResponse{Pets: pets}, nil
}

func (s *PetStoreImpl) CreatePet(ctx context.Context, req *petstore.CreatePetRequest) (*petstore.CreatePetResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    id := atomic.AddInt64(&s.nextID, 1)
    pet := &petstore.Pet{
        ID:   id,
        Name: req.Body.Name,
        Tag:  req.Body.Tag,
    }
    s.pets[id] = pet
    
    return &petstore.CreatePetResponse{Pet: *pet}, nil
}

func (s *PetStoreImpl) GetPet(ctx context.Context, req *petstore.GetPetRequest) (*petstore.GetPetResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    pet, ok := s.pets[req.PetID]
    if !ok {
        return nil, ErrNotFound
    }
    
    return &petstore.GetPetResponse{Pet: *pet}, nil
}
```

### Client with Security Helpers

Generate a client with authentication support:

```go
package main

import (
    "log"
    
    "github.com/erraggy/oastools/generator"
)

func main() {
    result, err := generator.GenerateWithOptions(
        generator.WithFilePath("secure-api.yaml"),
        generator.WithPackageName("api"),
        generator.WithClient(true),
        generator.WithSecurity(true),  // Enable security helpers
    )
    if err != nil {
        log.Fatal(err)
    }
    
    if err := result.WriteFiles("./generated/api"); err != nil {
        log.Fatal(err)
    }
}
```

**Example Spec with Security:**
```yaml
openapi: 3.0.3
info:
  title: Secure API
  version: 1.0.0
components:
  securitySchemes:
    apiKeyHeader:
      type: apiKey
      in: header
      name: X-API-Key
    bearerAuth:
      type: http
      scheme: bearer
    oauth2:
      type: oauth2
      flows:
        clientCredentials:
          tokenUrl: https://auth.example.com/oauth/token
          scopes:
            read: Read access
            write: Write access
```

**Generated Security Helpers:**
```go
// Code generated by oastools. DO NOT EDIT.

package api

import (
    "context"
    "encoding/base64"
    "net/http"
)

// WithApiKeyHeaderAPIKey sets the X-API-Key header for API key authentication.
func WithApiKeyHeaderAPIKey(key string) ClientOption {
    return func(c *Client) {
        c.requestMiddleware = append(c.requestMiddleware, 
            func(ctx context.Context, req *http.Request) error {
                req.Header.Set("X-API-Key", key)
                return nil
            })
    }
}

// WithBearerAuthBearerToken sets the Authorization header with a Bearer token.
func WithBearerAuthBearerToken(token string) ClientOption {
    return func(c *Client) {
        c.requestMiddleware = append(c.requestMiddleware,
            func(ctx context.Context, req *http.Request) error {
                req.Header.Set("Authorization", "Bearer "+token)
                return nil
            })
    }
}

// WithOauth2OAuth2Token sets the Authorization header with an OAuth2 token.
func WithOauth2OAuth2Token(token string) ClientOption {
    return func(c *Client) {
        c.requestMiddleware = append(c.requestMiddleware,
            func(ctx context.Context, req *http.Request) error {
                req.Header.Set("Authorization", "Bearer "+token)
                return nil
            })
    }
}
```

**Using Security Helpers:**
```go
package main

import (
    "context"
    "log"
    
    "myproject/generated/api"
)

func main() {
    // Create client with API key authentication
    client := api.NewClient(
        "https://api.example.com",
        api.WithApiKeyHeaderAPIKey("your-api-key-here"),
    )
    
    // Or use Bearer token
    client = api.NewClient(
        "https://api.example.com",
        api.WithBearerAuthBearerToken("your-jwt-token"),
    )
    
    // Or use OAuth2 token
    client = api.NewClient(
        "https://api.example.com",
        api.WithOauth2OAuth2Token("your-oauth-token"),
    )
    
    // Make authenticated requests
    result, err := client.GetProtectedResource(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Result: %+v", result)
}
```

### Advanced Security: OAuth2 Flows

Generate OAuth2 token flow helpers for complete authentication workflows:

```go
package main

import (
    "log"
    
    "github.com/erraggy/oastools/generator"
)

func main() {
    result, err := generator.GenerateWithOptions(
        generator.WithFilePath("oauth-api.yaml"),
        generator.WithPackageName("api"),
        generator.WithClient(true),
        generator.WithOAuth2Flows(true),  // Generate token flow helpers
    )
    if err != nil {
        log.Fatal(err)
    }
    
    if err := result.WriteFiles("./generated/api"); err != nil {
        log.Fatal(err)
    }
}
```

**Generated OAuth2 Flow Helpers:**
```go
// OAuth2TokenResponse represents an OAuth2 token response.
type OAuth2TokenResponse struct {
    AccessToken  string `json:"access_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
    RefreshToken string `json:"refresh_token,omitempty"`
    Scope        string `json:"scope,omitempty"`
}

// OAuth2Client handles OAuth2 authentication flows.
type OAuth2Client struct {
    TokenURL     string
    ClientID     string
    ClientSecret string
    HTTPClient   *http.Client
}

// ClientCredentialsGrant performs the client credentials grant flow.
func (c *OAuth2Client) ClientCredentialsGrant(ctx context.Context, scopes []string) (*OAuth2TokenResponse, error) {
    // Implementation handles token request
}

// RefreshToken refreshes an existing token.
func (c *OAuth2Client) RefreshToken(ctx context.Context, refreshToken string) (*OAuth2TokenResponse, error) {
    // Implementation handles token refresh
}

// WithAutoRefresh returns a ClientOption that automatically refreshes tokens.
func WithAutoRefresh() ClientOption {
    // Implementation handles automatic token refresh
}
```

**Using OAuth2 Flows:**
```go
package main

import (
    "context"
    "log"
    "time"
    
    "myproject/generated/api"
)

func main() {
    // Create OAuth2 client
    oauth := &api.OAuth2Client{
        TokenURL:     "https://auth.example.com/oauth/token",
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
    }
    
    // Get initial token
    token, err := oauth.ClientCredentialsGrant(context.Background(), []string{"read", "write"})
    if err != nil {
        log.Fatal(err)
    }
    
    // Create API client with token
    client := api.NewClient(
        "https://api.example.com",
        api.WithOauth2OAuth2Token(token.AccessToken),
    )
    
    // Or use auto-refresh for long-running applications
    client = api.NewClient(
        "https://api.example.com",
        api.WithAutoRefresh(),
    )
}
```

### Credential Management

Generate credential provider interfaces for flexible authentication:

```go
package main

import (
    "log"
    
    "github.com/erraggy/oastools/generator"
)

func main() {
    result, err := generator.GenerateWithOptions(
        generator.WithFilePath("api.yaml"),
        generator.WithPackageName("api"),
        generator.WithClient(true),
        generator.WithCredentialMgmt(true),  // Generate credential management
    )
    if err != nil {
        log.Fatal(err)
    }
    
    if err := result.WriteFiles("./generated/api"); err != nil {
        log.Fatal(err)
    }
}
```

**Generated Credential Management:**
```go
// CredentialProvider retrieves credentials for API authentication.
type CredentialProvider interface {
    GetCredential(ctx context.Context, scheme string) (string, error)
}

// MemoryCredentialProvider stores credentials in memory.
// Useful for testing and simple applications.
type MemoryCredentialProvider struct {
    credentials map[string]string
}

// EnvCredentialProvider retrieves credentials from environment variables.
type EnvCredentialProvider struct {
    // EnvMapping maps security scheme names to environment variable names.
    EnvMapping map[string]string
}

// CredentialChain tries multiple providers in order.
type CredentialChain struct {
    Providers []CredentialProvider
}

// WithCredentialProvider returns a ClientOption that uses the given provider.
func WithCredentialProvider(provider CredentialProvider) ClientOption {
    // Implementation
}
```

**Using Credential Providers:**
```go
package main

import (
    "context"
    "log"
    
    "myproject/generated/api"
)

func main() {
    // Use environment variables for credentials
    envProvider := &api.EnvCredentialProvider{
        EnvMapping: map[string]string{
            "apiKey":     "API_KEY",
            "bearerAuth": "AUTH_TOKEN",
        },
    }
    
    // Or create a chain for fallback
    chain := &api.CredentialChain{
        Providers: []api.CredentialProvider{
            envProvider,
            &api.MemoryCredentialProvider{
                credentials: map[string]string{
                    "apiKey": "fallback-key",
                },
            },
        },
    }
    
    client := api.NewClient(
        "https://api.example.com",
        api.WithCredentialProvider(chain),
    )
    
    // Credentials are resolved automatically per request
    result, err := client.GetResource(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Result: %+v", result)
}
```

### File Splitting for Large APIs

Configure file splitting when generating from large specifications:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/erraggy/oastools/generator"
)

func main() {
    result, err := generator.GenerateWithOptions(
        generator.WithFilePath("large-api.yaml"),  // 1000+ operations
        generator.WithPackageName("api"),
        generator.WithClient(true),
        
        // File splitting configuration
        generator.WithMaxLinesPerFile(2000),      // Split at 2000 lines
        generator.WithMaxTypesPerFile(200),       // Max 200 types per file
        generator.WithMaxOperationsPerFile(100),  // Max 100 operations per file
        generator.WithSplitByTag(true),           // Group by operation tags
        generator.WithSplitByPathPrefix(true),    // Fallback: group by path prefix
    )
    if err != nil {
        log.Fatal(err)
    }
    
    if err := result.WriteFiles("./generated/api"); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Generated %d files:\n", len(result.Files))
    for _, file := range result.Files {
        fmt.Printf("  %s (%d lines, %d types)\n", 
            file.Name, file.LineCount, file.TypeCount)
    }
}
```

**Example Output Structure for Large API:**
```
generated/api/
├── client.go                 # Core client type and helpers
├── users_client.go           # Operations tagged "users"
├── orders_client.go          # Operations tagged "orders"
├── products_client.go        # Operations tagged "products"
├── admin_client.go           # Operations tagged "admin"
├── types.go                  # Shared types
├── users_types.go            # Types for users operations
├── orders_types.go           # Types for orders operations
├── security_helpers.go       # Authentication helpers
└── README.md                 # Usage documentation
```

### Server-Side Security Enforcement

Generate security validation middleware for server implementations:

```go
package main

import (
    "log"
    
    "github.com/erraggy/oastools/generator"
)

func main() {
    result, err := generator.GenerateWithOptions(
        generator.WithFilePath("api.yaml"),
        generator.WithPackageName("api"),
        generator.WithServer(true),
        generator.WithSecurityEnforce(true),  // Generate enforcement middleware
    )
    if err != nil {
        log.Fatal(err)
    }
    
    if err := result.WriteFiles("./generated/api"); err != nil {
        log.Fatal(err)
    }
}
```

**Generated Security Enforcement:**
```go
// SecurityRequirement describes a security requirement for an operation.
type SecurityRequirement struct {
    Scheme string
    Scopes []string
}

// OperationSecurityRequirements maps operation IDs to their security requirements.
var OperationSecurityRequirements = map[string][]SecurityRequirement{
    "listUsers":   {{Scheme: "bearerAuth", Scopes: []string{"read"}}},
    "createUser":  {{Scheme: "bearerAuth", Scopes: []string{"write"}}},
    "adminAction": {{Scheme: "bearerAuth", Scopes: []string{"admin"}}},
}

// SecurityValidator validates security requirements for requests.
type SecurityValidator struct {
    // TokenValidator validates bearer tokens.
    TokenValidator func(token string, scopes []string) error
    // APIKeyValidator validates API keys.
    APIKeyValidator func(key string) error
}

// RequireSecurityMiddleware returns HTTP middleware that enforces security.
func RequireSecurityMiddleware(validator *SecurityValidator) func(http.Handler) http.Handler {
    // Implementation validates security per-operation
}
```

### High-Performance Generation with Pre-Parsed Documents

For workflows that combine parsing, validation, and generation:

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/erraggy/oastools/generator"
    "github.com/erraggy/oastools/parser"
    "github.com/erraggy/oastools/validator"
)

func main() {
    // Parse once
    parseResult, err := parser.ParseWithOptions(
        parser.WithFilePath("api.yaml"),
        parser.WithValidateStructure(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate
    valResult, err := validator.ValidateWithOptions(
        validator.WithParsed(*parseResult),
    )
    if err != nil {
        log.Fatal(err)
    }
    if !valResult.Valid {
        log.Fatal("Specification has validation errors")
    }
    
    // Generate from pre-parsed document
    start := time.Now()
    genResult, err := generator.GenerateWithOptions(
        generator.WithParsed(*parseResult),
        generator.WithPackageName("api"),
        generator.WithClient(true),
        generator.WithServer(true),
    )
    elapsed := time.Since(start)
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Generation completed in %v\n", elapsed)
    fmt.Printf("Files generated: %d\n", len(genResult.Files))
    
    if err := genResult.WriteFiles("./generated/api"); err != nil {
        log.Fatal(err)
    }
}
```

[↑ Back to top](#top)

## Source Map Integration

Source maps enable **precise issue locations** by tracking line and column numbers from your YAML/JSON source. Without source maps, generation issues only show JSON paths. With source maps, issues include file:line:column positions that IDEs can click to jump directly to the problematic schema or operation.

**Without source maps:**
```
warning: components.schemas.Order: schema has no properties defined
```

**With source maps:**
```
openapi.yaml:142:5: warning: schema has no properties defined
```

To enable source map tracking:

```go
parseResult, _ := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
    parser.WithSourceMap(true),  // Enable line tracking during parse
)

result, _ := generator.GenerateWithOptions(
    generator.WithParsed(*parseResult),
    generator.WithSourceMap(parseResult.SourceMap),  // Pass to generator
    generator.WithPackageName("api"),
    generator.WithClient(true),
)

// Issues now include line/column/file info
for _, issue := range result.Issues {
    if issue.HasLocation() {
        // IDE-friendly format: file:line:column
        fmt.Printf("%s: %s: %s\n", issue.Location(), issue.Severity, issue.Message)
    } else {
        // Fallback to JSON path
        fmt.Printf("%s: %s: %s\n", issue.Path, issue.Severity, issue.Message)
    }
}
```

The `Location()` method returns the IDE-friendly `file:line:column` format. The `HasLocation()` method checks if line info is available (returns `true` when `Line > 0`).

[Back to top](#top)

## Configuration Reference

### Generator Fields

```go
type Generator struct {
    // Package name for generated code (default: "api")
    PackageName string
    
    // Generation modes
    GenerateClient bool
    GenerateServer bool
    GenerateTypes  bool  // Auto-enabled with client or server
    
    // Type options
    UsePointers       bool  // Pointer types for optional fields (default: true)
    IncludeValidation bool  // Validation tags on structs (default: true)
    
    // Behavior
    StrictMode  bool  // Fail on any issues
    IncludeInfo bool  // Include informational messages
    
    // File splitting
    MaxLinesPerFile      int   // Default: 2000, 0 = no limit
    MaxTypesPerFile      int   // Default: 200
    MaxOperationsPerFile int   // Default: 100
    SplitByTag           bool  // Default: true
    SplitByPathPrefix    bool  // Default: true
    
    // Security options
    GenerateSecurity        bool  // Default: true when GenerateClient
    GenerateOAuth2Flows     bool
    GenerateCredentialMgmt  bool
    GenerateSecurityEnforce bool
    GenerateOIDCDiscovery   bool
    GenerateReadme          bool  // Default: true
}
```

### Available Options

| Option | Description |
|--------|-------------|
| `WithFilePath(string)` | Input specification path or URL |
| `WithParsed(ParseResult)` | Pre-parsed specification |
| `WithPackageName(string)` | Go package name |
| `WithClient(bool)` | Enable client generation |
| `WithServer(bool)` | Enable server generation |
| `WithTypes(bool)` | Enable types-only generation |
| `WithSecurity(bool)` | Enable security helpers |
| `WithOAuth2Flows(bool)` | Enable OAuth2 flow helpers |
| `WithCredentialMgmt(bool)` | Enable credential management |
| `WithSecurityEnforce(bool)` | Enable security enforcement |
| `WithOIDCDiscovery(bool)` | Enable OIDC discovery client |
| `WithMaxLinesPerFile(int)` | File splitting threshold |
| `WithSplitByTag(bool)` | Group operations by tag |
| `WithReadme(bool)` | Generate README.md |
| `WithServerResponses(bool)` | Generate typed response writers |
| `WithServerBinder(bool)` | Generate request parameter binding |
| `WithServerMiddleware(bool)` | Generate validation middleware |
| `WithServerRouter(string)` | Generate HTTP router ("stdlib", "chi") |
| `WithServerStubs(bool)` | Generate stub server for testing |
| `WithServerAll()` | Enable all server extensions |

[↑ Back to top](#top)

## GenerateResult Structure

```go
type GenerateResult struct {
    // Files contains all generated files
    Files []GeneratedFile
    
    // Version info
    Version    string
    OASVersion parser.OASVersion
    PackageName string
    
    // Statistics
    TypeCount      int
    OperationCount int
    SchemaCount    int
    
    // Generation info
    GenerateTime  time.Duration
    Success       bool
    
    // Issues encountered
    Issues        []GenerateIssue
    InfoCount     int
    WarningCount  int
    CriticalCount int
}

type GeneratedFile struct {
    Name      string
    Content   []byte
    LineCount int
    TypeCount int
}
```

[↑ Back to top](#top)

## Best Practices

**Always validate before generating** to ensure your specification is correct. Generation from invalid specs may produce incorrect or incomplete code.

**Use meaningful operation IDs** in your OpenAPI spec. These become method names in the generated client and server interfaces.

**Choose pointer usage based on your needs.** Pointers for optional fields (`UsePointers: true`) allows distinguishing between "not set" and "set to zero value", but requires nil checks. Value types are simpler but lose this distinction.

**Enable security helpers when generating clients** to get type-safe authentication configuration rather than manually setting headers.

**Use file splitting for large APIs** to improve compilation times and code organization. The default thresholds work well for most cases.

**Keep generated code in a separate package** from your hand-written code. This makes regeneration safe and keeps the boundary clear.

**Commit generated code to version control** so that consumers don't need the generator installed. Tag generated files with `// Code generated by oastools. DO NOT EDIT.`

**Use the README generation feature** to provide usage documentation alongside the generated code.

---

## Learn More

For additional examples and complete API documentation:

- 📦 [API Reference on pkg.go.dev](https://pkg.go.dev/github.com/erraggy/oastools/generator) - Complete API documentation with all examples
- 🖥️ [Server extensions example](https://pkg.go.dev/github.com/erraggy/oastools/generator#example-package-WithServerExtensions) - Full server framework generation
- 🔀 [Server router example](https://pkg.go.dev/github.com/erraggy/oastools/generator#example-package-WithServerRouter) - HTTP routing generation
- 🦊 [Chi router example](https://pkg.go.dev/github.com/erraggy/oastools/generator#example-package-WithChiRouter) - Chi framework support
- 🧪 [Server stubs example](https://pkg.go.dev/github.com/erraggy/oastools/generator#example-package-WithServerStubs) - Test stub generation

### Full Working Examples

The [`examples/`](../examples/) directory contains complete, generated modules you can browse:

- **[examples/petstore/](../examples/petstore/)** - Full client/server with OAuth2 flows, OIDC discovery, credential management, security enforcement, and all server extensions generated from the [Swagger Petstore API](https://petstore.swagger.io/)
