package differ

import (
	"fmt"

	"github.com/erraggy/oastools/internal/severity"
	"github.com/erraggy/oastools/parser"
)

// DiffMode indicates the type of diff operation to perform
type DiffMode int

const (
	// ModeSimple reports all semantic differences between documents
	ModeSimple DiffMode = iota
	// ModeBreaking categorizes changes and identifies breaking API changes
	ModeBreaking
)

// ChangeType indicates whether a change is an addition, removal, or modification
type ChangeType string

const (
	// ChangeTypeAdded indicates a new element was added
	ChangeTypeAdded ChangeType = "added"
	// ChangeTypeRemoved indicates an element was removed
	ChangeTypeRemoved ChangeType = "removed"
	// ChangeTypeModified indicates an existing element was changed
	ChangeTypeModified ChangeType = "modified"
)

// ChangeCategory indicates which part of the spec was changed
type ChangeCategory string

const (
	// CategoryEndpoint indicates a path/endpoint change
	CategoryEndpoint ChangeCategory = "endpoint"
	// CategoryOperation indicates an HTTP operation change
	CategoryOperation ChangeCategory = "operation"
	// CategoryParameter indicates a parameter change
	CategoryParameter ChangeCategory = "parameter"
	// CategoryRequestBody indicates a request body change
	CategoryRequestBody ChangeCategory = "request_body"
	// CategoryResponse indicates a response change
	CategoryResponse ChangeCategory = "response"
	// CategorySchema indicates a schema/definition change
	CategorySchema ChangeCategory = "schema"
	// CategorySecurity indicates a security scheme change
	CategorySecurity ChangeCategory = "security"
	// CategoryServer indicates a server change
	CategoryServer ChangeCategory = "server"
	// CategoryInfo indicates metadata change (info, contact, license, etc.)
	CategoryInfo ChangeCategory = "info"
	// CategoryExtension indicates a specification extension (x-*) change
	CategoryExtension ChangeCategory = "extension"
)

// Severity indicates the severity level of a change
type Severity = severity.Severity

const (
	// SeverityInfo indicates informational changes (additions, relaxed constraints)
	SeverityInfo = severity.SeverityInfo
	// SeverityWarning indicates potentially problematic changes
	SeverityWarning = severity.SeverityWarning
	// SeverityError indicates breaking changes (removed features, stricter constraints)
	SeverityError = severity.SeverityError
	// SeverityCritical indicates critical breaking changes (removed endpoints, operations)
	SeverityCritical = severity.SeverityCritical
)

// Change represents a single difference between two OpenAPI specifications
type Change struct {
	// Path is the JSON path to the changed element (e.g., "paths./pets.get")
	Path string
	// Type indicates if this is an addition, removal, or modification
	Type ChangeType
	// Category indicates which part of the spec was changed
	Category ChangeCategory
	// Severity indicates the impact level (only used in ModeBreaking)
	Severity Severity
	// OldValue is the value in the source document (nil for additions)
	OldValue any
	// NewValue is the value in the target document (nil for removals)
	NewValue any
	// Message is a human-readable description of the change
	Message string
}

// String returns a formatted string representation of the change
func (c Change) String() string {
	var symbol string
	switch c.Severity {
	case SeverityError, SeverityCritical:
		symbol = "✗"
	case SeverityWarning:
		symbol = "⚠"
	case SeverityInfo:
		symbol = "ℹ"
	default:
		symbol = "·"
	}

	typeStr := ""
	switch c.Type {
	case ChangeTypeAdded:
		typeStr = "added"
	case ChangeTypeRemoved:
		typeStr = "removed"
	case ChangeTypeModified:
		typeStr = "modified"
	}

	return fmt.Sprintf("%s %s [%s] %s: %s", symbol, c.Path, typeStr, c.Category, c.Message)
}

