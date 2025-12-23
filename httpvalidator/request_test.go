package httpvalidator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// matchMediaType Tests
// =============================================================================

func TestMatchMediaType(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		mediaType string
		want      bool
	}{
		// Full wildcard
		{"full wildcard matches json", "*/*", "application/json", true},
		{"full wildcard matches xml", "*/*", "application/xml", true},
		{"full wildcard matches text", "*/*", "text/plain", true},

		// Prefix wildcard
		{"application/* matches json", "application/*", "application/json", true},
		{"application/* matches xml", "application/*", "application/xml", true},
		{"application/* does not match text", "application/*", "text/plain", false},
		{"text/* matches plain", "text/*", "text/plain", true},
		{"text/* matches html", "text/*", "text/html", true},

		// Exact match
		{"exact match json", "application/json", "application/json", true},
		{"exact match xml", "application/xml", "application/xml", true},
		{"no match different types", "application/json", "application/xml", false},
		{"no match different categories", "application/json", "text/plain", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchMediaType(tt.pattern, tt.mediaType)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// getRequestBodySchema Tests
// =============================================================================

func TestGetRequestBodySchema(t *testing.T) {
	t.Run("returns nil for nil request body", func(t *testing.T) {
		v := &Validator{}
		schema := v.getRequestBodySchema(nil, "application/json")
		assert.Nil(t, schema)
	})

	t.Run("returns nil for nil content", func(t *testing.T) {
		v := &Validator{}
		reqBody := &parser.RequestBody{Content: nil}
		schema := v.getRequestBodySchema(reqBody, "application/json")
		assert.Nil(t, schema)
	})

	t.Run("returns exact match schema", func(t *testing.T) {
		v := &Validator{}
		expectedSchema := &parser.Schema{Type: "object"}
		reqBody := &parser.RequestBody{
			Content: map[string]*parser.MediaType{
				"application/json": {Schema: expectedSchema},
			},
		}
		schema := v.getRequestBodySchema(reqBody, "application/json")
		assert.Equal(t, expectedSchema, schema)
	})

	t.Run("returns wildcard match schema when exact not found", func(t *testing.T) {
		v := &Validator{}
		expectedSchema := &parser.Schema{Type: "string"}
		reqBody := &parser.RequestBody{
			Content: map[string]*parser.MediaType{
				"application/*": {Schema: expectedSchema},
			},
		}
		schema := v.getRequestBodySchema(reqBody, "application/json")
		assert.Equal(t, expectedSchema, schema)
	})

	t.Run("returns nil when no match found", func(t *testing.T) {
		v := &Validator{}
		reqBody := &parser.RequestBody{
			Content: map[string]*parser.MediaType{
				"text/plain": {Schema: &parser.Schema{Type: "string"}},
			},
		}
		schema := v.getRequestBodySchema(reqBody, "application/json")
		assert.Nil(t, schema)
	})
}

// =============================================================================
// validateJSONBody Tests
// =============================================================================

func TestValidateJSONBody(t *testing.T) {
	v := &Validator{
		schemaValidator: NewSchemaValidator(),
	}

	t.Run("empty body returns error", func(t *testing.T) {
		result := newRequestResult()
		schema := &parser.Schema{Type: "object"}
		v.validateJSONBody([]byte{}, schema, result)

		assert.False(t, result.Valid)
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "empty")
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		result := newRequestResult()
		schema := &parser.Schema{Type: "object"}
		v.validateJSONBody([]byte(`{invalid json}`), schema, result)

		assert.False(t, result.Valid)
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "invalid JSON")
	})

	t.Run("valid JSON with schema validation", func(t *testing.T) {
		result := newRequestResult()
		schema := &parser.Schema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
			},
		}
		v.validateJSONBody([]byte(`{"name": "test"}`), schema, result)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("valid JSON failing schema validation", func(t *testing.T) {
		result := newRequestResult()
		schema := &parser.Schema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
			},
		}
		v.validateJSONBody([]byte(`{"age": 25}`), schema, result)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Errors), 1)
		assert.Contains(t, result.Errors[0].Message, "required")
	})
}

// =============================================================================
// validateFormBody Tests
// =============================================================================

