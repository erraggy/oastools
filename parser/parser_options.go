package parser

import (
	"fmt"
	"io"
	"net/http"

	"github.com/erraggy/oastools"

	"github.com/erraggy/oastools/internal/options"
)

// Option is a function that configures a parse operation
type Option func(*parseConfig) error

// parseConfig holds configuration for a parse operation
type parseConfig struct {
	// Input source (exactly one must be set)
	filePath *string
	reader   io.Reader
	bytes    []byte

	// Configuration options
	resolveRefs        bool
	resolveHTTPRefs    bool
	insecureSkipVerify bool
	validateStructure  bool
	userAgent          string
	httpClient         *http.Client
	logger             Logger

	// Resource limits (0 means use default)
	maxRefDepth        int
	maxCachedDocuments int
	maxFileSize        int64

	// Source map building
	buildSourceMap bool

	// Order preservation
	preserveOrder bool

	// Source identification
	sourceName *string // Override SourcePath in the result
}

// ParseWithOptions parses an OpenAPI specification using functional options.
// This provides a flexible, extensible API that combines input source selection
// and configuration in a single function call.
//
// Example:
//
//	result, err := parser.ParseWithOptions(
//	    parser.WithFilePath("openapi.yaml"),
//	    parser.WithResolveRefs(true),
//	)
func ParseWithOptions(opts ...Option) (*ParseResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("parser: invalid options: %w", err)
	}

	p := &Parser{
		ResolveRefs:        cfg.resolveRefs,
		ResolveHTTPRefs:    cfg.resolveHTTPRefs,
		InsecureSkipVerify: cfg.insecureSkipVerify,
		ValidateStructure:  cfg.validateStructure,
		UserAgent:          cfg.userAgent,
		HTTPClient:         cfg.httpClient,
		Logger:             cfg.logger,
		MaxRefDepth:        cfg.maxRefDepth,
		MaxCachedDocuments: cfg.maxCachedDocuments,
		MaxFileSize:        cfg.maxFileSize,
		BuildSourceMap:     cfg.buildSourceMap,
		PreserveOrder:      cfg.preserveOrder,
	}

	// Route to appropriate parsing method based on input source
	var result *ParseResult
	var parseErr error
	switch {
	case cfg.filePath != nil:
		result, parseErr = p.Parse(*cfg.filePath)
	case cfg.reader != nil:
		result, parseErr = p.ParseReader(cfg.reader)
	case cfg.bytes != nil:
		result, parseErr = p.ParseBytes(cfg.bytes)
	default:
		// Should never reach here due to validation in applyOptions
		return nil, fmt.Errorf("parser: no input source specified")
	}

	if parseErr != nil {
		return result, parseErr
	}

	// Apply source name override if specified
	if result != nil && cfg.sourceName != nil {
		result.SourcePath = *cfg.sourceName
	}

	return result, nil
}

// applyOptions applies option functions and validates configuration
func applyOptions(opts ...Option) (*parseConfig, error) {
	cfg := &parseConfig{
		// Set defaults to match existing behavior
		resolveRefs:       false,
		validateStructure: true,
		userAgent:         oastools.UserAgent(),
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate exactly one input source is specified
	if err := options.ValidateSingleInputSource(
		"parser: must specify an input source (use WithFilePath, WithReader, or WithBytes)",
		"parser: must specify exactly one input source",
		cfg.filePath != nil, cfg.reader != nil, cfg.bytes != nil,
	); err != nil {
		return nil, err
	}

	return cfg, nil
}

// WithFilePath specifies a file path or URL as the input source
func WithFilePath(path string) Option {
	return func(cfg *parseConfig) error {
		cfg.filePath = &path
		return nil
	}
}

// WithReader specifies an io.Reader as the input source
func WithReader(r io.Reader) Option {
	return func(cfg *parseConfig) error {
		if r == nil {
			return fmt.Errorf("parser: reader cannot be nil")
		}
		cfg.reader = r
		return nil
	}
}

// WithBytes specifies a byte slice as the input source
func WithBytes(data []byte) Option {
	return func(cfg *parseConfig) error {
		if data == nil {
			return fmt.Errorf("parser: bytes cannot be nil")
		}
		cfg.bytes = data
		return nil
	}
}

// WithResolveRefs enables or disables reference resolution ($ref)
// Default: false
func WithResolveRefs(enabled bool) Option {
	return func(cfg *parseConfig) error {
		cfg.resolveRefs = enabled
		return nil
	}
}

// WithValidateStructure enables or disables basic structure validation
// Default: true
func WithValidateStructure(enabled bool) Option {
	return func(cfg *parseConfig) error {
		cfg.validateStructure = enabled
		return nil
	}
}

// WithUserAgent sets the User-Agent string for HTTP requests
// Default: "oastools/vX.Y.Z"
func WithUserAgent(ua string) Option {
	return func(cfg *parseConfig) error {
		cfg.userAgent = ua
		return nil
	}
}

// WithHTTPClient sets a custom HTTP client for fetching URLs.
// When set, the client is used as-is for all HTTP requests.
// The InsecureSkipVerify option is ignored when a custom client is provided
// (configure TLS settings on your client's transport instead).
//
// If the client is nil, this option has no effect (default client is used).
//
// Example with custom timeout:
//
//	client := &http.Client{Timeout: 60 * time.Second}
//	result, err := parser.ParseWithOptions(
//	    parser.WithFilePath("https://example.com/api.yaml"),
//	    parser.WithHTTPClient(client),
//	)
//
// Example with proxy:
//
//	proxyURL, _ := url.Parse("http://proxy.example.com:8080")
//	client := &http.Client{
//	    Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
//	}
//	result, err := parser.ParseWithOptions(
//	    parser.WithFilePath("https://internal.corp/api.yaml"),
//	    parser.WithHTTPClient(client),
//	)
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *parseConfig) error {
		cfg.httpClient = client
		return nil
	}
}

