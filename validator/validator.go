package validator

import (
	"fmt"
	"time"

	"github.com/erraggy/oastools/internal/issues"
	"github.com/erraggy/oastools/internal/severity"
	"github.com/erraggy/oastools/parser"
)

// Severity indicates the severity level of a validation issue
type Severity = severity.Severity

const (
	// SeverityError indicates a spec violation that makes the document invalid
	SeverityError = severity.SeverityError
	// SeverityWarning indicates a best practice violation or recommendation
	SeverityWarning = severity.SeverityWarning
	// SeverityInfo indicates informational messages
	SeverityInfo = severity.SeverityInfo
	// SeverityCritical indicates critical issues
	SeverityCritical = severity.SeverityCritical
)

const (
	// defaultErrorCapacity is the initial capacity for error slices
	defaultErrorCapacity = 10
	// defaultWarningCapacity is the initial capacity for warning slices
	defaultWarningCapacity = 10

	// Resource exhaustion protection
	maxSchemaNestingDepth = 100 // Maximum depth for nested schemas to prevent stack overflow
)

// ValidationError represents a single validation issue
type ValidationError = issues.Issue

// ValidationResult contains the results of validating an OpenAPI specification
type ValidationResult struct {
	// Valid is true if no errors were found (warnings are allowed)
	Valid bool
	// Version is the detected OAS version string
	Version string
	// OASVersion is the enumerated OAS version
	OASVersion parser.OASVersion
	// Errors contains all validation errors
	Errors []ValidationError
	// Warnings contains all validation warnings
	Warnings []ValidationError
	// ErrorCount is the total number of errors
	ErrorCount int
	// WarningCount is the total number of warnings
	WarningCount int
	// LoadTime is the time taken to load the source data
	LoadTime time.Duration
	// SourceSize is the size of the source data in bytes
	SourceSize int64
	// Stats contains statistical information about the document
	Stats parser.DocumentStats
	// Document contains the validated document (*parser.OAS2Document or *parser.OAS3Document).
	// Added to enable ToParseResult() for package chaining.
	Document any
	// SourceFormat is the format of the source file (JSON or YAML)
	SourceFormat parser.SourceFormat
	// SourcePath is the original source path from the parsed document.
	// Used by ToParseResult() to preserve the original path for package chaining.
	SourcePath string
}

// ToParseResult converts the ValidationResult to a ParseResult for use with
// other packages like fixer, converter, joiner, and differ.
// The returned ParseResult has Document populated but Data is nil
// (consumers use Document, not Data).
// Validation errors and warnings are converted to string warnings with
// severity prefixes for programmatic filtering:
// "[error] path: message", "[warning] path: message", etc.
func (r *ValidationResult) ToParseResult() *parser.ParseResult {
	// Convert validation errors/warnings to string warnings with severity prefix
	warnings := make([]string, 0, len(r.Errors)+len(r.Warnings))
	for _, e := range r.Errors {
		warnings = append(warnings, "["+e.Severity.String()+"] "+e.String())
	}
	for _, w := range r.Warnings {
		warnings = append(warnings, "["+w.Severity.String()+"] "+w.String())
	}

	// Use original source path, falling back to "validator" if not set
	sourcePath := r.SourcePath
	if sourcePath == "" {
		sourcePath = "validator"
	}

	return &parser.ParseResult{
		SourcePath:   sourcePath,
		SourceFormat: r.SourceFormat,
		Version:      r.Version,
		OASVersion:   r.OASVersion,
		Document:     r.Document,
		Errors:       make([]error, 0),
		Warnings:     warnings,
		Stats:        r.Stats,
		LoadTime:     r.LoadTime,
		SourceSize:   r.SourceSize,
	}
}

// Validator handles OpenAPI specification validation
type Validator struct {
	// IncludeWarnings determines whether to include best practice warnings
	IncludeWarnings bool
	// StrictMode enables stricter validation beyond the spec requirements
	StrictMode bool
	// ValidateStructure controls whether the parser performs basic structure validation.
	// When true (default), the parser validates required fields and correct types.
	// When false, parsing is more lenient and skips structure validation.
	ValidateStructure bool
	// UserAgent is the User-Agent string used when fetching URLs
	// Defaults to "oastools" if not set
	UserAgent string
	// SourceMap provides source location lookup for validation errors.
	// When set, validation errors will include Line, Column, and File fields.
	SourceMap *parser.SourceMap
	// refTracker tracks which operations reference which components.
	// Built during ValidateParsed for populating OperationContext on issues.
	refTracker *refTracker
}

// New creates a new Validator instance with default settings
func New() *Validator {
	return &Validator{
		IncludeWarnings:   true,
		StrictMode:        false,
		ValidateStructure: true,
	}
}

// ValidateWithOptions validates an OpenAPI specification using functional options.
// This provides a flexible, extensible API that combines input source selection
// and configuration in a single function call.
//
// Example:
//
//	result, err := validator.ValidateWithOptions(
//	    validator.WithFilePath("openapi.yaml"),
//	    validator.WithStrictMode(true),
//	)
func ValidateWithOptions(opts ...Option) (*ValidationResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("validator: invalid options: %w", err)
	}

	v := &Validator{
		IncludeWarnings:   cfg.includeWarnings,
		StrictMode:        cfg.strictMode,
		ValidateStructure: cfg.validateStructure,
		UserAgent:         cfg.userAgent,
		SourceMap:         cfg.sourceMap,
	}

	// Route to appropriate validation method based on input source
	// Parsed input is checked first as it's the preferred high-performance path
	if cfg.parsed != nil {
		return v.ValidateParsed(*cfg.parsed)
	}
	// cfg.filePath must be non-nil here (validated by applyOptions)
	return v.Validate(*cfg.filePath)
}

