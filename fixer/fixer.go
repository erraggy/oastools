package fixer

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// FixType identifies the type of fix applied
type FixType string

const (
	// FixTypeMissingPathParameter indicates a missing path parameter was added
	FixTypeMissingPathParameter FixType = "missing-path-parameter"
)

// Fix represents a single fix applied to the document
type Fix struct {
	// Type identifies the category of fix
	Type FixType
	// Path is the JSON path to the fixed location (e.g., "paths./users/{id}.get.parameters")
	Path string
	// Description is a human-readable description of the fix
	Description string
	// Before is the state before the fix (nil if adding new element)
	Before any
	// After is the value that was added or changed
	After any
}

// FixResult contains the results of a fix operation
type FixResult struct {
	// Document contains the fixed document (*parser.OAS2Document or *parser.OAS3Document)
	Document any
	// SourceVersion is the detected source OAS version string
	SourceVersion string
	// SourceOASVersion is the enumerated source OAS version
	SourceOASVersion parser.OASVersion
	// SourceFormat is the format of the source file (JSON or YAML)
	SourceFormat parser.SourceFormat
	// SourcePath is the path to the source file
	SourcePath string
	// Fixes contains all fixes applied
	Fixes []Fix
	// FixCount is the total number of fixes applied
	FixCount int
	// Success is true if fixing completed without errors
	Success bool
	// Stats contains statistical information about the document
	Stats parser.DocumentStats
}

// HasFixes returns true if any fixes were applied
func (r *FixResult) HasFixes() bool {
	return r.FixCount > 0
}

// Fixer handles automatic fixing of OAS validation issues
type Fixer struct {
	// InferTypes enables type inference for path parameters based on naming conventions.
	// When true, parameter names ending in "id"/"Id"/"ID" become integer type,
	// names containing "uuid"/"guid" become string with format uuid,
	// and all others become string type.
	InferTypes bool
	// EnabledFixes specifies which fix types to apply.
	// If nil or empty, all fix types are enabled.
	EnabledFixes []FixType
	// UserAgent is the User-Agent string used when fetching URLs.
	// Defaults to "oastools" if not set.
	UserAgent string
}

// New creates a new Fixer instance with default settings
func New() *Fixer {
	return &Fixer{
		InferTypes:   false,
		EnabledFixes: nil, // all fixes enabled
	}
}

// Option is a function that configures a fix operation
type Option func(*fixConfig) error

// fixConfig holds configuration for a fix operation
type fixConfig struct {
	// Input source (exactly one must be set)
	filePath *string
	parsed   *parser.ParseResult

	// Configuration options
	inferTypes   bool
	enabledFixes []FixType
	userAgent    string
}

// FixWithOptions fixes an OpenAPI specification using functional options.
// This provides a flexible, extensible API that combines input source selection
// and configuration in a single function call.
//
// Example:
//
//	result, err := fixer.FixWithOptions(
//	    fixer.WithFilePath("openapi.yaml"),
//	    fixer.WithInferTypes(true),
//	)
func FixWithOptions(opts ...Option) (*FixResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("fixer: invalid options: %w", err)
	}

	f := &Fixer{
		InferTypes:   cfg.inferTypes,
		EnabledFixes: cfg.enabledFixes,
		UserAgent:    cfg.userAgent,
	}

	// Route to appropriate fix method based on input source
	if cfg.filePath != nil {
		return f.Fix(*cfg.filePath)
	}
	if cfg.parsed != nil {
		return f.FixParsed(*cfg.parsed)
	}

	// Should never reach here due to validation in applyOptions
	return nil, fmt.Errorf("fixer: no input source specified")
}

