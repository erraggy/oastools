package httpvalidator

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
		// With nil/empty body, the JSON validator reports "request body is empty"
		// rather than the old "required but missing" since we no longer short-circuit
		// on ContentLength==0 (which caused false negatives for chunked transfers).
		require.NotEmpty(t, result.Errors)
		assert.True(t,
			strings.Contains(result.Errors[0].Message, "required") ||
				strings.Contains(result.Errors[0].Message, "empty"),
			"expected error about required or empty body, got: %s", result.Errors[0].Message,
		)
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

// =============================================================================
// Validator Helper Method Tests
// =============================================================================

func TestValidator_IsOAS3(t *testing.T) {
	t.Run("returns true for OAS 3.x", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
		v, _ := New(parsed)
		assert.True(t, v.IsOAS3())
		assert.False(t, v.IsOAS2())
	})
}

func TestValidator_IsOAS2(t *testing.T) {
	t.Run("returns true for OAS 2.0", func(t *testing.T) {
		parsed := mustParse(t, `
swagger: "2.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
		v, _ := New(parsed)
		assert.True(t, v.IsOAS2())
		assert.False(t, v.IsOAS3())
	})
}

func TestValidator_GetOperation_AllMethods(t *testing.T) {
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
    post:
      responses:
        "201":
          description: Created
    put:
      responses:
        "200":
          description: OK
    delete:
      responses:
        "204":
          description: No Content
    patch:
      responses:
        "200":
          description: OK
    head:
      responses:
        "200":
          description: OK
    options:
      responses:
        "200":
          description: OK
    trace:
      responses:
        "200":
          description: OK
`)

	v, _ := New(parsed)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			op := v.getOperation("/test", method)
			assert.NotNil(t, op, "operation for %s should exist", method)
		})
	}

	t.Run("unknown method returns nil", func(t *testing.T) {
		op := v.getOperation("/test", "CUSTOM")
		assert.Nil(t, op)
	})

	t.Run("lowercase method works", func(t *testing.T) {
		op := v.getOperation("/test", "get")
		assert.NotNil(t, op)
	})
}

func TestValidator_GetParameters_Merging(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /items/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: integer
      - name: version
        in: header
        schema:
          type: string
    get:
      parameters:
        - name: version
          in: header
          schema:
            type: integer
      responses:
        "200":
          description: OK
`)

	v, _ := New(parsed)

	t.Run("merges path-level and operation-level parameters", func(t *testing.T) {
		op := v.getOperation("/items/{id}", "GET")
		params := v.getParameters("/items/{id}", op)

		// Should have 2 params: id (from path level) and version (operation overrides path)
		assert.Len(t, params, 2)

		// Find the version param - it should have integer type from operation level
		var versionParam *parser.Parameter
		for _, p := range params {
			if p.Name == "version" {
				versionParam = p
				break
			}
		}
		require.NotNil(t, versionParam)
		assert.Equal(t, "integer", versionParam.Schema.Type)
	})

	t.Run("returns operation params when no path item", func(t *testing.T) {
		op := &parser.Operation{
			Parameters: []*parser.Parameter{
				{Name: "test", In: "query"},
			},
		}
		params := v.getParameters("/nonexistent", op)
		require.Len(t, params, 1)
		assert.Equal(t, "test", params[0].Name)
	})
}

func TestValidator_MatchPath_NilMatcherSet(t *testing.T) {
	v := &Validator{
		pathMatcherSet: nil,
	}

	template, params, found := v.matchPath("/any/path")
	assert.False(t, found)
	assert.Empty(t, template)
	assert.Nil(t, params)
}

// =============================================================================
// truncateForError Tests
// =============================================================================

func TestTruncateForError(t *testing.T) {
	t.Run("short string unchanged", func(t *testing.T) {
		assert.Equal(t, "short", truncateForError("short", 200))
	})

	t.Run("exact length unchanged", func(t *testing.T) {
		s := strings.Repeat("x", 200)
		assert.Equal(t, s, truncateForError(s, 200))
	})

	t.Run("long string truncated with ellipsis", func(t *testing.T) {
		long := strings.Repeat("x", 300)
		got := truncateForError(long, 200)
		assert.Equal(t, 203, len(got)) // 200 + "..."
		assert.True(t, strings.HasSuffix(got, "..."))
	})

	t.Run("empty string unchanged", func(t *testing.T) {
		assert.Equal(t, "", truncateForError("", 200))
	})

	t.Run("maxLen zero truncates everything", func(t *testing.T) {
		assert.Equal(t, "...", truncateForError("hello", 0))
	})
}

// =============================================================================
// Error message sanitization tests
// =============================================================================

func TestValidateRequest_PathSanitized(t *testing.T) {
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
	v, _ := New(parsed)

	t.Run("long path is truncated in error", func(t *testing.T) {
		longPath := "/" + strings.Repeat("x", 300)
		req := httptest.NewRequest("GET", longPath, nil)
		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		require.NotEmpty(t, result.Errors)
		// Error path should use static key, not raw user input
		assert.Equal(t, "request.path", result.Errors[0].Path)
		// Error message should be truncated and quoted
		assert.Contains(t, result.Errors[0].Message, "...")
		assert.Less(t, len(result.Errors[0].Message), 300)
	})
}

func TestValidateRequest_ContentTypeSanitized(t *testing.T) {
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
      responses:
        "200":
          description: OK
`)
	v, _ := New(parsed)

	t.Run("invalid Content-Type is quoted and truncated", func(t *testing.T) {
		body := bytes.NewBufferString(`{"test": true}`)
		req := httptest.NewRequest("POST", "/test", body)
		longCT := "invalid/" + strings.Repeat("x", 300) + ";;;"
		req.Header.Set("Content-Type", longCT)

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		require.NotEmpty(t, result.Errors)
		// Should contain quoted truncated value
		assert.Contains(t, result.Errors[0].Message, "...")
		assert.Contains(t, result.Errors[0].Message, "invalid Content-Type")
	})

	t.Run("unsupported Content-Type is quoted in strict mode", func(t *testing.T) {
		v.StrictMode = true
		defer func() { v.StrictMode = false }()

		body := bytes.NewBufferString(`<xml>test</xml>`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/xml")

		result, err := v.ValidateRequest(req)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		require.NotEmpty(t, result.Errors)
		// The media type should be quoted with %q
		assert.Contains(t, result.Errors[0].Message, `"application/xml"`)
	})
}

