package parser

//go:generate go run ../internal/codegen/deepcopy
//go:generate go run ../internal/codegen/decode

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	// HTTPClient is the HTTP client used for fetching URLs.
	// If nil, a default client with 30-second timeout is created.
	// When set, InsecureSkipVerify is ignored (configure TLS on your client's transport).
	HTTPClient *http.Client
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
	// BuildSourceMap enables source location tracking during parsing.
	// When enabled, the ParseResult.SourceMap will contain line/column
	// information for each JSON path in the document.
	// Default: false
	BuildSourceMap bool
	// PreserveOrder enables order-preserving marshaling.
	// When enabled, ParseResult stores the original yaml.Node structure,
	// allowing MarshalOrderedJSON/MarshalOrderedYAML to emit fields
	// in the same order as the source document.
	// This is useful for hash-based caching where roundtrip identity matters.
	// Default: false
	PreserveOrder bool
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
	// SourceMap contains JSON path to source location mappings.
	// Only populated when Parser.BuildSourceMap is true.
	SourceMap *SourceMap
	// sourceNode holds the original yaml.Node tree for order-preserving marshaling.
	// Only populated when Parser.PreserveOrder is true.
	// Use MarshalOrderedJSON/MarshalOrderedYAML to marshal with preserved order.
	sourceNode *yaml.Node
}

// OAS2Document returns the parsed document as an OAS2Document if the specification
// is version 2.0 (Swagger), and a boolean indicating whether the type assertion succeeded.
// This is a convenience method that provides a safe type assertion pattern.
//
// Example:
//
//	result, _ := parser.ParseWithOptions(parser.WithFilePath("swagger.yaml"))
//	if doc, ok := result.OAS2Document(); ok {
//	    fmt.Println("API Title:", doc.Info.Title)
//	}
func (pr *ParseResult) OAS2Document() (*OAS2Document, bool) {
	doc, ok := pr.Document.(*OAS2Document)
	return doc, ok
}

// OAS3Document returns the parsed document as an OAS3Document if the specification
// is version 3.x, and a boolean indicating whether the type assertion succeeded.
// This is a convenience method that provides a safe type assertion pattern.
//
// Example:
//
//	result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//	if doc, ok := result.OAS3Document(); ok {
//	    fmt.Println("API Title:", doc.Info.Title)
//	}
func (pr *ParseResult) OAS3Document() (*OAS3Document, bool) {
	doc, ok := pr.Document.(*OAS3Document)
	return doc, ok
}

// IsOAS2 returns true if the parsed document is an OpenAPI 2.0 (Swagger) specification.
// This is a convenience method for checking the document version without type assertions.
func (pr *ParseResult) IsOAS2() bool {
	return pr.OASVersion == OASVersion20
}

