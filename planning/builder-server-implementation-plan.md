# Builder Server Extension Implementation Plan

**oastools v1.34.0 Feature Proposal**

**Author:** erraggy  
**Date:** December 2025  
**Status:** Draft

---

## Executive Summary

This document proposes extending the `builder` package to produce runnable HTTP servers directly from the fluent API, enabling a "code-first" development workflow. Developers would define their API specification and implementation handlers simultaneously in Go, resulting in a complete HTTP server with OAS-powered validation middleware, type-safe routing, and an automatically generated OpenAPI specification that stays synchronized with the implementation.

The extension bridges two complementary approaches: the builder's programmatic API construction with the generator's server framework output. Rather than generating code files, this approach constructs the server at runtime, providing immediate feedback, hot-reloading capabilities, and a unified development experience.

---

## Table of Contents

1. [Motivation and Goals](#1-motivation-and-goals)
2. [Design Principles](#2-design-principles)
3. [Architecture Overview](#3-architecture-overview)
4. [API Design](#4-api-design)
5. [Implementation Phases](#5-implementation-phases)
6. [Phase 1: Core Server Builder](#phase-1-core-server-builder)
7. [Phase 2: Validation Integration](#phase-2-validation-integration)
8. [Phase 3: Router Strategies](#phase-3-router-strategies)
9. [Phase 4: Handler Binding](#phase-4-handler-binding)
10. [Phase 5: Response Helpers](#phase-5-response-helpers)
11. [Phase 6: Testing Support](#phase-6-testing-support)
12. [File Structure](#7-file-structure)
13. [Testing Strategy](#8-testing-strategy)
14. [Documentation](#9-documentation)
15. [Migration Path](#10-migration-path)
16. [Future Considerations](#11-future-considerations)
17. [Risk Assessment](#12-risk-assessment)
18. [Timeline Estimate](#13-timeline-estimate)

---

## 1. Motivation and Goals

### Current State

The oastools ecosystem currently supports two distinct workflows. In the "spec-first" approach, developers write OpenAPI specifications and use the `generator` package to produce server interfaces, types, middleware, and routers. In the "code-first" approach, developers use the `builder` package to construct OpenAPI documents programmatically from Go types, then separately implement the server.

These workflows remain disconnected. A developer using the builder must still manually wire up routing, validation middleware, and handler dispatch. The generator produces excellent server scaffolding, but requires a specification file as input rather than accepting the in-memory document the builder produces.

### Goals

This extension aims to achieve several objectives. First, it should provide a unified code-first experience where developers define API operations and their implementations in a single fluent API, with the OpenAPI specification generated as a byproduct. Second, it should enable runtime server construction that builds an `http.Handler` directly from builder state without code generation, supporting hot-reloading and rapid iteration. Third, it should provide automatic validation integration where all requests are validated against the builder's accumulated specification using the existing `httpvalidator` package. Fourth, it should offer flexible routing with stdlib `net/http` as the default router with zero dependencies, and optional chi integration for teams already using that framework. Fifth, it should maintain full specification access so the built OpenAPI document remains accessible for serving via `/openapi.yaml`, integration testing, and documentation generation.

### Non-Goals

This extension explicitly excludes certain capabilities. It does not aim to replace the generator package, which remains the appropriate choice for spec-first workflows and static code generation. It does not support async/streaming patterns in the initial implementation, though the architecture should accommodate future extension. It does not include automatic client generation, which remains the generator's responsibility.

---

## 2. Design Principles

### Principle 1: Zero Runtime Dependencies for Core Functionality

The stdlib router implementation must not introduce any new dependencies beyond what oastools already requires. Chi support is opt-in and isolated in a separate file with build constraints if desired, or simply documented as requiring users to import chi themselves.

### Principle 2: Composition Over Generation

Rather than generating code files, the server builder composes existing oastools components at runtime. This approach enables immediate feedback during development, supports hot-reloading without rebuild cycles, keeps the API surface minimal and discoverable, and maintains a single source of truth in the builder state.

### Principle 3: Type Safety Where Possible

While full compile-time type safety for request/response types requires code generation, the runtime approach should leverage generics and type assertions to provide as much safety as practical. Handler registration should accept generic handler functions that receive typed requests.

### Principle 4: Graceful Degradation

If validation fails or middleware encounters errors, the server should provide meaningful error responses rather than panicking. Unhandled operations should return 501 Not Implemented with helpful context.

### Principle 5: Testability

The resulting server must be easily testable with `httptest`, support stub handlers for integration testing, and expose the underlying specification for contract testing.

---

## 3. Architecture Overview

### Component Relationships

```
┌─────────────────────────────────────────────────────────────────────┐
│                         ServerBuilder                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │
│  │   Builder   │  │  Handlers   │  │ Middleware  │  │   Router   │ │
│  │  (OAS doc)  │  │   (map)     │  │   (chain)   │  │  Strategy  │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬──────┘ │
│         │                │                │                │        │
│         ▼                ▼                ▼                ▼        │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    BuildServer()                              │  │
│  │  1. Build OAS document (BuildOAS3/BuildOAS2)                  │  │
│  │  2. Create ParseResult for httpvalidator                      │  │
│  │  3. Initialize httpvalidator.Validator                        │  │
│  │  4. Build router with path matching                           │  │
│  │  5. Wire middleware chain                                     │  │
│  │  6. Return http.Handler + ServerResult                        │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                        ┌───────────────────────┐
                        │    ServerResult       │
                        │  - Handler            │
                        │  - Spec (OAS3/OAS2)   │
                        │  - ParseResult        │
                        │  - Validator          │
                        └───────────────────────┘
```

### Data Flow

Request processing follows this sequence. The HTTP request arrives at the router, which matches the path to a registered operation using `httpvalidator.PathMatcherSet` (stdlib) or native chi matching. The validation middleware validates parameters and body against the OAS document. The handler receives a context-enriched request with validated, deserialized parameters. The handler returns a response that is optionally validated before sending. The response is serialized and sent to the client.

---

## 4. API Design

### Core Types

```go
// ServerBuilder extends Builder to support server construction.
// It embeds Builder, inheriting all specification construction methods.
type ServerBuilder struct {
    *Builder
    handlers    map[string]HandlerFunc  // operationID -> handler
    middleware  []Middleware
    router      RouterStrategy
    errorHandler ErrorHandler
    config      ServerConfig
}

// HandlerFunc is the signature for operation handlers.
// The Request contains validated parameters and the raw http.Request.
// The Response interface allows type-safe response construction.
type HandlerFunc func(ctx context.Context, req *Request) Response

// Request contains validated request data.
type Request struct {
    HTTPRequest  *http.Request
    PathParams   map[string]any
    QueryParams  map[string]any
    HeaderParams map[string]any
    CookieParams map[string]any
    Body         any  // Unmarshaled request body
    RawBody      []byte
    OperationID  string
    MatchedPath  string
}

// Response is implemented by response types.
type Response interface {
    StatusCode() int
    Headers() http.Header
    Body() any
    WriteTo(w http.ResponseWriter) error
}

// ServerResult contains the built server and related artifacts.
type ServerResult struct {
    Handler     http.Handler
    Spec        any  // *parser.OAS3Document or *parser.OAS2Document
    ParseResult *parser.ParseResult
    Validator   *httpvalidator.Validator
}

// Middleware wraps handlers with additional behavior.
type Middleware func(http.Handler) http.Handler

// RouterStrategy defines how paths are matched to handlers.
type RouterStrategy interface {
    Build(operations []operationRoute, dispatcher http.Handler) http.Handler
    PathParam(r *http.Request, name string) string
}

// ErrorHandler handles errors during request processing.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
```

### Builder Extension API

```go
// NewServerBuilder creates a ServerBuilder for the specified OAS version.
func NewServerBuilder(version parser.OASVersion, opts ...ServerBuilderOption) *ServerBuilder

// FromBuilder creates a ServerBuilder from an existing Builder.
func FromBuilder(b *Builder, opts ...ServerBuilderOption) *ServerBuilder

// Handle registers a handler for an operation.
// The operation must have been added via AddOperation with an operationID.
func (s *ServerBuilder) Handle(operationID string, handler HandlerFunc) *ServerBuilder

// HandleFunc registers a handler using a standard http.HandlerFunc.
// This is useful for operations that don't need typed parameters.
func (s *ServerBuilder) HandleFunc(operationID string, handler http.HandlerFunc) *ServerBuilder

// Use adds middleware to the server.
// Middleware is applied in order: first added = outermost.
func (s *ServerBuilder) Use(mw ...Middleware) *ServerBuilder

// WithValidation adds validation middleware using the built specification.
// This is added automatically unless disabled via WithoutValidation().
func (s *ServerBuilder) WithValidation(opts ...ValidationOption) *ServerBuilder

// BuildServer constructs the http.Handler and related artifacts.
// Returns an error if required handlers are missing or the spec is invalid.
func (s *ServerBuilder) BuildServer() (*ServerResult, error)

// MustBuildServer is like BuildServer but panics on error.
// Useful for main() or init() where errors are fatal.
func (s *ServerBuilder) MustBuildServer() *ServerResult
```

### Configuration Options

```go
// ServerBuilderOption configures a ServerBuilder.
type ServerBuilderOption func(*serverBuilderConfig)

// WithRouter sets the routing strategy.
// Default: StdlibRouter
func WithRouter(strategy RouterStrategy) ServerBuilderOption

// WithStdlibRouter uses net/http with PathMatcherSet for routing.
// This is the default and adds no dependencies.
func WithStdlibRouter() ServerBuilderOption

// WithChiRouter uses go-chi/chi for routing.
// Requires: go get github.com/go-chi/chi/v5
func WithChiRouter() ServerBuilderOption

// WithoutValidation disables automatic request validation.
// Use when validation is handled elsewhere or for maximum performance.
func WithoutValidation() ServerBuilderOption

// WithValidationConfig sets validation middleware configuration.
func WithValidationConfig(cfg ValidationConfig) ServerBuilderOption

// WithErrorHandler sets the error handler for handler panics and errors.
func WithErrorHandler(handler ErrorHandler) ServerBuilderOption

// WithNotFoundHandler sets the handler for unmatched paths.
func WithNotFoundHandler(handler http.Handler) ServerBuilderOption

// WithMethodNotAllowedHandler sets the handler for unmatched methods.
func WithMethodNotAllowedHandler(handler http.Handler) ServerBuilderOption

// WithRecovery enables panic recovery middleware.
// Recovered panics are passed to the error handler.
func WithRecovery() ServerBuilderOption

// WithRequestLogging enables request logging middleware.
func WithRequestLogging(logger func(method, path string, status int, duration time.Duration)) ServerBuilderOption
```

### Response Helpers

```go
// JSON creates a JSON response with the given status and body.
func JSON(status int, body any) Response

// NoContent creates a 204 No Content response.
func NoContent() Response

// Error creates an error response with status and message.
func Error(status int, message string) Response

// ErrorWithDetails creates an error response with additional details.
func ErrorWithDetails(status int, message string, details any) Response

// Redirect creates a redirect response.
func Redirect(status int, location string) Response

// Stream creates a streaming response.
func Stream(status int, contentType string, reader io.Reader) Response

// ResponseBuilder provides fluent response construction.
type ResponseBuilder struct {
    status  int
    headers http.Header
    body    any
}

func NewResponse(status int) *ResponseBuilder
func (r *ResponseBuilder) Header(key, value string) *ResponseBuilder
func (r *ResponseBuilder) JSON(body any) Response
func (r *ResponseBuilder) XML(body any) Response
func (r *ResponseBuilder) Text(body string) Response
func (r *ResponseBuilder) Binary(contentType string, data []byte) Response
```

### Complete Usage Example

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/erraggy/oastools/builder"
    "github.com/erraggy/oastools/parser"
)

// Domain types (automatically generate OAS schemas)
type Pet struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
    Tag  string `json:"tag,omitempty"`
}

type Error struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

func main() {
    // Create server builder
    srv := builder.NewServerBuilder(parser.OASVersion320).
        SetTitle("Pet Store API").
        SetVersion("1.0.0").
        AddServer("http://localhost:8080", builder.WithServerDescription("Local development"))

    // Define operations with handlers inline
    srv.AddOperation(http.MethodGet, "/pets",
        builder.WithOperationID("listPets"),
        builder.WithQueryParam("limit", int32(0), builder.WithParamDescription("Maximum items to return")),
        builder.WithResponse(http.StatusOK, []Pet{}),
        builder.WithResponse(http.StatusDefault, Error{}),
    ).Handle("listPets", listPetsHandler)

    srv.AddOperation(http.MethodPost, "/pets",
        builder.WithOperationID("createPet"),
        builder.WithRequestBody("application/json", Pet{}),
        builder.WithResponse(http.StatusCreated, Pet{}),
        builder.WithResponse(http.StatusDefault, Error{}),
    ).Handle("createPet", createPetHandler)

    srv.AddOperation(http.MethodGet, "/pets/{petId}",
        builder.WithOperationID("getPet"),
        builder.WithPathParam("petId", int64(0)),
        builder.WithResponse(http.StatusOK, Pet{}),
        builder.WithResponse(http.StatusNotFound, Error{}),
    ).Handle("getPet", getPetHandler)

    // Build the server
    result, err := srv.BuildServer()
    if err != nil {
        log.Fatal(err)
    }

    // Optionally serve the spec
    http.Handle("/openapi.yaml", serveSpec(result.Spec))
    http.Handle("/", result.Handler)

    log.Println("Server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func listPetsHandler(ctx context.Context, req *builder.Request) builder.Response {
    limit := int32(100)
    if l, ok := req.QueryParams["limit"].(int64); ok {
        limit = int32(l)
    }

    pets := fetchPets(limit)
    return builder.JSON(http.StatusOK, pets)
}

func createPetHandler(ctx context.Context, req *builder.Request) builder.Response {
    pet, ok := req.Body.(*Pet)
    if !ok {
        return builder.Error(http.StatusBadRequest, "invalid request body")
    }

    created := savePet(pet)
    return builder.JSON(http.StatusCreated, created)
}

func getPetHandler(ctx context.Context, req *builder.Request) builder.Response {
    petID := req.PathParams["petId"].(int64)

    pet, found := findPet(petID)
    if !found {
        return builder.Error(http.StatusNotFound, "pet not found")
    }
    return builder.JSON(http.StatusOK, pet)
}

func serveSpec(spec any) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/yaml")
        // Marshal spec to YAML and write
    })
}
```

---

## 5. Implementation Phases

The implementation is divided into six phases, each building on the previous. This phased approach allows for incremental delivery and testing.

---

## Phase 1: Core Server Builder

### Objective

Establish the foundational `ServerBuilder` type that extends `Builder` and supports handler registration. This phase delivers a minimal working server without validation.

### New Files

**builder/server_builder.go**

```go
package builder

import (
    "context"
    "fmt"
    "net/http"
    "sync"

    "github.com/erraggy/oastools/httpvalidator"
    "github.com/erraggy/oastools/parser"
)

// ServerBuilder extends Builder to support server construction.
type ServerBuilder struct {
    *Builder
    mu           sync.RWMutex
    handlers     map[string]HandlerFunc
    middleware   []Middleware
    router       RouterStrategy
    errorHandler ErrorHandler
    config       serverBuilderConfig
}

type serverBuilderConfig struct {
    enableValidation     bool
    validationConfig     ValidationConfig
    notFoundHandler      http.Handler
    methodNotAllowed     http.Handler
    enableRecovery       bool
    requestLogger        func(method, path string, status int, duration time.Duration)
}

// NewServerBuilder creates a ServerBuilder for the specified OAS version.
func NewServerBuilder(version parser.OASVersion, opts ...ServerBuilderOption) *ServerBuilder {
    cfg := defaultServerBuilderConfig()
    for _, opt := range opts {
        opt(&cfg)
    }

    return &ServerBuilder{
        Builder:      New(version),
        handlers:     make(map[string]HandlerFunc),
        middleware:   make([]Middleware, 0),
        router:       cfg.router,
        errorHandler: cfg.errorHandler,
        config:       cfg,
    }
}

func defaultServerBuilderConfig() serverBuilderConfig {
    return serverBuilderConfig{
        enableValidation: true,
        validationConfig: DefaultValidationConfig(),
        errorHandler:     defaultErrorHandler,
        router:           &stdlibRouter{},
    }
}

// Handle registers a handler for an operation by operationID.
func (s *ServerBuilder) Handle(operationID string, handler HandlerFunc) *ServerBuilder {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.handlers[operationID] = handler
    return s
}

// HandleFunc registers a handler using standard http.HandlerFunc signature.
func (s *ServerBuilder) HandleFunc(operationID string, handler http.HandlerFunc) *ServerBuilder {
    return s.Handle(operationID, func(ctx context.Context, req *Request) Response {
        // Create a response recorder to capture the http.HandlerFunc output
        rec := &responseCapture{header: make(http.Header)}
        handler(rec, req.HTTPRequest)
        return &capturedResponse{
            status:  rec.status,
            headers: rec.header,
            body:    rec.body,
        }
    })
}

// Use adds middleware to the server.
func (s *ServerBuilder) Use(mw ...Middleware) *ServerBuilder {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.middleware = append(s.middleware, mw...)
    return s
}

// BuildServer constructs the http.Handler and related artifacts.
func (s *ServerBuilder) BuildServer() (*ServerResult, error) {
    // Build the OAS document
    doc, err := s.buildDocument()
    if err != nil {
        return nil, fmt.Errorf("builder: failed to build OAS document: %w", err)
    }

    // Create ParseResult for httpvalidator
    parseResult, err := s.createParseResult(doc)
    if err != nil {
        return nil, fmt.Errorf("builder: failed to create parse result: %w", err)
    }

    // Create validator if enabled
    var validator *httpvalidator.Validator
    if s.config.enableValidation {
        validator, err = httpvalidator.New(parseResult)
        if err != nil {
            return nil, fmt.Errorf("builder: failed to create validator: %w", err)
        }
    }

    // Build route table
    routes, err := s.buildRoutes(parseResult)
    if err != nil {
        return nil, fmt.Errorf("builder: failed to build routes: %w", err)
    }

    // Create dispatcher
    dispatcher := s.buildDispatcher(routes, validator)

    // Build router
    handler := s.router.Build(routes, dispatcher)

    // Apply middleware (in reverse order so first added is outermost)
    for i := len(s.middleware) - 1; i >= 0; i-- {
        handler = s.middleware[i](handler)
    }

    return &ServerResult{
        Handler:     handler,
        Spec:        doc,
        ParseResult: parseResult,
        Validator:   validator,
    }, nil
}

// MustBuildServer is like BuildServer but panics on error.
func (s *ServerBuilder) MustBuildServer() *ServerResult {
    result, err := s.BuildServer()
    if err != nil {
        panic(err)
    }
    return result
}
```

**builder/server_types.go**

```go
package builder

import (
    "context"
    "net/http"

    "github.com/erraggy/oastools/httpvalidator"
    "github.com/erraggy/oastools/parser"
)

// HandlerFunc is the signature for operation handlers.
type HandlerFunc func(ctx context.Context, req *Request) Response

// Request contains validated request data.
type Request struct {
    HTTPRequest  *http.Request
    PathParams   map[string]any
    QueryParams  map[string]any
    HeaderParams map[string]any
    CookieParams map[string]any
    Body         any
    RawBody      []byte
    OperationID  string
    MatchedPath  string
}

// Response is implemented by response types.
type Response interface {
    StatusCode() int
    Headers() http.Header
    Body() any
    WriteTo(w http.ResponseWriter) error
}

// ServerResult contains the built server and related artifacts.
type ServerResult struct {
    Handler     http.Handler
    Spec        any
    ParseResult *parser.ParseResult
    Validator   *httpvalidator.Validator
}

// Middleware wraps handlers with additional behavior.
type Middleware func(http.Handler) http.Handler

// ErrorHandler handles errors during request processing.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)
```

### Acceptance Criteria

1. `NewServerBuilder` creates a functional builder that inherits all `Builder` methods
2. `Handle` registers handlers by operation ID
3. `BuildServer` produces a working `http.Handler`
4. The handler correctly routes requests to registered handlers
5. Unhandled operations return 501 Not Implemented
6. The `ServerResult` contains the built OAS document
7. All existing builder tests continue to pass

---

## Phase 2: Validation Integration

### Objective

Integrate `httpvalidator` for automatic request validation. Validated and deserialized parameters should be available in the `Request` struct passed to handlers.

### Changes

**builder/server_validation.go**

```go
package builder

import (
    "context"
    "net/http"

    "github.com/erraggy/oastools/httpvalidator"
)

// ValidationConfig configures request/response validation.
type ValidationConfig struct {
    IncludeRequestValidation  bool
    IncludeResponseValidation bool
    StrictMode                bool
    OnValidationError         ValidationErrorHandler
}

// ValidationErrorHandler handles validation failures.
type ValidationErrorHandler func(w http.ResponseWriter, r *http.Request, result *httpvalidator.RequestValidationResult)

// DefaultValidationConfig returns sensible defaults.
func DefaultValidationConfig() ValidationConfig {
    return ValidationConfig{
        IncludeRequestValidation:  true,
        IncludeResponseValidation: false,
        StrictMode:                false,
        OnValidationError:         nil,
    }
}

// validationMiddleware creates the validation middleware.
func validationMiddleware(v *httpvalidator.Validator, cfg ValidationConfig) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !cfg.IncludeRequestValidation {
                next.ServeHTTP(w, r)
                return
            }

            result, err := v.ValidateRequest(r)
            if err != nil {
                writeValidationError(w, http.StatusInternalServerError, err.Error())
                return
            }

            hasErrors := len(result.Errors) > 0
            hasWarnings := len(result.Warnings) > 0 && cfg.StrictMode

            if hasErrors || hasWarnings {
                if cfg.OnValidationError != nil {
                    cfg.OnValidationError(w, r, result)
                    return
                }
                writeValidationResult(w, result)
                return
            }

            // Store validated params in context for handler access
            ctx := contextWithValidationResult(r.Context(), result)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

type validationResultKey struct{}

func contextWithValidationResult(ctx context.Context, result *httpvalidator.RequestValidationResult) context.Context {
    return context.WithValue(ctx, validationResultKey{}, result)
}

func validationResultFromContext(ctx context.Context) *httpvalidator.RequestValidationResult {
    if result, ok := ctx.Value(validationResultKey{}).(*httpvalidator.RequestValidationResult); ok {
        return result
    }
    return nil
}
```

### Acceptance Criteria

1. Requests are automatically validated against the OAS specification
2. Invalid requests receive 400 Bad Request with error details
3. Validated parameters are available in `Request.PathParams`, `QueryParams`, etc.
4. `WithoutValidation()` option disables validation middleware
5. Custom validation error handlers are supported
6. Strict mode treats warnings as errors
7. Response validation is optional and works when enabled

---

## Phase 3: Router Strategies

### Objective

Implement the stdlib router using `httpvalidator.PathMatcherSet` and provide an interface for chi integration.

### New Files

**builder/server_router_stdlib.go**

```go
package builder

import (
    "context"
    "net/http"
    "strings"

    "github.com/erraggy/oastools/httpvalidator"
)

// RouterStrategy defines how paths are matched to handlers.
type RouterStrategy interface {
    Build(routes []operationRoute, dispatcher http.Handler) http.Handler
    PathParam(r *http.Request, name string) string
}

// operationRoute represents a registered route.
type operationRoute struct {
    Method      string
    Path        string
    OperationID string
    Handler     HandlerFunc
}

// stdlibRouter implements RouterStrategy using net/http and PathMatcherSet.
type stdlibRouter struct {
    notFound         http.Handler
    methodNotAllowed http.Handler
}

func (r *stdlibRouter) Build(routes []operationRoute, dispatcher http.Handler) http.Handler {
    // Build PathMatcherSet from routes
    patterns := make([]string, 0, len(routes))
    for _, route := range routes {
        patterns = append(patterns, route.Path)
    }
    matcher := httpvalidator.NewPathMatcherSet(patterns)

    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        // Match path
        matched, params, found := matcher.Match(req.URL.Path)
        if !found {
            if r.notFound != nil {
                r.notFound.ServeHTTP(w, req)
            } else {
                http.NotFound(w, req)
            }
            return
        }

        // Store matched path and params in context
        ctx := req.Context()
        ctx = context.WithValue(ctx, matchedPathKey{}, matched)
        ctx = context.WithValue(ctx, pathParamsKey{}, params)

        dispatcher.ServeHTTP(w, req.WithContext(ctx))
    })
}

func (r *stdlibRouter) PathParam(req *http.Request, name string) string {
    if params, ok := req.Context().Value(pathParamsKey{}).(map[string]string); ok {
        return params[name]
    }
    return ""
}

type matchedPathKey struct{}
type pathParamsKey struct{}

// PathParam extracts a path parameter from the request.
// This is a package-level function for convenience.
func PathParam(r *http.Request, name string) string {
    if params, ok := r.Context().Value(pathParamsKey{}).(map[string]string); ok {
        return params[name]
    }
    return ""
}
```

**builder/server_router_chi.go**

```go
package builder

import (
    "net/http"

    "github.com/go-chi/chi/v5"
)

// chiRouter implements RouterStrategy using go-chi/chi.
type chiRouter struct {
    middleware []func(http.Handler) http.Handler
}

// ChiRouter returns a RouterStrategy that uses chi for routing.
// Requires: go get github.com/go-chi/chi/v5
func ChiRouter(opts ...ChiRouterOption) RouterStrategy {
    r := &chiRouter{}
    for _, opt := range opts {
        opt(r)
    }
    return r
}

// ChiRouterOption configures the chi router.
type ChiRouterOption func(*chiRouter)

// WithChiMiddleware adds chi-compatible middleware.
func WithChiMiddleware(mw ...func(http.Handler) http.Handler) ChiRouterOption {
    return func(r *chiRouter) {
        r.middleware = append(r.middleware, mw...)
    }
}

func (r *chiRouter) Build(routes []operationRoute, dispatcher http.Handler) http.Handler {
    router := chi.NewRouter()

    // Apply chi middleware
    for _, mw := range r.middleware {
        router.Use(mw)
    }

    // Register routes
    for _, route := range routes {
        method := route.Method
        path := route.Path

        router.Method(method, path, dispatcher)
    }

    return router
}

func (r *chiRouter) PathParam(req *http.Request, name string) string {
    return chi.URLParam(req, name)
}
```

### Acceptance Criteria

1. Stdlib router matches paths using `PathMatcherSet`
2. Path parameters are extracted and available via `PathParam()`
3. 404 Not Found is returned for unmatched paths
4. Chi router integration compiles and routes correctly
5. Both routers support the same `RouterStrategy` interface
6. Path parameter extraction works for both routers

---

## Phase 4: Handler Binding

### Objective

Create the dispatcher that binds requests to handlers, populating the `Request` struct with validated parameters and unmarshaled request body.

### Changes

**builder/server_dispatcher.go**

```go
package builder

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"

    "github.com/erraggy/oastools/httpvalidator"
    "github.com/erraggy/oastools/parser"
)

// dispatcher handles routing validated requests to handlers.
type dispatcher struct {
    routes       map[string]map[string]operationRoute // path -> method -> route
    errorHandler ErrorHandler
    bodySchemas  map[string]*parser.Schema            // operationID -> request body schema
}

func (s *ServerBuilder) buildDispatcher(routes []operationRoute, v *httpvalidator.Validator) http.Handler {
    d := &dispatcher{
        routes:       make(map[string]map[string]operationRoute),
        errorHandler: s.errorHandler,
        bodySchemas:  make(map[string]*parser.Schema),
    }

    // Index routes by path and method
    for _, route := range routes {
        if d.routes[route.Path] == nil {
            d.routes[route.Path] = make(map[string]operationRoute)
        }
        d.routes[route.Path][route.Method] = route
    }

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get matched path from context
        matchedPath, _ := r.Context().Value(matchedPathKey{}).(string)
        if matchedPath == "" {
            http.NotFound(w, r)
            return
        }

        // Find route
        methods, ok := d.routes[matchedPath]
        if !ok {
            http.NotFound(w, r)
            return
        }

        route, ok := methods[r.Method]
        if !ok {
            w.Header().Set("Allow", strings.Join(allowedMethods(methods), ", "))
            http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
            return
        }

        // Check if handler is registered
        if route.Handler == nil {
            http.Error(w, fmt.Sprintf("Operation %q not implemented", route.OperationID), http.StatusNotImplemented)
            return
        }

        // Build Request struct
        req := d.buildRequest(r, route)

        // Call handler
        ctx := r.Context()
        resp := route.Handler(ctx, req)

        // Write response
        if err := resp.WriteTo(w); err != nil {
            if d.errorHandler != nil {
                d.errorHandler(w, r, err)
            }
        }
    })
}

func (d *dispatcher) buildRequest(r *http.Request, route operationRoute) *Request {
    req := &Request{
        HTTPRequest: r,
        OperationID: route.OperationID,
        MatchedPath: route.Path,
        PathParams:  make(map[string]any),
        QueryParams: make(map[string]any),
        HeaderParams: make(map[string]any),
        CookieParams: make(map[string]any),
    }

    // Get validation result from context (if validation is enabled)
    if result := validationResultFromContext(r.Context()); result != nil {
        req.PathParams = result.PathParams
        req.QueryParams = result.QueryParams
        req.HeaderParams = result.HeaderParams
        req.CookieParams = result.CookieParams
    } else {
        // Fallback: extract raw path params from context
        if params, ok := r.Context().Value(pathParamsKey{}).(map[string]string); ok {
            for k, v := range params {
                req.PathParams[k] = v
            }
        }
    }

    // Read and unmarshal body if present
    if r.Body != nil && r.ContentLength > 0 {
        body, err := io.ReadAll(r.Body)
        if err == nil {
            req.RawBody = body
            // Attempt JSON unmarshal
            var parsed any
            if json.Unmarshal(body, &parsed) == nil {
                req.Body = parsed
            }
        }
    }

    return req
}

func allowedMethods(methods map[string]operationRoute) []string {
    result := make([]string, 0, len(methods))
    for method := range methods {
        result = append(result, method)
    }
    return result
}
```

### Acceptance Criteria

1. Requests are dispatched to the correct handler based on path and method
2. Path parameters are populated from validation result or router
3. Query, header, and cookie parameters are populated when validation is enabled
4. Request body is read and available as both `RawBody` and unmarshaled `Body`
5. Method Not Allowed returns 405 with Allow header
6. Not Implemented returns 501 for operations without handlers
7. Error handler is called for response write errors

---

## Phase 5: Response Helpers

### Objective

Provide convenient response construction helpers that implement the `Response` interface.

### New Files

**builder/server_response.go**

```go
package builder

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io"
    "net/http"
)

// jsonResponse implements Response for JSON bodies.
type jsonResponse struct {
    status  int
    headers http.Header
    body    any
}

// JSON creates a JSON response.
func JSON(status int, body any) Response {
    return &jsonResponse{
        status:  status,
        headers: make(http.Header),
        body:    body,
    }
}

func (r *jsonResponse) StatusCode() int      { return r.status }
func (r *jsonResponse) Headers() http.Header { return r.headers }
func (r *jsonResponse) Body() any            { return r.body }

func (r *jsonResponse) WriteTo(w http.ResponseWriter) error {
    for k, v := range r.headers {
        w.Header()[k] = v
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(r.status)

    if r.body == nil {
        return nil
    }
    return json.NewEncoder(w).Encode(r.body)
}

// noContentResponse implements Response for 204 No Content.
type noContentResponse struct{}

// NoContent creates a 204 No Content response.
func NoContent() Response {
    return &noContentResponse{}
}

func (r *noContentResponse) StatusCode() int      { return http.StatusNoContent }
func (r *noContentResponse) Headers() http.Header { return nil }
func (r *noContentResponse) Body() any            { return nil }

func (r *noContentResponse) WriteTo(w http.ResponseWriter) error {
    w.WriteHeader(http.StatusNoContent)
    return nil
}

// errorResponse implements Response for error messages.
type errorResponse struct {
    status  int
    message string
    details any
}

// Error creates an error response.
func Error(status int, message string) Response {
    return &errorResponse{status: status, message: message}
}

// ErrorWithDetails creates an error response with additional details.
func ErrorWithDetails(status int, message string, details any) Response {
    return &errorResponse{status: status, message: message, details: details}
}

func (r *errorResponse) StatusCode() int      { return r.status }
func (r *errorResponse) Headers() http.Header { return nil }
func (r *errorResponse) Body() any {
    body := map[string]any{
        "error": r.message,
    }
    if r.details != nil {
        body["details"] = r.details
    }
    return body
}

func (r *errorResponse) WriteTo(w http.ResponseWriter) error {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(r.status)
    return json.NewEncoder(w).Encode(r.Body())
}

// redirectResponse implements Response for redirects.
type redirectResponse struct {
    status   int
    location string
}

// Redirect creates a redirect response.
func Redirect(status int, location string) Response {
    return &redirectResponse{status: status, location: location}
}

func (r *redirectResponse) StatusCode() int      { return r.status }
func (r *redirectResponse) Headers() http.Header { return http.Header{"Location": {r.location}} }
func (r *redirectResponse) Body() any            { return nil }

func (r *redirectResponse) WriteTo(w http.ResponseWriter) error {
    w.Header().Set("Location", r.location)
    w.WriteHeader(r.status)
    return nil
}

// streamResponse implements Response for streaming bodies.
type streamResponse struct {
    status      int
    contentType string
    reader      io.Reader
}

// Stream creates a streaming response.
func Stream(status int, contentType string, reader io.Reader) Response {
    return &streamResponse{status: status, contentType: contentType, reader: reader}
}

func (r *streamResponse) StatusCode() int      { return r.status }
func (r *streamResponse) Headers() http.Header { return nil }
func (r *streamResponse) Body() any            { return r.reader }

func (r *streamResponse) WriteTo(w http.ResponseWriter) error {
    w.Header().Set("Content-Type", r.contentType)
    w.WriteHeader(r.status)
    _, err := io.Copy(w, r.reader)
    return err
}

// ResponseBuilder provides fluent response construction.
type ResponseBuilder struct {
    status  int
    headers http.Header
    body    any
    encoder func(w io.Writer, v any) error
}

// NewResponse creates a new ResponseBuilder.
func NewResponse(status int) *ResponseBuilder {
    return &ResponseBuilder{
        status:  status,
        headers: make(http.Header),
    }
}

// Header adds a header to the response.
func (b *ResponseBuilder) Header(key, value string) *ResponseBuilder {
    b.headers.Add(key, value)
    return b
}

// JSON sets a JSON body.
func (b *ResponseBuilder) JSON(body any) Response {
    b.body = body
    b.headers.Set("Content-Type", "application/json")
    b.encoder = func(w io.Writer, v any) error {
        return json.NewEncoder(w).Encode(v)
    }
    return b
}

// XML sets an XML body.
func (b *ResponseBuilder) XML(body any) Response {
    b.body = body
    b.headers.Set("Content-Type", "application/xml")
    b.encoder = func(w io.Writer, v any) error {
        return xml.NewEncoder(w).Encode(v)
    }
    return b
}

// Text sets a plain text body.
func (b *ResponseBuilder) Text(body string) Response {
    b.body = body
    b.headers.Set("Content-Type", "text/plain")
    b.encoder = func(w io.Writer, v any) error {
        _, err := w.Write([]byte(v.(string)))
        return err
    }
    return b
}

// Binary sets a binary body.
func (b *ResponseBuilder) Binary(contentType string, data []byte) Response {
    b.body = data
    b.headers.Set("Content-Type", contentType)
    b.encoder = func(w io.Writer, v any) error {
        _, err := w.Write(v.([]byte))
        return err
    }
    return b
}

func (b *ResponseBuilder) StatusCode() int      { return b.status }
func (b *ResponseBuilder) Headers() http.Header { return b.headers }
func (b *ResponseBuilder) Body() any            { return b.body }

func (b *ResponseBuilder) WriteTo(w http.ResponseWriter) error {
    for k, v := range b.headers {
        w.Header()[k] = v
    }
    w.WriteHeader(b.status)
    if b.body == nil || b.encoder == nil {
        return nil
    }
    return b.encoder(w, b.body)
}
```

### Acceptance Criteria

1. `JSON()` creates proper JSON responses with Content-Type header
2. `NoContent()` creates 204 responses without body
3. `Error()` creates JSON error responses with consistent structure
4. `Redirect()` creates redirects with Location header
5. `Stream()` streams content without buffering
6. `ResponseBuilder` supports fluent construction with custom headers
7. All response types implement the `Response` interface correctly

---

## Phase 6: Testing Support

### Objective

Provide utilities for testing servers built with `ServerBuilder`, including stub handlers and request builders.

### New Files

**builder/server_testing.go**

```go
package builder

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
)

// TestRequest builds requests for testing.
type TestRequest struct {
    method  string
    path    string
    headers http.Header
    body    io.Reader
    query   map[string]string
}

// NewTestRequest creates a new test request builder.
func NewTestRequest(method, path string) *TestRequest {
    return &TestRequest{
        method:  method,
        path:    path,
        headers: make(http.Header),
        query:   make(map[string]string),
    }
}

// Header adds a header.
func (r *TestRequest) Header(key, value string) *TestRequest {
    r.headers.Add(key, value)
    return r
}

// Query adds a query parameter.
func (r *TestRequest) Query(key, value string) *TestRequest {
    r.query[key] = value
    return r
}

// JSONBody sets a JSON request body.
func (r *TestRequest) JSONBody(body any) *TestRequest {
    data, _ := json.Marshal(body)
    r.body = bytes.NewReader(data)
    r.headers.Set("Content-Type", "application/json")
    return r
}

// Body sets a raw request body.
func (r *TestRequest) Body(contentType string, body io.Reader) *TestRequest {
    r.body = body
    r.headers.Set("Content-Type", contentType)
    return r
}

// Build creates the http.Request.
func (r *TestRequest) Build() *http.Request {
    path := r.path
    if len(r.query) > 0 {
        path += "?"
        first := true
        for k, v := range r.query {
            if !first {
                path += "&"
            }
            path += k + "=" + v
            first = false
        }
    }

    req := httptest.NewRequest(r.method, path, r.body)
    for k, v := range r.headers {
        req.Header[k] = v
    }
    return req
}

// Execute runs the request against a handler and returns the response.
func (r *TestRequest) Execute(handler http.Handler) *httptest.ResponseRecorder {
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, r.Build())
    return rec
}

// StubHandler creates a handler that returns a fixed response.
func StubHandler(response Response) HandlerFunc {
    return func(ctx context.Context, req *Request) Response {
        return response
    }
}

// StubHandlerFunc creates a handler that calls a function.
// Useful for asserting request contents in tests.
func StubHandlerFunc(fn func(req *Request) Response) HandlerFunc {
    return func(ctx context.Context, req *Request) Response {
        return fn(req)
    }
}

// ErrorStubHandler creates a handler that returns an error response.
func ErrorStubHandler(status int, message string) HandlerFunc {
    return StubHandler(Error(status, message))
}

// ServerTest provides testing utilities for a built server.
type ServerTest struct {
    Result *ServerResult
}

// NewServerTest creates a ServerTest from a ServerResult.
func NewServerTest(result *ServerResult) *ServerTest {
    return &ServerTest{Result: result}
}

// Request creates a test request builder.
func (t *ServerTest) Request(method, path string) *TestRequest {
    return NewTestRequest(method, path)
}

// Execute runs a request and returns the recorder.
func (t *ServerTest) Execute(req *TestRequest) *httptest.ResponseRecorder {
    return req.Execute(t.Result.Handler)
}

// GetJSON performs a GET and unmarshals the JSON response.
func (t *ServerTest) GetJSON(path string, target any) (*httptest.ResponseRecorder, error) {
    rec := t.Execute(NewTestRequest(http.MethodGet, path))
    if target != nil && rec.Code >= 200 && rec.Code < 300 {
        if err := json.NewDecoder(rec.Body).Decode(target); err != nil {
            return rec, err
        }
    }
    return rec, nil
}

// PostJSON performs a POST with a JSON body and unmarshals the response.
func (t *ServerTest) PostJSON(path string, body any, target any) (*httptest.ResponseRecorder, error) {
    rec := t.Execute(NewTestRequest(http.MethodPost, path).JSONBody(body))
    if target != nil && rec.Code >= 200 && rec.Code < 300 {
        if err := json.NewDecoder(rec.Body).Decode(target); err != nil {
            return rec, err
        }
    }
    return rec, nil
}
```

### Acceptance Criteria

1. `TestRequest` builds valid HTTP requests for testing
2. `StubHandler` creates handlers that return fixed responses
3. `StubHandlerFunc` allows request inspection in tests
4. `ServerTest` provides convenient test utilities
5. JSON request/response helpers work correctly
6. Tests can be written using standard `testing` package and `httptest`

---

## 7. File Structure

The implementation adds the following files to the `builder` package:

```
builder/
├── builder.go                 # Existing - unchanged
├── builder_options.go         # Existing - unchanged
├── builder_test.go            # Existing - unchanged
├── deep_dive.md               # Update with server builder docs
├── doc.go                     # Update package documentation
├── example_test.go            # Update with server builder examples
├── operation.go               # Existing - unchanged
├── schema.go                  # Existing - unchanged
├── server.go                  # Existing (OAS server object) - unchanged
├── server_builder.go          # NEW: Core ServerBuilder type
├── server_builder_options.go  # NEW: ServerBuilderOption functions
├── server_builder_test.go     # NEW: ServerBuilder tests
├── server_dispatcher.go       # NEW: Request dispatcher
├── server_response.go         # NEW: Response helpers
├── server_router_stdlib.go    # NEW: Stdlib router implementation
├── server_router_chi.go       # NEW: Chi router implementation
├── server_testing.go          # NEW: Testing utilities
├── server_types.go            # NEW: Type definitions
├── server_validation.go       # NEW: Validation middleware
└── tags.go                    # Existing - unchanged
```

---

## 8. Testing Strategy

### Unit Tests

Each new file should have corresponding test coverage.

**server_builder_test.go** tests `NewServerBuilder`, builder method chaining, `Handle` registration, `BuildServer` output, error cases for invalid configurations.

**server_dispatcher_test.go** tests request routing, parameter extraction, body unmarshaling, method not allowed handling, not implemented handling.

**server_response_test.go** tests all response helpers, response builder fluent API, WriteTo implementations, header propagation.

**server_router_stdlib_test.go** tests path matching, parameter extraction, 404 handling, method routing.

**server_router_chi_test.go** tests chi router integration, path parameter extraction, middleware application.

**server_validation_test.go** tests validation middleware, error responses, strict mode, custom error handlers.

**server_testing_test.go** tests TestRequest builder, stub handlers, ServerTest utilities.

### Integration Tests

**server_integration_test.go** tests end-to-end server construction, full request/response cycle with validation, spec serving from ServerResult, concurrent request handling.

### Benchmark Tests

**server_benchmark_test.go** tests server construction time, request dispatch overhead, validation overhead comparison, router strategy comparison.

### Test Coverage Target

All new code should achieve at least 85% test coverage, measured by `go test -cover`.

---

## 9. Documentation

### Package Documentation

Update `builder/doc.go` to include server builder documentation in the package-level comment.

### Deep Dive Guide

Update `builder/deep_dive.md` with a new section covering server builder concepts, complete example walkthrough, configuration reference, testing patterns, and migration from generator.

### Example Tests

Add comprehensive `Example_*` functions demonstrating basic server construction, handler registration patterns, validation configuration, testing approaches, and chi router usage.

### README Updates

Update the main `README.md` with a brief mention of server builder capability and link to deep dive.

### Whitepaper Updates

Add a new section "12.6 Server Construction" to the whitepaper covering the code-first workflow.

---

## 10. Migration Path

### From Generator-Based Servers

Teams currently using the `generator` package for server code can migrate incrementally. The generated `ServerInterface` pattern maps directly to `Handle` registrations. Validation middleware is equivalent. Router behavior is consistent. Types generated by the generator work with ServerBuilder handlers.

### Migration Steps

Step 1: Keep existing generated types. Step 2: Create ServerBuilder with same operations. Step 3: Move handlers from ServerInterface implementation to Handle registrations. Step 4: Verify behavior matches. Step 5: Optionally remove generated server code.

### Coexistence

Both approaches can coexist. Use generator for complex specs received from external sources. Use ServerBuilder for internally-defined APIs. Mix approaches within the same application.

---

## 11. Future Considerations

### Potential Extensions

**Type-Safe Generic Handlers** could use Go generics for compile-time type-safe request/response handling when Go's type system allows.

**OpenTelemetry Integration** could add built-in tracing and metrics middleware options.

**WebSocket Support** could extend for WebSocket operations defined in OAS.

**gRPC Gateway** could consider gRPC-JSON transcoding for hybrid APIs.

**Hot Reload** could support rebuilding server on spec changes during development.

### Out of Scope for Initial Release

Automatic client generation from ServerBuilder. GraphQL schema generation. Service mesh integration.

---

## 12. Risk Assessment

### Technical Risks

**Performance Overhead:** Runtime construction adds startup cost compared to generated code.
*Mitigation:* Benchmark thoroughly; construction happens once at startup.

**Type Safety Limitations:** Without code generation, full compile-time type safety is not achievable.
*Mitigation:* Provide clear documentation; suggest generator for type-critical use cases.

**Chi Dependency:** Chi router adds an external dependency.
*Mitigation:* Make chi opt-in; stdlib router has zero dependencies.

### Compatibility Risks

**API Stability:** New API may need iteration based on user feedback.
*Mitigation:* Mark as experimental in v1.34.0; stabilize in v1.35.0.

**httpvalidator Integration:** Changes to httpvalidator could affect server builder.
*Mitigation:* Use stable httpvalidator APIs; add integration tests.

---

## 13. Timeline Estimate

| Phase | Effort | Dependencies |
|-------|--------|--------------|
| Phase 1: Core Server Builder | 4-6 hours | None |
| Phase 2: Validation Integration | 3-4 hours | Phase 1 |
| Phase 3: Router Strategies | 4-5 hours | Phase 1 |
| Phase 4: Handler Binding | 3-4 hours | Phases 1-3 |
| Phase 5: Response Helpers | 2-3 hours | Phase 1 |
| Phase 6: Testing Support | 2-3 hours | Phases 1-5 |
| Documentation | 3-4 hours | All phases |
| Integration Testing | 2-3 hours | All phases |
| **Total** | **23-32 hours** | |

### Suggested Session Breakdown

Session 1 (Phases 1-2): Core builder and validation (~8 hours). Session 2 (Phases 3-4): Routing and dispatch (~8 hours). Session 3 (Phases 5-6): Responses and testing (~5 hours). Session 4: Documentation and polish (~6 hours).

---

## Appendix A: Alternative Approaches Considered

### A.1 Code Generation Extension

**Approach:** Extend generator to accept Builder output directly.

**Pros:** Full type safety; consistent with existing generator workflow.

**Cons:** Requires code generation step; loses runtime flexibility; duplicates generator functionality.

**Decision:** Rejected in favor of runtime approach for simpler developer experience.

### A.2 Reflection-Based Type Safety

**Approach:** Use reflection to generate typed handlers at runtime.

**Pros:** Better type safety without code generation.

**Cons:** Complex implementation; runtime overhead; unclear error messages.

**Decision:** Deferred for future consideration; initial release uses interface-based approach.

### A.3 Separate Package

**Approach:** Create new `builder/server` subpackage.

**Pros:** Cleaner separation; optional import.

**Cons:** Breaks fluent API (can't chain `AddOperation().Handle()`); complicates discovery.

**Decision:** Rejected; server builder integrates directly into builder package.

---

## Appendix B: Chi Router Integration Details

Chi integration is provided through the `WithChiRouter()` option. Users must install chi separately:

```bash
go get github.com/go-chi/chi/v5
```

The chi router provides several advantages over the stdlib router: native path parameter extraction via `chi.URLParam()`, extensive middleware ecosystem, route grouping and mounting, and better performance for large route tables.

The oastools package itself does not depend on chi. The `server_router_chi.go` file imports chi, but this is an opt-in feature that users enable explicitly.

---

## Appendix C: Comparison with Similar Projects

| Feature | oastools ServerBuilder | oapi-codegen | go-swagger |
|---------|----------------------|--------------|------------|
| Runtime construction | ✓ | ✗ | ✗ |
| Code generation | Via generator | ✓ | ✓ |
| Validation integration | Built-in | External | Built-in |
| Router flexibility | Stdlib + Chi | Echo/Chi/etc | Built-in |
| Type safety | Runtime | Compile-time | Compile-time |
| Hot reload | ✓ | ✗ | ✗ |
| Spec access | Direct | Generated | Generated |

The ServerBuilder occupies a unique position: enabling rapid iteration during development while the generator remains available for production deployments requiring full type safety.

---

*End of Implementation Plan*
