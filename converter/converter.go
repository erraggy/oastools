package converter

import (
	"fmt"
	"time"

	"github.com/erraggy/oastools/internal/issues"
	"github.com/erraggy/oastools/internal/options"
	"github.com/erraggy/oastools/internal/severity"
	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
)

// Severity indicates the severity level of a conversion issue
type Severity = severity.Severity

const (
	// SeverityInfo indicates informational messages about conversion choices
	SeverityInfo = severity.SeverityInfo
	// SeverityWarning indicates lossy conversions or best-effort transformations
	SeverityWarning = severity.SeverityWarning
	// SeverityError indicates validation errors
	SeverityError = severity.SeverityError
	// SeverityCritical indicates features that cannot be converted (data loss)
	SeverityCritical = severity.SeverityCritical
)

// ConversionIssue represents a single conversion issue or limitation
type ConversionIssue = issues.Issue

// ConversionResult contains the results of converting an OpenAPI specification
type ConversionResult struct {
	// Document contains the converted document (*parser.OAS2Document or *parser.OAS3Document)
	Document any
	// SourceVersion is the detected source OAS version string
	SourceVersion string
	// SourceOASVersion is the enumerated source OAS version
	SourceOASVersion parser.OASVersion
	// SourceFormat is the format of the source file (JSON or YAML)
	SourceFormat parser.SourceFormat
	// TargetVersion is the target OAS version string
	TargetVersion string
	// TargetOASVersion is the enumerated target OAS version
	TargetOASVersion parser.OASVersion
	// Issues contains all conversion issues grouped by severity
	Issues []ConversionIssue
	// InfoCount is the total number of info messages
	InfoCount int
	// WarningCount is the total number of warnings
	WarningCount int
	// CriticalCount is the total number of critical issues
	CriticalCount int
	// Success is true if conversion completed without critical issues
	Success bool
	// LoadTime is the time taken to load the source data
	LoadTime time.Duration
	// SourceSize is the size of the source data in bytes
	SourceSize int64
	// Stats contains statistical information about the source document
	Stats parser.DocumentStats
}

// HasCriticalIssues returns true if there are any critical issues
func (r *ConversionResult) HasCriticalIssues() bool {
	return r.CriticalCount > 0
}

// HasWarnings returns true if there are any warnings
func (r *ConversionResult) HasWarnings() bool {
	return r.WarningCount > 0
}

// Converter handles OpenAPI specification version conversion
type Converter struct {
	// StrictMode causes conversion to fail on any issues (even warnings)
	StrictMode bool
	// IncludeInfo determines whether to include informational messages
	IncludeInfo bool
	// UserAgent is the User-Agent string used when fetching URLs
	// Defaults to "oastools" if not set
	UserAgent string
	// SourceMap provides source location lookup for conversion issues.
	// When set, issues will include Line, Column, and File information.
	SourceMap *parser.SourceMap
}

// New creates a new Converter instance with default settings
func New() *Converter {
	return &Converter{
		StrictMode:  false,
		IncludeInfo: true,
	}
}

// Option is a function that configures a conversion operation
type Option func(*convertConfig) error

// convertConfig holds configuration for a conversion operation
type convertConfig struct {
	// Input source (exactly one must be set)
	filePath *string
	parsed   *parser.ParseResult

	// Target version (required)
	targetVersion string

	// Configuration options
	strictMode  bool
	includeInfo bool
	userAgent   string

	// Source map for line/column tracking
	sourceMap *parser.SourceMap

	// Overlay integration options
	preConversionOverlay      *overlay.Overlay // Applied before conversion
	preConversionOverlayFile  *string          // File path for pre-conversion overlay
	postConversionOverlay     *overlay.Overlay // Applied after conversion
	postConversionOverlayFile *string          // File path for post-conversion overlay
}

