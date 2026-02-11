package httpvalidator

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erraggy/oastools/internal/testutil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// getResponseDefinition Tests
// =============================================================================

func TestGetResponseDefinition(t *testing.T) {
	t.Run("returns nil for nil responses", func(t *testing.T) {
		v := &Validator{}
		op := &parser.Operation{Responses: nil}
		resp := v.getResponseDefinition(op, 200)
		assert.Nil(t, resp)
	})

	t.Run("finds exact status code match", func(t *testing.T) {
		expected := &parser.Response{Description: "OK"}
		v := &Validator{}
		op := &parser.Operation{
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": expected,
					"400": {Description: "Bad Request"},
				},
			},
		}
		resp := v.getResponseDefinition(op, 200)
		assert.Equal(t, expected, resp)
	})

	t.Run("finds 2XX wildcard match", func(t *testing.T) {
		expected := &parser.Response{Description: "Success"}
		v := &Validator{}
		op := &parser.Operation{
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"2XX": expected,
				},
			},
		}
		resp := v.getResponseDefinition(op, 201)
		assert.Equal(t, expected, resp)
	})

	t.Run("finds lowercase 2xx wildcard match", func(t *testing.T) {
		expected := &parser.Response{Description: "Success"}
		v := &Validator{}
		op := &parser.Operation{
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"2xx": expected,
				},
			},
		}
		resp := v.getResponseDefinition(op, 204)
		assert.Equal(t, expected, resp)
	})

	t.Run("finds 4XX wildcard match", func(t *testing.T) {
		expected := &parser.Response{Description: "Client Error"}
		v := &Validator{}
		op := &parser.Operation{
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"4XX": expected,
				},
			},
		}
		resp := v.getResponseDefinition(op, 404)
		assert.Equal(t, expected, resp)
	})

	t.Run("finds 5XX wildcard match", func(t *testing.T) {
		expected := &parser.Response{Description: "Server Error"}
		v := &Validator{}
		op := &parser.Operation{
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"5XX": expected,
				},
			},
		}
		resp := v.getResponseDefinition(op, 503)
		assert.Equal(t, expected, resp)
	})

	t.Run("falls back to default response", func(t *testing.T) {
		expected := &parser.Response{Description: "Default"}
		v := &Validator{}
		op := &parser.Operation{
			Responses: &parser.Responses{
				Default: expected,
				Codes: map[string]*parser.Response{
					"200": {Description: "OK"},
				},
			},
		}
		resp := v.getResponseDefinition(op, 500)
		assert.Equal(t, expected, resp)
	})

	t.Run("prefers exact match over wildcard", func(t *testing.T) {
		exact := &parser.Response{Description: "Exact 200"}
		wildcard := &parser.Response{Description: "2XX"}
		v := &Validator{}
		op := &parser.Operation{
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": exact,
					"2XX": wildcard,
				},
			},
		}
		resp := v.getResponseDefinition(op, 200)
		assert.Equal(t, exact, resp)
	})

	t.Run("returns nil when no match found", func(t *testing.T) {
		v := &Validator{}
		op := &parser.Operation{
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {Description: "OK"},
				},
			},
		}
		resp := v.getResponseDefinition(op, 500)
		assert.Nil(t, resp)
	})
}

// =============================================================================
// getResponseSchema Tests
// =============================================================================

func TestGetResponseSchema(t *testing.T) {
	t.Run("returns nil for nil content", func(t *testing.T) {
		v := &Validator{}
		resp := &parser.Response{Content: nil}
		schema := v.getResponseSchema(resp, "application/json")
		assert.Nil(t, schema)
	})

	t.Run("finds exact media type match", func(t *testing.T) {
		expected := &parser.Schema{Type: "object"}
		v := &Validator{}
		resp := &parser.Response{
			Content: map[string]*parser.MediaType{
				"application/json": {Schema: expected},
			},
		}
		schema := v.getResponseSchema(resp, "application/json")
		assert.Equal(t, expected, schema)
	})

	t.Run("finds wildcard media type match", func(t *testing.T) {
		expected := &parser.Schema{Type: "object"}
		v := &Validator{}
		resp := &parser.Response{
			Content: map[string]*parser.MediaType{
				"application/*": {Schema: expected},
			},
		}
		schema := v.getResponseSchema(resp, "application/json")
		assert.Equal(t, expected, schema)
	})

	t.Run("returns nil when no match", func(t *testing.T) {
		v := &Validator{}
		resp := &parser.Response{
			Content: map[string]*parser.MediaType{
				"text/plain": {Schema: &parser.Schema{Type: "string"}},
			},
		}
		schema := v.getResponseSchema(resp, "application/json")
		assert.Nil(t, schema)
	})
}

