package parser

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher)
	doc := map[string]any{}

	// Test resolving the whole document
	result, err := resolver.Resolve(doc, server.URL)
	if err != nil {
		t.Fatalf("Failed to resolve HTTP ref: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map[string]any, got %T", result)
	}

	components, ok := resultMap["components"].(map[string]any)
	if !ok {
		t.Fatal("Expected components in result")
	}

	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatal("Expected schemas in components")
	}

	if _, ok := schemas["Pet"]; !ok {
		t.Error("Expected Pet schema in result")
	}
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

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher)
	doc := map[string]any{}

	// Test resolving with fragment
	ref := server.URL + "#/components/schemas/Pet"
	result, err := resolver.Resolve(doc, ref)
	if err != nil {
		t.Fatalf("Failed to resolve HTTP ref with fragment: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map[string]any, got %T", result)
	}

	if resultMap["type"] != "object" {
		t.Errorf("Expected type: object, got: %v", resultMap["type"])
	}

	props, ok := resultMap["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties in Pet schema")
	}

	if _, ok := props["name"]; !ok {
		t.Error("Expected name property in Pet schema")
	}
	if _, ok := props["age"]; !ok {
		t.Error("Expected age property in Pet schema")
	}
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

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher)
	doc := map[string]any{}

	// Resolve the same URL multiple times
	for i := 0; i < 3; i++ {
		_, err := resolver.Resolve(doc, server.URL)
		if err != nil {
			t.Fatalf("Failed to resolve on attempt %d: %v", i, err)
		}
	}

	// Should only have fetched once due to caching
	if callCount != 1 {
		t.Errorf("Expected 1 HTTP call due to caching, got %d", callCount)
	}
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

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher)
	doc := map[string]any{}

	// First resolution should work
	_, err := resolver.Resolve(doc, server.URL)
	if err != nil {
		t.Fatalf("First resolution should succeed: %v", err)
	}

	// Simulating visiting the same ref again (circular) - mark as visited
	resolver.visited[server.URL] = true
	_, err = resolver.Resolve(doc, server.URL)
	if err == nil {
		t.Error("Expected circular reference error")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("Expected 'circular reference' error, got: %v", err)
	}
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

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher)
	doc := map[string]any{}

	_, err := resolver.Resolve(doc, server.URL)
	if err == nil {
		t.Error("Expected error for oversized HTTP response")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size limit") {
		t.Errorf("Expected 'exceeds maximum size limit' error, got: %v", err)
	}
}

// TestResolveHTTPFetchError tests handling of HTTP fetch errors
func TestResolveHTTPFetchError(t *testing.T) {
	fetcher := func(url string) ([]byte, string, error) {
		return nil, "", http.ErrServerClosed
	}

	resolver := NewRefResolverWithHTTP(".", "http://example.com", fetcher)
	doc := map[string]any{}

	_, err := resolver.Resolve(doc, "http://example.com/api.yaml")
	if err == nil {
		t.Error("Expected error when fetcher fails")
	}
	if !strings.Contains(err.Error(), "failed to fetch HTTP reference") {
		t.Errorf("Expected 'failed to fetch HTTP reference' error, got: %v", err)
	}
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

	resolver := NewRefResolverWithHTTP(".", server.URL, fetcher)
	doc := map[string]any{}

	_, err := resolver.Resolve(doc, server.URL)
	if err == nil {
		t.Error("Expected error for invalid content")
	}
	if !strings.Contains(err.Error(), "failed to parse HTTP reference") {
		t.Errorf("Expected 'failed to parse HTTP reference' error, got: %v", err)
	}
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
			resolver := NewRefResolverWithHTTP(".", tt.baseURL, nil)
			result, err := resolver.resolveRelativeURL(tt.ref)
			if err != nil {
				t.Fatalf("Failed to resolve relative URL: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestNewRefResolverWithHTTP tests the constructor
func TestNewRefResolverWithHTTP(t *testing.T) {
	fetcher := func(url string) ([]byte, string, error) {
		return nil, "", nil
	}

	resolver := NewRefResolverWithHTTP("/base/dir", "https://example.com/api.yaml", fetcher)

	if resolver.baseDir != "/base/dir" {
		t.Errorf("Expected baseDir '/base/dir', got '%s'", resolver.baseDir)
	}

	if resolver.baseURL != "https://example.com/api.yaml" {
		t.Errorf("Expected baseURL 'https://example.com/api.yaml', got '%s'", resolver.baseURL)
	}

	if resolver.httpFetch == nil {
		t.Error("Expected httpFetch to be set")
	}

	if resolver.visited == nil {
		t.Error("Expected visited map to be initialized")
	}

	if resolver.resolving == nil {
		t.Error("Expected resolving map to be initialized")
	}

	if resolver.documents == nil {
		t.Error("Expected documents map to be initialized")
	}
}

// TestResolveHTTPRelativeRefFromHTTPSource tests that relative refs are resolved against baseURL
func TestResolveHTTPRelativeRefFromHTTPSource(t *testing.T) {
	callPaths := []string{}

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
	resolver := NewRefResolverWithHTTP(".", baseURL, fetcher)
	doc := map[string]any{}

	// Resolve a relative reference - should resolve against baseURL
	ref := "schemas/pet.yaml"
	result, err := resolver.Resolve(doc, ref)
	if err != nil {
		t.Fatalf("Failed to resolve relative ref: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map[string]any, got %T", result)
	}

	if resultMap["type"] != "pet" {
		t.Errorf("Expected type 'pet', got '%v'", resultMap["type"])
	}

	// Verify the correct path was called
	found := false
	for _, path := range callPaths {
		if path == "/api/schemas/pet.yaml" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected call to /api/schemas/pet.yaml, got calls to: %v", callPaths)
	}
}