// ConvertWithOptions converts an OpenAPI specification using functional options.
// This provides a flexible, extensible API that combines input source selection
// and configuration in a single function call.
//
// When overlay options are provided, the conversion process follows these steps:
//  1. Parse the source specification
//  2. Apply pre-conversion overlay (if specified)
//  3. Perform the version conversion
//  4. Apply post-conversion overlay (if specified)
//
// Example:
//
//	result, err := converter.ConvertWithOptions(
//	    converter.WithFilePath("swagger.yaml"),
//	    converter.WithTargetVersion("3.0.3"),
//	    converter.WithStrictMode(true),
//	    converter.WithPostConversionOverlayFile("enhance-v3.yaml"),
//	)
func ConvertWithOptions(opts ...Option) (*ConversionResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("converter: invalid options: %w", err)
	}

	c := &Converter{
		StrictMode:  cfg.strictMode,
		IncludeInfo: cfg.includeInfo,
		UserAgent:   cfg.userAgent,
		SourceMap:   cfg.sourceMap,
	}

	// Check if any overlays are configured
	hasOverlays := cfg.preConversionOverlay != nil ||
		cfg.preConversionOverlayFile != nil ||
		cfg.postConversionOverlay != nil ||
		cfg.postConversionOverlayFile != nil

	// Fast path: no overlays configured, use original logic
	// Parsed input is checked first as it's the preferred high-performance path
	if !hasOverlays {
		if cfg.parsed != nil {
			return c.ConvertParsed(*cfg.parsed, cfg.targetVersion)
		}
		return c.Convert(*cfg.filePath, cfg.targetVersion)
	}

	// Slow path: overlays require us to parse, transform, convert, transform
	return convertWithOverlays(c, cfg)
}

// convertWithOverlays handles conversion with overlay processing
func convertWithOverlays(c *Converter, cfg *convertConfig) (*ConversionResult, error) {
	// Step 1: Parse overlay files if specified
	preOverlay, err := overlay.ParseOverlaySingle(cfg.preConversionOverlay, cfg.preConversionOverlayFile)
	if err != nil {
		return nil, fmt.Errorf("converter: pre-conversion overlay: %w", err)
	}

	postOverlay, err := overlay.ParseOverlaySingle(cfg.postConversionOverlay, cfg.postConversionOverlayFile)
	if err != nil {
		return nil, fmt.Errorf("converter: post-conversion overlay: %w", err)
	}

	// Step 2: Parse the source specification
	var parsed *parser.ParseResult
	if cfg.filePath != nil {
		p := parser.New()
		p.UserAgent = c.UserAgent
		parsed, err = p.Parse(*cfg.filePath)
		if err != nil {
			return nil, fmt.Errorf("converter: failed to parse source: %w", err)
		}
	} else {
		parsed = cfg.parsed
	}

	// Step 3: Apply pre-conversion overlay
	if preOverlay != nil {
		applier := overlay.NewApplier()
		result, err := applier.ApplyParsed(parsed, preOverlay)
		if err != nil {
			return nil, fmt.Errorf("converter: applying pre-conversion overlay: %w", err)
		}
		// Re-parse to restore typed document
		reparsed, err := overlay.ReparseDocument(parsed, result.Document)
		if err != nil {
			return nil, err
		}
		parsed = reparsed
	}

	// Step 4: Perform the conversion
	convResult, err := c.ConvertParsed(*parsed, cfg.targetVersion)
	if err != nil {
		return nil, err
	}

	// Step 5: Apply post-conversion overlay
	if postOverlay != nil {
		applier := overlay.NewApplier()
		postResult, err := applier.ApplyParsed(&parser.ParseResult{
			Document:     convResult.Document,
			SourceFormat: convResult.SourceFormat,
		}, postOverlay)
		if err != nil {
			return nil, fmt.Errorf("converter: applying post-conversion overlay: %w", err)
		}
		convResult.Document = postResult.Document
	}

	return convResult, nil
}

