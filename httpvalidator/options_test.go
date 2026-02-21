package httpvalidator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Option Functions Tests
// =============================================================================

func TestWithFilePath(t *testing.T) {
	cfg := &config{}
	opt := WithFilePath("/path/to/spec.yaml")
	err := opt(cfg)

	assert.NoError(t, err)
	assert.Equal(t, "/path/to/spec.yaml", cfg.filePath)
}

func TestWithParsed(t *testing.T) {
	t.Run("accepts valid parsed result", func(t *testing.T) {
		parsed := &parser.ParseResult{}
		cfg := &config{}
		opt := WithParsed(parsed)
		err := opt(cfg)

		assert.NoError(t, err)
		assert.Equal(t, parsed, cfg.parsed)
	})

	t.Run("returns error for nil parsed result", func(t *testing.T) {
		cfg := &config{}
		opt := WithParsed(nil)
		err := opt(cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})
}

func TestWithIncludeWarnings(t *testing.T) {
	t.Run("sets include warnings to true", func(t *testing.T) {
		cfg := &config{}
		opt := WithIncludeWarnings(true)
		err := opt(cfg)

		assert.NoError(t, err)
		assert.True(t, cfg.includeWarnings)
	})

	t.Run("sets include warnings to false", func(t *testing.T) {
		cfg := &config{includeWarnings: true}
		opt := WithIncludeWarnings(false)
		err := opt(cfg)

		assert.NoError(t, err)
		assert.False(t, cfg.includeWarnings)
	})
}

func TestWithStrictMode(t *testing.T) {
	t.Run("enables strict mode", func(t *testing.T) {
		cfg := &config{}
		opt := WithStrictMode(true)
		err := opt(cfg)

		assert.NoError(t, err)
		assert.True(t, cfg.strictMode)
	})

	t.Run("disables strict mode", func(t *testing.T) {
		cfg := &config{strictMode: true}
		opt := WithStrictMode(false)
		err := opt(cfg)

		assert.NoError(t, err)
		assert.False(t, cfg.strictMode)
	})
}

func TestWithSkipBodyValidation(t *testing.T) {
	cfg := &config{}
	opt := WithSkipBodyValidation(true)
	err := opt(cfg)

	assert.NoError(t, err)
	assert.True(t, cfg.skipBodyValidation)
}

func TestWithSkipQueryValidation(t *testing.T) {
	cfg := &config{}
	opt := WithSkipQueryValidation(true)
	err := opt(cfg)

	assert.NoError(t, err)
	assert.True(t, cfg.skipQueryValidation)
}

func TestWithSkipHeaderValidation(t *testing.T) {
	cfg := &config{}
	opt := WithSkipHeaderValidation(true)
	err := opt(cfg)

	assert.NoError(t, err)
	assert.True(t, cfg.skipHeaderValidation)
}

func TestWithSkipCookieValidation(t *testing.T) {
	cfg := &config{}
	opt := WithSkipCookieValidation(true)
	err := opt(cfg)

	assert.NoError(t, err)
	assert.True(t, cfg.skipCookieValidation)
}

func TestWithMaxBodySize(t *testing.T) {
	t.Run("sets max body size", func(t *testing.T) {
		cfg := &config{}
		opt := WithMaxBodySize(1024)
		err := opt(cfg)

		assert.NoError(t, err)
		assert.Equal(t, int64(1024), cfg.maxBodySize)
	})

	t.Run("allows zero (means default)", func(t *testing.T) {
		cfg := &config{}
		opt := WithMaxBodySize(0)
		err := opt(cfg)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), cfg.maxBodySize)
	})

	t.Run("returns error for negative value", func(t *testing.T) {
		cfg := &config{}
		opt := WithMaxBodySize(-1)
		err := opt(cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "negative")
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	assert.True(t, cfg.includeWarnings, "includeWarnings should default to true")
	assert.False(t, cfg.strictMode, "strictMode should default to false")
	assert.Empty(t, cfg.filePath)
	assert.Nil(t, cfg.parsed)
	assert.False(t, cfg.skipBodyValidation)
	assert.False(t, cfg.skipQueryValidation)
	assert.False(t, cfg.skipHeaderValidation)
	assert.False(t, cfg.skipCookieValidation)
	assert.Equal(t, int64(0), cfg.maxBodySize, "maxBodySize should default to 0 (meaning default 10 MiB)")
}

// =============================================================================
// getParsedSpec Tests
// =============================================================================

func TestGetParsedSpec(t *testing.T) {
	t.Run("returns parsed when set", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
		cfg := &config{parsed: parsed}
		result, err := getParsedSpec(cfg)

		assert.NoError(t, err)
		assert.Equal(t, parsed, result)
	})

	t.Run("parses file when filePath set", func(t *testing.T) {
		// Create a temp file with valid spec
		content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test-spec.yaml")
		err := os.WriteFile(tmpFile, []byte(content), 0644)
		require.NoError(t, err)

		cfg := &config{filePath: tmpFile}
		result, err := getParsedSpec(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("returns error when nothing set", func(t *testing.T) {
		cfg := &config{}
		result, err := getParsedSpec(cfg)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no specification provided")
	})

	t.Run("prefers parsed over filePath", func(t *testing.T) {
		parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Parsed
  version: "1.0"
paths: {}
`)
		cfg := &config{
			parsed:   parsed,
			filePath: "/nonexistent/file.yaml",
		}
		result, err := getParsedSpec(cfg)

		assert.NoError(t, err)
		assert.Equal(t, parsed, result)
	})
}

// =============================================================================
// ValidateRequestWithOptions Tests
// =============================================================================

func TestValidateRequestWithOptions(t *testing.T) {
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

	t.Run("validates request with parsed spec", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets?limit=10", nil)
		result, err := ValidateRequestWithOptions(req, WithParsed(parsed))

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "/pets", result.MatchedPath)
	})

	t.Run("respects strict mode option", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets?limit=10&unknown=foo", nil)
		result, err := ValidateRequestWithOptions(
			req,
			WithParsed(parsed),
			WithStrictMode(true),
		)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		hasUnknownError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "unknown") {
				hasUnknownError = true
				break
			}
		}
		assert.True(t, hasUnknownError)
	})

	t.Run("respects include warnings option", func(t *testing.T) {
		// We'd need a scenario that generates warnings
		req := httptest.NewRequest("GET", "/pets", nil)
		result, err := ValidateRequestWithOptions(
			req,
			WithParsed(parsed),
			WithIncludeWarnings(false),
		)

		require.NoError(t, err)
		assert.True(t, result.Valid)
		// With warnings disabled, we shouldn't see any
		assert.Empty(t, result.Warnings)
	})

	t.Run("returns error when option fails", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		result, err := ValidateRequestWithOptions(req, WithParsed(nil))

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error for no spec", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		result, err := ValidateRequestWithOptions(req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no specification")
	})
}

// =============================================================================
// validateRequestWithSkips Tests
// =============================================================================

func TestValidateRequestWithSkips(t *testing.T) {
	parsed := mustParse(t, `
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /users/{userId}:
    post:
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: integer
        - name: filter
          in: query
          required: true
          schema:
            type: string
        - name: X-API-Key
          in: header
          required: true
          schema:
            type: string
        - name: session
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
              required: [name]
              properties:
                name:
                  type: string
      responses:
        "201":
          description: Created
`)

	t.Run("skips query validation", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users/123", bytes.NewBufferString(`{"name": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "secret")
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})
		// Missing required query param 'filter'

		result, err := ValidateRequestWithOptions(
			req,
			WithParsed(parsed),
			WithSkipQueryValidation(true),
		)

		require.NoError(t, err)
		assert.True(t, result.Valid, "should pass when query validation is skipped: %v", result.Errors)
	})

	t.Run("skips header validation", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users/123?filter=active", bytes.NewBufferString(`{"name": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		// Missing required header X-API-Key
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})

		result, err := ValidateRequestWithOptions(
			req,
			WithParsed(parsed),
			WithSkipHeaderValidation(true),
		)

		require.NoError(t, err)
		assert.True(t, result.Valid, "should pass when header validation is skipped: %v", result.Errors)
	})

	t.Run("skips cookie validation", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users/123?filter=active", bytes.NewBufferString(`{"name": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "secret")
		// Missing required cookie 'session'

		result, err := ValidateRequestWithOptions(
			req,
			WithParsed(parsed),
			WithSkipCookieValidation(true),
		)

		require.NoError(t, err)
		assert.True(t, result.Valid, "should pass when cookie validation is skipped: %v", result.Errors)
	})

	t.Run("skips body validation", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users/123?filter=active", nil) // Missing body
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "secret")
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})

		result, err := ValidateRequestWithOptions(
			req,
			WithParsed(parsed),
			WithSkipBodyValidation(true),
		)

		require.NoError(t, err)
		assert.True(t, result.Valid, "should pass when body validation is skipped: %v", result.Errors)
	})

	t.Run("validates path params even when skipping others", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users/not-a-number?filter=active", bytes.NewBufferString(`{"name": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "secret")
		req.AddCookie(&http.Cookie{Name: "session", Value: "abc"})

		result, err := ValidateRequestWithOptions(
			req,
			WithParsed(parsed),
			WithSkipQueryValidation(true),
			WithSkipHeaderValidation(true),
			WithSkipCookieValidation(true),
			WithSkipBodyValidation(true),
		)

		require.NoError(t, err)
		// Path params are always validated
		assert.False(t, result.Valid, "should fail on invalid path param")
	})

	t.Run("handles unknown path in skips mode", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/unknown", nil)

		result, err := ValidateRequestWithOptions(
			req,
			WithParsed(parsed),
			WithSkipBodyValidation(true),
		)

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

	t.Run("handles unknown method in skips mode", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/users/123", nil)

		result, err := ValidateRequestWithOptions(
			req,
			WithParsed(parsed),
			WithSkipBodyValidation(true),
		)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		hasMethodError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "not allowed") {
				hasMethodError = true
				break
			}
		}
		assert.True(t, hasMethodError)
	})
}