// applyOptions applies option functions and validates configuration
func applyOptions(opts ...Option) (*fixConfig, error) {
	cfg := &fixConfig{
		// Set defaults
		inferTypes:   false,
		enabledFixes: nil,
		userAgent:    "",
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate that exactly one input source is specified
	sources := 0
	if cfg.filePath != nil {
		sources++
	}
	if cfg.parsed != nil {
		sources++
	}

	if sources == 0 {
		return nil, fmt.Errorf("no input source specified: use WithFilePath or WithParsed")
	}
	if sources > 1 {
		return nil, fmt.Errorf("multiple input sources specified: use only one of WithFilePath or WithParsed")
	}

	return cfg, nil
}

// WithFilePath specifies the file path (local file or URL) to fix
func WithFilePath(path string) Option {
	return func(cfg *fixConfig) error {
		if path == "" {
			return fmt.Errorf("file path cannot be empty")
		}
		cfg.filePath = &path
		return nil
	}
}

// WithParsed specifies an already-parsed specification to fix
func WithParsed(result parser.ParseResult) Option {
	return func(cfg *fixConfig) error {
		cfg.parsed = &result
		return nil
	}
}

// WithInferTypes enables type inference for path parameters
func WithInferTypes(infer bool) Option {
	return func(cfg *fixConfig) error {
		cfg.inferTypes = infer
		return nil
	}
}

// WithEnabledFixes specifies which fix types to apply
func WithEnabledFixes(fixes ...FixType) Option {
	return func(cfg *fixConfig) error {
		cfg.enabledFixes = fixes
		return nil
	}
}

// WithUserAgent sets the User-Agent string for HTTP requests
func WithUserAgent(userAgent string) Option {
	return func(cfg *fixConfig) error {
		cfg.userAgent = userAgent
		return nil
	}
}

// Fix fixes an OpenAPI specification file and returns the result
func (f *Fixer) Fix(specPath string) (*FixResult, error) {
	// Parse the specification
	p := parser.New()
	if f.UserAgent != "" {
		p.UserAgent = f.UserAgent
	}

	parseResult, err := p.Parse(specPath)
	if err != nil {
		return nil, fmt.Errorf("fixer: failed to parse specification: %w", err)
	}

	return f.FixParsed(*parseResult)
}

// FixParsed fixes an already-parsed OpenAPI specification.
// The fixer operates on the parsed document structure and does not require
// a valid specification - it will attempt to fix issues even if validation
// errors exist (since that's often the reason for using the fixer).
func (f *Fixer) FixParsed(parseResult parser.ParseResult) (*FixResult, error) {
	result := &FixResult{
		SourceVersion:    parseResult.Version,
		SourceOASVersion: parseResult.OASVersion,
		SourceFormat:     parseResult.SourceFormat,
		SourcePath:       parseResult.SourcePath,
		Stats:            parseResult.Stats,
		Fixes:            make([]Fix, 0),
		Success:          true,
	}

	// Only fail if the document couldn't be parsed at all
	if parseResult.Document == nil {
		return nil, fmt.Errorf("fixer: specification could not be parsed (nil document)")
	}

	// Route based on OAS version
	if parseResult.OASVersion == parser.OASVersion20 {
		return f.fixOAS2(parseResult, result)
	}
	return f.fixOAS3(parseResult, result)
}

// isFixEnabled checks if a fix type is enabled.
// The fixType parameter is used for extensibility when more fix types are added.
//
//nolint:unparam // fixType is parameterized for future extensibility
func (f *Fixer) isFixEnabled(fixType FixType) bool {
	if len(f.EnabledFixes) == 0 {
		return true // all fixes enabled by default
	}
	for _, ft := range f.EnabledFixes {
		if ft == fixType {
			return true
		}
	}
	return false
}

// fixOAS2 applies fixes to an OAS 2.0 document
func (f *Fixer) fixOAS2(parseResult parser.ParseResult, result *FixResult) (*FixResult, error) {
	// Extract the OAS 2.0 document from the generic Document field
	srcDoc, ok := parseResult.Document.(*parser.OAS2Document)
	if !ok {
		return nil, fmt.Errorf("fixer: expected *parser.OAS2Document, got %T", parseResult.Document)
	}

	// Deep copy the document to avoid mutating the original
	doc, err := deepCopyOAS2Document(srcDoc)
	if err != nil {
		return nil, fmt.Errorf("fixer: failed to copy document: %w", err)
	}

	// Apply enabled fixes
	if f.isFixEnabled(FixTypeMissingPathParameter) {
		f.fixMissingPathParametersOAS2(doc, result)
	}

	// Update result
	result.Document = doc
	result.FixCount = len(result.Fixes)

	return result, nil
}

// fixOAS3 applies fixes to an OAS 3.x document
func (f *Fixer) fixOAS3(parseResult parser.ParseResult, result *FixResult) (*FixResult, error) {
	// Extract the OAS 3.x document from the generic Document field
	srcDoc, ok := parseResult.Document.(*parser.OAS3Document)
	if !ok {
		return nil, fmt.Errorf("fixer: expected *parser.OAS3Document, got %T", parseResult.Document)
	}

	// Deep copy the document to avoid mutating the original
	doc, err := deepCopyOAS3Document(srcDoc)
	if err != nil {
		return nil, fmt.Errorf("fixer: failed to copy document: %w", err)
	}

	// Apply enabled fixes
	if f.isFixEnabled(FixTypeMissingPathParameter) {
		f.fixMissingPathParametersOAS3(doc, result)
	}

	// Update result
	result.Document = doc
	result.FixCount = len(result.Fixes)

	return result, nil
}
