package httpvalidator_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/erraggy/oastools/httpvalidator"
	"github.com/erraggy/oastools/parser"
)

func ExampleNew() {
	// Create a minimal spec inline for the example
	specYAML := `
openapi: "3.0.0"
info:
  title: Pet Store
  version: "1.0"
paths:
  /pets:
    get:
      responses:
        "200":
          description: Success
`
	// Parse an OpenAPI specification
	parsed, err := parser.ParseWithOptions(parser.WithBytes([]byte(specYAML)))
	if err != nil {
		fmt.Println("Parse error:", err)
		return
	}

	// Create a validator
	v, err := httpvalidator.New(parsed)
	if err != nil {
		fmt.Println("Validator error:", err)
		return
	}

	// The validator is ready to validate requests and responses
	fmt.Println("Validator created, strict mode:", v.StrictMode)
	// Output: Validator created, strict mode: false
}

func ExampleValidator_ValidateRequest() {
	// Create a minimal OAS 3.0 spec for the example
	specYAML := `
openapi: "3.0.0"
info:
  title: Pet Store
  version: "1.0"
paths:
  /pets/{petId}:
    get:
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
        - name: include
          in: query
          schema:
            type: string
            enum: [owner, vaccinations, all]
      responses:
        "200":
          description: Success
`
	parsed, _ := parser.ParseWithOptions(parser.WithBytes([]byte(specYAML)))
	v, _ := httpvalidator.New(parsed)

	// Create a test request
	req := httptest.NewRequest("GET", "/pets/123?include=owner", nil)

	// Validate the request
	result, err := v.ValidateRequest(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Valid:", result.Valid)
	fmt.Println("Matched path:", result.MatchedPath)
	fmt.Println("Pet ID:", result.PathParams["petId"])
	// Output:
	// Valid: true
	// Matched path: /pets/{petId}
	// Pet ID: 123
}

func ExampleValidator_ValidateRequest_invalid() {
	specYAML := `
openapi: "3.0.0"
info:
  title: Pet Store
  version: "1.0"
paths:
  /pets/{petId}:
    get:
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
            minimum: 1
      responses:
        "200":
          description: Success
`
	parsed, _ := parser.ParseWithOptions(parser.WithBytes([]byte(specYAML)))
	v, _ := httpvalidator.New(parsed)

	// Request with invalid petId (not an integer)
	req := httptest.NewRequest("GET", "/pets/abc", nil)

	result, _ := v.ValidateRequest(req)

	fmt.Println("Valid:", result.Valid)
	if len(result.Errors) > 0 {
		fmt.Println("First error:", result.Errors[0].Message)
	}
	// Output:
	// Valid: false
	// First error: expected type integer but got string
}

func ExampleValidator_ValidateResponseData() {
	specYAML := `
openapi: "3.0.0"
info:
  title: Pet Store
  version: "1.0"
paths:
  /pets/{petId}:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                type: object
                required: [id, name]
                properties:
                  id:
                    type: integer
                  name:
                    type: string
`
	parsed, _ := parser.ParseWithOptions(parser.WithBytes([]byte(specYAML)))
	v, _ := httpvalidator.New(parsed)

	// Original request
	req := httptest.NewRequest("GET", "/pets/123", nil)

	// Captured response data (simulating middleware capture)
	statusCode := 200
	headers := http.Header{"Content-Type": []string{"application/json"}}
	body := []byte(`{"id": 123, "name": "Fluffy"}`)

	// Validate the response
	result, _ := v.ValidateResponseData(req, statusCode, headers, body)

	fmt.Println("Valid:", result.Valid)
	fmt.Println("Status code:", result.StatusCode)
	// Output:
	// Valid: true
	// Status code: 200
}

func ExampleValidateRequestWithOptions() {
	specYAML := `
openapi: "3.0.0"
info:
  title: API
  version: "1.0"
paths:
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [email]
              properties:
                email:
                  type: string
                  format: email
      responses:
        "201":
          description: Created
`
	parsed, _ := parser.ParseWithOptions(parser.WithBytes([]byte(specYAML)))

	// Create request with JSON body
	body := strings.NewReader(`{"email": "user@example.com"}`)
	req := httptest.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", "application/json")

	// Validate using functional options
	result, _ := httpvalidator.ValidateRequestWithOptions(
		req,
		httpvalidator.WithParsed(parsed),
		httpvalidator.WithStrictMode(true),
	)

	fmt.Println("Valid:", result.Valid)
	fmt.Println("Matched path:", result.MatchedPath)
	// Output:
	// Valid: true
	// Matched path: /users
}

func ExamplePathMatcher() {
	// Create a path matcher from a template
	matcher, err := httpvalidator.NewPathMatcher("/users/{userId}/posts/{postId}")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Test matching
	matched, params := matcher.Match("/users/42/posts/101")

	fmt.Println("Matched:", matched)
	fmt.Println("userId:", params["userId"])
	fmt.Println("postId:", params["postId"])
	// Output:
	// Matched: true
	// userId: 42
	// postId: 101
}

func ExampleParamDeserializer() {
	d := httpvalidator.NewParamDeserializer()

	// Deserialize a matrix-style path parameter: ;color=blue
	// Explode is nil (uses default: false for path params)
	pathParam := &parser.Parameter{
		Name:   "color",
		In:     "path",
		Style:  "matrix",
		Schema: &parser.Schema{Type: "string"},
	}
	result := d.DeserializePathParam(";color=blue", pathParam)
	fmt.Println("Matrix path param:", result)

	// Deserialize a pipe-delimited query parameter: red|green|blue
	// Explode nil uses default (true for form, false for others)
	queryParam := &parser.Parameter{
		Name:   "colors",
		In:     "query",
		Style:  "pipeDelimited",
		Schema: &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}},
	}
	queryResult := d.DeserializeQueryParam([]string{"red|green|blue"}, queryParam)
	fmt.Println("Pipe-delimited query param:", queryResult)

	// Deserialize a simple header parameter: X-Rate-Limit: 100
	headerParam := &parser.Parameter{
		Name:   "X-Rate-Limit",
		In:     "header",
		Schema: &parser.Schema{Type: "integer"},
	}
	headerResult := d.DeserializeHeaderParam("100", headerParam)
	fmt.Println("Header param:", headerResult)
	// Output:
	// Matrix path param: blue
	// Pipe-delimited query param: [red green blue]
	// Header param: 100
}

