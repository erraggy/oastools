package parser

//go:generate go run ../internal/codegen/deepcopy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/erraggy/oastools"
	"go.yaml.in/yaml/v4"

	"github.com/erraggy/oastools/internal/httputil"
)

// Parser handles OpenAPI specification parsing
type Parser struct {
	// ResolveRefs determines whether to resolve $ref references
	ResolveRefs bool
	// ResolveHTTPRefs determines whether to resolve HTTP/HTTPS $ref URLs
	// This is disabled by default for security (SSRF protection)
	// Must be explicitly enabled when parsing specifications with HTTP refs
	ResolveHTTPRefs bool
	// InsecureSkipVerify disables TLS certificate verification for HTTP refs
	// Use with caution - only enable for testing or internal servers with self-signed certs
	InsecureSkipVerify bool
	// ValidateStructure determines whether to perform basic structure validation
	ValidateStructure bool
	// UserAgent is the User-Agent string used when fetching URLs
	// Defaults to "oastools" if not set
	UserAgent string
	// Logger is the structured logger for debug output
	// If nil, logging is disabled (default)
	Logger Logger

	// Resource limits (0 means use default)

	// MaxRefDepth is the maximum depth for resolving nested $ref pointers.
	// Default: 100
	MaxRefDepth int
	// MaxCachedDocuments is the maximum number of external documents to cache.
	// Default: 100
	MaxCachedDocuments int
	// MaxFileSize is the maximum file size in bytes for external references.
	// Default: 10MB
	MaxFileSize int64
}

// New creates a new Parser instance with default settings
func New() *Parser {
	return &Parser{
		ResolveRefs:       false,
		ValidateStructure: true,
		UserAgent:         oastools.UserAgent(),
	}
}

// log returns the configured logger, or a no-op logger if none is set.
func (p *Parser) log() Logger {
	if p.Logger != nil {
		return p.Logger
	}
	return NopLogger{}
}

// SourceFormat represents the format of the source OpenAPI specification file
type SourceFormat string

const (
	// SourceFormatYAML indicates the source was in YAML format
	SourceFormatYAML SourceFormat = "yaml"
	// SourceFormatJSON indicates the source was in JSON format
	SourceFormatJSON SourceFormat = "json"
	// SourceFormatUnknown indicates the source format could not be determined
	SourceFormatUnknown SourceFormat = "unknown"
)

// ParseResult contains the parsed OpenAPI specification and metadata.
// This structure provides both the raw parsed data and version-specific
// typed representations of the OpenAPI document.
//
// # Immutability
//
// While Go does not enforce immutability, callers should treat ParseResult as
// read-only after parsing. Modifying the returned document may lead to unexpected
// behavior if the document is cached or shared across multiple operations.
//
// For document modification use cases:
//   - Version conversion: Use the converter package
//   - Document merging: Use the joiner package
//   - Manual modification: Create a deep copy first using Copy() method
//
// Example of safe modification:
//
//	original, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//	modified := original.Copy()  // Deep copy
//	// Now safe to modify 'modified' without affecting 'original'
type ParseResult struct {
	// SourcePath is the document's input source path that it was read from.
	// Note: if the source was not a file path, this will be set to the name of the method
	// and end in '.yaml' or '.json' based on the detected format
	SourcePath string
	// SourceFormat is the format of the source file (JSON or YAML)
	SourceFormat SourceFormat
	// Version is the detected OAS version string (e.g., "2.0", "3.0.3", "3.1.0")
	Version string
	// Data contains the raw parsed data as a map, potentially with resolved $refs
	Data map[string]any
	// Document contains the version-specific parsed document:
	// - *OAS2Document for OpenAPI 2.0
	// - *OAS3Document for OpenAPI 3.x
	Document any
	// Errors contains any parsing or validation errors encountered
	Errors []error
	// Warnings contains non-fatal issues such as ref resolution failures
	Warnings []string
	// OASVersion is the enumerated version of the OpenAPI specification
	OASVersion OASVersion
	// LoadTime is the time taken to load the source data (file, URL, etc.)
	LoadTime time.Duration
	// SourceSize is the size of the source data in bytes
	SourceSize int64
	// Stats contains statistical information about the document
	Stats DocumentStats
}