// DiffResult contains the results of comparing two OpenAPI specifications
type DiffResult struct {
	// SourceVersion is the source document's OAS version string
	SourceVersion string
	// SourceOASVersion is the enumerated source OAS version
	SourceOASVersion parser.OASVersion
	// SourceStats contains statistical information about the source document
	SourceStats parser.DocumentStats
	// SourceSize is the size of the source document in bytes
	SourceSize int64
	// TargetVersion is the target document's OAS version string
	TargetVersion string
	// TargetOASVersion is the enumerated target OAS version
	TargetOASVersion parser.OASVersion
	// TargetStats contains statistical information about the target document
	TargetStats parser.DocumentStats
	// TargetSize is the size of the target document in bytes
	TargetSize int64
	// Changes contains all detected changes
	Changes []Change
	// BreakingCount is the number of breaking changes (Critical + Error severity)
	BreakingCount int
	// WarningCount is the number of warnings
	WarningCount int
	// InfoCount is the number of informational changes
	InfoCount int
	// HasBreakingChanges is true if any breaking changes were detected
	HasBreakingChanges bool
}

// Differ handles OpenAPI specification comparison
type Differ struct {
	// Mode determines the type of diff operation (Simple or Breaking)
	Mode DiffMode
	// IncludeInfo determines whether to include informational changes
	IncludeInfo bool
	// UserAgent is the User-Agent string used when fetching URLs
	// Defaults to "oastools" if not set
	UserAgent string
}

// New creates a new Differ instance with default settings
func New() *Differ {
	return &Differ{
		Mode:        ModeSimple,
		IncludeInfo: true,
	}
}

// Option is a function that configures a diff operation
type Option func(*diffConfig) error

// diffConfig holds configuration for a diff operation
type diffConfig struct {
	// Input sources (exactly one source and one target must be set)
	sourceFilePath *string
	sourceParsed   *parser.ParseResult
	targetFilePath *string
	targetParsed   *parser.ParseResult

	// Configuration options
	mode        DiffMode
	includeInfo bool
	userAgent   string
}

// DiffWithOptions compares two OpenAPI specifications using functional options.
// This provides a flexible, extensible API that combines input source selection
// and configuration in a single function call.
//
// Example:
//
//	result, err := differ.DiffWithOptions(
//	    differ.WithSourceFilePath("api-v1.yaml"),
//	    differ.WithTargetFilePath("api-v2.yaml"),
//	    differ.WithMode(differ.ModeBreaking),
//	)
func DiffWithOptions(opts ...Option) (*DiffResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("differ: invalid options: %w", err)
	}

	d := &Differ{
		Mode:        cfg.mode,
		IncludeInfo: cfg.includeInfo,
		UserAgent:   cfg.userAgent,
	}

	// Determine source
	var source parser.ParseResult
	if cfg.sourceFilePath != nil {
		p := parser.New()
		if d.UserAgent != "" {
			p.UserAgent = d.UserAgent
		}
		sourceResult, err := p.Parse(*cfg.sourceFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse source: %w", err)
		}
		if len(sourceResult.Errors) > 0 {
			return nil, fmt.Errorf("source document has %d parse error(s)", len(sourceResult.Errors))
		}
		source = *sourceResult
	} else {
		source = *cfg.sourceParsed
	}

	// Determine target
	var target parser.ParseResult
	if cfg.targetFilePath != nil {
		p := parser.New()
		if d.UserAgent != "" {
			p.UserAgent = d.UserAgent
		}
		targetResult, err := p.Parse(*cfg.targetFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse target: %w", err)
		}
		if len(targetResult.Errors) > 0 {
			return nil, fmt.Errorf("target document has %d parse error(s)", len(targetResult.Errors))
		}
		target = *targetResult
	} else {
		target = *cfg.targetParsed
	}

	return d.DiffParsed(source, target)
}