// populateIssueLocation looks up the source location for an issue's path
// and populates the Line, Column, and File fields if found.
func (v *Validator) populateIssueLocation(issue *ValidationError) {
	if v.SourceMap == nil {
		return
	}
	// Convert validation path to JSON path format (add $ prefix if needed)
	jsonPath := issue.Path
	if !hasJSONPathPrefix(jsonPath) {
		jsonPath = "$." + issue.Path
	}
	loc := v.SourceMap.Get(jsonPath)
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

// populateOperationContext attaches operation context to an issue if applicable.
func (v *Validator) populateOperationContext(issue *ValidationError, doc any) {
	if v.refTracker == nil {
		return
	}
	issue.OperationContext = v.refTracker.getOperationContext(issue.Path, doc)
}

// addError appends a validation error and populates its source location.
func (v *Validator) addError(result *ValidationResult, path, message string, opts ...func(*ValidationError)) {
	err := ValidationError{
		Path:     path,
		Message:  message,
		Severity: SeverityError,
	}
	for _, opt := range opts {
		opt(&err)
	}
	v.populateIssueLocation(&err)
	v.populateOperationContext(&err, result.Document)
	result.Errors = append(result.Errors, err)
}

// addWarning appends a validation warning and populates its source location.
func (v *Validator) addWarning(result *ValidationResult, path, message string, opts ...func(*ValidationError)) {
	warn := ValidationError{
		Path:     path,
		Message:  message,
		Severity: SeverityWarning,
	}
	for _, opt := range opts {
		opt(&warn)
	}
	v.populateIssueLocation(&warn)
	v.populateOperationContext(&warn, result.Document)
	result.Warnings = append(result.Warnings, warn)
}

// withField sets the Field on a ValidationError.
func withField(field string) func(*ValidationError) {
	return func(e *ValidationError) { e.Field = field }
}

// withValue sets the Value on a ValidationError.
func withValue(value any) func(*ValidationError) {
	return func(e *ValidationError) { e.Value = value }
}

// withSpecRef sets the SpecRef on a ValidationError.
func withSpecRef(ref string) func(*ValidationError) {
	return func(e *ValidationError) { e.SpecRef = ref }
}

// ValidateParsed validates an already parsed OpenAPI specification
func (v *Validator) ValidateParsed(parseResult parser.ParseResult) (*ValidationResult, error) {
	result := &ValidationResult{
		Version:      parseResult.Version,
		OASVersion:   parseResult.OASVersion,
		Errors:       make([]ValidationError, 0, defaultErrorCapacity),
		Warnings:     make([]ValidationError, 0, defaultWarningCapacity),
		LoadTime:     parseResult.LoadTime,
		SourceSize:   parseResult.SourceSize,
		Stats:        parseResult.Stats,
		Document:     parseResult.Document,
		SourceFormat: parseResult.SourceFormat,
		SourcePath:   parseResult.SourcePath,
	}

	// Build reference tracker for operation context
	switch doc := parseResult.Document.(type) {
	case *parser.OAS3Document:
		v.refTracker = buildRefTrackerOAS3(doc)
	case *parser.OAS2Document:
		v.refTracker = buildRefTrackerOAS2(doc)
	}

	// Add parser errors to validation result
	for _, parseErr := range parseResult.Errors {
		result.Errors = append(result.Errors, ValidationError{
			Path:     "document",
			Message:  parseErr.Error(),
			Severity: SeverityError,
		})
	}

	// Add parser warnings to validation result
	for _, warning := range parseResult.Warnings {
		result.Warnings = append(result.Warnings, ValidationError{
			Path:     "document",
			Message:  warning,
			Severity: SeverityWarning,
		})
	}

	// Perform additional validation based on OAS version
	// Check both version and document type to ensure consistency
	if parseResult.IsOAS2() {
		if doc, ok := parseResult.OAS2Document(); ok {
			v.validateOAS2(doc, result)
		} else {
			return nil, fmt.Errorf("validator: failed to cast document to OAS2Document")
		}
	} else if parseResult.IsOAS3() {
		if doc, ok := parseResult.OAS3Document(); ok {
			v.validateOAS3(doc, result)
		} else {
			return nil, fmt.Errorf("validator: failed to cast document to OAS3Document")
		}
	} else {
		// in reality this should never happen, since the parser's `Parse` would have errored as well
		return nil, fmt.Errorf("validator: unsupported OAS version: %s", parseResult.OASVersion)
	}

	// Calculate counts
	result.ErrorCount = len(result.Errors)
	result.WarningCount = len(result.Warnings)
	result.Valid = result.ErrorCount == 0

	// Filter warnings if not included
	if !v.IncludeWarnings {
		result.Warnings = nil
		result.WarningCount = 0
	}

	return result, nil
}

// Validate validates an OpenAPI specification file
func (v *Validator) Validate(specPath string) (*ValidationResult, error) {
	// Create parser and configure it
	p := parser.New()
	p.ValidateStructure = v.ValidateStructure
	if v.UserAgent != "" {
		p.UserAgent = v.UserAgent
	}

	// Parse the document
	parseResult, err := p.Parse(specPath)
	if err != nil {
		return nil, fmt.Errorf("validator: failed to parse specification: %w", err)
	}

	return v.ValidateParsed(*parseResult)
}