// =============================================================================
// validateJSONResponseBody Tests
// =============================================================================

func TestValidateJSONResponseBody(t *testing.T) {
	v := &Validator{
		schemaValidator: NewSchemaValidator(),
	}

	t.Run("validates valid JSON", func(t *testing.T) {
		result := newResponseResult()
		schema := &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id": {Type: "integer"},
			},
		}
		v.validateJSONResponseBody([]byte(`{"id": 123}`), schema, result)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("reports invalid JSON", func(t *testing.T) {
		result := newResponseResult()
		schema := &parser.Schema{Type: "object"}
		v.validateJSONResponseBody([]byte(`{invalid json}`), schema, result)

		assert.False(t, result.Valid)
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "invalid JSON")
	})

	t.Run("validates against schema", func(t *testing.T) {
		result := newResponseResult()
		schema := &parser.Schema{
			Type:     "object",
			Required: []string{"id"},
			Properties: map[string]*parser.Schema{
				"id": {Type: "integer"},
			},
		}
		v.validateJSONResponseBody([]byte(`{"name": "test"}`), schema, result)

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
// validateResponseHeaders Tests
// =============================================================================

func TestValidateResponseHeaders(t *testing.T) {
	v := &Validator{
		schemaValidator: NewSchemaValidator(),
	}

	t.Run("validates required headers present", func(t *testing.T) {
		result := newResponseResult()
		responseDef := &parser.Response{
			Headers: map[string]*parser.Header{
				"X-Request-ID": {Required: true, Schema: &parser.Schema{Type: "string"}},
			},
		}
		headers := http.Header{"X-Request-Id": []string{"123"}}
		v.validateResponseHeaders(headers, responseDef, result)

		assert.True(t, result.Valid)
	})

	t.Run("reports missing required headers", func(t *testing.T) {
		result := newResponseResult()
		responseDef := &parser.Response{
			Headers: map[string]*parser.Header{
				"X-Request-ID": {Required: true, Schema: &parser.Schema{Type: "string"}},
			},
		}
		headers := http.Header{}
		v.validateResponseHeaders(headers, responseDef, result)

		assert.False(t, result.Valid)
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "required response header")
	})

	t.Run("validates header against schema", func(t *testing.T) {
		result := newResponseResult()
		responseDef := &parser.Response{
			Headers: map[string]*parser.Header{
				"X-Count": {Schema: &parser.Schema{Type: "integer"}},
			},
		}
		headers := http.Header{"X-Count": []string{"not-a-number"}}
		v.validateResponseHeaders(headers, responseDef, result)

		assert.False(t, result.Valid)
	})

	t.Run("skips validation when no headers defined", func(t *testing.T) {
		result := newResponseResult()
		responseDef := &parser.Response{Headers: nil}
		headers := http.Header{"X-Custom": []string{"value"}}
		v.validateResponseHeaders(headers, responseDef, result)

		assert.True(t, result.Valid)
	})

	t.Run("ignores missing optional headers", func(t *testing.T) {
		result := newResponseResult()
		responseDef := &parser.Response{
			Headers: map[string]*parser.Header{
				"X-Optional": {Required: false, Schema: &parser.Schema{Type: "string"}},
			},
		}
		headers := http.Header{}
		v.validateResponseHeaders(headers, responseDef, result)

		assert.True(t, result.Valid)
	})
}

// =============================================================================
// validateResponseBody Tests
// =============================================================================

func TestValidateResponseBody(t *testing.T) {
	t.Run("skips validation when no schema defined", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
		v, _ := New(parsed)

		result := newResponseResult()
		responseDef := &parser.Response{Content: nil}
		v.validateResponseBody([]byte(`{"any": "data"}`), "application/json", responseDef, result)

		assert.True(t, result.Valid)
	})

	t.Run("warns on empty body with schema defined", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
		v, _ := New(parsed)
		v.IncludeWarnings = true

		result := newResponseResult()
		responseDef := &parser.Response{
			Content: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{Type: "object"}},
			},
		}
		v.validateResponseBody(nil, "application/json", responseDef, result)

		assert.True(t, result.Valid) // Empty body is a warning, not error
		hasEmptyWarning := false
		for _, w := range result.Warnings {
			if containsSubstring(w.Message, "empty") {
				hasEmptyWarning = true
				break
			}
		}
		assert.True(t, hasEmptyWarning)
	})

	t.Run("validates JSON response body", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
		v, _ := New(parsed)

		result := newResponseResult()
		responseDef := &parser.Response{
			Content: map[string]*parser.MediaType{
				"application/json": {Schema: &parser.Schema{
					Type:     "object",
					Required: []string{"id"},
					Properties: map[string]*parser.Schema{
						"id": {Type: "integer"},
					},
				}},
			},
		}
		v.validateResponseBody([]byte(`{"id": 123}`), "application/json", responseDef, result)

		assert.True(t, result.Valid)
	})

	t.Run("validates +json suffix content types", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
		v, _ := New(parsed)

		result := newResponseResult()
		responseDef := &parser.Response{
			Content: map[string]*parser.MediaType{
				"application/vnd.api+json": {Schema: &parser.Schema{Type: "object"}},
			},
		}
		v.validateResponseBody([]byte(`{"data": "test"}`), "application/vnd.api+json", responseDef, result)

		assert.True(t, result.Valid)
	})

	t.Run("validates text response with string schema", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
		v, _ := New(parsed)

		result := newResponseResult()
		responseDef := &parser.Response{
			Content: map[string]*parser.MediaType{
				"text/plain": {Schema: &parser.Schema{
					Type:      "string",
					MinLength: testutil.Ptr(5),
				}},
			},
		}
		v.validateResponseBody([]byte("hello world"), "text/plain", responseDef, result)

		assert.True(t, result.Valid)
	})

	t.Run("warns on unsupported content type", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
		v, _ := New(parsed)
		v.IncludeWarnings = true

		result := newResponseResult()
		responseDef := &parser.Response{
			Content: map[string]*parser.MediaType{
				"application/octet-stream": {Schema: &parser.Schema{Type: "string", Format: "binary"}},
			},
		}
		v.validateResponseBody([]byte{0x01, 0x02, 0x03}, "application/octet-stream", responseDef, result)

		hasWarning := false
		for _, w := range result.Warnings {
			if containsSubstring(w.Message, "cannot validate") {
				hasWarning = true
				break
			}
		}
		assert.True(t, hasWarning)
	})
}