// IsOAS3 returns true if the parsed document is an OpenAPI 3.x specification
// (including 3.0.x, 3.1.x, and 3.2.x).
// This is a convenience method for checking the document version without type assertions.
func (pr *ParseResult) IsOAS3() bool {
	switch pr.OASVersion {
	case OASVersion300, OASVersion301, OASVersion302, OASVersion303, OASVersion304,
		OASVersion310, OASVersion311, OASVersion312, OASVersion320:
		return true
	default:
		return false
	}
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

	// Deep copy the SourceMap
	if pr.SourceMap != nil {
		result.SourceMap = pr.SourceMap.Copy()
	}

	return result
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

	// Update source map file paths
	updateSourceMapFilePath(res.SourceMap, specPath)

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
	// Update source map file paths
	updateSourceMapFilePath(res.SourceMap, res.SourcePath)
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
	// Update source map file paths
	updateSourceMapFilePath(res.SourceMap, res.SourcePath)
	return res, nil
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

	// Detect format early for potential JSON fast-path
	format := detectFormatFromContent(data)

	// JSON fast-path: skip YAML AST overhead when:
	// - Input is detected as JSON
	// - BuildSourceMap is disabled (source maps require YAML node tracking)
	// - PreserveOrder is disabled (order preservation requires YAML node tracking)
	//
	// This optimization reduces memory allocation by ~93% and parse time by ~88%
	// for JSON input by using encoding/json directly instead of building a YAML AST.
	if format == SourceFormatJSON && !p.BuildSourceMap && !p.PreserveOrder {
		return p.parseJSONFastPath(data, baseDir, baseURL, result)
	}

	// Build source map and/or preserve order if enabled (both require parsing to yaml.Node first)
	if p.BuildSourceMap || p.PreserveOrder {
		var rootNode yaml.Node
		if err := yaml.Unmarshal(data, &rootNode); err != nil {
			// Don't fail parsing, just add a warning
			result.Warnings = append(result.Warnings, fmt.Sprintf("yaml node parsing: failed to parse YAML nodes: %v", err))
		} else {
			// Build the source map with empty source path (will be updated later)
			if p.BuildSourceMap {
				result.SourceMap = buildSourceMap(&rootNode, "")
			}
			// Store node tree for order-preserving marshaling
			if p.PreserveOrder {
				result.sourceNode = &rootNode
			}
		}
	}

	// First pass: parse to generic map to detect OAS version
	var rawData map[string]any
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("parser: failed to parse YAML/JSON: %w", err)
	}

	// Resolve references if enabled (before semver-specific parsing)
	var hasCircularRefs bool
	if p.ResolveRefs {
		rawData, hasCircularRefs = p.resolveRefsInMap(rawData, data, baseDir, baseURL, result)
	}

	result.Data = rawData

	// Detect semver
	version, err := p.detectVersion(rawData)
	if err != nil {
		return nil, fmt.Errorf("parser: failed to detect OAS version: %w", err)
	}
	result.Version = version

	if hasCircularRefs {
		result.Warnings = append(result.Warnings, "Warning: Circular references detected. Non-circular references resolved normally. Circular references remain as $ref pointers.")
	}

	// Parse to version-specific structure
	var doc any
	var oasVersion OASVersion
	if p.ResolveRefs {
		// Decode directly from the resolved map, avoiding the marshal->unmarshal roundtrip.
		// This eliminates the intermediate []byte allocation that inflates memory for large specs.
		doc, oasVersion, err = decodeDocumentFromMap(rawData, version)
	} else {
		doc, oasVersion, err = p.parseVersionSpecific(data, version)
	}
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

// resolveRefsInMap resolves $ref references in a parsed map. It first attempts
// shallow-copy resolution (which avoids duplicating resolved content). If circular
// references are detected, shallow copy creates Go pointer cycles in the map, so
// it falls back to re-parsing and resolving with deep copy.
//
// Parameters:
//   - rawData: the parsed map to resolve refs in (modified in place)
//   - originalBytes: the original YAML/JSON bytes, used for re-parsing on fallback
//   - baseDir, baseURL: for resolving relative file and HTTP refs
//   - result: the ParseResult to append warnings to and set source maps on
//
// Returns the (possibly re-parsed) map and whether circular refs were detected.
func (p *Parser) resolveRefsInMap(rawData map[string]any, originalBytes []byte, baseDir, baseURL string, result *ParseResult) (map[string]any, bool) {
	resolver := p.newRefResolver(baseDir, baseURL)

	// If building source maps, pass the source map to the resolver
	if p.BuildSourceMap && result.SourceMap != nil {
		resolver.SourceMap = result.SourceMap
	}

	// Try shallow copy first — avoids deep-copying every resolved ref subtree.
	resolver.ShallowCopy = true

	resolveErr := resolver.ResolveAllRefs(rawData)

	// Shallow copy is only safe when there are no circular refs AND the
	// resolution completed without error. With circular schemas, shallow copy
	// creates Go pointer cycles in the map data. A MaxRefDepth error may also
	// indicate cycles created by shallow copy (the resolver's value-iteration
	// loop follows shared map pointers, inflating depth).
	needsFallback := resolver.HasCircularRefs() || resolveErr != nil
	if !needsFallback {
		if p.BuildSourceMap && result.SourceMap != nil {
			resolver.updateAllRefTargets()
		}
		return rawData, false
	}

	// Circular refs or resolution error — shallow copy may have created Go
	// pointer cycles. Re-parse from original bytes and re-resolve with deep copy.
	var freshData map[string]any
	if err := yaml.Unmarshal(originalBytes, &freshData); err != nil {
		// If re-parse fails, return the cyclic data with a warning.
		// This shouldn't happen since the same bytes parsed successfully before.
		result.Warnings = append(result.Warnings, fmt.Sprintf("ref resolution warning: failed to re-parse for deep copy fallback: %v", err))
		return rawData, resolver.HasCircularRefs()
	}

	freshResolver := p.newRefResolver(baseDir, baseURL)
	if p.BuildSourceMap && result.SourceMap != nil {
		freshResolver.SourceMap = result.SourceMap
	}
	// Deep copy (ShallowCopy=false is the default) — safe for circular refs.
	if err := freshResolver.ResolveAllRefs(freshData); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("ref resolution warning: %v", err))
	}
	if p.BuildSourceMap && result.SourceMap != nil {
		freshResolver.updateAllRefTargets()
	}
	return freshData, freshResolver.HasCircularRefs()
}