// applyOptions applies option functions and validates configuration
func applyOptions(opts ...Option) (*convertConfig, error) {
	cfg := &convertConfig{
		// Set defaults to match existing behavior
		strictMode:  false,
		includeInfo: true,
		userAgent:   "",
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate exactly one input source is specified
	if err := options.ValidateSingleInputSource(
		"must specify an input source (use WithFilePath or WithParsed)",
		"must specify exactly one input source",
		cfg.filePath != nil, cfg.parsed != nil,
	); err != nil {
		return nil, err
	}

	// Validate target version is specified
	if cfg.targetVersion == "" {
		return nil, fmt.Errorf("must specify a target version (use WithTargetVersion)")
	}

	return cfg, nil
}

// WithFilePath specifies a file path or URL as the input source
func WithFilePath(path string) Option {
	return func(cfg *convertConfig) error {
		cfg.filePath = &path
		return nil
	}
}

// WithParsed specifies a parsed ParseResult as the input source
func WithParsed(result parser.ParseResult) Option {
	return func(cfg *convertConfig) error {
		cfg.parsed = &result
		return nil
	}
}

// WithTargetVersion specifies the target OAS version for conversion
// Required option - must be one of: "2.0", "3.0.0", "3.0.1", "3.0.2", "3.0.3", "3.1.0", etc.
func WithTargetVersion(version string) Option {
	return func(cfg *convertConfig) error {
		if version == "" {
			return fmt.Errorf("target version cannot be empty")
		}
		cfg.targetVersion = version
		return nil
	}
}

// WithStrictMode enables or disables strict mode (fail on any issues)
// Default: false
func WithStrictMode(enabled bool) Option {
	return func(cfg *convertConfig) error {
		cfg.strictMode = enabled
		return nil
	}
}

// WithIncludeInfo enables or disables informational messages
// Default: true
func WithIncludeInfo(enabled bool) Option {
	return func(cfg *convertConfig) error {
		cfg.includeInfo = enabled
		return nil
	}
}

// WithUserAgent sets the User-Agent string for HTTP requests
// Default: "" (uses parser default)
func WithUserAgent(ua string) Option {
	return func(cfg *convertConfig) error {
		cfg.userAgent = ua
		return nil
	}
}

// WithSourceMap provides a SourceMap for populating line/column information
// in conversion issues. When set, issues will include source location details
// that enable IDE-friendly error reporting.
func WithSourceMap(sm *parser.SourceMap) Option {
	return func(cfg *convertConfig) error {
		cfg.sourceMap = sm
		return nil
	}
}

// Convert converts an OpenAPI specification file to a target version
func (c *Converter) Convert(specPath string, targetVersion string) (*ConversionResult, error) {
	// Create parser and set UserAgent if specified
	p := parser.New()
	if c.UserAgent != "" {
		p.UserAgent = c.UserAgent
	}

	// Parse the source document
	parseResult, err := p.Parse(specPath)
	if err != nil {
		return nil, fmt.Errorf("converter: failed to parse specification: %w", err)
	}

	// Check for parse errors
	if len(parseResult.Errors) > 0 {
		return nil, fmt.Errorf("converter: source document has %d parse error(s), cannot convert", len(parseResult.Errors))
	}

	return c.ConvertParsed(*parseResult, targetVersion)
}

// ConvertParsed converts an already-parsed OpenAPI specification to a target version
func (c *Converter) ConvertParsed(parseResult parser.ParseResult, targetVersionStr string) (*ConversionResult, error) {
	// Parse target version
	targetVersion, ok := parser.ParseVersion(targetVersionStr)
	if !ok {
		return nil, fmt.Errorf("converter: invalid target version: %s", targetVersionStr)
	}

	// Initialize result
	result := &ConversionResult{
		SourceVersion:    parseResult.Version,
		SourceOASVersion: parseResult.OASVersion,
		SourceFormat:     parseResult.SourceFormat,
		TargetVersion:    targetVersionStr,
		TargetOASVersion: targetVersion,
		Issues:           make([]ConversionIssue, 0),
		LoadTime:         parseResult.LoadTime,
		SourceSize:       parseResult.SourceSize,
		Stats:            parseResult.Stats,
	}

	// Check if conversion is needed
	if parseResult.OASVersion == targetVersion {
		// No conversion needed, just copy the document
		result.Document = parseResult.Document
		result.Success = true
		c.addIssue(result, "document", fmt.Sprintf("Source and target versions are the same (%s), no conversion needed", targetVersionStr), SeverityInfo)
		c.updateCounts(result)
		return result, nil
	}

	// Determine conversion direction
	sourceIsOAS2 := parseResult.OASVersion == parser.OASVersion20
	targetIsOAS2 := targetVersion == parser.OASVersion20

	var err error
	switch {
	case sourceIsOAS2 && !targetIsOAS2:
		// OAS 2.0 → OAS 3.x
		err = c.convertOAS2ToOAS3(parseResult, targetVersion, result)
	case !sourceIsOAS2 && targetIsOAS2:
		// OAS 3.x → OAS 2.0
		err = c.convertOAS3ToOAS2(parseResult, result)
	default:
		// OAS 3.x → OAS 3.y (version update)
		// Note: all other cases handled above (OAS 2.0↔3.x, same version handled earlier)
		err = c.convertOAS3ToOAS3(parseResult, targetVersion, result)
	}

	if err != nil {
		return nil, err
	}

	// Update counts and success status
	c.updateCounts(result)
	result.Success = result.CriticalCount == 0

	// In strict mode, fail on any issues
	if c.StrictMode && (result.CriticalCount > 0 || result.WarningCount > 0) {
		return result, fmt.Errorf("conversion failed in strict mode: %d critical issue(s), %d warning(s)",
			result.CriticalCount, result.WarningCount)
	}

	// Filter info messages if not included
	if !c.IncludeInfo {
		filtered := make([]ConversionIssue, 0, len(result.Issues))
		for _, issue := range result.Issues {
			if issue.Severity != SeverityInfo {
				filtered = append(filtered, issue)
			}
		}
		result.Issues = filtered
		result.InfoCount = 0
	}

	return result, nil
}

// updateCounts updates the issue counts in the result
func (c *Converter) updateCounts(result *ConversionResult) {
	result.InfoCount = 0
	result.WarningCount = 0
	result.CriticalCount = 0

	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityInfo:
			result.InfoCount++
		case SeverityWarning:
			result.WarningCount++
		case SeverityCritical:
			result.CriticalCount++
		}
	}
}

