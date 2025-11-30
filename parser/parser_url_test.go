package parser

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsURL tests the isURL function
func TestIsURL(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"HTTP URL", "http://example.com/api.yaml", true},
		{"HTTPS URL", "https://example.com/api.yaml", true},
		{"File path", "/path/to/file.yaml", false},
		{"Relative path", "../testdata/api.yaml", false},
		{"Windows path", "C:\\path\\to\\file.yaml", false},
		{"FTP URL (not supported)", "ftp://example.com/file.yaml", false},
		{"Empty string", "", false},
		{"Just http", "http", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isURL(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDetectFormatFromURL tests format detection from URLs
func TestDetectFormatFromURL(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		contentType    string
		expectedFormat SourceFormat
	}{
		{
			name:           "JSON extension in URL",
			url:            "https://example.com/api/spec.json",
			contentType:    "",
			expectedFormat: SourceFormatJSON,
		},
		{
			name:           "YAML extension in URL",
			url:            "https://example.com/api/spec.yaml",
			contentType:    "",
			expectedFormat: SourceFormatYAML,
		},
		{
			name:           "YML extension in URL",
			url:            "https://example.com/api/spec.yml",
			contentType:    "",
			expectedFormat: SourceFormatYAML,
		},
		{
			name:           "No extension, JSON content-type",
			url:            "https://example.com/api/spec",
			contentType:    "application/json",
			expectedFormat: SourceFormatJSON,
		},
		{
			name:           "No extension, YAML content-type",
			url:            "https://example.com/api/spec",
			contentType:    "application/yaml",
			expectedFormat: SourceFormatYAML,
		},
		{
			name:           "No extension, x-yaml content-type",
			url:            "https://example.com/api/spec",
			contentType:    "application/x-yaml",
			expectedFormat: SourceFormatYAML,
		},
		{
			name:           "No extension, text/yaml content-type",
			url:            "https://example.com/api/spec",
			contentType:    "text/yaml",
			expectedFormat: SourceFormatYAML,
		},
		{
			name:           "Content-type with charset",
			url:            "https://example.com/api/spec",
			contentType:    "application/json; charset=utf-8",
			expectedFormat: SourceFormatJSON,
		},
		{
			name:           "No extension, no content-type",
			url:            "https://example.com/api/spec",
			contentType:    "",
			expectedFormat: SourceFormatUnknown,
		},
		{
			name:           "Extension overrides content-type",
			url:            "https://example.com/api/spec.json",
			contentType:    "application/yaml",
			expectedFormat: SourceFormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := detectFormatFromURL(tt.url, tt.contentType)
			assert.Equal(t, tt.expectedFormat, format)
		})
	}
}

// TestFetchURL tests URL fetching with a test server
func TestFetchURL(t *testing.T) {
	// Create test content
	yamlContent := `openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
`

	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		expectError   bool
		errorContains string
	}{
		{
			name: "successful fetch with 200 OK",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/yaml")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(yamlContent))
				}))
			},
			expectError: false,
		},
		{
			name: "404 not found",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte("Not Found"))
				}))
			},
			expectError:   true,
			errorContains: "HTTP 404",
		},
		{
			name: "500 internal server error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte("Internal Server Error"))
				}))
			},
			expectError:   true,
			errorContains: "HTTP 500",
		},
		{
			name: "401 unauthorized",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte("Unauthorized"))
				}))
			},
			expectError:   true,
			errorContains: "HTTP 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			p := New()
			data, contentType, err := p.fetchURL(server.URL)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, data)
				assert.Contains(t, string(data), "Test API")
				assert.Equal(t, "application/yaml", contentType)
			}
		})
	}
}