// Copy creates a deep copy of the ParseResult, including all nested documents and data.
// This is useful when you need to modify a parsed document without affecting the original.
//
// The deep copy is performed using JSON marshaling and unmarshaling to ensure
// all nested structures and maps are properly copied.
//
// Example:
//
//	original, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//	modified := original.Copy()
//	// Modify the copy without affecting the original
//	if doc, ok := modified.Document.(*parser.OAS3Document); ok {
//	    doc.Info.Title = "Modified API"
//	}
func (pr *ParseResult) Copy() *ParseResult {
	if pr == nil {
		return nil
	}

	// Create a shallow copy of the result
	result := &ParseResult{
		SourcePath:   pr.SourcePath,
		SourceFormat: pr.SourceFormat,
		Version:      pr.Version,
		OASVersion:   pr.OASVersion,
		LoadTime:     pr.LoadTime,
		SourceSize:   pr.SourceSize,
		Stats:        pr.Stats, // DocumentStats is a value type, copied by value
	}

	// Deep copy the Document using generated DeepCopy methods
	switch doc := pr.Document.(type) {
	case *OAS2Document:
		result.Document = doc.DeepCopy()
	case *OAS3Document:
		result.Document = doc.DeepCopy()
	default:
		// Unknown document type, leave as nil
		result.Document = nil
	}

	// Deep copy the Data map
	if pr.Data != nil {
		result.Data = deepCopyExtensions(pr.Data)
	}

	// Deep copy errors slice
	if pr.Errors != nil {
		result.Errors = make([]error, len(pr.Errors))
		copy(result.Errors, pr.Errors)
	}

	// Deep copy warnings slice
	if pr.Warnings != nil {
		result.Warnings = make([]string, len(pr.Warnings))
		copy(result.Warnings, pr.Warnings)
	}

	return result
}

// FormatBytes formats a byte count into a human-readable string using binary units (KiB, MiB, etc.)
func FormatBytes(bytes int64) string {
	// Handle negative values
	if bytes < 0 {
		return fmt.Sprintf("%d B", bytes)
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit && exp < 5; n /= unit {
		div *= unit
		exp++
	}

	// Use proper binary unit notation (KiB, MiB, GiB, etc.)
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// detectFormatFromPath detects the source format from a file path
func detectFormatFromPath(path string) SourceFormat {
	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		return SourceFormatJSON
	case ".yaml", ".yml":
		return SourceFormatYAML
	default:
		return SourceFormatUnknown
	}
}

// detectFormatFromContent attempts to detect the format from the content bytes
// JSON typically starts with '{' or '[', while YAML does not
func detectFormatFromContent(data []byte) SourceFormat {
	// Trim leading whitespace
	trimmed := bytes.TrimLeft(data, " \t\n\r")

	if len(trimmed) == 0 {
		return SourceFormatUnknown
	}

	// JSON objects/arrays start with { or [
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return SourceFormatJSON
	}

	// Otherwise assume YAML (could be more sophisticated, but this covers most cases)
	return SourceFormatYAML
}

// isURL determines if the given path is a URL (http:// or https://)
func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// fetchURL fetches content from a URL and returns the bytes and Content-Type header
func (p *Parser) fetchURL(urlStr string) ([]byte, string, error) {
	// Create HTTP client with timeout
	// Configure TLS if InsecureSkipVerify is enabled
	var client *http.Client
	if p.InsecureSkipVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // User explicitly requested insecure mode
			},
		}
		client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	} else {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Create request
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, "", fmt.Errorf("parser: failed to create request: %w", err)
	}

	// Set user agent (use default if not set)
	userAgent := p.UserAgent
	if userAgent == "" {
		userAgent = oastools.UserAgent()
	}
	req.Header.Set("User-Agent", userAgent)

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("parser: failed to fetch URL: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("parser: HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("parser: failed to read response body: %w", err)
	}

	// Return data and Content-Type header
	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}

// detectFormatFromURL attempts to detect the format from a URL path and Content-Type header
func detectFormatFromURL(urlStr string, contentType string) SourceFormat {
	// First try to detect from URL path extension
	parsedURL, err := url.Parse(urlStr)
	if err == nil && parsedURL.Path != "" {
		format := detectFormatFromPath(parsedURL.Path)
		if format != SourceFormatUnknown {
			return format
		}
	}

	// Try to detect from Content-Type header
	if contentType != "" {
		contentType = strings.ToLower(contentType)
		// Remove charset and other parameters
		if idx := strings.Index(contentType, ";"); idx != -1 {
			contentType = contentType[:idx]
		}
		contentType = strings.TrimSpace(contentType)

		switch contentType {
		case "application/json":
			return SourceFormatJSON
		case "application/yaml", "application/x-yaml", "text/yaml", "text/x-yaml":
			return SourceFormatYAML
		}
	}

	return SourceFormatUnknown
}

