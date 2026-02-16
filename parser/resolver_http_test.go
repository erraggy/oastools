package parser

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erraggy/oastools/oaserrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveHTTPBasic tests basic HTTP ref resolution
func TestResolveHTTPBasic(t *testing.T) {
	// Create a mock server that serves a simple schema
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"components": {
				"schemas": {
					"Pet": {
						"type": "object",
						"properties": {
							"name": {"type": "string"}
						}
					}
				}
			}
		}`))
	}))
	defer server.Close()

	// Create fetcher that uses HTTP client
	fetcher := func(url string) ([]byte, string, error) {
		resp, err := http.Get(url) //nolint:noctx // test helper
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = resp.Body.Close() }()
		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		return body[:n], resp.Header.Get("Content-Type"), nil
	}

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher, 0, 0, 0)
	doc := map[string]any{}

	// Test resolving the whole document
	result, err := resolver.Resolve(doc, server.URL)
	require.NoError(t, err, "Failed to resolve HTTP ref")

	resultMap, ok := result.(map[string]any)
	require.True(t, ok, "Expected map[string]any, got %T", result)

	components, ok := resultMap["components"].(map[string]any)
	require.True(t, ok, "Expected components in result")

	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok, "Expected schemas in components")

	_, ok = schemas["Pet"]
	assert.True(t, ok, "Expected Pet schema in result")
}

// TestResolveHTTPWithFragment tests HTTP ref resolution with JSON pointer fragment
func TestResolveHTTPWithFragment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write([]byte(`
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer
`))
	}))
	defer server.Close()

	fetcher := func(url string) ([]byte, string, error) {
		resp, err := http.Get(url) //nolint:noctx // test helper
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = resp.Body.Close() }()
		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		return body[:n], resp.Header.Get("Content-Type"), nil
	}

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher, 0, 0, 0)
	doc := map[string]any{}

	// Test resolving with fragment
	ref := server.URL + "#/components/schemas/Pet"
	result, err := resolver.Resolve(doc, ref)
	require.NoError(t, err, "Failed to resolve HTTP ref with fragment")

	resultMap, ok := result.(map[string]any)
	require.True(t, ok, "Expected map[string]any, got %T", result)

	assert.Equal(t, "object", resultMap["type"], "Expected type: object")

	props, ok := resultMap["properties"].(map[string]any)
	require.True(t, ok, "Expected properties in Pet schema")

	_, ok = props["name"]
	assert.True(t, ok, "Expected name property in Pet schema")
	_, ok = props["age"]
	assert.True(t, ok, "Expected age property in Pet schema")
}

// TestResolveHTTPCaching tests that HTTP documents are cached
func TestResolveHTTPCaching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"schemas": {"Test": {"type": "string"}}}`))
	}))
	defer server.Close()

	fetcher := func(url string) ([]byte, string, error) {
		resp, err := http.Get(url) //nolint:noctx // test helper
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = resp.Body.Close() }()
		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		return body[:n], resp.Header.Get("Content-Type"), nil
	}

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher, 0, 0, 0)
	doc := map[string]any{}

	// Resolve the same URL multiple times
	for i := 0; i < 3; i++ {
		_, err := resolver.Resolve(doc, server.URL)
		require.NoError(t, err, "Failed to resolve on attempt %d", i)
	}

	// Should only have fetched once due to caching
	assert.Equal(t, 1, callCount, "Expected 1 HTTP call due to caching")
}