func TestValidateFormBody_OAS3(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    post:
      requestBody:
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
                age:
                  type: string
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)

	t.Run("validates form data against schema", func(t *testing.T) {
		result := newRequestResult()
		schema := &parser.Schema{
			Type:     "object",
			Required: []string{"name"},
			Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
			},
		}
		v.validateFormBody([]byte("name=test"), schema, "/test", nil, result)

		assert.True(t, result.Valid)
	})

	t.Run("validates empty pairs in form data", func(t *testing.T) {
		result := newRequestResult()
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
			},
		}
		v.validateFormBody([]byte("name=test&&"), schema, "/test", nil, result)

		assert.True(t, result.Valid)
	})

	t.Run("handles key without value", func(t *testing.T) {
		result := newRequestResult()
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"flag": {Type: "string"},
			},
		}
		v.validateFormBody([]byte("flag"), schema, "/test", nil, result)

		assert.True(t, result.Valid)
	})
}

// =============================================================================
// validateFormDataParams Tests (OAS 2.0)
// =============================================================================

func TestValidateFormDataParams(t *testing.T) {
	// Note: OAS 2.0 formData parameters without a body parameter schema are currently
	// not validated because validateRequestBody returns early when bodySchema is nil.
	// These tests document the unit test coverage for validateFormDataParams when
	// called directly.

	parsed := mustParse(t, `
swagger: "2.0"
info:
  title: Test
  version: "1.0"
paths:
  /upload:
    post:
      consumes:
        - application/x-www-form-urlencoded
      parameters:
        - name: username
          in: formData
          type: string
          required: true
        - name: email
          in: formData
          type: string
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)

	t.Run("direct validateFormDataParams with valid data", func(t *testing.T) {
		result := newRequestResult()
		op := v.getOperation("/upload", "POST")
		require.NotNil(t, op)

		v.validateFormDataParams([]byte("username=john&email=john@example.com"), "/upload", op, result)
		assert.True(t, result.Valid)
	})

	t.Run("direct validateFormDataParams with missing required field", func(t *testing.T) {
		result := newRequestResult()
		op := v.getOperation("/upload", "POST")
		require.NotNil(t, op)

		v.validateFormDataParams([]byte("email=john@example.com"), "/upload", op, result)
		assert.False(t, result.Valid, "expected validation to fail for missing required form field")
		require.NotEmpty(t, result.Errors, "expected at least one error")
		hasRequiredError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "required") || containsSubstring(e.Message, "missing") {
				hasRequiredError = true
				break
			}
		}
		assert.True(t, hasRequiredError, "expected required/missing error, got: %v", result.Errors)
	})

	t.Run("direct validateFormDataParams with empty body", func(t *testing.T) {
		result := newRequestResult()
		op := v.getOperation("/upload", "POST")
		require.NotNil(t, op)

		v.validateFormDataParams([]byte(""), "/upload", op, result)
		assert.False(t, result.Valid) // Missing required username
	})

	t.Run("direct validateFormDataParams handles empty pairs", func(t *testing.T) {
		result := newRequestResult()
		op := v.getOperation("/upload", "POST")
		require.NotNil(t, op)

		v.validateFormDataParams([]byte("username=john&&"), "/upload", op, result)
		assert.True(t, result.Valid)
	})

	t.Run("direct validateFormDataParams handles key without value", func(t *testing.T) {
		result := newRequestResult()
		op := v.getOperation("/upload", "POST")
		require.NotNil(t, op)

		v.validateFormDataParams([]byte("username"), "/upload", op, result)
		// username is present but with empty value
		assert.True(t, result.Valid)
	})
}

// =============================================================================
// validatePathParams Tests
// =============================================================================

func TestValidatePathParams_UndefinedParam(t *testing.T) {
	// Create a minimal spec where we can inject an undefined path param
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /users/{userId}:
    get:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: integer
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)

	t.Run("handles path param with schema validation errors", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/not-a-number", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		// Check that there's a type error for the path param
		hasTypeError := false
		for _, e := range result.Errors {
			if e.Path == "path.userId" {
				hasTypeError = true
				break
			}
		}
		assert.True(t, hasTypeError, "expected type error for path.userId")
	})

	t.Run("missing required path parameter reports error", func(t *testing.T) {
		// This scenario tests when a defined path param isn't extracted
		// We test this by directly calling validatePathParams with incomplete data
		result := newRequestResult()
		op := v.getOperation("/users/{userId}", "GET")
		require.NotNil(t, op)

		// Pass empty path params - userId is required but missing
		v.validatePathParams(map[string]string{}, "/users/{userId}", op, result)

		assert.False(t, result.Valid)
		hasRequiredError := false
		for _, e := range result.Errors {
			if e.Path == "path.userId" && containsSubstring(e.Message, "required") {
				hasRequiredError = true
				break
			}
		}
		assert.True(t, hasRequiredError, "expected required error for path.userId")
	})
}

// =============================================================================
// validateQueryParams Tests
// =============================================================================

func TestValidateQueryParams_DeepObject(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /search:
    get:
      parameters:
        - name: filter
          in: query
          style: deepObject
          explode: true
          schema:
            type: object
            properties:
              status:
                type: string
              minPrice:
                type: string
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)

	t.Run("parses deepObject query params", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search?filter[status]=active&filter[minPrice]=100", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)

		filterVal, ok := result.QueryParams["filter"].(map[string]any)
		require.True(t, ok, "filter should be a map")
		assert.Equal(t, "active", filterVal["status"])
		assert.Equal(t, "100", filterVal["minPrice"])
	})
}

func TestValidateQueryParams_EmptyValue(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      parameters:
        - name: flag
          in: query
          allowEmptyValue: true
          schema:
            type: string
        - name: noEmpty
          in: query
          allowEmptyValue: false
          schema:
            type: string
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)
	v.IncludeWarnings = true

	t.Run("allows empty value when allowEmptyValue is true", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?flag=", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		// Should not have warnings for allowed empty values
		hasEmptyWarning := false
		for _, w := range result.Warnings {
			if containsSubstring(w.Message, "empty value") && containsSubstring(w.Path, "flag") {
				hasEmptyWarning = true
				break
			}
		}
		assert.False(t, hasEmptyWarning)
	})

	t.Run("warns on empty value when allowEmptyValue is false", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?noEmpty=", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		// Check for empty value warning
		hasEmptyWarning := false
		for _, w := range result.Warnings {
			if containsSubstring(w.Message, "empty value") {
				hasEmptyWarning = true
				break
			}
		}
		assert.True(t, hasEmptyWarning, "expected empty value warning")
	})
}

// =============================================================================
// validateHeaderParams Tests
// =============================================================================

func TestValidateHeaderParams_StrictMode(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      parameters:
        - name: X-Custom-Header
          in: header
          schema:
            type: string
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)
	v.StrictMode = true

	t.Run("allows standard headers in strict mode", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "test-client")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("rejects unknown non-standard headers in strict mode", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Unknown-Header", "value")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "unknown header")
	})

	t.Run("allows defined custom headers in strict mode", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Custom-Header", "my-value")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "my-value", result.HeaderParams["X-Custom-Header"])
	})

	t.Run("allows sec- prefixed headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Sec-Fetch-Mode", "cors")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})
}

func TestValidateHeaderParams_SchemaValidation(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      parameters:
        - name: X-Request-ID
          in: header
          required: true
          schema:
            type: string
            enum: [valid-id-1, valid-id-2]
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)

	t.Run("validates header against enum schema", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "invalid-value") // Not in enum

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		hasEnumError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "allowed values") || containsSubstring(e.Message, "enum") {
				hasEnumError = true
				break
			}
		}
		assert.True(t, hasEnumError, "expected enum validation error, got: %v", result.Errors)
	})

	t.Run("valid header passes schema validation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "valid-id-1")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})
}

// =============================================================================
// validateCookieParams Tests
// =============================================================================

func TestValidateCookieParams_StrictMode(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      parameters:
        - name: session
          in: cookie
          schema:
            type: string
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)
	v.StrictMode = true

	t.Run("rejects unknown cookies in strict mode", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "unknown_cookie", Value: "value"})

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "unknown cookie")
	})

	t.Run("allows defined cookies", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})
}

func TestValidateCookieParams_SchemaValidation(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      parameters:
        - name: prefs
          in: cookie
          required: true
          schema:
            type: string
            enum: [light, dark]
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)

	t.Run("validates cookie against enum schema", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "prefs", Value: "invalid"})

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		hasEnumError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "allowed values") {
				hasEnumError = true
				break
			}
		}
		assert.True(t, hasEnumError)
	})
}

// =============================================================================
// validateRequestBody Tests
// =============================================================================

func TestValidateRequestBody_ContentTypes(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
          text/plain:
            schema:
              type: string
          multipart/form-data:
            schema:
              type: object
              properties:
                file:
                  type: string
                  format: binary
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)
	v.IncludeWarnings = true

	t.Run("missing Content-Type header adds warning", func(t *testing.T) {
		body := bytes.NewBufferString(`{"test": true}`)
		req := httptest.NewRequest("POST", "/test", body)
		// No Content-Type header set

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		// Should have a warning about missing Content-Type
		hasWarning := false
		for _, w := range result.Warnings {
			if containsSubstring(w.Message, "Content-Type") {
				hasWarning = true
				break
			}
		}
		assert.True(t, hasWarning, "expected Content-Type warning")
	})

	t.Run("invalid Content-Type header returns error", func(t *testing.T) {
		body := bytes.NewBufferString(`{"test": true}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "invalid;;;") // Invalid media type

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "invalid Content-Type")
	})

	t.Run("text content type with empty body warns", func(t *testing.T) {
		body := bytes.NewBufferString("")
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "text/plain")
		req.ContentLength = 0 // Explicitly set to 0

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		// Body is required but empty - this returns an error
		assert.False(t, result.Valid)
	})

	t.Run("unsupported content type in strict mode", func(t *testing.T) {
		v.StrictMode = true
		defer func() { v.StrictMode = false }()

		body := bytes.NewBufferString(`<xml>test</xml>`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/xml")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "unsupported Content-Type")
	})

	t.Run("multipart form-data adds warning", func(t *testing.T) {
		body := bytes.NewBufferString("--boundary\r\nContent-Disposition: form-data; name=\"file\"\r\n\r\ndata\r\n--boundary--")
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		hasWarning := false
		for _, w := range result.Warnings {
			if containsSubstring(w.Message, "multipart") {
				hasWarning = true
				break
			}
		}
		assert.True(t, hasWarning, "expected multipart warning")
	})
}