// Parse parses an OpenAPI specification file or URL
// For URLs (http:// or https://), the content is fetched and parsed
// For local files, the file is read and parsed
func (p *Parser) Parse(specPath string) (*ParseResult, error) {
	var data []byte
	var err error
	var format SourceFormat
	var baseDir string
	var baseURL string
	var loadStart time.Time
	var loadTime time.Duration

	// Check if specPath is a URL
	if isURL(specPath) {
		// Fetch content from URL
		var contentType string
		loadStart = time.Now()
		data, contentType, err = p.fetchURL(specPath)
		loadTime = time.Since(loadStart)
		if err != nil {
			return nil, err
		}

		// For URLs, use current directory for local file refs
		// but store the URL for resolving relative HTTP refs
		baseDir = "."
		baseURL = specPath

		// Try to detect format from URL path and Content-Type header
		format = detectFormatFromURL(specPath, contentType)
	} else {
		// Read from file
		loadStart = time.Now()
		data, err = os.ReadFile(specPath)
		loadTime = time.Since(loadStart)
		if err != nil {
			return nil, fmt.Errorf("parser: failed to read file: %w", err)
		}

		// Get the directory of the spec file for resolving relative refs
		baseDir = filepath.Dir(specPath)
		// No base URL for local files
		baseURL = ""

		// Detect format from file extension
		format = detectFormatFromPath(specPath)
	}

	// Parse the data
	res, err := p.parseBytesWithBaseDirAndURL(data, baseDir, baseURL)
	if err != nil {
		return nil, err
	}

	// Set source path and format
	res.SourcePath = specPath
	res.LoadTime = loadTime
	res.SourceSize = int64(len(data))

	// If format was detected from path/URL, use it; otherwise use content-based detection
	if format != SourceFormatUnknown {
		res.SourceFormat = format
	} else if res.SourceFormat == SourceFormatUnknown {
		// Fallback to content-based detection if not already set
		res.SourceFormat = detectFormatFromContent(data)
	}

	return res, nil
}

// ParseReader parses an OpenAPI specification from an io.Reader
// Note: since there is no actual ParseResult.SourcePath, it will be set to: ParseReader.yaml or ParseReader.json
func (p *Parser) ParseReader(r io.Reader) (*ParseResult, error) {
	loadStart := time.Now()
	data, err := io.ReadAll(r)
	loadTime := time.Since(loadStart)
	if err != nil {
		return nil, fmt.Errorf("parser: failed to read data: %w", err)
	}
	res, err := p.ParseBytes(data)
	if err != nil {
		return nil, err
	}
	// Update timing and size info
	res.LoadTime = loadTime
	res.SourceSize = int64(len(data))
	// Update SourcePath to match detected format
	if res.SourceFormat == SourceFormatJSON {
		res.SourcePath = "ParseReader.json"
	} else {
		res.SourcePath = "ParseReader.yaml"
	}
	return res, nil
}

// ParseBytes parses an OpenAPI specification from a byte slice
// For external references to work, use Parse() with a file path instead
// Note: since there is no actual ParseResult.SourcePath, it will be set to: ParseBytes.yaml or ParseBytes.json
func (p *Parser) ParseBytes(data []byte) (*ParseResult, error) {
	res, err := p.parseBytesWithBaseDir(data, ".")
	if err != nil {
		return nil, err
	}
	// Detect format from content
	res.SourceFormat = detectFormatFromContent(data)
	// Set size (no load time since data is already in memory)
	res.SourceSize = int64(len(data))
	// Update SourcePath to match detected format
	if res.SourceFormat == SourceFormatJSON {
		res.SourcePath = "ParseBytes.json"
	} else {
		res.SourcePath = "ParseBytes.yaml"
	}
	return res, nil
}

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
	logger             Logger

	// Resource limits (0 means use default)
	maxRefDepth        int
	maxCachedDocuments int
	maxFileSize        int64
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
		Logger:             cfg.logger,
		MaxRefDepth:        cfg.maxRefDepth,
		MaxCachedDocuments: cfg.maxCachedDocuments,
		MaxFileSize:        cfg.maxFileSize,
	}

	// Route to appropriate parsing method based on input source
	if cfg.filePath != nil {
		return p.Parse(*cfg.filePath)
	}
	if cfg.reader != nil {
		return p.ParseReader(cfg.reader)
	}
	if cfg.bytes != nil {
		return p.ParseBytes(cfg.bytes)
	}

	// Should never reach here due to validation in applyOptions
	return nil, fmt.Errorf("parser: no input source specified")
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
	sourceCount := 0
	if cfg.filePath != nil {
		sourceCount++
	}
	if cfg.reader != nil {
		sourceCount++
	}
	if cfg.bytes != nil {
		sourceCount++
	}

	if sourceCount == 0 {
		return nil, fmt.Errorf("must specify an input source (use WithFilePath, WithReader, or WithBytes)")
	}
	if sourceCount > 1 {
		return nil, fmt.Errorf("must specify exactly one input source")
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
			return fmt.Errorf("reader cannot be nil")
		}
		cfg.reader = r
		return nil
	}
}

