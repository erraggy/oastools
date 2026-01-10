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
	// FixTypePrunedEmptyPath indicates an empty path item was removed
	FixTypePrunedEmptyPath FixType = "pruned-empty-path"
	// FixTypePrunedUnusedSchema indicates an orphaned schema was removed
	FixTypePrunedUnusedSchema FixType = "pruned-unused-schema"
	// FixTypeRenamedGenericSchema indicates a generic type name was simplified
	FixTypeRenamedGenericSchema FixType = "renamed-generic-schema"
	// FixTypeEnumCSVExpanded indicates a CSV enum string was expanded to individual values
	FixTypeEnumCSVExpanded FixType = "enum-csv-expanded"
	// FixTypeDuplicateOperationId indicates a duplicate operationId was renamed
	FixTypeDuplicateOperationId FixType = "duplicate-operation-id"
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
	// Line is the 1-based line number in the source file (0 if unknown)
	Line int
	// Column is the 1-based column number in the source file (0 if unknown)
	Column int
	// File is the source file path (empty for main document)
	File string
}

// HasLocation returns true if this fix has source location information.
func (f Fix) HasLocation() bool {
	return f.Line > 0
}

// Location returns an IDE-friendly location string.
// Returns "file:line:column" if file is set, "line:column" if only line is set,
// or the Path if location is unknown.
func (f Fix) Location() string {
	if f.Line == 0 {
		return f.Path
	}
	if f.File != "" {
		return fmt.Sprintf("%s:%d:%d", f.File, f.Line, f.Column)
	}
	return fmt.Sprintf("%d:%d", f.Line, f.Column)
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

// ToParseResult converts the FixResult to a ParseResult for use with
// other packages like validator, converter, joiner, and differ.
// The returned ParseResult has Document populated but Data is nil
// (consumers use Document, not Data).
// Errors and Warnings are empty slices since fixes are informational,
// not validation errors.
func (r *FixResult) ToParseResult() *parser.ParseResult {
	sourcePath := r.SourcePath
	if sourcePath == "" {
		sourcePath = "fixer"
	}
	warnings := make([]string, 0)
	if r.Document == nil {
		warnings = append(warnings, "fixer: ToParseResult: Document is nil, downstream operations may fail")
	}
	return &parser.ParseResult{
		SourcePath:   sourcePath,
		SourceFormat: r.SourceFormat,
		Version:      r.SourceVersion,
		OASVersion:   r.SourceOASVersion,
		Document:     r.Document,
		Errors:       make([]error, 0),
		Warnings:     warnings,
		Stats:        r.Stats,
	}
}

// Fixer handles automatic fixing of OAS validation issues
type Fixer struct {
	// InferTypes enables type inference for path parameters based on naming conventions.
	// When true, parameter names ending in "id"/"Id"/"ID" become integer type,
	// names containing "uuid"/"guid" become string with format uuid,
	// and all others become string type.
	InferTypes bool
	// EnabledFixes specifies which fix types to apply.
	// Defaults to only FixTypeMissingPathParameter for performance.
	// Set to include other FixType values to enable additional fixes.
	// Set to empty slice ([]FixType{}) or nil to enable all fix types
	// (for backward compatibility with pre-v1.28.1 behavior).
	EnabledFixes []FixType
	// UserAgent is the User-Agent string used when fetching URLs.
	// Defaults to "oastools" if not set.
	UserAgent string
	// SourceMap provides source location lookup for fix issues.
	// When set, fixes will include Line, Column, and File information.
	SourceMap *parser.SourceMap
	// GenericNamingConfig configures how generic type names are transformed
	// when fixing invalid schema names (e.g., Response[User] → ResponseOfUser).
	GenericNamingConfig GenericNamingConfig
	// OperationIdNamingConfig configures how duplicate operationId values are renamed.
	// Uses template-based naming with placeholders like {operationId}, {method}, {path}, {n}.
	OperationIdNamingConfig OperationIdNamingConfig
	// DryRun when true, collects fixes without modifying the document.
	// Useful for previewing what would be changed.
	DryRun bool
}

// New creates a new Fixer instance with default settings
func New() *Fixer {
	return &Fixer{
		InferTypes:              false,
		EnabledFixes:            []FixType{FixTypeMissingPathParameter}, // only missing params by default
		GenericNamingConfig:     DefaultGenericNamingConfig(),
		OperationIdNamingConfig: DefaultOperationIdNamingConfig(),
		DryRun:                  false,
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
	inferTypes              bool
	enabledFixes            []FixType
	userAgent               string
	genericNamingConfig     GenericNamingConfig
	operationIdNamingConfig OperationIdNamingConfig
	dryRun                  bool

	// Source map for line/column tracking
	sourceMap *parser.SourceMap
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
		InferTypes:              cfg.inferTypes,
		EnabledFixes:            cfg.enabledFixes,
		UserAgent:               cfg.userAgent,
		SourceMap:               cfg.sourceMap,
		GenericNamingConfig:     cfg.genericNamingConfig,
		OperationIdNamingConfig: cfg.operationIdNamingConfig,
		DryRun:                  cfg.dryRun,
	}

	// Route to appropriate fix method based on input source
	// Parsed input is checked first as it's the preferred high-performance path
	if cfg.parsed != nil {
		return f.FixParsed(*cfg.parsed)
	}
	if cfg.filePath != nil {
		return f.Fix(*cfg.filePath)
	}

	// Should never reach here due to validation in applyOptions
	return nil, fmt.Errorf("fixer: no input source specified")
}

// applyOptions applies option functions and validates configuration
func applyOptions(opts ...Option) (*fixConfig, error) {
	cfg := &fixConfig{
		// Set defaults
		inferTypes:              false,
		enabledFixes:            []FixType{FixTypeMissingPathParameter}, // only missing params by default
		userAgent:               "",
		genericNamingConfig:     DefaultGenericNamingConfig(),
		operationIdNamingConfig: DefaultOperationIdNamingConfig(),
		dryRun:                  false,
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
		return nil, fmt.Errorf("fixer: no input source specified: use WithFilePath or WithParsed")
	}
	if sources > 1 {
		return nil, fmt.Errorf("fixer: multiple input sources specified: use only one of WithFilePath or WithParsed")
	}

	return cfg, nil
}

// WithFilePath specifies the file path (local file or URL) to fix
func WithFilePath(path string) Option {
	return func(cfg *fixConfig) error {
		if path == "" {
			return fmt.Errorf("fixer: file path cannot be empty")
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

// WithSourceMap provides a SourceMap for populating line/column information
// in fix records. When set, fixes will include source location details
// that enable IDE-friendly error reporting.
func WithSourceMap(sm *parser.SourceMap) Option {
	return func(cfg *fixConfig) error {
		cfg.sourceMap = sm
		return nil
	}
}

// WithGenericNaming sets the naming strategy for fixing invalid schema names.
// This applies to schemas with names like "Response[User]" that contain
// characters requiring URL encoding in $ref values.
func WithGenericNaming(strategy GenericNamingStrategy) Option {
	return func(cfg *fixConfig) error {
		cfg.genericNamingConfig.Strategy = strategy
		return nil
	}
}

// WithGenericNamingConfig sets the full generic naming configuration.
func WithGenericNamingConfig(config GenericNamingConfig) Option {
	return func(cfg *fixConfig) error {
		cfg.genericNamingConfig = config
		return nil
	}
}

// WithOperationIdNamingConfig sets the configuration for renaming duplicate operationIds.
// The config includes a template with placeholders: {operationId}, {method}, {path}, {tag}, {tags}, {n}.
func WithOperationIdNamingConfig(config OperationIdNamingConfig) Option {
	return func(cfg *fixConfig) error {
		if err := ParseOperationIdNamingTemplate(config.Template); err != nil {
			return err
		}
		cfg.operationIdNamingConfig = config
		return nil
	}
}

// WithDryRun enables dry-run mode, which collects fixes without
// actually modifying the document. Useful for previewing changes.
func WithDryRun(dryRun bool) Option {
	return func(cfg *fixConfig) error {
		cfg.dryRun = dryRun
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
		return true // if explicitly set to empty/nil, enable all fixes
	}
	for _, ft := range f.EnabledFixes {
		if ft == fixType {
			return true
		}
	}
	return false
}

// populateFixLocation fills in Line/Column/File from the SourceMap if available.
// The path parameter is the JSON path in the source document.
func (f *Fixer) populateFixLocation(fix *Fix) {
	if f.SourceMap == nil {
		return
	}

	// Convert path format if needed (fixer uses dotted paths like "paths./users.get",
	// while SourceMap uses JSON path notation like "$.paths./users.get")
	jsonPath := fix.Path
	if !hasJSONPathPrefix(jsonPath) {
		jsonPath = "$." + fix.Path
	}

	loc := f.SourceMap.Get(jsonPath)
	if loc.IsKnown() {
		fix.Line = loc.Line
		fix.Column = loc.Column
		fix.File = loc.File
	}
}

// hasJSONPathPrefix returns true if the path already has a JSON path prefix.
func hasJSONPathPrefix(path string) bool {
	return len(path) > 0 && path[0] == '$'
}
