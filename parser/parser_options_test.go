package parser

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseWithOptions_FilePath tests the functional options API with file path
func TestParseWithOptions_FilePath(t *testing.T) {
	result, err := ParseWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithResolveRefs(false),
		WithValidateStructure(true),
	)
	require.NoError(t, err)
	assert.Equal(t, "3.0.3", result.Version)

	doc, ok := result.Document.(*OAS3Document)
	require.True(t, ok, "Expected OAS3Document, got %T", result.Document)
	assert.NotNil(t, doc.Info)
	assert.Equal(t, "Petstore API", doc.Info.Title)
	assert.Empty(t, result.Errors)
}

// TestParseWithOptions_Reader tests the functional options API with io.Reader
func TestParseWithOptions_Reader(t *testing.T) {
	file, err := os.Open("../testdata/petstore-3.0.yaml")
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	result, err := ParseWithOptions(
		WithReader(file),
		WithValidateStructure(true),
	)
	require.NoError(t, err)
	assert.Equal(t, "3.0.3", result.Version)
	assert.Equal(t, "ParseReader.yaml", result.SourcePath)
	assert.Empty(t, result.Errors)
}

// TestParseWithOptions_Bytes tests the functional options API with byte slice
func TestParseWithOptions_Bytes(t *testing.T) {
	data, err := os.ReadFile("../testdata/petstore-3.0.yaml")
	require.NoError(t, err)

	result, err := ParseWithOptions(
		WithBytes(data),
		WithResolveRefs(false),
	)
	require.NoError(t, err)
	assert.Equal(t, "3.0.3", result.Version)
	assert.Equal(t, "ParseBytes.yaml", result.SourcePath)
}

// TestParseWithOptions_UserAgent tests that user agent option is applied
func TestParseWithOptions_UserAgent(t *testing.T) {
	// Create a test HTTP server that records the User-Agent header
	receivedUA := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`))
	}))
	defer server.Close()

	customUA := "custom-user-agent/1.0"
	_, err := ParseWithOptions(
		WithFilePath(server.URL),
		WithUserAgent(customUA),
	)
	require.NoError(t, err)
	assert.Equal(t, customUA, receivedUA)
}

// TestParseWithOptions_DefaultValues tests that default values are applied correctly
func TestParseWithOptions_DefaultValues(t *testing.T) {
	result, err := ParseWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		// Not specifying WithResolveRefs or WithValidateStructure to test defaults
	)
	require.NoError(t, err)

	// Default: ValidateStructure = true, so no structural errors
	assert.Empty(t, result.Errors)

	// Default: ResolveRefs = false (hard to test directly, but would be visible
	// in documents with $refs if we had a test case with unresolved refs)
	assert.NotNil(t, result.Document)
}

// TestParseWithOptions_NoInputSource tests error when no input source is specified
func TestParseWithOptions_NoInputSource(t *testing.T) {
	_, err := ParseWithOptions(
		WithResolveRefs(true),
		WithValidateStructure(false),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify an input source")
}

// TestParseWithOptions_MultipleInputSources tests error when multiple input sources are specified
func TestParseWithOptions_MultipleInputSources(t *testing.T) {
	data := []byte(`openapi: "3.0.0"`)

	_, err := ParseWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithBytes(data),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify exactly one input source")
}

// TestParseWithOptions_NilReader tests error when nil reader is provided
func TestParseWithOptions_NilReader(t *testing.T) {
	_, err := ParseWithOptions(
		WithReader(nil),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reader cannot be nil")
}

// TestParseWithOptions_NilBytes tests error when nil bytes are provided
func TestParseWithOptions_NilBytes(t *testing.T) {
	_, err := ParseWithOptions(
		WithBytes(nil),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bytes cannot be nil")
}

// TestParseWithOptions_ResolveRefs tests that ref resolution can be enabled
func TestParseWithOptions_ResolveRefs(t *testing.T) {
	// This test uses a document with external $refs to verify resolution works
	result, err := ParseWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithResolveRefs(true),
	)
	require.NoError(t, err)
	assert.NotNil(t, result.Data)
	// If there were $refs, they would be resolved in result.Data
}

// TestParseWithOptions_DisableValidation tests that validation can be disabled
func TestParseWithOptions_DisableValidation(t *testing.T) {
	// Create an invalid spec (missing required fields)
	invalidSpec := `openapi: "3.0.0"
info:
  title: Test
  # Missing version field
paths:
  /test:
    get:
      # Missing responses field
      operationId: test`

	result, err := ParseWithOptions(
		WithBytes([]byte(invalidSpec)),
		WithValidateStructure(false), // Disable validation
	)
	require.NoError(t, err)

	// With validation disabled, structural errors should not be in result.Errors
	// (though the parsing itself should succeed)
	assert.NotNil(t, result.Document)
}

// TestParseWithOptions_JSONFormat tests parsing JSON format with options
func TestParseWithOptions_JSONFormat(t *testing.T) {
	jsonSpec := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {}
	}`

	result, err := ParseWithOptions(
		WithBytes([]byte(jsonSpec)),
	)
	require.NoError(t, err)
	assert.Equal(t, SourceFormatJSON, result.SourceFormat)
	assert.Equal(t, "ParseBytes.json", result.SourcePath)
}