// WithBytes specifies a byte slice as the input source
func WithBytes(data []byte) Option {
	return func(cfg *parseConfig) error {
		if data == nil {
			return fmt.Errorf("bytes cannot be nil")
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
			return fmt.Errorf("maxRefDepth cannot be negative")
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
			return fmt.Errorf("maxCachedDocuments cannot be negative")
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
			return fmt.Errorf("maxFileSize cannot be negative")
		}
		cfg.maxFileSize = size
		return nil
	}
}

// parseBytesWithBaseDir parses data with a specified base directory for ref resolution
func (p *Parser) parseBytesWithBaseDir(data []byte, baseDir string) (*ParseResult, error) {
	return p.parseBytesWithBaseDirAndURL(data, baseDir, "")
}

// parseBytesWithBaseDirAndURL parses data with base directory and optional base URL for HTTP refs
func (p *Parser) parseBytesWithBaseDirAndURL(data []byte, baseDir, baseURL string) (*ParseResult, error) {
	result := &ParseResult{
		Errors:   make([]error, 0),
		Warnings: make([]string, 0),
	}

	// First pass: parse to generic map to detect OAS version
	var rawData map[string]any
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("parser: failed to parse YAML/JSON: %w", err)
	}

	// Resolve references if enabled (before semver-specific parsing)
	var hasCircularRefs bool
	if p.ResolveRefs {
		var resolver *RefResolver
		if p.ResolveHTTPRefs {
			// Use HTTP-enabled resolver when HTTP refs are enabled
			// This supports both: local files with absolute HTTP $refs, and
			// HTTP-sourced specs with relative $refs (resolved against baseURL)
			resolver = NewRefResolverWithHTTP(baseDir, baseURL, p.fetchURL)
		} else {
			resolver = NewRefResolver(baseDir)
		}
		if err := resolver.ResolveAllRefs(rawData); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("ref resolution warning: %v", err))
		}
		hasCircularRefs = resolver.HasCircularRefs()
	}

	result.Data = rawData

	// Detect semver
	version, err := p.detectVersion(rawData)
	if err != nil {
		return nil, fmt.Errorf("parser: failed to detect OAS version: %w", err)
	}
	result.Version = version

	// Prepare data for version-specific parsing
	// Only re-marshal if we resolved refs AND no circular refs were detected
	//
	// Performance trade-off: When ResolveRefs is enabled, we must re-marshal the
	// rawData map after reference resolution to ensure the resolved content is
	// available to the version-specific parsers. This adds overhead (especially
	// for large documents), but is necessary for correct reference resolution.
	// When ResolveRefs is disabled, we skip this step and use the original data.
	//
	// IMPORTANT: If circular references were detected, we MUST skip re-marshaling
	// because yaml.Marshal will enter an infinite loop on circular Go structures.
	var parseData []byte
	if p.ResolveRefs && !hasCircularRefs {
		// Re-marshal the data with resolved refs
		parseData, err = yaml.Marshal(rawData)
		if err != nil {
			// If marshaling fails, fall back to using the original data.
			parseData = data
			result.Warnings = append(result.Warnings, fmt.Sprintf("Warning: Could not re-marshal document after reference resolution: %v. Using original document structure. Some references may not be fully resolved.", err))
		}
	} else if hasCircularRefs {
		// Use original data when circular refs detected to avoid infinite loop in yaml.Marshal
		parseData = data
		result.Warnings = append(result.Warnings, "Warning: Circular references detected. Using original document structure. Some references may not be fully resolved.")
	} else {
		// Use original data directly
		parseData = data
	}

	// Parse to semver-specific structure
	doc, oasVersion, err := p.parseVersionSpecific(parseData, version)
	if err != nil {
		return nil, err
	}
	result.Document = doc
	result.OASVersion = oasVersion

	// Validate structure if enabled
	if p.ValidateStructure {
		validationErrors := p.validateStructure(result)
		result.Errors = append(result.Errors, validationErrors...)
	}

	// Calculate document statistics
	result.Stats = GetDocumentStats(result.Document)

	return result, nil
}

