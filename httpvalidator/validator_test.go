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

// Helper to create a parsed spec from YAML content
func mustParse(t *testing.T, yaml string) *parser.ParseResult {
	t.Helper()
	result, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)
	return result
}

func TestNew(t *testing.T) {
	t.Run("creates validator from parsed spec", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
`)
		v, err := New(parsed)
		require.NoError(t, err)
		assert.NotNil(t, v)
		assert.True(t, v.IncludeWarnings)
		assert.False(t, v.StrictMode)
	})

	t.Run("returns error for nil parsed result", func(t *testing.T) {
		v, err := New(nil)
		assert.Nil(t, v)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("handles empty paths", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
		v, err := New(parsed)
		require.NoError(t, err)
		assert.NotNil(t, v)
	})
}

func TestValidateRequest_PathMatching(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
  /pets/{petId}:
    get:
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        "200":
          description: OK
`)
	v, _ := New(parsed)

	t.Run("matches exact path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "/pets", result.MatchedPath)
	})

	t.Run("matches parameterized path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets/123", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "/pets/{petId}", result.MatchedPath)
		assert.Equal(t, int64(123), result.PathParams["petId"])
	})

	t.Run("returns error for unknown path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/unknown", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "no matching path")
	})

	t.Run("returns error for unknown method", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/pets", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "not allowed")
	})
}

func TestValidateRequest_QueryParams(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
            minimum: 1
            maximum: 100
        - name: status
          in: query
          required: true
          schema:
            type: string
            enum: [available, pending, sold]
      responses:
        "200":
          description: OK
`)
	v, _ := New(parsed)

	t.Run("validates valid query params", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets?limit=10&status=available", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, int64(10), result.QueryParams["limit"])
		assert.Equal(t, "available", result.QueryParams["status"])
	})

	t.Run("returns error for missing required param", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets?limit=10", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "required")
	})

	t.Run("returns error for invalid enum value", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets?status=invalid", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "not one of the allowed values")
	})

	t.Run("returns error for out of range value", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets?limit=200&status=available", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "maximum")
	})
}

func TestValidateRequest_StrictMode(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
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
          description: OK
`)
	v, _ := New(parsed)
	v.StrictMode = true

	t.Run("rejects unknown query params in strict mode", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets?limit=10&unknown=foo", nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "unknown query parameter")
	})
}

func TestValidateRequest_RequestBody(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
                  minLength: 1
                age:
                  type: integer
                  minimum: 0
      responses:
        "201":
          description: Created
`)
	v, _ := New(parsed)

	t.Run("validates valid request body", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name": "Fluffy", "age": 3}`)
		req := httptest.NewRequest("POST", "/pets", body)
		req.Header.Set("Content-Type", "application/json")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		if !result.Valid {
			for _, e := range result.Errors {
				t.Logf("Error: %s: %s", e.Path, e.Message)
			}
		}
		assert.True(t, result.Valid)
	})

	t.Run("returns error for missing required field", func(t *testing.T) {
		body := bytes.NewBufferString(`{"age": 3}`)
		req := httptest.NewRequest("POST", "/pets", body)
		req.Header.Set("Content-Type", "application/json")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "required")
	})

	t.Run("returns error for missing required body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/pets", nil)
		req.Header.Set("Content-Type", "application/json")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "required")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		body := bytes.NewBufferString(`{invalid json}`)
		req := httptest.NewRequest("POST", "/pets", body)
		req.Header.Set("Content-Type", "application/json")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "invalid JSON")
	})
}

func TestValidateResponse(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
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
        "404":
          description: Not Found
        default:
          description: Error
`)
	v, _ := New(parsed)

	t.Run("validates valid response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets/123", nil)
		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       newReadCloser(`{"id": 123, "name": "Fluffy"}`),
		}

		result, err := v.ValidateResponse(req, resp)
		require.NoError(t, err)
		if !result.Valid {
			for _, e := range result.Errors {
				t.Logf("Error: %s: %s", e.Path, e.Message)
			}
		}
		assert.True(t, result.Valid)
		assert.Equal(t, 200, result.StatusCode)
	})

	t.Run("validates response with different status code", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets/123", nil)
		resp := &http.Response{
			StatusCode: 404,
			Header:     http.Header{},
			Body:       http.NoBody,
		}

		result, err := v.ValidateResponse(req, resp)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, 404, result.StatusCode)
	})

	t.Run("uses default response for unknown status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets/123", nil)
		resp := &http.Response{
			StatusCode: 500,
			Header:     http.Header{},
			Body:       http.NoBody,
		}

		result, err := v.ValidateResponse(req, resp)
		require.NoError(t, err)
		assert.True(t, result.Valid) // default response exists
	})
}

func TestValidateResponseData(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: integer
`)
	v, _ := New(parsed)

	t.Run("validates response data without http.Response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		statusCode := 200
		headers := http.Header{"Content-Type": []string{"application/json"}}
		body := []byte(`[{"id": 1}, {"id": 2}]`)

		result, err := v.ValidateResponseData(req, statusCode, headers, body)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "/pets", result.MatchedPath)
		assert.Equal(t, "GET", result.MatchedMethod)
	})
}

func TestValidateRequest_HeaderParams(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      parameters:
        - name: X-API-Key
          in: header
          required: true
          schema:
            type: string
            minLength: 10
      responses:
        "200":
          description: OK
`)
	v, _ := New(parsed)

	t.Run("validates valid header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		req.Header.Set("X-API-Key", "my-secret-api-key")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "my-secret-api-key", result.HeaderParams["X-API-Key"])
	})

	t.Run("returns error for missing required header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "required")
	})
}

func TestValidateRequest_CookieParams(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      parameters:
        - name: session
          in: cookie
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
`)
	v, _ := New(parsed)

	t.Run("validates valid cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "abc123", result.CookieParams["session"])
	})

	t.Run("returns error for missing required cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors[0].Message, "required")
	})
}

func TestValidateRequest_OAS2(t *testing.T) {
	parsed := mustParse(t, `
swagger: "2.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    post:
      consumes:
        - application/json
      parameters:
        - name: body
          in: body
          required: true
          schema:
            type: object
            required: [name]
            properties:
              name:
                type: string
      responses:
        "201":
          description: Created
`)
	v, _ := New(parsed)

	t.Run("validates OAS 2.0 body parameter", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name": "Fluffy"}`)
		req := httptest.NewRequest("POST", "/pets", body)
		req.Header.Set("Content-Type", "application/json")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})
}

// Helper to create an io.ReadCloser from a string
type stringReadCloser struct {
	*bytes.Reader
}

func (s *stringReadCloser) Close() error { return nil }

func newReadCloser(s string) *stringReadCloser {
	return &stringReadCloser{bytes.NewReader([]byte(s))}
}