func ExampleValidator_ValidateRequest_strictMode() {
	specYAML := `
openapi: "3.0.0"
info:
  title: Pet Store
  version: "1.0"
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: Success
`
	parsed, _ := parser.ParseWithOptions(parser.WithBytes([]byte(specYAML)))
	v, _ := httpvalidator.New(parsed)

	// Enable strict mode: unknown parameters cause errors
	v.StrictMode = true

	// Request with an unknown query parameter "color"
	req := httptest.NewRequest("GET", "/pets?limit=10&color=red", nil)
	result, _ := v.ValidateRequest(req)

	fmt.Println("Strict mode valid:", result.Valid)
	if len(result.Errors) > 0 {
		fmt.Println("Error:", result.Errors[0].Message)
	}

	// Now with strict mode off (default)
	v.StrictMode = false
	result2, _ := v.ValidateRequest(req)
	fmt.Println("Lenient mode valid:", result2.Valid)
	// Output:
	// Strict mode valid: false
	// Error: unknown query parameter "color"
	// Lenient mode valid: true
}

func ExamplePathMatcherSet() {
	// Create matchers for multiple paths
	templates := []string{
		"/pets",
		"/pets/{petId}",
		"/pets/{petId}/owner",
		"/users/{userId}",
	}

	set, _ := httpvalidator.NewPathMatcherSet(templates)

	// Match a request path
	template, params, found := set.Match("/pets/123/owner")

	fmt.Println("Found:", found)
	fmt.Println("Template:", template)
	fmt.Println("petId:", params["petId"])
	// Output:
	// Found: true
	// Template: /pets/{petId}/owner
	// petId: 123
}