// detectVersion determines the OAS semver from the raw data
func (p *Parser) detectVersion(data map[string]any) (string, error) {
	// Check for OAS 2.0 (Swagger)
	if swagger, ok := data["swagger"].(string); ok {
		return swagger, nil
	}

	// Check for OAS 3.x
	if openapi, ok := data["openapi"].(string); ok {
		return openapi, nil
	}

	// Neither field was found - provide helpful error message
	return "", fmt.Errorf("parser: unable to detect OpenAPI version: document must contain either 'swagger: \"2.0\"' (for OAS 2.0) or 'openapi: \"3.x.x\"' (for OAS 3.x) at the root level")
}

// parseSemVer parses a semver string into a semantic version
func parseSemVer(v string) (*version, error) {
	return parseVersion(v)
}

// versionInRangeExclusive checks if a semver string is within the specified range: minVersion <= v < maxVersion
// If maxVersion is empty string, no upper bound is enforced (v >= minVersion)
func versionInRangeExclusive(v, minVersion, maxVersion string) bool {
	ver, err := parseSemVer(v)
	if err != nil {
		return false
	}

	min, err := parseSemVer(minVersion)
	if err != nil {
		return false
	}

	// Check lower bound
	if !ver.greaterThanOrEqual(min) {
		return false
	}

	// If maxVersion is empty, no upper bound
	if maxVersion == "" {
		return true
	}

	max, err := parseSemVer(maxVersion)
	if err != nil {
		return false
	}
	return ver.lessThan(max)
}

// parseVersionSpecific parses the data into a semver-specific structure
func (p *Parser) parseVersionSpecific(data []byte, version string) (any, OASVersion, error) {
	v, ok := ParseVersion(version)
	if !ok {
		return nil, 0, fmt.Errorf("parser: invalid OAS version: %s", version)
	}
	switch v {
	case OASVersion20:
		var doc OAS2Document
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, 0, fmt.Errorf("parser: failed to parse OAS 2.0 document structure: %w", err)
		}
		doc.OASVersion = v
		return &doc, v, nil

	case OASVersion300, OASVersion301, OASVersion302, OASVersion303, OASVersion304, OASVersion310, OASVersion311, OASVersion312, OASVersion320:
		var doc OAS3Document
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, 0, fmt.Errorf("parser: failed to parse OAS %s document structure: %w", version, err)
		}
		doc.OASVersion = v
		return &doc, v, nil

	default:
		return nil, 0, fmt.Errorf("parser: unsupported OpenAPI version: %s (only 2.0 and 3.x versions are supported)", version)
	}
}

// validateStructure performs basic structure validation
func (p *Parser) validateStructure(result *ParseResult) []error {
	errors := make([]error, 0)

	// Validate required fields based on semver
	switch {
	case result.OASVersion == OASVersion20:
		doc, ok := result.Document.(*OAS2Document)
		if !ok {
			errors = append(errors, fmt.Errorf("parser: internal error: document type mismatch for OAS 2.0 (expected *OAS2Document, got %T)", result.Document))
			return errors
		}
		errors = append(errors, p.validateOAS2(doc)...)

	case result.OASVersion.IsValid():
		doc, ok := result.Document.(*OAS3Document)
		if !ok {
			errors = append(errors, fmt.Errorf("parser: internal error: document type mismatch for OAS 3.x (expected *OAS3Document, got %T)", result.Document))
			return errors
		}
		errors = append(errors, p.validateOAS3(doc)...)

	default:
		errors = append(errors, fmt.Errorf("parser: unsupported OpenAPI version: %s (only versions 2.0 and 3.x are supported)", result.Version))
	}

	return errors
}