// TestParseFromURL tests end-to-end parsing from URLs
func TestParseFromURL(t *testing.T) {
	// Create test OAS documents
	oas30YAML := `openapi: "3.0.3"
info:
  title: URL Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: Success
`

	oas20JSON := `{
  "swagger": "2.0",
  "info": {
    "title": "URL Test API",
    "version": "1.0.0"
  },
  "paths": {
    "/users": {
      "get": {
        "responses": {
          "200": {
            "description": "Success"
          }
        }
      }
    }
  }
}`

	tests := []struct {
		name           string
		urlPath        string
		content        string
		contentType    string
		expectError    bool
		validateResult func(*testing.T, *ParseResult)
	}{
		{
			name:        "parse OAS 3.0 YAML from URL",
			urlPath:     "/api/spec.yaml",
			content:     oas30YAML,
			contentType: "application/yaml",
			expectError: false,
			validateResult: func(t *testing.T, result *ParseResult) {
				assert.Equal(t, "3.0.3", result.Version)
				assert.Equal(t, OASVersion303, result.OASVersion)
				doc, ok := result.Document.(*OAS3Document)
				assert.True(t, ok)
				assert.Equal(t, "URL Test API", doc.Info.Title)
				assert.Empty(t, result.Errors)
				assert.Equal(t, SourceFormatYAML, result.SourceFormat)
			},
		},
		{
			name:        "parse OAS 2.0 JSON from URL",
			urlPath:     "/api/spec.json",
			content:     oas20JSON,
			contentType: "application/json",
			expectError: false,
			validateResult: func(t *testing.T, result *ParseResult) {
				assert.Equal(t, "2.0", result.Version)
				assert.Equal(t, OASVersion20, result.OASVersion)
				doc, ok := result.Document.(*OAS2Document)
				assert.True(t, ok)
				assert.Equal(t, "URL Test API", doc.Info.Title)
				assert.Empty(t, result.Errors)
				assert.Equal(t, SourceFormatJSON, result.SourceFormat)
			},
		},
		{
			name:        "URL is preserved in SourcePath",
			urlPath:     "/api/openapi.yaml",
			content:     oas30YAML,
			contentType: "application/yaml",
			expectError: false,
			validateResult: func(t *testing.T, result *ParseResult) {
				assert.Contains(t, result.SourcePath, "http://")
				assert.Contains(t, result.SourcePath, "/api/openapi.yaml")
			},
		},
		{
			name:        "format detection from URL extension",
			urlPath:     "/spec.json",
			content:     oas20JSON,
			contentType: "",
			expectError: false,
			validateResult: func(t *testing.T, result *ParseResult) {
				assert.Equal(t, SourceFormatJSON, result.SourceFormat)
			},
		},
		{
			name:        "format detection from Content-Type (no extension)",
			urlPath:     "/api/spec",
			content:     oas30YAML,
			contentType: "application/yaml",
			expectError: false,
			validateResult: func(t *testing.T, result *ParseResult) {
				assert.Equal(t, SourceFormatYAML, result.SourceFormat)
				assert.Equal(t, "3.0.3", result.Version)
			},
		},
		{
			name:        "format detection from Content-Type with charset (no extension)",
			urlPath:     "/openapi",
			content:     oas20JSON,
			contentType: "application/json; charset=utf-8",
			expectError: false,
			validateResult: func(t *testing.T, result *ParseResult) {
				assert.Equal(t, SourceFormatJSON, result.SourceFormat)
				assert.Equal(t, "2.0", result.Version)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == tt.urlPath {
					if tt.contentType != "" {
						w.Header().Set("Content-Type", tt.contentType)
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(tt.content))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			// Parse from URL
			p := New()
			url := server.URL + tt.urlPath
			result, err := p.Parse(url)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}
		})
	}
}

// TestParseURLErrors tests error handling when parsing from URLs
func TestParseURLErrors(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		errorContains string
	}{
		{
			name: "invalid YAML from URL",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("{{{invalid yaml"))
				}))
			},
			errorContains: "failed to parse YAML/JSON",
		},
		{
			name: "missing version field",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`info:
  title: No Version
  version: 1.0.0
paths: {}`))
				}))
			},
			errorContains: "unable to detect OpenAPI version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			p := New()
			result, err := p.Parse(server.URL + "/api/spec.yaml")

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
			assert.Nil(t, result)
		})
	}
}

// TestFetchURLWithInvalidURL tests error handling for malformed URLs
func TestFetchURLWithInvalidURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"invalid scheme", "ht!tp://invalid-url"},
		{"malformed URL", "://no-scheme"},
		{"empty URL", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			_, _, err := p.fetchURL(tt.url)
			assert.Error(t, err)
		})
	}
}

// TestCustomUserAgent tests that custom User-Agent is used when fetching URLs
func TestCustomUserAgent(t *testing.T) {
	yamlContent := `openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`

	var receivedUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(yamlContent))
	}))
	defer server.Close()

	tests := []struct {
		name              string
		userAgent         string
		expectedUserAgent string
	}{
		{
			name:              "custom user agent",
			userAgent:         "oastools/1.5.0",
			expectedUserAgent: "oastools/1.5.0",
		},
		{
			name:              "default user agent when not set",
			userAgent:         "",
			expectedUserAgent: "oastools/dev",
		},
		{
			name:              "default user agent from New()",
			userAgent:         "default",
			expectedUserAgent: "oastools/dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			if tt.userAgent == "default" {
				// Use default from New()
			} else {
				p.UserAgent = tt.userAgent
			}

			receivedUserAgent = "" // Reset
			_, err := p.Parse(server.URL + "/spec.yaml")
			require.NoError(t, err)
			assert.Equal(t, tt.expectedUserAgent, receivedUserAgent)
		})
	}
}

// TestParseURLvsFilePath tests that the parser correctly distinguishes between URLs and file paths
func TestParseURLvsFilePath(t *testing.T) {
	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")
	fileContent := `openapi: "3.0.0"
info:
  title: File Test API
  version: 1.0.0
paths: {}`
	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	// Create a test server
	urlContent := `openapi: "3.0.0"
info:
  title: URL Test API
  version: 1.0.0
paths: {}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(urlContent))
	}))
	defer server.Close()

	p := New()

	// Test file path
	fileResult, err := p.Parse(testFile)
	require.NoError(t, err)
	assert.Equal(t, testFile, fileResult.SourcePath)
	doc1, ok := fileResult.Document.(*OAS3Document)
	require.True(t, ok)
	assert.Equal(t, "File Test API", doc1.Info.Title)

	// Test URL
	urlResult, err := p.Parse(server.URL + "/spec.yaml")
	require.NoError(t, err)
	assert.Contains(t, urlResult.SourcePath, "http://")
	doc2, ok := urlResult.Document.(*OAS3Document)
	require.True(t, ok)
	assert.Equal(t, "URL Test API", doc2.Info.Title)
}