// =============================================================================
// ValidateResponseWithOptions Tests
// =============================================================================

func TestValidateResponseWithOptions(t *testing.T) {
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

	t.Run("validates response with parsed spec", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       newReadCloser(`[{"id": 1}]`),
		}

		result, err := ValidateResponseWithOptions(req, resp, WithParsed(parsed))

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, 200, result.StatusCode)
	})

	t.Run("returns error for no spec", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{},
			Body:       http.NoBody,
		}

		result, err := ValidateResponseWithOptions(req, resp)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("respects strict mode", func(t *testing.T) {
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
		req := httptest.NewRequest("GET", "/pets", nil)
		resp := &http.Response{
			StatusCode: 404, // Not in spec
			Header:     http.Header{},
			Body:       http.NoBody,
		}

		result, err := ValidateResponseWithOptions(
			req, resp,
			WithParsed(parsed),
			WithStrictMode(true),
		)

		require.NoError(t, err)
		assert.False(t, result.Valid)
	})
}

// =============================================================================
// ValidateResponseDataWithOptions Tests
// =============================================================================

func TestValidateResponseDataWithOptions(t *testing.T) {
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
                type: object
                required: [id]
                properties:
                  id:
                    type: integer
`)

	t.Run("validates response data with parsed spec", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		headers := http.Header{"Content-Type": []string{"application/json"}}
		body := []byte(`{"id": 123}`)

		result, err := ValidateResponseDataWithOptions(
			req, 200, headers, body,
			WithParsed(parsed),
		)

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, 200, result.StatusCode)
		assert.Equal(t, "/pets", result.MatchedPath)
	})

	t.Run("returns error for no spec", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)

		result, err := ValidateResponseDataWithOptions(
			req, 200, http.Header{}, nil,
		)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("validates response data against schema", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/pets", nil)
		headers := http.Header{"Content-Type": []string{"application/json"}}
		body := []byte(`{"name": "missing id"}`) // Missing required field

		result, err := ValidateResponseDataWithOptions(
			req, 200, headers, body,
			WithParsed(parsed),
		)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		hasRequiredError := false
		for _, e := range result.Errors {
			if containsSubstring(e.Message, "required") {
				hasRequiredError = true
				break
			}
		}
		assert.True(t, hasRequiredError, "expected required field error")
	})
}

// =============================================================================
// File-based Validation Tests
// =============================================================================

func TestValidateWithFilePath(t *testing.T) {
	// Create a temp file with valid spec
	content := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths:
  /hello:
    get:
      responses:
        "200":
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-spec.yaml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	t.Run("validates request from file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/hello", nil)
		result, err := ValidateRequestWithOptions(req, WithFilePath(tmpFile))

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "/hello", result.MatchedPath)
	})

	t.Run("validates response from file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/hello", nil)
		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{},
			Body:       http.NoBody,
		}

		result, err := ValidateResponseWithOptions(req, resp, WithFilePath(tmpFile))

		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("returns error for invalid file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/hello", nil)
		result, err := ValidateRequestWithOptions(req, WithFilePath("/nonexistent/file.yaml"))

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