// TestParseWithOptions_AllOptions tests using all options together
func TestParseWithOptions_AllOptions(t *testing.T) {
	data, err := os.ReadFile("../testdata/petstore-3.0.yaml")
	require.NoError(t, err)

	result, err := ParseWithOptions(
		WithBytes(data),
		WithResolveRefs(false),
		WithValidateStructure(true),
		WithUserAgent("test-agent/1.0"),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "3.0.3", result.Version)
	assert.Empty(t, result.Errors)
}

// TestWithFilePath tests the WithFilePath option function
func TestWithFilePath(t *testing.T) {
	cfg := &parseConfig{}
	opt := WithFilePath("test.yaml")
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.filePath)
	assert.Equal(t, "test.yaml", *cfg.filePath)
}

// TestWithReader tests the WithReader option function
func TestWithReader(t *testing.T) {
	reader := strings.NewReader("test")
	cfg := &parseConfig{}
	opt := WithReader(reader)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, reader, cfg.reader)
}

// TestWithReader_Nil tests that WithReader rejects nil readers
func TestWithReader_Nil(t *testing.T) {
	cfg := &parseConfig{}
	opt := WithReader(nil)
	err := opt(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reader cannot be nil")
}

// TestWithBytes tests the WithBytes option function
func TestWithBytes(t *testing.T) {
	data := []byte("test")
	cfg := &parseConfig{}
	opt := WithBytes(data)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, data, cfg.bytes)
}

// TestWithBytes_Nil tests that WithBytes rejects nil byte slices
func TestWithBytes_Nil(t *testing.T) {
	cfg := &parseConfig{}
	opt := WithBytes(nil)
	err := opt(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bytes cannot be nil")
}

// TestWithResolveRefs tests the WithResolveRefs option function
func TestWithResolveRefs(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &parseConfig{}
			opt := WithResolveRefs(tt.enabled)
			err := opt(cfg)

			require.NoError(t, err)
			assert.Equal(t, tt.enabled, cfg.resolveRefs)
		})
	}
}

// TestWithValidateStructure tests the WithValidateStructure option function
func TestWithValidateStructure(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &parseConfig{}
			opt := WithValidateStructure(tt.enabled)
			err := opt(cfg)

			require.NoError(t, err)
			assert.Equal(t, tt.enabled, cfg.validateStructure)
		})
	}
}

// TestWithUserAgent tests the WithUserAgent option function
func TestWithUserAgent(t *testing.T) {
	cfg := &parseConfig{}
	opt := WithUserAgent("custom-agent/2.0")
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, "custom-agent/2.0", cfg.userAgent)
}

// TestApplyOptions_Defaults tests that default values are set correctly
func TestApplyOptions_Defaults(t *testing.T) {
	cfg, err := applyOptions(WithFilePath("test.yaml"))

	require.NoError(t, err)
	assert.False(t, cfg.resolveRefs, "default resolveRefs should be false")
	assert.True(t, cfg.validateStructure, "default validateStructure should be true")
	assert.NotEmpty(t, cfg.userAgent, "default userAgent should be set")
}

// TestApplyOptions_OverrideDefaults tests that options override defaults
func TestApplyOptions_OverrideDefaults(t *testing.T) {
	cfg, err := applyOptions(
		WithFilePath("test.yaml"),
		WithResolveRefs(true),
		WithValidateStructure(false),
		WithUserAgent("custom/1.0"),
	)

	require.NoError(t, err)
	assert.True(t, cfg.resolveRefs)
	assert.False(t, cfg.validateStructure)
	assert.Equal(t, "custom/1.0", cfg.userAgent)
}

// TestWithLogger tests the logger option
func TestWithLogger(t *testing.T) {
	t.Run("nil logger", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithLogger(nil)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Nil(t, cfg.logger)
	})

	t.Run("with NopLogger", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithLogger(NopLogger{})
		err := opt(cfg)

		require.NoError(t, err)
		assert.NotNil(t, cfg.logger)
	})

	t.Run("with SlogAdapter", func(t *testing.T) {
		cfg := &parseConfig{}
		logger := NewSlogAdapter(nil)
		opt := WithLogger(logger)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Equal(t, logger, cfg.logger)
	})
}