// =============================================================================
// validateResponseParts Tests
// =============================================================================

func TestValidateResponseParts(t *testing.T) {
	t.Run("strict mode errors on undocumented status code", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
`)
		v, _ := New(parsed)
		v.StrictMode = true

		result := newResponseResult()
		op := v.getOperation("/test", "GET")
		v.validateResponseParts(500, http.Header{}, nil, "/test", op, result)

		assert.False(t, result.Valid)
		hasUndocumentedError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "undocumented") {
				hasUndocumentedError = true
				break
			}
		}
		assert.True(t, hasUndocumentedError)
	})

	t.Run("non-strict mode warns on undocumented status code", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
`)
		v, _ := New(parsed)
		v.IncludeWarnings = true
		v.StrictMode = false

		result := newResponseResult()
		op := v.getOperation("/test", "GET")
		v.validateResponseParts(500, http.Header{}, nil, "/test", op, result)

		assert.True(t, result.Valid) // Warning doesn't mark as invalid
		hasWarning := false
		for _, w := range result.Warnings {
			if containsSubstring(w.Message, "not documented") {
				hasWarning = true
				break
			}
		}
		assert.True(t, hasWarning)
	})
}

// =============================================================================
// validateResponse Tests (with *http.Response)
// =============================================================================

func TestValidateResponse_BodyRead(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties:
                  id:
                    type: integer
`)

	v, _ := New(parsed)

	t.Run("reads and validates response body", func(t *testing.T) {
		result := newResponseResult()
		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       newReadCloser(`{"id": 123}`),
		}
		op := v.getOperation("/test", "GET")
		v.validateResponse(resp, "/test", op, result)

		assert.True(t, result.Valid, "errors: %v", result.Errors)
	})

	t.Run("handles nil body", func(t *testing.T) {
		result := newResponseResult()
		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{},
			Body:       nil,
		}
		op := v.getOperation("/test", "GET")
		v.validateResponse(resp, "/test", op, result)

		// Should not panic, just skip body validation
		assert.True(t, result.Valid)
	})
}

// =============================================================================
// ValidateResponseData Tests
// =============================================================================

func TestValidateResponseData_Comprehensive(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /users/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        "200":
          description: OK
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
`)

	v, _ := New(parsed)

	t.Run("validates complete response data", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/123", nil)
		headers := http.Header{"Content-Type": []string{"application/json"}}
		body := []byte(`{"id": 123, "name": "John"}`)

		result, err := v.ValidateResponseData(req, 200, headers, body)

		require.NoError(t, err)
		assert.True(t, result.Valid, "errors: %v", result.Errors)
		assert.Equal(t, "/users/{id}", result.MatchedPath)
		assert.Equal(t, "GET", result.MatchedMethod)
		assert.Equal(t, 200, result.StatusCode)
		assert.Equal(t, "application/json", result.ContentType)
	})

	t.Run("returns error for unknown path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/unknown", nil)

		result, err := v.ValidateResponseData(req, 200, http.Header{}, nil)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		hasPathError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "no matching path") {
				hasPathError = true
				break
			}
		}
		assert.True(t, hasPathError)
	})

	t.Run("returns error for unknown method", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/users/123", nil)

		result, err := v.ValidateResponseData(req, 200, http.Header{}, nil)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		hasMethodError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "no operation found") {
				hasMethodError = true
				break
			}
		}
		assert.True(t, hasMethodError)
	})
}