func TestValidateRequestBody_JSONSuffixContentType(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    post:
      requestBody:
        content:
          application/vnd.api+json:
            schema:
              type: object
              properties:
                data:
                  type: string
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)

	t.Run("validates +json suffix content types", func(t *testing.T) {
		body := bytes.NewBufferString(`{"data": "test"}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/vnd.api+json")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})
}

func TestValidateRequestBody_WildcardContentType(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    post:
      requestBody:
        content:
          "*/*":
            schema:
              type: object
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)

	t.Run("matches wildcard content type", func(t *testing.T) {
		body := bytes.NewBufferString(`{"test": true}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		// Should match the wildcard
		assert.True(t, result.Valid)
	})
}

func TestValidateRequestBody_OAS2BodyParam(t *testing.T) {
	parsed := mustParse(t, `
swagger: "2.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    post:
      consumes:
        - application/json
      parameters:
        - name: body
          in: body
          required: true
          schema:
            type: object
            required: [id]
            properties:
              id:
                type: integer
      responses:
        "200":
          description: OK
`)

	v, err := New(parsed)
	require.NoError(t, err)

	t.Run("validates OAS 2.0 body parameter", func(t *testing.T) {
		body := bytes.NewBufferString(`{"id": 123}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("fails on missing required body field in OAS 2.0", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name": "test"}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		hasRequiredError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "required") {
				hasRequiredError = true
				break
			}
		}
		assert.True(t, hasRequiredError)
	})
}

// =============================================================================
// Integration Tests for Complete Request Validation Flow
// =============================================================================

func TestValidateRequest_CompleteFlow(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Complete Test API
  version: "1.0"
paths:
  /users/{userId}/posts:
    post:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: integer
        - name: draft
          in: query
          schema:
            type: boolean
        - name: X-Correlation-ID
          in: header
          required: true
          schema:
            type: string
        - name: auth
          in: cookie
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [title]
              properties:
                title:
                  type: string
                content:
                  type: string
      responses:
        "201":
          description: Created
`)

	v, err := New(parsed)
	require.NoError(t, err)

	t.Run("validates complete valid request", func(t *testing.T) {
		body := bytes.NewBufferString(`{"title": "My Post", "content": "Hello world"}`)
		req := httptest.NewRequest("POST", "/users/123/posts?draft=true", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Correlation-ID", "corr-123")
		req.AddCookie(&http.Cookie{Name: "auth", Value: "token123"})

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid, "errors: %v", result.Errors)
		assert.Equal(t, "/users/{userId}/posts", result.MatchedPath)
		assert.Equal(t, int64(123), result.PathParams["userId"])
		assert.Equal(t, true, result.QueryParams["draft"])
		assert.Equal(t, "corr-123", result.HeaderParams["X-Correlation-ID"])
		assert.Equal(t, "token123", result.CookieParams["auth"])
	})

	t.Run("fails with multiple validation errors", func(t *testing.T) {
		body := bytes.NewBufferString(`{"content": "Missing title"}`)
		req := httptest.NewRequest("POST", "/users/abc/posts", body) // Invalid userId type
		req.Header.Set("Content-Type", "application/json")
		// Missing X-Correlation-ID header
		// Missing auth cookie

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		// Should have multiple errors: invalid path param type, missing header, missing cookie, missing required field
		assert.GreaterOrEqual(t, len(result.Errors), 3, "errors: %v", result.Errors)
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// containsSubstring is a helper to check if a string contains a substring.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