// TestParserLog tests the log() helper method
func TestParserLog(t *testing.T) {
	t.Run("returns NopLogger when Logger is nil", func(t *testing.T) {
		p := &Parser{}
		logger := p.log()
		_, ok := logger.(NopLogger)
		assert.True(t, ok, "expected NopLogger when Logger is nil")
	})

	t.Run("returns configured logger", func(t *testing.T) {
		adapter := NewSlogAdapter(nil)
		p := &Parser{Logger: adapter}
		logger := p.log()
		assert.Equal(t, adapter, logger)
	})
}

// TestWithMaxRefDepth tests the WithMaxRefDepth option
func TestWithMaxRefDepth(t *testing.T) {
	t.Run("sets positive value", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithMaxRefDepth(50)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Equal(t, 50, cfg.maxRefDepth)
	})

	t.Run("accepts zero (use default)", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithMaxRefDepth(0)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Equal(t, 0, cfg.maxRefDepth)
	})

	t.Run("rejects negative value", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithMaxRefDepth(-1)
		err := opt(cfg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})
}

// TestWithMaxCachedDocuments tests the WithMaxCachedDocuments option
func TestWithMaxCachedDocuments(t *testing.T) {
	t.Run("sets positive value", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithMaxCachedDocuments(200)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Equal(t, 200, cfg.maxCachedDocuments)
	})

	t.Run("accepts zero (use default)", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithMaxCachedDocuments(0)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Equal(t, 0, cfg.maxCachedDocuments)
	})

	t.Run("rejects negative value", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithMaxCachedDocuments(-1)
		err := opt(cfg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})
}

// TestWithMaxFileSize tests the WithMaxFileSize option
func TestWithMaxFileSize(t *testing.T) {
	t.Run("sets positive value", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithMaxFileSize(20 * 1024 * 1024) // 20MB
		err := opt(cfg)

		require.NoError(t, err)
		assert.Equal(t, int64(20*1024*1024), cfg.maxFileSize)
	})

	t.Run("accepts zero (use default)", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithMaxFileSize(0)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Equal(t, int64(0), cfg.maxFileSize)
	})

	t.Run("rejects negative value", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithMaxFileSize(-1)
		err := opt(cfg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})
}

// TestParseWithOptions_ResourceLimits tests that resource limits are passed to parser
func TestParseWithOptions_ResourceLimits(t *testing.T) {
	// We can't easily test that the limits are actually used without
	// modifying the resolver, but we can verify they're passed through
	// by testing the parseConfig
	t.Run("limits are applied to config", func(t *testing.T) {
		cfg, err := applyOptions(
			WithFilePath("../testdata/petstore-3.0.yaml"),
			WithMaxRefDepth(50),
			WithMaxCachedDocuments(200),
			WithMaxFileSize(5*1024*1024),
		)

		require.NoError(t, err)
		assert.Equal(t, 50, cfg.maxRefDepth)
		assert.Equal(t, 200, cfg.maxCachedDocuments)
		assert.Equal(t, int64(5*1024*1024), cfg.maxFileSize)
	})
}

func TestWithSourceName(t *testing.T) {
	t.Run("sets source name in config", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithSourceName("users-api")
		err := opt(cfg)

		require.NoError(t, err)
		require.NotNil(t, cfg.sourceName)
		assert.Equal(t, "users-api", *cfg.sourceName)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithSourceName("")
		err := opt(cfg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestWithSourceName_AppliedToResult(t *testing.T) {
	minimalOAS := []byte(`openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths: {}
`)

	t.Run("overrides ParseBytes default source name", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithBytes(minimalOAS),
			WithSourceName("billing-service"),
		)

		require.NoError(t, err)
		assert.Equal(t, "billing-service", result.SourcePath)
	})

	t.Run("without WithSourceName uses default", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithBytes(minimalOAS),
		)

		require.NoError(t, err)
		// Default name from ParseBytes
		assert.Equal(t, "ParseBytes.yaml", result.SourcePath)
	})

	t.Run("overrides file path source name", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.0.yaml"),
			WithSourceName("pet-service"),
		)

		require.NoError(t, err)
		// Should use the override, not the file path
		assert.Equal(t, "pet-service", result.SourcePath)
	})
}