// =============================================================================
// OAS 2.0 Response Validation Tests
// =============================================================================

func TestValidateResponseBody_OAS2(t *testing.T) {
	parsed := mustParse(t, `
swagger: "2.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      produces:
        - application/json
      responses:
        "200":
          description: OK
          schema:
            type: array
            items:
              type: object
              properties:
                id:
                  type: integer
`)

	v, _ := New(parsed)

	t.Run("validates OAS 2.0 response with direct schema", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       newReadCloser(`[{"id": 1}, {"id": 2}]`),
		}

		result, err := v.ValidateResponse(req, resp)

		require.NoError(t, err)
		assert.True(t, result.Valid, "errors: %v", result.Errors)
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestValidateResponse_IntegrationWithHeaders(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /items:
    get:
      responses:
        "200":
          description: OK
          headers:
            X-Total-Count:
              required: true
              schema:
                type: integer
            X-Page:
              schema:
                type: integer
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
`)

	v, _ := New(parsed)

	t.Run("validates response with required headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/items", nil)
		resp := &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type":  []string{"application/json"},
				"X-Total-Count": []string{"42"},
			},
			Body: newReadCloser(`[{"id": 1}]`),
		}

		result, err := v.ValidateResponse(req, resp)

		require.NoError(t, err)
		assert.True(t, result.Valid, "errors: %v", result.Errors)
	})

	t.Run("fails when required header is missing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/items", nil)
		resp := &http.Response{
			StatusCode: 200,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: newReadCloser(`[{"id": 1}]`),
		}

		result, err := v.ValidateResponse(req, resp)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		hasHeaderError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "required response header") {
				hasHeaderError = true
				break
			}
		}
		assert.True(t, hasHeaderError, "errors: %v", result.Errors)
	})
}
