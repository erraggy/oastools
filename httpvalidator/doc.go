// Package httpvalidator validates HTTP requests and responses against OpenAPI specifications.
//
// This package enables runtime validation of HTTP traffic in API gateways, middleware,
// and testing scenarios. It supports both OAS 2.0 (Swagger) and OAS 3.x specifications.
//
// # Features
//
//   - Request validation: path, query, header, cookie parameters and request body
//   - Response validation: status codes, headers, and response body
//   - Parameter deserialization: all OAS serialization styles (simple, form, matrix, label, etc.)
//   - Schema validation: type checking, constraints, enum, composition (allOf/anyOf/oneOf)
//   - Middleware-friendly: works with standard net/http patterns
//   - Strict mode: reject unknown parameters and undocumented responses
//
// # Basic Usage
//
// Create a validator from a parsed OpenAPI specification:
//
//	parsed, _ := parser.ParseWithOptions(parser.WithFilePath("openapi.yaml"))
//	v, err := httpvalidator.New(parsed)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate incoming request
//	result, err := v.ValidateRequest(req)
//	if !result.Valid {
//	    for _, e := range result.Errors {
//	        log.Printf("Validation error: %s: %s", e.Path, e.Message)
//	    }
//	}
//
//	// Access validated and deserialized parameters
//	userID := result.PathParams["userId"]
//	page := result.QueryParams["page"]
//
// # Middleware Pattern
//
// The validator integrates naturally with HTTP middleware:
//
//	func ValidateMiddleware(v *httpvalidator.Validator) func(http.Handler) http.Handler {
//	    return func(next http.Handler) http.Handler {
//	        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	            result, _ := v.ValidateRequest(r)
//	            if !result.Valid {
//	                http.Error(w, "Invalid request", http.StatusBadRequest)
//	                return
//	            }
//	            next.ServeHTTP(w, r)
//	        })
//	    }
//	}
//
// For response validation in middleware, use ValidateResponseData which accepts
// captured response parts instead of *http.Response:
//
//	result, _ := v.ValidateResponseData(req, recorder.Code, recorder.Header(), recorder.Body.Bytes())
//
// # Functional Options
//
// For one-off validations, use the functional options API:
//
//	result, err := httpvalidator.ValidateRequestWithOptions(
//	    req,
//	    httpvalidator.WithFilePath("openapi.yaml"),
//	    httpvalidator.WithStrictMode(true),
//	)
//
// # Strict Mode
//
// Enable strict mode for stricter validation:
//
//	v.StrictMode = true
//
// In strict mode:
//   - Unknown query parameters cause validation errors
//   - Unknown headers (except standard HTTP headers) cause errors
//   - Unknown cookies cause errors
//   - Undocumented response status codes cause errors
//
// # Parameter Deserialization
//
// The validator automatically deserializes parameters according to their OAS style:
//
//   - path: simple (default), label, matrix
//   - query: form (default), spaceDelimited, pipeDelimited, deepObject
//   - header: simple (default)
//   - cookie: form (default)
//
// Deserialized values are available in the result:
//
//	result.PathParams["id"]      // Deserialized path parameter
//	result.QueryParams["page"]   // Deserialized query parameter
//	result.HeaderParams["X-API"] // Deserialized header parameter
//	result.CookieParams["token"] // Deserialized cookie parameter
//
// # Schema Validation
//
// The validator performs JSON Schema validation on request/response bodies including:
//
//   - Type checking (string, number, integer, boolean, array, object, null)
//   - String constraints (minLength, maxLength, pattern, format, enum)
//   - Number constraints (minimum, maximum, exclusiveMin/Max, multipleOf)
//   - Array constraints (minItems, maxItems, uniqueItems)
//   - Object constraints (required, properties, additionalProperties)
//   - Composition (allOf, anyOf, oneOf)
//   - Nullable fields (OAS 3.0 nullable, OAS 3.1 type arrays)
package httpvalidator