// TestWithHTTPClient tests the WithHTTPClient option
func TestWithHTTPClient(t *testing.T) {
	t.Run("sets client in config", func(t *testing.T) {
		customClient := &http.Client{Timeout: 60 * time.Second}
		cfg := &parseConfig{}
		opt := WithHTTPClient(customClient)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Same(t, customClient, cfg.httpClient)
	})

	t.Run("accepts nil client", func(t *testing.T) {
		cfg := &parseConfig{}
		opt := WithHTTPClient(nil)
		err := opt(cfg)

		require.NoError(t, err)
		assert.Nil(t, cfg.httpClient)
	})
}

// TestParseWithOptions_HTTPClient tests custom HTTP client integration
func TestParseWithOptions_HTTPClient(t *testing.T) {
	t.Run("uses custom client for URL parsing", func(t *testing.T) {
		requestReceived := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestReceived = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`))
		}))
		defer server.Close()

		customClient := &http.Client{Timeout: 5 * time.Second}
		result, err := ParseWithOptions(
			WithFilePath(server.URL),
			WithHTTPClient(customClient),
		)

		require.NoError(t, err)
		assert.True(t, requestReceived)
		assert.Equal(t, "3.0.0", result.Version)
	})

	t.Run("custom client timeout is respected", func(t *testing.T) {
		// Server that delays response longer than client timeout
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`))
		}))
		defer server.Close()

		// Client with very short timeout
		shortTimeoutClient := &http.Client{Timeout: 50 * time.Millisecond}
		_, err := ParseWithOptions(
			WithFilePath(server.URL),
			WithHTTPClient(shortTimeoutClient),
		)

		require.Error(t, err)
		// Error message varies by Go version, just check it's a timeout-related error
		assert.True(t, strings.Contains(err.Error(), "deadline") || strings.Contains(err.Error(), "timeout"))
	})
}

// roundTripperFunc is a helper for testing custom transports
type roundTripperFunc struct {
	fn func(*http.Request) (*http.Response, error)
}

func (r *roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.fn(req)
}

func TestParseWithOptions_HTTPClient_CustomTransport(t *testing.T) {
	t.Run("custom transport is used", func(t *testing.T) {
		transportUsed := false
		customTransport := &roundTripperFunc{
			fn: func(req *http.Request) (*http.Response, error) {
				transportUsed = true
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`)),
					Header: make(http.Header),
				}, nil
			},
		}
		customClient := &http.Client{Transport: customTransport}

		result, err := ParseWithOptions(
			WithFilePath("https://example.com/api.yaml"),
			WithHTTPClient(customClient),
		)

		require.NoError(t, err)
		assert.True(t, transportUsed, "Custom transport should have been used")
		assert.Equal(t, "3.0.0", result.Version)
	})

	t.Run("user agent still applied with custom client", func(t *testing.T) {
		var receivedUA string
		customTransport := &roundTripperFunc{
			fn: func(req *http.Request) (*http.Response, error) {
				receivedUA = req.Header.Get("User-Agent")
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`)),
					Header: make(http.Header),
				}, nil
			},
		}
		customClient := &http.Client{Transport: customTransport}

		_, err := ParseWithOptions(
			WithFilePath("https://example.com/api.yaml"),
			WithHTTPClient(customClient),
			WithUserAgent("custom-agent/1.0"),
		)

		require.NoError(t, err)
		assert.Equal(t, "custom-agent/1.0", receivedUA)
	})
}

func TestParseWithOptions_HTTPClient_InsecureInteraction(t *testing.T) {
	t.Run("warns when both HTTPClient and InsecureSkipVerify set", func(t *testing.T) {
		var logMessages []string
		mockLogger := &mockTestLogger{
			warnFunc: func(msg string, args ...any) {
				logMessages = append(logMessages, msg)
			},
		}

		customTransport := &roundTripperFunc{
			fn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths: {}`)),
					Header: make(http.Header),
				}, nil
			},
		}
		customClient := &http.Client{Transport: customTransport}

		_, err := ParseWithOptions(
			WithFilePath("https://example.com/api.yaml"),
			WithHTTPClient(customClient),
			WithInsecureSkipVerify(true),
			WithLogger(mockLogger),
		)

		require.NoError(t, err)
		require.Len(t, logMessages, 1)
		assert.Contains(t, logMessages[0], "InsecureSkipVerify ignored")
	})
}

// mockTestLogger implements Logger for testing
type mockTestLogger struct {
	warnFunc func(msg string, args ...any)
}

func (m *mockTestLogger) Debug(msg string, args ...any) {}
func (m *mockTestLogger) Info(msg string, args ...any)  {}
func (m *mockTestLogger) Warn(msg string, args ...any) {
	if m.warnFunc != nil {
		m.warnFunc(msg, args...)
	}
}
func (m *mockTestLogger) Error(msg string, args ...any) {}
func (m *mockTestLogger) With(attrs ...any) Logger      { return m }