// validateOAS2 validates an OAS 2.0 document
func (p *Parser) validateOAS2(doc *OAS2Document) []error {
	errors := make([]error, 0)

	// Validate swagger version field
	if doc.Swagger == "" {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required root field 'swagger': must be set to \"2.0\""))
	} else if doc.Swagger != "2.0" {
		errors = append(errors, fmt.Errorf("oas 2.0: invalid 'swagger' field value: expected \"2.0\", got \"%s\"", doc.Swagger))
	}

	// Validate info object
	errors = append(errors, p.validateOAS2Info(doc.Info)...)

	// Validate paths object
	if doc.Paths == nil {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required root field 'paths': Paths object is required per spec (https://spec.openapis.org/oas/v2.0.html#pathsObject)"))
	} else {
		errors = append(errors, p.validateOAS2Paths(doc.Paths)...)
	}

	return errors
}

func (p *Parser) validateOAS2Info(info *Info) []error {
	errors := make([]error, 0)
	if info == nil {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required root field 'info': Info object is required per spec (https://spec.openapis.org/oas/v2.0.html#infoObject)"))
	} else {
		if info.Title == "" {
			errors = append(errors, fmt.Errorf("oas 2.0: missing required field 'info.title': Info object must have a title per spec"))
		}
		if info.Version == "" {
			errors = append(errors, fmt.Errorf("oas 2.0: missing required field 'info.version': Info object must have a version string per spec"))
		}
	}
	return errors
}

func (p *Parser) validateOAS2Paths(paths map[string]*PathItem) []error {
	errors := make([]error, 0)
	operationIDs := make(map[string]string)

	for pathPattern, pathItem := range paths {
		if pathItem == nil {
			continue
		}

		// Validate path pattern
		if pathPattern != "" && pathPattern[0] != '/' {
			errors = append(errors, fmt.Errorf("oas 2.0: invalid path pattern 'paths.%s': path must begin with '/'", pathPattern))
		}

		// Check all operations in this path
		errors = append(errors, p.validateOAS2PathItem(pathItem, pathPattern, operationIDs)...)
	}

	return errors
}

func (p *Parser) validateOAS2PathItem(pathItem *PathItem, pathPattern string, operationIDs map[string]string) []error {
	errors := make([]error, 0)
	operations := GetOAS2Operations(pathItem)

	for method, op := range operations {
		if op == nil {
			continue
		}

		opPath := fmt.Sprintf("paths.%s.%s", pathPattern, method)
		errors = append(errors, p.validateOAS2Operation(op, opPath, operationIDs)...)
	}

	return errors
}

func (p *Parser) validateOAS2Operation(op *Operation, opPath string, operationIDs map[string]string) []error {
	errors := make([]error, 0)

	// Validate operationId uniqueness
	if op.OperationID != "" {
		if existingPath, exists := operationIDs[op.OperationID]; exists {
			errors = append(errors, fmt.Errorf("oas 2.0: duplicate operationId '%s' at '%s': previously defined at '%s'",
				op.OperationID, opPath, existingPath))
		} else {
			operationIDs[op.OperationID] = opPath
		}
	}

	// Validate responses object exists
	if op.Responses == nil {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required field '%s.responses': Operation must have a responses object", opPath))
	} else {
		// Validate status codes in responses
		for code := range op.Responses.Codes {
			if !httputil.ValidateStatusCode(code) {
				errors = append(errors, fmt.Errorf("oas 2.0: invalid status code '%s' in '%s.responses': must be a valid HTTP status code (e.g., \"200\", \"404\") or wildcard pattern (e.g., \"2XX\")", code, opPath))
			}
		}
	}

	// Validate parameters
	for i, param := range op.Parameters {
		if param == nil {
			continue
		}
		errors = append(errors, p.validateOAS2Parameter(param, opPath, i)...)
	}

	return errors
}

func (p *Parser) validateOAS2Parameter(param *Parameter, opPath string, index int) []error {
	errors := make([]error, 0)
	paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, index)

	// Skip validation for $ref parameters - they reference definitions elsewhere
	if param.Ref != "" {
		return errors
	}

	if param.Name == "" {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required field '%s.name': Parameter must have a name", paramPath))
	}
	if param.In == "" {
		errors = append(errors, fmt.Errorf("oas 2.0: missing required field '%s.in': Parameter must specify location (query, header, path, formData, body)", paramPath))
	} else {
		validLocations := map[string]bool{
			ParamInQuery:    true,
			ParamInHeader:   true,
			ParamInPath:     true,
			ParamInFormData: true,
			ParamInBody:     true,
		}
		if !validLocations[param.In] {
			errors = append(errors, fmt.Errorf("oas 2.0: invalid value for '%s.in': \"%s\" is not a valid parameter location (must be query, header, path, formData, or body)", paramPath, param.In))
		}
	}

	return errors
}

