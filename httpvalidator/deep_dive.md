<a id="top"></a>

# HTTP Validator Package Deep Dive

## Table of Contents

- [Overview](#overview)
- [Key Concepts](#key-concepts)
- [API Styles](#api-styles)
- [Practical Examples](#practical-examples)
- [Parameter Deserialization](#parameter-deserialization)
- [Schema Validation](#schema-validation)
- [Middleware Integration](#middleware-integration)
- [Validation Result Structure](#validation-result-structure)
- [Configuration Reference](#configuration-reference)
- [Best Practices](#best-practices)

---

The `httpvalidator` package validates HTTP requests and responses against OpenAPI Specification documents at runtime. It enables API gateways, middleware, and testing frameworks to enforce API contracts, ensuring that HTTP traffic conforms to the declared specification.

## Overview

Runtime HTTP validation catches contract violations before they reach application logic. The httpvalidator package supports both request validation (incoming traffic) and response validation (outgoing traffic), with comprehensive parameter deserialization and JSON Schema validation.

The validator supports OAS 2.0 (Swagger) through OAS 3.2, automatically adapting its behavior to match the specification version. It handles all OAS parameter serialization styles (simple, form, matrix, label, deepObject, spaceDelimited, pipeDelimited) and performs schema validation for request and response bodies.

Key features include:
- **Request validation**: Path parameters, query parameters, headers, cookies, request body
- **Response validation**: Status codes, headers, response body
- **Parameter deserialization**: Automatic type conversion according to OAS style and schema
- **Schema validation**: Type checking, constraints (min/max, pattern, enum), composition (allOf/anyOf/oneOf)
- **Middleware-friendly API**: Designed for use with standard `net/http` patterns
- **Strict mode**: Optionally reject unknown parameters and undocumented responses

[↑ Back to top](#top)

## Key Concepts

### Request vs Response Validation

**Request validation** examines incoming HTTP requests against the specification's path and operation definitions. It verifies that:
- The request path matches a defined path template
- The HTTP method is supported for that path
- All required path, query, header, and cookie parameters are present
- Parameter values match their declared types and constraints
- The request body (if present) matches the declared schema

**Response validation** checks outgoing HTTP responses against the operation's response definitions. It ensures that:
- The response status code is documented in the operation
- Response headers match declared headers
- The response body conforms to the declared schema for that status code

### Validation vs Deserialization

The validator performs two distinct operations:

**Deserialization** converts raw parameter strings into typed values according to the OAS parameter style and schema. For example, a query parameter `?tags=foo,bar,baz` with style `form` and `explode: false` deserializes to `[]string{"foo", "bar", "baz"}`.

**Validation** then checks the deserialized values against the parameter's schema constraints (type, format, minimum, maximum, pattern, enum, etc.).

Deserialized values are available in the validation result for use by your application, eliminating the need for manual parameter parsing.

### Strict Mode

By default, the validator is permissive: it validates declared parameters but allows undeclared parameters to pass through. This accommodates real-world scenarios where clients send extra headers or where responses include additional status codes not documented in the specification.

**Strict mode** changes this behavior:
- **Unknown query parameters**: Rejected (error)
- **Unknown headers**: Rejected (error), except for standard HTTP headers like `Content-Type`, `Content-Length`, `User-Agent`
- **Unknown cookies**: Rejected (error)
- **Undocumented response status codes**: Rejected (error)

Use strict mode when you need exact contract enforcement, such as in testing scenarios or strict API gateway policies.

### Path Matching Specificity

When multiple path templates could match a request (e.g., `/users/123` could match both `/users/{id}` and `/users/new`), the validator uses specificity-based ordering following OpenAPI best practices:

1. Static paths (no parameters) have highest priority
2. Paths with fewer parameters have higher priority
3. Paths with more static segments have higher priority

This ensures that `/users/new` (static) matches before `/users/{id}` (parameterized).

[↑ Back to top](#top)

## API Styles

### Functional Options API

Best for one-off validations with inline configuration:

```go
result, err := httpvalidator.ValidateRequestWithOptions(
    req,
    httpvalidator.WithFilePath("openapi.yaml"),
    httpvalidator.WithStrictMode(true),
)
```

### Struct-Based API

Best for reusable validators in middleware or long-running services:

```go
// Parse specification once at startup
parsed, _ := parser.ParseWithOptions(
    parser.WithFilePath("openapi.yaml"),
)

// Create validator instance
v, _ := httpvalidator.New(parsed)
v.StrictMode = true

// Reuse for all requests
result1, _ := v.ValidateRequest(req1)
result2, _ := v.ValidateRequest(req2)
```

[↑ Back to top](#top)

## Practical Examples

### Basic Request Validation

Validate an incoming HTTP request against the specification:

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/erraggy/oastools/httpvalidator"
    "github.com/erraggy/oastools/parser"
)

func main() {
    // Parse specification
    parsed, err := parser.ParseWithOptions(
        parser.WithFilePath("openapi.yaml"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create validator
    v, err := httpvalidator.New(parsed)
    if err != nil {
        log.Fatal(err)
    }

    // Create sample request
    req, _ := http.NewRequest("GET", "/users/123?page=1&limit=10", nil)

    // Validate request
    result, err := v.ValidateRequest(req)
    if err != nil {
        log.Fatalf("Validation failed: %v", err)
    }

    if result.Valid {
        fmt.Println("✓ Request is valid")
        fmt.Printf("  Path params: %v\n", result.PathParams)
        fmt.Printf("  Query params: %v\n", result.QueryParams)
    } else {
        fmt.Println("✗ Request validation failed:")
        for _, e := range result.Errors {
            fmt.Printf("  [%s] %s: %s\n", e.Severity, e.Path, e.Message)
        }
    }
}
```

### Response Validation in Middleware

Validate responses using captured response data:

```go
package main

import (
    "bytes"
    "net/http"

    "github.com/erraggy/oastools/httpvalidator"
)

// ResponseRecorder captures response data for validation
type ResponseRecorder struct {
    http.ResponseWriter
    StatusCode int
    Body       *bytes.Buffer
}

func (r *ResponseRecorder) WriteHeader(code int) {
    r.StatusCode = code
    r.ResponseWriter.WriteHeader(code)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
    r.Body.Write(b)
    return r.ResponseWriter.Write(b)
}

// ValidationMiddleware validates both requests and responses
func ValidationMiddleware(v *httpvalidator.Validator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Validate request
            reqResult, _ := v.ValidateRequest(r)
            if !reqResult.Valid {
                http.Error(w, "Invalid request", http.StatusBadRequest)
                return
            }

            // Record response for validation
            recorder := &ResponseRecorder{
                ResponseWriter: w,
                StatusCode:     http.StatusOK,
                Body:           new(bytes.Buffer),
            }

            // Call next handler
            next.ServeHTTP(recorder, r)

            // Validate response
            respResult, _ := v.ValidateResponseData(
                r,
                recorder.StatusCode,
                recorder.Header(),
                recorder.Body.Bytes(),
            )

            if !respResult.Valid {
                // Log validation failures
                for _, e := range respResult.Errors {
                    log.Printf("Response validation error: %s", e.Message)
                }
            }
        })
    }
}
```

### Request Body Validation

Validate JSON request bodies against schemas:

```go
package main

import (
    "bytes"
    "fmt"
    "log"
    "net/http"

    "github.com/erraggy/oastools/httpvalidator"
    "github.com/erraggy/oastools/parser"
)

func main() {
    parsed, _ := parser.ParseWithOptions(
        parser.WithFilePath("openapi.yaml"),
    )
    v, _ := httpvalidator.New(parsed)

    // Create request with JSON body
    body := bytes.NewBufferString(`{
        "name": "John Doe",
        "email": "john@example.com",
        "age": 30
    }`)

    req, _ := http.NewRequest("POST", "/users", body)
    req.Header.Set("Content-Type", "application/json")

    // Validate request (including body schema)
    result, err := v.ValidateRequest(req)
    if err != nil {
        log.Fatal(err)
    }

    if result.Valid {
        fmt.Println("✓ Request body is valid")
    } else {
        fmt.Println("✗ Request body validation failed:")
        for _, e := range result.Errors {
            fmt.Printf("  %s: %s\n", e.Path, e.Message)
        }
    }
}
```

**Example OpenAPI Specification:**
```yaml
openapi: 3.0.3
info:
  title: User API
  version: 1.0.0
paths:
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name, email]
              properties:
                name:
                  type: string
                  minLength: 1
                email:
                  type: string
                  format: email
                age:
                  type: integer
                  minimum: 0
                  maximum: 150
```

### Parameter Validation with Type Conversion

Access deserialized and validated parameters:

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/erraggy/oastools/httpvalidator"
    "github.com/erraggy/oastools/parser"
)

func main() {
    parsed, _ := parser.ParseWithOptions(
        parser.WithFilePath("openapi.yaml"),
    )
    v, _ := httpvalidator.New(parsed)

    // Request with various parameter types
    req, _ := http.NewRequest(
        "GET",
        "/users/123/posts?published=true&limit=10&tags=golang,api",
        nil,
    )
    req.Header.Set("X-API-Version", "v1")

    result, err := v.ValidateRequest(req)
    if err != nil {
        log.Fatal(err)
    }

    if result.Valid {
        // Access typed, deserialized parameters
        userID := result.PathParams["userId"]       // "123" (string)
        published := result.QueryParams["published"] // true (bool)
        limit := result.QueryParams["limit"]        // 10 (int)
        tags := result.QueryParams["tags"]          // []string{"golang", "api"}
        apiVersion := result.HeaderParams["X-API-Version"] // "v1" (string)

        fmt.Printf("User ID: %v (%T)\n", userID, userID)
        fmt.Printf("Published: %v (%T)\n", published, published)
        fmt.Printf("Limit: %v (%T)\n", limit, limit)
        fmt.Printf("Tags: %v (%T)\n", tags, tags)
        fmt.Printf("API Version: %v (%T)\n", apiVersion, apiVersion)
    }
}
```

### Strict Mode for Contract Enforcement

Reject requests with undeclared parameters:

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/erraggy/oastools/httpvalidator"
    "github.com/erraggy/oastools/parser"
)

func main() {
    parsed, _ := parser.ParseWithOptions(
        parser.WithFilePath("openapi.yaml"),
    )

    v, _ := httpvalidator.New(parsed)
    v.StrictMode = true  // Enable strict validation

    // Request with an undeclared query parameter
    req, _ := http.NewRequest(
        "GET",
        "/users?page=1&undeclared=value",
        nil,
    )

    result, _ := v.ValidateRequest(req)

    if !result.Valid {
        fmt.Println("✗ Validation failed (strict mode):")
        for _, e := range result.Errors {
            fmt.Printf("  %s: %s\n", e.Path, e.Message)
        }
        // Output:
        // ✗ Validation failed (strict mode):
        //   query.undeclared: unknown query parameter 'undeclared'
    }
}
```

### Functional Options for One-Off Validation

Use functional options when you don't need a reusable validator:

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/erraggy/oastools/httpvalidator"
)

func main() {
    req, _ := http.NewRequest("GET", "/users/123", nil)

    // Validate without creating a validator instance
    result, err := httpvalidator.ValidateRequestWithOptions(
        req,
        httpvalidator.WithFilePath("openapi.yaml"),
        httpvalidator.WithStrictMode(true),
        httpvalidator.WithIncludeWarnings(true),
    )

    if err != nil {
        panic(err)
    }

    fmt.Printf("Valid: %v\n", result.Valid)
}
```

### Integration with Testing

Use httpvalidator in API tests to verify request/response conformance:

```go
package main

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/erraggy/oastools/httpvalidator"
    "github.com/erraggy/oastools/parser"
)

func TestAPIConformance(t *testing.T) {
    // Parse specification once for all tests
    parsed, _ := parser.ParseWithOptions(
        parser.WithFilePath("openapi.yaml"),
    )
    v, _ := httpvalidator.New(parsed)
    v.StrictMode = true

    // Test request validation
    req := httptest.NewRequest("GET", "/users/123", nil)
    result, err := v.ValidateRequest(req)
    if err != nil {
        t.Fatalf("Validation error: %v", err)
    }
    if !result.Valid {
        t.Errorf("Request validation failed: %v", result.Errors)
    }

    // Test response validation
    recorder := httptest.NewRecorder()
    recorder.WriteHeader(http.StatusOK)
    recorder.Write([]byte(`{"id": "123", "name": "John"}`))
    recorder.Header().Set("Content-Type", "application/json")

    respResult, _ := v.ValidateResponseData(
        req,
        recorder.Code,
        recorder.Header(),
        recorder.Body.Bytes(),
    )

    if !respResult.Valid {
        t.Errorf("Response validation failed: %v", respResult.Errors)
    }
}
```

[↑ Back to top](#top)

## Parameter Deserialization

The validator automatically deserializes parameters according to their OpenAPI style and schema definition. Understanding the deserialization rules helps you configure your specification correctly and interpret validation results.

### Style Reference

| Location | Default Style | Supported Styles |
|----------|---------------|------------------|
| path | simple | simple, label, matrix |
| query | form | form, spaceDelimited, pipeDelimited, deepObject |
| header | simple | simple |
| cookie | form | form |

### Simple Style

**Simple style** (default for path and header parameters) serializes values without prefixes or delimiters.

**Primitive values:**
```
param=value          → "value"
```

**Array with explode=false:**
```
param=red,green,blue → []string{"red", "green", "blue"}
```

**Array with explode=true:**
```
param=red&param=green&param=blue → []string{"red", "green", "blue"}
```

**Object with explode=false:**
```
param=role,admin,enabled,true → map[string]interface{}{"role": "admin", "enabled": "true"}
```

**Object with explode=true:**
```
role=admin&enabled=true → map[string]interface{}{"role": "admin", "enabled": "true"}
```

### Form Style

**Form style** (default for query and cookie parameters) uses ampersand-separated key-value pairs.

**Primitive values:**
```
param=value → "value"
```

**Array with explode=false:**
```
param=red,green,blue → []string{"red", "green", "blue"}
```

**Array with explode=true:**
```
param=red&param=green&param=blue → []string{"red", "green", "blue"}
```

**Object with explode=true:**
```
role=admin&enabled=true → map[string]interface{}{"role": "admin", "enabled": "true"}
```

### Matrix Style

**Matrix style** (path parameters only) uses semicolon-prefixed parameters.

**Primitive values:**
```
;param=value → "value"
```

**Array with explode=false:**
```
;param=red,green,blue → []string{"red", "green", "blue"}
```

**Array with explode=true:**
```
;param=red;param=green;param=blue → []string{"red", "green", "blue"}
```

### Label Style

**Label style** (path parameters only) uses dot-prefixed parameters.

**Primitive values:**
```
.value → "value"
```

**Array with explode=false:**
```
.red.green.blue → []string{"red", "green", "blue"}
```

**Array with explode=true:**
```
.red.green.blue → []string{"red", "green", "blue"}
```

### DeepObject Style

**DeepObject style** (query parameters only) uses bracket notation for nested objects.

**Nested object:**
```
filter[name]=John&filter[age]=30 → map[string]interface{}{"name": "John", "age": "30"}
```

This style is particularly useful for complex query parameters representing structured data.

### SpaceDelimited and PipeDelimited Styles

**SpaceDelimited** and **PipeDelimited** (query parameters only) use space or pipe as array delimiters.

**SpaceDelimited:**
```
tags=red%20green%20blue → []string{"red", "green", "blue"}
```

**PipeDelimited:**
```
tags=red|green|blue → []string{"red", "green", "blue"}
```

### Type Coercion

After deserialization, the validator performs type coercion based on the parameter's schema:

- **string**: No conversion (raw string value)
- **integer**: Parsed with `strconv.Atoi`
- **number**: Parsed with `strconv.ParseFloat`
- **boolean**: Parsed with `strconv.ParseBool` (accepts "true", "false", "1", "0")
- **array**: Elements are recursively type-coerced based on the array item schema
- **object**: Properties are recursively type-coerced based on their schemas

If type coercion fails (e.g., "abc" cannot be parsed as an integer), a validation error is added.

[↑ Back to top](#top)

## Schema Validation

The validator includes a minimal JSON Schema implementation that validates request and response bodies against their declared schemas. This implementation focuses on the JSON Schema features commonly used in OpenAPI specifications.

### Supported Schema Features

**Type Validation:**
- Primitive types: `string`, `number`, `integer`, `boolean`, `null`
- Structured types: `array`, `object`
- OAS 3.1 type arrays: `type: ["string", "null"]`
- OAS 3.0 nullable: `nullable: true`

**String Constraints:**
- `minLength`, `maxLength`
- `pattern` (regex validation)
- `format` (email, uri, uuid, date-time, date, etc.)
- `enum` (allowed values)

**Number Constraints:**
- `minimum`, `maximum`
- `exclusiveMinimum`, `exclusiveMaximum`
- `multipleOf`

**Array Constraints:**
- `minItems`, `maxItems`
- `uniqueItems`
- `items` (schema for array elements)

**Object Constraints:**
- `required` (required property names)
- `properties` (property schemas)
- `additionalProperties` (schema for undeclared properties, or boolean to allow/disallow)
- `minProperties`, `maxProperties`

**Composition:**
- `allOf` (must match all schemas)
- `anyOf` (must match at least one schema)
- `oneOf` (must match exactly one schema)

**References:**
- `$ref` (references to component schemas)

### Validation Behavior

**Type checking** validates that the data type matches the declared type. For OAS 3.1, type can be an array (e.g., `["string", "null"]`), allowing multiple types.

**Constraint checking** validates that values meet their declared constraints. For example, a string with `minLength: 5` must have at least 5 characters.

**Format validation** performs limited format checking for common formats:
- `email`: Basic email pattern
- `uri`: URL format check
- `uuid`: UUID v4 pattern
- `date-time`, `date`, `time`: ISO 8601 format

**Composition** applies all composition rules:
- `allOf`: Value must validate against every schema in the array
- `anyOf`: Value must validate against at least one schema in the array
- `oneOf`: Value must validate against exactly one schema (not zero, not multiple)

**Recursion** handles nested schemas for objects and arrays. The validator recursively descends into properties and array items to validate nested structures.

### Schema Validation Example

```yaml
openapi: 3.0.3
paths:
  /products:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [name, price]
              properties:
                name:
                  type: string
                  minLength: 3
                  maxLength: 100
                price:
                  type: number
                  minimum: 0
                  exclusiveMinimum: true
                tags:
                  type: array
                  items:
                    type: string
                  minItems: 1
                  uniqueItems: true
```

**Valid request body:**
```json
{
  "name": "Widget",
  "price": 19.99,
  "tags": ["electronics", "gadgets"]
}
```

**Invalid request body (multiple violations):**
```json
{
  "name": "AB",           // Too short (minLength: 3)
  "price": 0,             // Not greater than 0 (exclusiveMinimum: true)
  "tags": []              // Too few items (minItems: 1)
}
```

[↑ Back to top](#top)

## Middleware Integration

The httpvalidator package is designed for seamless integration with standard Go middleware patterns. This section demonstrates common middleware scenarios.

### Request Validation Middleware

Validate all incoming requests before they reach handlers:

```go
func RequestValidationMiddleware(v *httpvalidator.Validator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            result, err := v.ValidateRequest(r)
            if err != nil {
                http.Error(w, "Internal validation error", http.StatusInternalServerError)
                return
            }

            if !result.Valid {
                // Return 400 Bad Request with validation errors
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]interface{}{
                    "error": "Validation failed",
                    "details": result.Errors,
                })
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### Response Validation Middleware

Validate responses before sending them to clients:

```go
type ResponseRecorder struct {
    http.ResponseWriter
    StatusCode int
    Body       *bytes.Buffer
    Headers    http.Header
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
    return &ResponseRecorder{
        ResponseWriter: w,
        StatusCode:     http.StatusOK,
        Body:           new(bytes.Buffer),
        Headers:        make(http.Header),
    }
}

func (r *ResponseRecorder) WriteHeader(code int) {
    r.StatusCode = code
    for k, v := range r.ResponseWriter.Header() {
        r.Headers[k] = v
    }
    r.ResponseWriter.WriteHeader(code)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
    r.Body.Write(b)
    return r.ResponseWriter.Write(b)
}

func ResponseValidationMiddleware(v *httpvalidator.Validator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            recorder := NewResponseRecorder(w)

            next.ServeHTTP(recorder, r)

            // Validate response after handler completes
            result, err := v.ValidateResponseData(
                r,
                recorder.StatusCode,
                recorder.Headers,
                recorder.Body.Bytes(),
            )

            if err != nil || !result.Valid {
                log.Printf("Response validation failed for %s %s: %v",
                    r.Method, r.URL.Path, result.Errors)
            }
        })
    }
}
```

### Combined Request and Response Validation

Validate both directions in a single middleware:

```go
func ValidationMiddleware(v *httpvalidator.Validator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Validate request
            reqResult, err := v.ValidateRequest(r)
            if err != nil || !reqResult.Valid {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]interface{}{
                    "error": "Invalid request",
                    "details": reqResult.Errors,
                })
                return
            }

            // Record response
            recorder := NewResponseRecorder(w)
            next.ServeHTTP(recorder, r)

            // Validate response
            respResult, _ := v.ValidateResponseData(
                r,
                recorder.StatusCode,
                recorder.Headers,
                recorder.Body.Bytes(),
            )

            if !respResult.Valid {
                log.Printf("Response validation failed: %v", respResult.Errors)
            }
        })
    }
}
```

### Passing Validated Parameters to Handlers

Store validated and deserialized parameters in the request context:

```go
type contextKey string

const validatedParamsKey contextKey = "validatedParams"

type ValidatedParams struct {
    PathParams   map[string]interface{}
    QueryParams  map[string]interface{}
    HeaderParams map[string]interface{}
    CookieParams map[string]interface{}
}

func ValidationMiddleware(v *httpvalidator.Validator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            result, err := v.ValidateRequest(r)
            if err != nil || !result.Valid {
                http.Error(w, "Invalid request", http.StatusBadRequest)
                return
            }

            // Store validated params in context
            params := &ValidatedParams{
                PathParams:   result.PathParams,
                QueryParams:  result.QueryParams,
                HeaderParams: result.HeaderParams,
                CookieParams: result.CookieParams,
            }

            ctx := context.WithValue(r.Context(), validatedParamsKey, params)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// In your handler
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    params := r.Context().Value(validatedParamsKey).(*ValidatedParams)
    userID := params.PathParams["userId"].(string)

    // Use the validated, typed parameter
    user, _ := getUserByID(userID)
    json.NewEncoder(w).Encode(user)
}
```

[↑ Back to top](#top)

## Validation Result Structure

```go
// ValidationResult contains the outcome of request or response validation.
type ValidationResult struct {
    // Valid indicates whether the request/response is valid.
    // For requests: true if all required parameters are present and valid.
    // For responses: true if status code is documented and body matches schema.
    Valid bool

    // Errors contains validation errors (severity: error).
    Errors []ValidationError

    // Warnings contains best practice warnings (severity: warning).
    Warnings []ValidationError

    // ErrorCount is the number of errors.
    ErrorCount int

    // WarningCount is the number of warnings.
    WarningCount int

    // PathParams contains deserialized path parameters (request only).
    PathParams map[string]interface{}

    // QueryParams contains deserialized query parameters (request only).
    QueryParams map[string]interface{}

    // HeaderParams contains deserialized header parameters.
    HeaderParams map[string]interface{}

    // CookieParams contains deserialized cookie parameters (request only).
    CookieParams map[string]interface{}

    // MatchedPath is the OpenAPI path template that matched (e.g., "/users/{id}").
    MatchedPath string

    // MatchedOperation is the HTTP method for the matched operation.
    MatchedOperation string
}

// ValidationError represents a single validation error or warning.
type ValidationError struct {
    // Path is the location of the error (e.g., "query.page", "body.name").
    Path string

    // Message describes the validation error.
    Message string

    // Severity indicates the error level (error, warning).
    Severity string
}
```

[↑ Back to top](#top)

## Configuration Reference

### Validator Fields

```go
type Validator struct {
    // IncludeWarnings determines whether to include best practice warnings
    // in validation results. Default is true.
    IncludeWarnings bool

    // StrictMode enables stricter validation behavior:
    // - Rejects requests with unknown query parameters
    // - Rejects requests with unknown headers
    // - Rejects responses with undocumented status codes
    // Default is false.
    StrictMode bool
}
```

### Available Options

| Option | Description |
|--------|-------------|
| `WithFilePath(string)` | Load specification from file path |
| `WithParsed(*ParseResult)` | Use pre-parsed specification |
| `WithStrictMode(bool)` | Enable/disable strict validation |
| `WithIncludeWarnings(bool)` | Include/exclude best practice warnings |

### Usage Examples

```go
// Functional options API
result, err := httpvalidator.ValidateRequestWithOptions(
    req,
    httpvalidator.WithFilePath("openapi.yaml"),
    httpvalidator.WithStrictMode(true),
)

// Struct-based API
parsed, _ := parser.ParseWithOptions(parser.WithFilePath("openapi.yaml"))
v, _ := httpvalidator.New(parsed)
v.StrictMode = true
v.IncludeWarnings = true
result, _ := v.ValidateRequest(req)
```

[↑ Back to top](#top)

## Best Practices

**Parse specifications once at startup.** In production services, parse the OpenAPI specification during initialization and create a single Validator instance for reuse. Parsing on every request is inefficient.

**Use strict mode in testing, permissive mode in production.** Strict mode catches contract violations during development and testing. In production, permissive mode accommodates real-world variance (extra headers, additional response codes) without breaking clients.

**Log response validation failures.** Unlike request validation failures (which should return 400 Bad Request), response validation failures indicate bugs in your service. Log these failures for monitoring and debugging.

**Store validated parameters in request context.** The validator deserializes and type-converts parameters for you. Pass these validated values through the request context to avoid duplicate parsing in handlers.

**Validate responses in integration tests.** Use httpvalidator in your test suite to ensure your API implementation matches its specification. This catches discrepancies early.

**Handle validation errors gracefully.** Return structured error responses that help API consumers understand what went wrong. Include the validation error paths and messages in your response body.

**Be aware of schema validation limitations.** The built-in schema validator handles common JSON Schema features but is not a complete implementation. For advanced schema validation (JSON Schema 2020-12 with all keywords), consider integrating a full-featured JSON Schema library.

**Consider performance for high-throughput services.** Path matching and schema validation add latency to request processing. Profile your service under load and consider:
- Validating requests but not responses in production (response validation in tests only)
- Selective validation (validate only critical paths or methods)
- Asynchronous response validation (log failures without blocking the response)

**Update validators when specifications change.** If your specification is updated at runtime (dynamic API configurations), recreate the Validator with the new specification. Validators are immutable once created.

[↑ Back to top](#top)