// convertOAS3ToOAS3 handles version updates within OAS 3.x (e.g., 3.0.3 → 3.1.0)
func (c *Converter) convertOAS3ToOAS3(parseResult parser.ParseResult, targetVersion parser.OASVersion, result *ConversionResult) error {
	// For OAS 3.x to 3.y conversions, we primarily just update the version string
	doc, ok := parseResult.OAS3Document()
	if !ok {
		return fmt.Errorf("source document is not an OAS3Document")
	}

	// Create a deep copy of the document
	converted, err := c.deepCopyOAS3Document(doc)
	if err != nil {
		return err
	}

	// Update the version string
	converted.OpenAPI = result.TargetVersion
	converted.OASVersion = targetVersion

	result.Document = converted

	// Add informational message about version update
	c.addInfoWithContext(result, "openapi", fmt.Sprintf("Updated version from %s to %s", parseResult.Version, result.TargetVersion), "OAS 3.x versions are generally compatible, but verify features are supported")

	// Check for nullable deprecation when converting 3.0.x to 3.1.x
	if c.isOAS30(parseResult.OASVersion) && c.isOAS31OrLater(targetVersion) {
		c.checkNullableDeprecation(converted, result)
	}

	return nil
}

// isOAS30 returns true if the version is OAS 3.0.x
func (c *Converter) isOAS30(v parser.OASVersion) bool {
	return v >= parser.OASVersion300 && v <= parser.OASVersion304
}

// isOAS31OrLater returns true if the version is OAS 3.1.x or later
func (c *Converter) isOAS31OrLater(v parser.OASVersion) bool {
	return v >= parser.OASVersion310
}

// checkNullableDeprecation walks the document and warns about nullable usage
func (c *Converter) checkNullableDeprecation(doc *parser.OAS3Document, result *ConversionResult) {
	// Check component schemas
	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schema := range doc.Components.Schemas {
			c.checkSchemaNullable(schema, fmt.Sprintf("components.schemas.%s", name), result)
		}
	}

	// Check paths
	for pathPattern, pathItem := range doc.Paths {
		c.checkPathItemNullable(pathItem, fmt.Sprintf("paths.%s", pathPattern), result)
	}
}

// checkPathItemNullable checks all operations in a path item for nullable schemas
func (c *Converter) checkPathItemNullable(pathItem *parser.PathItem, pathPrefix string, result *ConversionResult) {
	if pathItem == nil {
		return
	}
	ops := map[string]*parser.Operation{
		"get":     pathItem.Get,
		"put":     pathItem.Put,
		"post":    pathItem.Post,
		"delete":  pathItem.Delete,
		"options": pathItem.Options,
		"head":    pathItem.Head,
		"patch":   pathItem.Patch,
		"trace":   pathItem.Trace,
	}
	for method, op := range ops {
		if op != nil {
			c.checkOperationNullable(op, fmt.Sprintf("%s.%s", pathPrefix, method), result)
		}
	}
}