// validateOAS3 validates an OAS 3.x document
func (p *Parser) validateOAS3(doc *OAS3Document) []error {
	errors := make([]error, 0)
	version := doc.OpenAPI

	// Validate openapi version field
	if doc.OpenAPI == "" {
		errors = append(errors, fmt.Errorf("oas 3.x: missing required root field 'openapi': must be set to a valid 3.x version (e.g., \"3.0.3\", \"3.1.0\")"))
	} else if !versionInRangeExclusive(doc.OpenAPI, "3.0.0", "4.0.0") {
		errors = append(errors, fmt.Errorf("oas %s: invalid 'openapi' field value: \"%s\" is not a valid 3.x version", version, doc.OpenAPI))
	}

	// Validate info object
	errors = append(errors, p.validateOAS3Info(doc.Info, version)...)

	// Validate paths object - required in 3.0.x, optional in 3.1+
	errors = append(errors, p.validateOAS3PathsRequirement(doc, version)...)

	// Validate paths if present
	if doc.Paths != nil {
		errors = append(errors, p.validateOAS3Paths(doc.Paths, version)...)
	}

	// Validate webhooks if present (OAS 3.1+)
	if len(doc.Webhooks) > 0 {
		if versionInRangeExclusive(doc.OpenAPI, "0.0.0", "3.1.0") {
			errors = append(errors, fmt.Errorf("oas %s: 'webhooks' field is only supported in OAS 3.1.0 and later, not in version %s", version, doc.OpenAPI))
		} else {
			// Validate webhook structure (webhooks are PathItems like paths)
			errors = append(errors, p.validateOAS3Webhooks(doc.Webhooks, version)...)
		}
	}

	return errors
}

func (p *Parser) validateOAS3Info(info *Info, version string) []error {
	errors := make([]error, 0)
	if info == nil {
		errors = append(errors, fmt.Errorf("oas %s: missing required root field 'info': Info object is required per spec (https://spec.openapis.org/oas/v3.0.0.html#info-object)", version))
	} else {
		if info.Title == "" {
			errors = append(errors, fmt.Errorf("oas %s: missing required field 'info.title': Info object must have a title per spec", version))
		}
		if info.Version == "" {
			errors = append(errors, fmt.Errorf("oas %s: missing required field 'info.version': Info object must have a version string per spec", version))
		}
	}
	return errors
}

func (p *Parser) validateOAS3PathsRequirement(doc *OAS3Document, version string) []error {
	errors := make([]error, 0)
	if versionInRangeExclusive(doc.OpenAPI, "3.0.0", "3.1.0") {
		if doc.Paths == nil {
			errors = append(errors, fmt.Errorf("oas %s: missing required root field 'paths': Paths object is required in OAS 3.0.x (https://spec.openapis.org/oas/v3.0.0.html#paths-object)", version))
		}
	} else if versionInRangeExclusive(doc.OpenAPI, "3.1.0", "") {
		// In OAS 3.1+, either paths or webhooks must be present
		if doc.Paths == nil && len(doc.Webhooks) == 0 {
			errors = append(errors, fmt.Errorf("oas %s: document must have either 'paths' or 'webhooks': at least one is required in OAS 3.1+", version))
		}
	}
	return errors
}

func (p *Parser) validateOAS3Paths(paths map[string]*PathItem, version string) []error {
	errors := make([]error, 0)
	operationIDs := make(map[string]string)

	for pathPattern, pathItem := range paths {
		if pathItem == nil {
			continue
		}

		// Validate path pattern
		if pathPattern != "" && pathPattern[0] != '/' {
			errors = append(errors, fmt.Errorf("oas %s: invalid path pattern 'paths.%s': path must begin with '/'", version, pathPattern))
		}

		// Check all operations in this path
		errors = append(errors, p.validateOAS3PathItem(pathItem, pathPattern, operationIDs, version)...)
	}

	return errors
}