// WithResolveHTTPRefs enables resolution of HTTP/HTTPS $ref URLs
// This is disabled by default for security (SSRF protection)
// Must be explicitly enabled when parsing specifications with HTTP refs
// Note: This option only takes effect when ResolveRefs is also enabled
func WithResolveHTTPRefs(enabled bool) Option {
	return func(cfg *parseConfig) error {
		cfg.resolveHTTPRefs = enabled
		return nil
	}
}

// WithInsecureSkipVerify disables TLS certificate verification for HTTPS refs
// Use with caution - only enable for testing or internal servers with self-signed certs
// Note: This option only takes effect when ResolveHTTPRefs is also enabled
func WithInsecureSkipVerify(enabled bool) Option {
	return func(cfg *parseConfig) error {
		cfg.insecureSkipVerify = enabled
		return nil
	}
}

// WithLogger sets a structured logger for debug output during parsing.
// By default, no logging is performed (nil logger).
//
// The logger interface is compatible with log/slog, zap, and zerolog.
// Use NewSlogAdapter to wrap a *slog.Logger.
//
// Example:
//
//	logger := parser.NewSlogAdapter(slog.Default())
//	result, err := parser.ParseWithOptions(
//	    parser.WithFilePath("api.yaml"),
//	    parser.WithLogger(logger),
//	)
func WithLogger(l Logger) Option {
	return func(cfg *parseConfig) error {
		cfg.logger = l
		return nil
	}
}

// WithMaxRefDepth sets the maximum depth for resolving nested $ref pointers.
// This prevents stack overflow from deeply nested (but non-circular) references.
// A value of 0 means use the default (100).
// Returns an error if depth is negative.
func WithMaxRefDepth(depth int) Option {
	return func(cfg *parseConfig) error {
		if depth < 0 {
			return fmt.Errorf("parser: maxRefDepth cannot be negative")
		}
		cfg.maxRefDepth = depth
		return nil
	}
}

// WithMaxCachedDocuments sets the maximum number of external documents to cache
// during reference resolution. This prevents memory exhaustion from documents
// with many external references.
// A value of 0 means use the default (100).
// Returns an error if count is negative.
func WithMaxCachedDocuments(count int) Option {
	return func(cfg *parseConfig) error {
		if count < 0 {
			return fmt.Errorf("parser: maxCachedDocuments cannot be negative")
		}
		cfg.maxCachedDocuments = count
		return nil
	}
}

// WithMaxFileSize sets the maximum file size in bytes for external reference files.
// This prevents resource exhaustion from loading arbitrarily large files.
// A value of 0 means use the default (10MB).
// Returns an error if size is negative.
func WithMaxFileSize(size int64) Option {
	return func(cfg *parseConfig) error {
		if size < 0 {
			return fmt.Errorf("parser: maxFileSize cannot be negative")
		}
		cfg.maxFileSize = size
		return nil
	}
}

// WithSourceMap enables or disables source location tracking.
// When enabled, the ParseResult.SourceMap will contain line/column
// information for each JSON path in the document.
// Default: false
func WithSourceMap(enabled bool) Option {
	return func(cfg *parseConfig) error {
		cfg.buildSourceMap = enabled
		return nil
	}
}

// WithPreserveOrder enables order-preserving marshaling.
// When enabled, ParseResult stores the original yaml.Node structure,
// allowing MarshalOrderedJSON/MarshalOrderedYAML to emit fields
// in the same order as the source document.
//
// This is useful for:
//   - Hash-based caching where roundtrip identity matters
//   - Minimizing diffs when editing and re-serializing specs
//   - Maintaining human-friendly key ordering
//
// Default: false
//
// Example:
//
//	result, err := parser.ParseWithOptions(
//	    parser.WithFilePath("api.yaml"),
//	    parser.WithPreserveOrder(true),
//	)
//	orderedJSON, _ := result.MarshalOrderedJSON()
func WithPreserveOrder(enabled bool) Option {
	return func(cfg *parseConfig) error {
		cfg.preserveOrder = enabled
		return nil
	}
}

// WithSourceName specifies a meaningful name for the source document.
// This is particularly useful when parsing from bytes or reader, where
// the default names ("ParseBytes.yaml", "ParseReader.yaml") are not descriptive.
// The name is used in error messages, collision reports when joining, and other
// diagnostic output.
//
// Example:
//
//	result, err := parser.ParseWithOptions(
//	    parser.WithBytes(data),
//	    parser.WithSourceName("users-api"),
//	)
//
// This helps when joining multiple pre-parsed documents:
//
//	// Without WithSourceName, collision reports show "ParseBytes.yaml vs ParseBytes.yaml"
//	// With WithSourceName, collision reports show "users-api vs billing-api"
func WithSourceName(name string) Option {
	return func(cfg *parseConfig) error {
		if name == "" {
			return fmt.Errorf("parser: source name cannot be empty")
		}
		cfg.sourceName = &name
		return nil
	}
}