// =============================================================================
// Validation Flags Snapshot Tests
// =============================================================================

// mutatingReader is a reader that flips Validator fields when Read is called,
// simulating a concurrent mutation during validation.
type mutatingReader struct {
	data    []byte
	offset  int
	mutateF func()
	mutated bool
}

func (r *mutatingReader) Read(p []byte) (int, error) {
	if !r.mutated {
		r.mutateF()
		r.mutated = true
	}
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}

func TestValidateRequest_SnapshotStrictMode(t *testing.T) {
	// Verify that StrictMode is snapshotted at the start of ValidateRequest,
	// so a concurrent mutation after the method begins does not affect
	// the in-flight validation.
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    post:
      parameters:
        - name: known
          in: query
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
      responses:
        "200":
          description: OK
`)
	v, err := New(parsed)
	require.NoError(t, err)

	// Start with StrictMode enabled -- unknown query params should be rejected.
	v.StrictMode = true

	// The mutating reader flips StrictMode to false when the body is read.
	// Body reading happens AFTER query param validation, but within the
	// same ValidateRequest call. If the snapshot works, the query param
	// validation already captured StrictMode=true and will reject the
	// unknown parameter regardless of the mutation.
	body := &mutatingReader{
		data: []byte(`{"ok": true}`),
		mutateF: func() {
			v.StrictMode = false
		},
	}

	req := httptest.NewRequest("POST", "/test?known=a&unknown=b", body)
	req.Header.Set("Content-Type", "application/json")

	result, err := v.ValidateRequest(req)
	require.NoError(t, err)

	// The unknown query param should still cause an error because StrictMode
	// was true when ValidateRequest began (before the reader flipped it).
	assert.False(t, result.Valid, "expected validation to fail for unknown query param")
	hasUnknownQueryErr := false
	for _, e := range result.Errors {
		if containsSubstring(e.Message, "unknown query parameter") {
			hasUnknownQueryErr = true
			break
		}
	}
	assert.True(t, hasUnknownQueryErr,
		"expected 'unknown query parameter' error despite StrictMode being flipped during body read; errors: %v",
		result.Errors)
}

func TestValidateRequest_SnapshotIncludeWarnings(t *testing.T) {
	// Verify that IncludeWarnings is snapshotted at the start of ValidateRequest.
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    post:
      parameters:
        - name: flag
          in: query
          allowEmptyValue: false
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        "200":
          description: OK
`)
	v, err := New(parsed)
	require.NoError(t, err)

	// Start with IncludeWarnings enabled
	v.IncludeWarnings = true

	// The mutating reader flips IncludeWarnings to false during body read.
	body := &mutatingReader{
		data: []byte(`{"ok": true}`),
		mutateF: func() {
			v.IncludeWarnings = false
		},
	}

	req := httptest.NewRequest("POST", "/test?flag=", body)
	req.Header.Set("Content-Type", "application/json")

	result, err := v.ValidateRequest(req)
	require.NoError(t, err)

	// The empty value warning should still be present because IncludeWarnings
	// was true when ValidateRequest began.
	hasEmptyWarning := false
	for _, w := range result.Warnings {
		if containsSubstring(w.Message, "empty value") {
			hasEmptyWarning = true
			break
		}
	}
	assert.True(t, hasEmptyWarning,
		"expected 'empty value' warning despite IncludeWarnings being flipped during body read; warnings: %v",
		result.Warnings)
}

func TestValidateResponse_SnapshotStrictMode(t *testing.T) {
	// Verify that StrictMode is snapshotted at the start of ValidateResponse.
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
	v, err := New(parsed)
	require.NoError(t, err)

	// Start with StrictMode enabled -- undocumented status codes should error.
	v.StrictMode = true

	// The mutating reader flips StrictMode to false when the response body is read.
	body := &mutatingReader{
		data: []byte(`{"error": "not found"}`),
		mutateF: func() {
			v.StrictMode = false
		},
	}

	req := httptest.NewRequest("GET", "/test", nil)
	resp := &http.Response{
		StatusCode: 404, // Undocumented status code
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(body),
	}

	result, err := v.ValidateResponse(req, resp)
	require.NoError(t, err)

	// The undocumented status code should still cause an error because
	// StrictMode was true when ValidateResponse began.
	assert.False(t, result.Valid, "expected validation to fail for undocumented status code")
	hasUndocumentedErr := false
	for _, e := range result.Errors {
		if containsSubstring(e.Message, "undocumented") {
			hasUndocumentedErr = true
			break
		}
	}
	assert.True(t, hasUndocumentedErr,
		"expected 'undocumented' error despite StrictMode being flipped during body read; errors: %v",
		result.Errors)
}

// Helper to create an io.ReadCloser from a string
type stringReadCloser struct {
	*bytes.Reader
}

func (s *stringReadCloser) Close() error { return nil }

func newReadCloser(s string) *stringReadCloser {
	return &stringReadCloser{bytes.NewReader([]byte(s))}
}