// applyOptions applies option functions and validates configuration
func applyOptions(opts ...Option) (*diffConfig, error) {
	cfg := &diffConfig{
		// Set defaults to match existing behavior
		mode:        ModeSimple,
		includeInfo: true,
		userAgent:   "",
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate exactly one source is specified
	sourceCount := 0
	if cfg.sourceFilePath != nil {
		sourceCount++
	}
	if cfg.sourceParsed != nil {
		sourceCount++
	}

	if sourceCount == 0 {
		return nil, fmt.Errorf("must specify a source (use WithSourceFilePath or WithSourceParsed)")
	}
	if sourceCount > 1 {
		return nil, fmt.Errorf("must specify exactly one source")
	}

	// Validate exactly one target is specified
	targetCount := 0
	if cfg.targetFilePath != nil {
		targetCount++
	}
	if cfg.targetParsed != nil {
		targetCount++
	}

	if targetCount == 0 {
		return nil, fmt.Errorf("must specify a target (use WithTargetFilePath or WithTargetParsed)")
	}
	if targetCount > 1 {
		return nil, fmt.Errorf("must specify exactly one target")
	}

	return cfg, nil
}

// WithSourceFilePath specifies a file path or URL as the source document
func WithSourceFilePath(path string) Option {
	return func(cfg *diffConfig) error {
		cfg.sourceFilePath = &path
		return nil
	}
}

// WithSourceParsed specifies a parsed ParseResult as the source document
func WithSourceParsed(result parser.ParseResult) Option {
	return func(cfg *diffConfig) error {
		cfg.sourceParsed = &result
		return nil
	}
}

// WithTargetFilePath specifies a file path or URL as the target document
func WithTargetFilePath(path string) Option {
	return func(cfg *diffConfig) error {
		cfg.targetFilePath = &path
		return nil
	}
}

// WithTargetParsed specifies a parsed ParseResult as the target document
func WithTargetParsed(result parser.ParseResult) Option {
	return func(cfg *diffConfig) error {
		cfg.targetParsed = &result
		return nil
	}
}

// WithMode sets the diff mode (Simple or Breaking)
// Default: ModeSimple
func WithMode(mode DiffMode) Option {
	return func(cfg *diffConfig) error {
		cfg.mode = mode
		return nil
	}
}

// WithIncludeInfo enables or disables informational changes
// Default: true
func WithIncludeInfo(enabled bool) Option {
	return func(cfg *diffConfig) error {
		cfg.includeInfo = enabled
		return nil
	}
}

// WithUserAgent sets the User-Agent string for HTTP requests
// Default: "" (uses parser default)
func WithUserAgent(ua string) Option {
	return func(cfg *diffConfig) error {
		cfg.userAgent = ua
		return nil
	}
}

// Diff compares two OpenAPI specification files
func (d *Differ) Diff(sourcePath, targetPath string) (*DiffResult, error) {
	// Create parser and set UserAgent if specified
	p := parser.New()
	if d.UserAgent != "" {
		p.UserAgent = d.UserAgent
	}

	// Parse source document
	sourceResult, err := p.Parse(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source specification: %w", err)
	}
	if len(sourceResult.Errors) > 0 {
		return nil, fmt.Errorf("source document has %d parse error(s), cannot diff", len(sourceResult.Errors))
	}

	// Parse target document
	targetResult, err := p.Parse(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target specification: %w", err)
	}
	if len(targetResult.Errors) > 0 {
		return nil, fmt.Errorf("target document has %d parse error(s), cannot diff", len(targetResult.Errors))
	}

	return d.DiffParsed(*sourceResult, *targetResult)
}

// DiffParsed compares two already-parsed OpenAPI specifications
func (d *Differ) DiffParsed(source, target parser.ParseResult) (*DiffResult, error) {
	// Initialize result
	result := &DiffResult{
		SourceVersion:    source.Version,
		SourceOASVersion: source.OASVersion,
		SourceStats:      source.Stats,
		SourceSize:       source.SourceSize,
		TargetVersion:    target.Version,
		TargetOASVersion: target.OASVersion,
		TargetStats:      target.Stats,
		TargetSize:       target.SourceSize,
		Changes:          make([]Change, 0),
	}

	// Perform unified diff (handles both ModeSimple and ModeBreaking)
	d.diffUnified(source, target, result)

	// Filter out info-level changes if not requested
	if !d.IncludeInfo {
		filtered := make([]Change, 0, len(result.Changes))
		for _, change := range result.Changes {
			if change.Severity != SeverityInfo {
				filtered = append(filtered, change)
			}
		}
		result.Changes = filtered
	}

	// Calculate counts
	for _, change := range result.Changes {
		switch change.Severity {
		case SeverityCritical, SeverityError:
			result.BreakingCount++
		case SeverityWarning:
			result.WarningCount++
		case SeverityInfo:
			result.InfoCount++
		}
	}

	result.HasBreakingChanges = result.BreakingCount > 0

	return result, nil
}