// newRefResolver creates a RefResolver configured for this parser's settings.
func (p *Parser) newRefResolver(baseDir, baseURL string) *RefResolver {
	if p.ResolveHTTPRefs {
		return NewRefResolverWithHTTP(baseDir, baseURL, p.fetchURL, p.MaxRefDepth, p.MaxCachedDocuments, p.MaxFileSize)
	}
	return NewRefResolver(baseDir, p.MaxRefDepth, p.MaxCachedDocuments, p.MaxFileSize)
}

// parseJSONFastPath parses JSON input directly using encoding/json, bypassing YAML AST overhead.
// This method is called when:
// - Input is detected as JSON format
// - BuildSourceMap is disabled
// - PreserveOrder is disabled
//
// The JSON fast-path provides significant performance benefits:
// - ~93% reduction in memory allocation (e.g., 750MB → 50MB for large specs)
// - ~88% reduction in parse time (e.g., 2.5s → 0.3s)
//
// This works because the yaml.v4 library builds a complete AST with token tracking,
// which is necessary for YAML features (anchors, aliases, multiline strings) but
// wasteful for JSON input where encoding/json is more efficient.
func (p *Parser) parseJSONFastPath(data []byte, baseDir, baseURL string, result *ParseResult) (*ParseResult, error) {
	// First pass: parse to generic map to detect OAS version
	var rawData map[string]any
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("parser: failed to parse JSON: %w", err)
	}

	// Resolve references if enabled (before version-specific parsing)
	var hasCircularRefs bool
	if p.ResolveRefs {
		rawData, hasCircularRefs = p.resolveRefsInMap(rawData, data, baseDir, baseURL, result)
	}

	result.Data = rawData
	result.SourceFormat = SourceFormatJSON

	// Detect version
	version, err := p.detectVersion(rawData)
	if err != nil {
		return nil, fmt.Errorf("parser: failed to detect OAS version: %w", err)
	}
	result.Version = version

	if hasCircularRefs {
		result.Warnings = append(result.Warnings, "Warning: Circular references detected. Non-circular references resolved normally. Circular references remain as $ref pointers.")
	}

	// Parse to version-specific structure
	var doc any
	var oasVersion OASVersion
	if p.ResolveRefs {
		// Decode directly from the resolved map, avoiding the marshal->unmarshal roundtrip.
		doc, oasVersion, err = decodeDocumentFromMap(rawData, version)
	} else {
		doc, oasVersion, err = p.parseVersionSpecificJSON(data, version)
	}
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

// parseVersionSpecificJSON parses JSON data into a version-specific structure using encoding/json.
// This is the JSON equivalent of parseVersionSpecific, used by the JSON fast-path.
func (p *Parser) parseVersionSpecificJSON(data []byte, version string) (any, OASVersion, error) {
	v, ok := ParseVersion(version)
	if !ok {
		return nil, 0, fmt.Errorf("parser: invalid OAS version: %s", version)
	}
	switch v {
	case OASVersion20:
		var doc OAS2Document
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, 0, fmt.Errorf("parser: failed to parse OAS 2.0 JSON document: %w", err)
		}
		doc.OASVersion = v
		return &doc, v, nil

	case OASVersion300, OASVersion301, OASVersion302, OASVersion303, OASVersion304,
		OASVersion310, OASVersion311, OASVersion312, OASVersion320:
		var doc OAS3Document
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, 0, fmt.Errorf("parser: failed to parse OAS %s JSON document: %w", version, err)
		}
		doc.OASVersion = v
		return &doc, v, nil

	default:
		return nil, 0, fmt.Errorf("parser: unsupported OpenAPI version: %s (only 2.0 and 3.x versions are supported)", version)
	}
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

	minVer, err := parseSemVer(minVersion)
	if err != nil {
		return false
	}

	// Check lower bound
	if !ver.greaterThanOrEqual(minVer) {
		return false
	}

	// If maxVersion is empty, no upper bound
	if maxVersion == "" {
		return true
	}

	maxVer, err := parseSemVer(maxVersion)
	if err != nil {
		return false
	}
	return ver.lessThan(maxVer)
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
	operations := GetOperations(pathItem, OASVersion20)

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
	oasVersion, _ := ParseVersion(version)
	operations := GetOperations(pathItem, oasVersion)

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
