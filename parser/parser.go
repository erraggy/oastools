package parser

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/erraggy/oastools"
	semver "github.com/hashicorp/go-version"
	"gopkg.in/yaml.v3"

	"github.com/erraggy/oastools/internal/httputil"
)

// Parser handles OpenAPI specification parsing
type Parser struct {
	// ResolveRefs determines whether to resolve $ref references
	ResolveRefs bool
	// ValidateStructure determines whether to perform basic structure validation
	ValidateStructure bool
	// UserAgent is the User-Agent string used when fetching URLs
	// Defaults to "oastools" if not set
	UserAgent string
}

// New creates a new Parser instance with default settings
func New() *Parser {
	return &Parser{
		ResolveRefs:       false,
		ValidateStructure: true,
		UserAgent:         oastools.UserAgent(),
	}
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
// typed representations of the OpenAPI document, and should be treated as immutable.
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
	client := &http.Client{
		Timeout: 30 * time.Second,
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

		// For URLs, we can't resolve relative refs easily, so use current directory
		// Note: This means relative $ref paths in URL-loaded specs will attempt to
		// load from the local filesystem, not relative to the URL. This is a known
		// limitation that may be addressed in a future version.
		baseDir = "."

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

		// Detect format from file extension
		format = detectFormatFromPath(specPath)
	}

	// Parse the data
	res, err := p.parseBytesWithBaseDir(data, baseDir)
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

// Parse is a convenience function that parses an OpenAPI specification file
// with the specified options. It's equivalent to creating a Parser with
// New(), setting the options, and calling Parse().
//
// For one-off parsing operations, this function provides a simpler API.
// For parsing multiple files with the same configuration, create a Parser
// instance and reuse it.
//
// Example:
//
//	result, err := parser.Parse("openapi.yaml", false, true)
//	if err != nil {
//	    log.Fatal(err)
//	}
func Parse(specPath string, resolveRefs, validateStructure bool) (*ParseResult, error) {
	p := &Parser{
		ResolveRefs:       resolveRefs,
		ValidateStructure: validateStructure,
	}
	return p.Parse(specPath)
}

// ParseReader is a convenience function that parses an OpenAPI specification
// from an io.Reader with the specified options.
//
// Example:
//
//	file, _ := os.Open("openapi.yaml")
//	defer file.Close()
//	result, err := parser.ParseReader(file, false, true)
func ParseReader(r io.Reader, resolveRefs, validateStructure bool) (*ParseResult, error) {
	p := &Parser{
		ResolveRefs:       resolveRefs,
		ValidateStructure: validateStructure,
	}
	return p.ParseReader(r)
}

// ParseBytes is a convenience function that parses an OpenAPI specification
// from a byte slice with the specified options.
//
// Example:
//
//	data := []byte(`openapi: "3.0.0"...`)
//	result, err := parser.ParseBytes(data, false, true)
func ParseBytes(data []byte, resolveRefs, validateStructure bool) (*ParseResult, error) {
	p := &Parser{
		ResolveRefs:       resolveRefs,
		ValidateStructure: validateStructure,
	}
	return p.ParseBytes(data)
}

// parseBytesWithBaseDir parses data with a specified base directory for ref resolution
func (p *Parser) parseBytesWithBaseDir(data []byte, baseDir string) (*ParseResult, error) {
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
	if p.ResolveRefs {
		resolver := NewRefResolver(baseDir)
		if err := resolver.ResolveAllRefs(rawData); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("ref resolution warning: %v", err))
		}
	}

	result.Data = rawData

	// Detect semver
	version, err := p.detectVersion(rawData)
	if err != nil {
		return nil, fmt.Errorf("parser: failed to detect OAS version: %w", err)
	}
	result.Version = version

	// Prepare data for version-specific parsing
	// Only re-marshal if we resolved refs (to avoid unnecessary overhead)
	//
	// Performance trade-off: When ResolveRefs is enabled, we must re-marshal the
	// rawData map after reference resolution to ensure the resolved content is
	// available to the version-specific parsers. This adds overhead (especially
	// for large documents), but is necessary for correct reference resolution.
	// When ResolveRefs is disabled, we skip this step and use the original data.
	var parseData []byte
	if p.ResolveRefs {
		// Re-marshal the data with resolved refs
		parseData, err = yaml.Marshal(rawData)
		if err != nil {
			// If marshaling fails (e.g., due to circular references that couldn't be resolved),
			// fall back to using the original data. This can happen with complex circular structures
			// like $ref: "#" which we intentionally don't resolve to prevent infinite loops.
			parseData = data
			result.Warnings = append(result.Warnings, fmt.Sprintf("Warning: Could not re-marshal document after reference resolution (likely due to circular references): %v. Using original document structure. Some references may not be fully resolved.", err))
		}
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

// parseSemVer parses a semver string into a semantic semver
func parseSemVer(v string) (*semver.Version, error) {
	return semver.NewVersion(v)
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
	if !ver.GreaterThanOrEqual(min) {
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
	return ver.LessThan(max)
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

	// Validate requestBody if present
	if op.RequestBody != nil {
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
