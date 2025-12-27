# HTTP Validation

Demonstrates runtime HTTP request/response validation using the httpvalidator package.

## What You'll Learn

- How to create an HTTP validator from an OpenAPI spec
- Validating request parameters (path, query, header)
- Validating request bodies against schema
- Extracting typed path parameters
- Validating responses for contract compliance
- Securely logging validation errors without exposing credentials

## Prerequisites

- Go 1.24+

## Quick Start

```bash
cd examples/workflows/http-validation
go run main.go
```

## Expected Output

```
HTTP Validation Workflow
========================

[1/6] Creating HTTP validator...
      Validator created, strict mode: false

[2/6] Validating GET /todos?status=pending&limit=10...
      Valid: true
      Matched Path: /todos

[3/6] Validating GET /todos?status=invalid...
      Valid: false
      Matched Path: /todos
      Errors: 1 validation issue(s) found

[4/6] Validating POST /todos with valid body...
      Valid: true
      Matched Path: /todos

[5/6] Validating POST /todos with invalid body...
      Valid: true
      Matched Path: /todos

[6/6] Path parameter extraction...
      Matched Path: /todos/{todoId}
      todoId: 42
      Valid: true

[Bonus] Response validation...
      Response Valid: true
      Status Code: 200

---
HTTP Validation examples complete
```

## Files

| File | Purpose |
|------|---------|
| main.go | Demonstrates the HTTP validation workflow |
| specs/api.yaml | OpenAPI spec with validation constraints |

## Key Concepts

### Creating a Validator

```go
parsed, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
v, _ := httpvalidator.New(parsed)
v.StrictMode = false  // Allow unknown headers
```

### Request Validation

```go
req := httptest.NewRequest("POST", "/todos", body)
req.Header.Set("Content-Type", "application/json")

result, _ := v.ValidateRequest(req)
if !result.Valid {
    for _, err := range result.Errors {
        log.Printf("[%s] %s", err.Path, err.Message)
    }
}
```

### Path Parameter Extraction

```go
result, _ := v.ValidateRequest(req)
todoId := result.PathParams["todoId"]  // Extracted from /todos/{todoId}
```

### Response Validation

```go
result, _ := v.ValidateResponseData(req, statusCode, headers, body)
if !result.Valid {
    // Response doesn't match spec
}
```

### Validation Types

| Validation | Description |
|------------|-------------|
| Path parameters | Type, format, constraints (min/max) |
| Query parameters | Type, enum values, constraints |
| Request body | Required fields, schema validation |
| Response body | Schema compliance check |

### Secure Error Logging

The httpvalidator package automatically redacts values in error messages for potentially sensitive parameters (headers and cookies). This means validation error messages are safe to log:

```go
result, _ := v.ValidateRequest(req)
for _, err := range result.Errors {
    // Safe to log - sensitive header/cookie values are redacted at source
    log.Printf("[%s] %s", err.Path, err.Message)
}
```

**How it works:**
- Query params, path params, body: Full values included (helpful for debugging)
- Headers, cookies: Values redacted (e.g., "value is not one of the allowed values" instead of "value 'Bearer sk-xxx' is not...")

Note: This example prints only paths to satisfy static analysis tools, but in production you can safely log the full `err.Message`.

### Middleware Integration

```go
func validationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        result, _ := v.ValidateRequest(r)
        if !result.Valid {
            http.Error(w, "Invalid request", http.StatusBadRequest)
            return
        }
        // Store extracted params in context
        ctx := context.WithValue(r.Context(), "pathParams", result.PathParams)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Next Steps

- [HTTPValidator Deep Dive](https://erraggy.github.io/oastools/packages/httpvalidator/) - Complete documentation
- [Validate and Fix](../validate-and-fix/) - Fix spec validation errors
- [Builder](../../programmatic-api/builder/) - Build specs programmatically

---

*Generated for [oastools](https://github.com/erraggy/oastools)*