func (p *Parser) validateOAS3PathItem(pathItem *PathItem, pathPattern string, operationIDs map[string]string, version string) []error {
	errors := make([]error, 0)
	operations := GetOAS3Operations(pathItem)

	for method, op := range operations {
		if op == nil {
			continue
		}

		opPath := fmt.Sprintf("paths.%s.%s", pathPattern, method)
		errors = append(errors, p.validateOAS3Operation(op, opPath, operationIDs, version)...)
	}

	return errors
}

func (p *Parser) validateOAS3Operation(op *Operation, opPath string, operationIDs map[string]string, version string) []error {
	errors := make([]error, 0)

	// Validate operationId uniqueness
	if op.OperationID != "" {
		if existingPath, exists := operationIDs[op.OperationID]; exists {
			errors = append(errors, fmt.Errorf("oas %s: duplicate operationId '%s' at '%s': previously defined at '%s' (operationIds must be unique across all operations)",
				version, op.OperationID, opPath, existingPath))
		} else {
			operationIDs[op.OperationID] = opPath
		}
	}

	// Validate responses object exists
	if op.Responses == nil {
		errors = append(errors, fmt.Errorf("oas %s: missing required field '%s.responses': Operation must have a responses object", version, opPath))
	} else {
		// Validate status codes in responses
		for code := range op.Responses.Codes {
			if !httputil.ValidateStatusCode(code) {
				errors = append(errors, fmt.Errorf("oas %s: invalid status code '%s' in '%s.responses': must be a valid HTTP status code (e.g., \"200\", \"404\") or wildcard pattern (e.g., \"2XX\")", version, code, opPath))
			}
		}
	}

	// Validate parameters
	for i, param := range op.Parameters {
		if param == nil {
			continue
		}
		errors = append(errors, p.validateOAS3Parameter(param, opPath, i, version)...)
	}

	// Validate requestBody if present (skip if it's a $ref)
	if op.RequestBody != nil && op.RequestBody.Ref == "" {
		rbPath := fmt.Sprintf("%s.requestBody", opPath)
		if len(op.RequestBody.Content) == 0 {
			errors = append(errors, fmt.Errorf("oas %s: missing required field '%s.content': RequestBody must have at least one media type", version, rbPath))
		}
	}

	return errors
}

func (p *Parser) validateOAS3Parameter(param *Parameter, opPath string, index int, version string) []error {
	errors := make([]error, 0)
	paramPath := fmt.Sprintf("%s.parameters[%d]", opPath, index)

	// Skip validation for $ref parameters - they reference definitions elsewhere
	if param.Ref != "" {
		return errors
	}

	if param.Name == "" {
		errors = append(errors, fmt.Errorf("oas %s: missing required field '%s.name': Parameter must have a name", version, paramPath))
	}
	if param.In == "" {
		errors = append(errors, fmt.Errorf("oas %s: missing required field '%s.in': Parameter must specify location (query, header, path, cookie)", version, paramPath))
	} else {
		validLocations := map[string]bool{
			ParamInQuery:  true,
			ParamInHeader: true,
			ParamInPath:   true,
			ParamInCookie: true,
		}
		if !validLocations[param.In] {
			errors = append(errors, fmt.Errorf("oas %s: invalid value for '%s.in': \"%s\" is not a valid parameter location (must be query, header, path, or cookie)", version, paramPath, param.In))
		}
	}

	// Path parameters must be required
	if param.In == ParamInPath && !param.Required {
		errors = append(errors, fmt.Errorf("oas %s: invalid parameter '%s': path parameters must have 'required: true' per spec", version, paramPath))
	}

	return errors
}

// validateOAS3Webhooks validates webhooks structure (OAS 3.1+)
// Webhooks are similar to paths - they map webhook names to PathItems
func (p *Parser) validateOAS3Webhooks(webhooks map[string]*PathItem, version string) []error {
	errors := make([]error, 0)
	operationIDs := make(map[string]string)

	for webhookName, pathItem := range webhooks {
		if pathItem == nil {
			continue
		}

		// Validate each webhook's operations
		// Note: webhook names don't have the same path pattern requirements as paths
		// (they don't need to start with '/')
		errors = append(errors, p.validateOAS3PathItem(pathItem, "webhooks."+webhookName, operationIDs, version)...)
	}

	return errors
}