// TestResolveHTTPCircularDetection tests that circular HTTP refs are detected
func TestResolveHTTPCircularDetection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"type": "object"}`))
	}))
	defer server.Close()

	fetcher := func(url string) ([]byte, string, error) {
		resp, err := http.Get(url) //nolint:noctx // test helper
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = resp.Body.Close() }()
		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		return body[:n], resp.Header.Get("Content-Type"), nil
	}

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher, 0, 0, 0)
	doc := map[string]any{}

	// First resolution should work
	_, err := resolver.Resolve(doc, server.URL)
	require.NoError(t, err, "First resolution should succeed")

	// Simulating visiting the same ref again (circular) - mark as visited
	resolver.visited[server.URL] = true
	_, err = resolver.Resolve(doc, server.URL)
	require.Error(t, err, "Expected circular reference error")

	// Use errors.Is for sentinel error check
	assert.True(t, errors.Is(err, oaserrors.ErrCircularReference), "Expected ErrCircularReference, got: %v", err)

	// Use errors.As to verify error type and fields
	var refErr *oaserrors.ReferenceError
	require.True(t, errors.As(err, &refErr), "Expected *oaserrors.ReferenceError, got %T", err)
	assert.True(t, refErr.IsCircular, "Expected IsCircular=true")
	assert.Equal(t, "http", refErr.RefType, "Expected RefType='http'")
}

// TestResolveHTTPMaxSizeLimit tests that HTTP responses are size-limited
func TestResolveHTTPMaxSizeLimit(t *testing.T) {
	// Create a server that returns a response larger than MaxFileSize
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Write more than MaxFileSize bytes
		large := make([]byte, MaxFileSize+1000)
		for i := range large {
			large[i] = 'x'
		}
		_, _ = w.Write(large)
	}))
	defer server.Close()

	fetcher := func(url string) ([]byte, string, error) {
		resp, err := http.Get(url) //nolint:noctx // test helper
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = resp.Body.Close() }()
		// Read the entire response
		body := make([]byte, MaxFileSize+2000)
		n, _ := resp.Body.Read(body)
		return body[:n], resp.Header.Get("Content-Type"), nil
	}

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher, 0, 0, 0)
	doc := map[string]any{}

	_, err := resolver.Resolve(doc, server.URL)
	require.Error(t, err, "Expected error for oversized HTTP response")
	assert.Contains(t, err.Error(), "exceeds maximum size limit")
}

// TestResolveHTTPFetchError tests handling of HTTP fetch errors
func TestResolveHTTPFetchError(t *testing.T) {
	fetcher := func(url string) ([]byte, string, error) {
		return nil, "", http.ErrServerClosed
	}

	resolver := NewRefResolverWithHTTP(".", "http://example.com", fetcher, 0, 0, 0)
	doc := map[string]any{}

	_, err := resolver.Resolve(doc, "http://example.com/api.yaml")
	require.Error(t, err, "Expected error when fetcher fails")
	assert.Contains(t, err.Error(), "failed to fetch HTTP reference")
}

// TestResolveHTTPInvalidContent tests handling of invalid YAML/JSON content
func TestResolveHTTPInvalidContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	fetcher := func(url string) ([]byte, string, error) {
		resp, err := http.Get(url) //nolint:noctx // test helper
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = resp.Body.Close() }()
		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		return body[:n], resp.Header.Get("Content-Type"), nil
	}

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher, 0, 0, 0)
	doc := map[string]any{}

	_, err := resolver.Resolve(doc, server.URL)
	require.Error(t, err, "Expected error for invalid content")
	assert.Contains(t, err.Error(), "failed to parse HTTP reference")
}

// TestResolveRelativeURL tests relative URL resolution
func TestResolveRelativeURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		ref      string
		expected string
	}{
		{
			name:     "simple relative path",
			baseURL:  "https://example.com/api/spec.yaml",
			ref:      "schemas/pet.yaml",
			expected: "https://example.com/api/schemas/pet.yaml",
		},
		{
			name:     "relative path with fragment",
			baseURL:  "https://example.com/api/spec.yaml",
			ref:      "schemas/pet.yaml#/Pet",
			expected: "https://example.com/api/schemas/pet.yaml#/Pet",
		},
		{
			name:     "parent directory reference",
			baseURL:  "https://example.com/api/v1/spec.yaml",
			ref:      "../schemas/pet.yaml",
			expected: "https://example.com/api/schemas/pet.yaml",
		},
		{
			name:     "same directory reference",
			baseURL:  "https://example.com/api/spec.yaml",
			ref:      "./common.yaml",
			expected: "https://example.com/api/common.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewRefResolverWithHTTP(".", tt.baseURL, nil, 0, 0, 0)
			result, err := resolver.resolveRelativeURL(tt.ref)
			require.NoError(t, err, "Failed to resolve relative URL")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNewRefResolverWithHTTP tests the constructor
func TestNewRefResolverWithHTTP(t *testing.T) {
	fetcher := func(url string) ([]byte, string, error) {
		return nil, "", nil
	}

	resolver := NewRefResolverWithHTTP("/base/dir", "https://example.com/api.yaml", fetcher, 0, 0, 0)

	assert.Equal(t, "/base/dir", resolver.baseDir, "Expected baseDir '/base/dir'")
	assert.Equal(t, "https://example.com/api.yaml", resolver.baseURL, "Expected baseURL 'https://example.com/api.yaml'")
	assert.NotNil(t, resolver.httpFetch, "Expected httpFetch to be set")
	assert.NotNil(t, resolver.visited, "Expected visited map to be initialized")
	assert.NotNil(t, resolver.resolving, "Expected resolving map to be initialized")
	assert.NotNil(t, resolver.documents, "Expected documents map to be initialized")
}

// TestResolveHTTPRelativeRefFromHTTPSource tests that relative refs are resolved against baseURL
func TestResolveHTTPRelativeRefFromHTTPSource(t *testing.T) {
	var callPaths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/yaml")
		switch r.URL.Path {
		case "/api/spec.yaml":
			_, _ = w.Write([]byte(`{"type": "main"}`))
		case "/api/schemas/pet.yaml":
			_, _ = w.Write([]byte(`{"type": "pet"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	fetcher := func(url string) ([]byte, string, error) {
		resp, err := http.Get(url) //nolint:noctx // test helper
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = resp.Body.Close() }()
		body := make([]byte, 4096)
		n, _ := resp.Body.Read(body)
		return body[:n], resp.Header.Get("Content-Type"), nil
	}

	baseURL := server.URL + "/api/spec.yaml"
	resolver := NewRefResolverWithHTTP(".", baseURL, fetcher, 0, 0, 0)
	doc := map[string]any{}

	// Resolve a relative reference - should resolve against baseURL
	ref := "schemas/pet.yaml"
	result, err := resolver.Resolve(doc, ref)
	require.NoError(t, err, "Failed to resolve relative ref")

	resultMap, ok := result.(map[string]any)
	require.True(t, ok, "Expected map[string]any, got %T", result)

	assert.Equal(t, "pet", resultMap["type"], "Expected type 'pet'")

	// Verify the correct path was called
	assert.Contains(t, callPaths, "/api/schemas/pet.yaml", "Expected call to /api/schemas/pet.yaml")
}