// checkOperationNullable checks request body and responses for nullable schemas
func (c *Converter) checkOperationNullable(op *parser.Operation, opPath string, result *ConversionResult) {
	// Check request body
	if op.RequestBody != nil && op.RequestBody.Content != nil {
		for mediaType, content := range op.RequestBody.Content {
			if content.Schema != nil {
				c.checkSchemaNullable(content.Schema, fmt.Sprintf("%s.requestBody.content.%s.schema", opPath, mediaType), result)
			}
		}
	}

	// Check responses
	if op.Responses != nil {
		if op.Responses.Default != nil && op.Responses.Default.Content != nil {
			for mediaType, content := range op.Responses.Default.Content {
				if content.Schema != nil {
					c.checkSchemaNullable(content.Schema, fmt.Sprintf("%s.responses.default.content.%s.schema", opPath, mediaType), result)
				}
			}
		}
		for code, response := range op.Responses.Codes {
			if response != nil && response.Content != nil {
				for mediaType, content := range response.Content {
					if content.Schema != nil {
						c.checkSchemaNullable(content.Schema, fmt.Sprintf("%s.responses.%s.content.%s.schema", opPath, code, mediaType), result)
					}
				}
			}
		}
	}

	// Check parameters
	for i, param := range op.Parameters {
		if param != nil && param.Schema != nil {
			c.checkSchemaNullable(param.Schema, fmt.Sprintf("%s.parameters[%d].schema", opPath, i), result)
		}
	}
}

// checkSchemaNullable recursively checks a schema for nullable usage
func (c *Converter) checkSchemaNullable(schema *parser.Schema, path string, result *ConversionResult) {
	if schema == nil {
		return
	}

	if schema.Nullable {
		c.addIssueWithContext(result, path, "'nullable: true' is deprecated in OAS 3.1", "In OAS 3.1, use 'type: [\"<type>\", \"null\"]' instead of 'nullable: true'")
	}

	// Check nested schemas
	if schema.Items != nil {
		if itemsSchema, ok := schema.Items.(*parser.Schema); ok {
			c.checkSchemaNullable(itemsSchema, path+".items", result)
		}
	}
	for propName, propSchema := range schema.Properties {
		c.checkSchemaNullable(propSchema, fmt.Sprintf("%s.properties.%s", path, propName), result)
	}
	if schema.AdditionalProperties != nil {
		if additionalSchema, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			c.checkSchemaNullable(additionalSchema, path+".additionalProperties", result)
		}
	}
	for i, allOf := range schema.AllOf {
		c.checkSchemaNullable(allOf, fmt.Sprintf("%s.allOf[%d]", path, i), result)
	}
	for i, anyOf := range schema.AnyOf {
		c.checkSchemaNullable(anyOf, fmt.Sprintf("%s.anyOf[%d]", path, i), result)
	}
	for i, oneOf := range schema.OneOf {
		c.checkSchemaNullable(oneOf, fmt.Sprintf("%s.oneOf[%d]", path, i), result)
	}
	if schema.Not != nil {
		c.checkSchemaNullable(schema.Not, path+".not", result)
	}
}

// addIssue is a helper to add a conversion issue to the result
func (c *Converter) addIssue(result *ConversionResult, path, message string, severity Severity) {
	issue := ConversionIssue{
		Path:     path,
		Message:  message,
		Severity: severity,
	}
	c.populateIssueLocation(&issue, path)
	result.Issues = append(result.Issues, issue)
}

// addIssueWithContext is a helper to add a conversion issue with context
func (c *Converter) addIssueWithContext(result *ConversionResult, path, message, context string) {
	issue := ConversionIssue{
		Path:     path,
		Message:  message,
		Severity: SeverityWarning,
		Context:  context,
	}
	c.populateIssueLocation(&issue, path)
	result.Issues = append(result.Issues, issue)
}

// addInfoWithContext is a helper to add an informational conversion issue with context
func (c *Converter) addInfoWithContext(result *ConversionResult, path, message, context string) {
	issue := ConversionIssue{
		Path:     path,
		Message:  message,
		Severity: SeverityInfo,
		Context:  context,
	}
	c.populateIssueLocation(&issue, path)
	result.Issues = append(result.Issues, issue)
}

// populateIssueLocation fills in Line/Column/File from the SourceMap if available.
// The path parameter is the JSON path in the source document.
func (c *Converter) populateIssueLocation(issue *ConversionIssue, path string) {
	if c.SourceMap == nil {
		return
	}
	// Convert path format if needed (converter uses dotted paths like "paths./users.get",
	// while SourceMap uses JSON path notation like "$.paths./users.get")
	jsonPath := path
	if !hasJSONPathPrefix(jsonPath) {
		jsonPath = "$." + path
	}
	loc := c.SourceMap.Get(jsonPath)
	if loc.IsKnown() {
		issue.Line = loc.Line
		issue.Column = loc.Column
		issue.File = loc.File
	}
}

// hasJSONPathPrefix returns true if the path already has a JSON path prefix.
func hasJSONPathPrefix(path string) bool {
	return len(path) > 0 && path[0] == '$'
}
