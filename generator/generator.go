package generator

import (
	"fmt"
	"time"

	"github.com/erraggy/oastools/internal/issues"
	"github.com/erraggy/oastools/internal/severity"
	"github.com/erraggy/oastools/parser"
)

// Severity indicates the severity level of a generation issue
type Severity = severity.Severity

const (
	// SeverityInfo indicates informational messages about generation choices
	SeverityInfo = severity.SeverityInfo
	// SeverityWarning indicates features that may not generate perfectly
	SeverityWarning = severity.SeverityWarning
	// SeverityError indicates validation errors
	SeverityError = severity.SeverityError
	// SeverityCritical indicates features that cannot be generated
	SeverityCritical = severity.SeverityCritical
)

// GenerateIssue represents a single generation issue or limitation
type GenerateIssue = issues.Issue

// GeneratedFile represents a single generated file
type GeneratedFile struct {
	// Name is the file name (e.g., "types.go", "client.go")
	Name string
	// Content is the generated Go source code
	Content []byte
}

// GenerateResult contains the results of generating code from an OpenAPI specification
type GenerateResult struct {
	// Files contains all generated files
	Files []GeneratedFile
	// SourceVersion is the detected source OAS version string
	SourceVersion string
	// SourceOASVersion is the enumerated source OAS version
	SourceOASVersion parser.OASVersion
	// SourceFormat is the format of the source file (JSON or YAML)
	SourceFormat parser.SourceFormat
	// PackageName is the Go package name used in generation
	PackageName string
	// Issues contains all generation issues grouped by severity
	Issues []GenerateIssue
	// InfoCount is the total number of info messages
	InfoCount int
	// WarningCount is the total number of warnings
	WarningCount int
	// CriticalCount is the total number of critical issues
	CriticalCount int
	// Success is true if generation completed without critical issues
	Success bool
	// LoadTime is the time taken to load the source data
	LoadTime time.Duration
	// GenerateTime is the time taken to generate code
	GenerateTime time.Duration
	// SourceSize is the size of the source data in bytes
	SourceSize int64
	// Stats contains statistical information about the source document
	Stats parser.DocumentStats
	// GeneratedTypes is the count of types generated
	GeneratedTypes int
	// GeneratedOperations is the count of operations generated
	GeneratedOperations int
}

// HasCriticalIssues returns true if there are any critical issues
func (r *GenerateResult) HasCriticalIssues() bool {
	return r.CriticalCount > 0
}

// HasWarnings returns true if there are any warnings
func (r *GenerateResult) HasWarnings() bool {
	return r.WarningCount > 0
}

// GetFile returns the generated file with the given name, or nil if not found
func (r *GenerateResult) GetFile(name string) *GeneratedFile {
	for i := range r.Files {
		if r.Files[i].Name == name {
			return &r.Files[i]
		}
	}
	return nil
}

// Generator handles code generation from OpenAPI specifications
type Generator struct {
	// PackageName is the Go package name for generated code
	// If empty, defaults to "api"
	PackageName string

	// GenerateClient enables HTTP client generation
	GenerateClient bool

	// GenerateServer enables server interface generation
	GenerateServer bool

	// GenerateTypes enables schema/model type generation
	// This is always true when either client or server generation is enabled
	GenerateTypes bool

	// UsePointers uses pointer types for optional fields
	// Default: true
	UsePointers bool

	// IncludeValidation adds validation tags to generated structs
	// Default: true
	IncludeValidation bool

	// StrictMode causes generation to fail on any issues (even warnings)
	StrictMode bool

	// IncludeInfo determines whether to include informational messages
	IncludeInfo bool

	// UserAgent is the User-Agent string used when fetching URLs
	UserAgent string
}

// New creates a new Generator instance with default settings
func New() *Generator {
	return &Generator{
		PackageName:       "api",
		GenerateClient:    false,
		GenerateServer:    false,
		GenerateTypes:     true,
		UsePointers:       true,
		IncludeValidation: true,
		StrictMode:        false,
		IncludeInfo:       true,
	}
}

// Option is a function that configures a generate operation
type Option func(*generateConfig) error

// generateConfig holds configuration for a generate operation
type generateConfig struct {
	// Input source (exactly one must be set)
	filePath *string
	parsed   *parser.ParseResult

	// Configuration options
	packageName       string
	generateClient    bool
	generateServer    bool
	generateTypes     bool
	usePointers       bool
	includeValidation bool
	strictMode        bool
	includeInfo       bool
	userAgent         string
}

// GenerateWithOptions generates code from an OpenAPI specification using functional options.
// This provides a flexible, extensible API that combines input source selection
// and configuration in a single function call.
//
// Example:
//
//	result, err := generator.GenerateWithOptions(
//	    generator.WithFilePath("openapi.yaml"),
//	    generator.WithPackageName("petstore"),
//	    generator.WithClient(true),
//	)
func GenerateWithOptions(opts ...Option) (*GenerateResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("generator: invalid options: %w", err)
	}

	g := &Generator{
		PackageName:       cfg.packageName,
		GenerateClient:    cfg.generateClient,
		GenerateServer:    cfg.generateServer,
		GenerateTypes:     cfg.generateTypes,
		UsePointers:       cfg.usePointers,
		IncludeValidation: cfg.includeValidation,
		StrictMode:        cfg.strictMode,
		IncludeInfo:       cfg.includeInfo,
		UserAgent:         cfg.userAgent,
	}

	// Route to appropriate generation method based on input source
	if cfg.filePath != nil {
		return g.Generate(*cfg.filePath)
	}
	if cfg.parsed != nil {
		return g.GenerateParsed(*cfg.parsed)
	}

	// Should never reach here due to validation in applyOptions
	return nil, fmt.Errorf("generator: no input source specified")
}

// applyOptions applies option functions and validates configuration
func applyOptions(opts ...Option) (*generateConfig, error) {
	cfg := &generateConfig{
		// Set defaults
		packageName:       "api",
		generateClient:    false,
		generateServer:    false,
		generateTypes:     true,
		usePointers:       true,
		includeValidation: true,
		strictMode:        false,
		includeInfo:       true,
		userAgent:         "",
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
	if cfg.parsed != nil {
		sourceCount++
	}

	if sourceCount == 0 {
		return nil, fmt.Errorf("must specify an input source (use WithFilePath or WithParsed)")
	}
	if sourceCount > 1 {
		return nil, fmt.Errorf("must specify exactly one input source")
	}

	// Ensure types are generated if client or server is enabled
	if cfg.generateClient || cfg.generateServer {
		cfg.generateTypes = true
	}

	return cfg, nil
}

// WithFilePath specifies a file path or URL as the input source
func WithFilePath(path string) Option {
	return func(cfg *generateConfig) error {
		cfg.filePath = &path
		return nil
	}
}

// WithParsed specifies a parsed ParseResult as the input source
func WithParsed(result parser.ParseResult) Option {
	return func(cfg *generateConfig) error {
		cfg.parsed = &result
		return nil
	}
}

// WithPackageName specifies the Go package name for generated code
// Default: "api"
func WithPackageName(name string) Option {
	return func(cfg *generateConfig) error {
		if name == "" {
			return fmt.Errorf("package name cannot be empty")
		}
		cfg.packageName = name
		return nil
	}
}

// WithClient enables or disables HTTP client generation
// Default: false
func WithClient(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.generateClient = enabled
		return nil
	}
}

// WithServer enables or disables server interface generation
// Default: false
func WithServer(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.generateServer = enabled
		return nil
	}
}

// WithTypes enables or disables type-only generation
// Note: Types are always generated when client or server is enabled
// Default: true
func WithTypes(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.generateTypes = enabled
		return nil
	}
}

// WithPointers enables or disables pointer types for optional fields
// Default: true
func WithPointers(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.usePointers = enabled
		return nil
	}
}

// WithValidation enables or disables validation tags in generated structs
// Default: true
func WithValidation(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.includeValidation = enabled
		return nil
	}
}

// WithStrictMode enables or disables strict mode (fail on any issues)
// Default: false
func WithStrictMode(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.strictMode = enabled
		return nil
	}
}

// WithIncludeInfo enables or disables informational messages
// Default: true
func WithIncludeInfo(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.includeInfo = enabled
		return nil
	}
}

// WithUserAgent sets the User-Agent string for HTTP requests
// Default: "" (uses parser default)
func WithUserAgent(ua string) Option {
	return func(cfg *generateConfig) error {
		cfg.userAgent = ua
		return nil
	}
}

// Generate generates code from an OpenAPI specification file or URL
func (g *Generator) Generate(specPath string) (*GenerateResult, error) {
	// Create parser and set UserAgent if specified
	p := parser.New()
	if g.UserAgent != "" {
		p.UserAgent = g.UserAgent
	}

	// Parse the source document
	parseResult, err := p.Parse(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse specification: %w", err)
	}

	// Check for parse errors
	if len(parseResult.Errors) > 0 {
		return nil, fmt.Errorf("source document has %d parse error(s), cannot generate", len(parseResult.Errors))
	}

	return g.GenerateParsed(*parseResult)
}

// GenerateParsed generates code from an already-parsed OpenAPI specification
func (g *Generator) GenerateParsed(parseResult parser.ParseResult) (*GenerateResult, error) {
	startTime := time.Now()

	// Initialize result
	result := &GenerateResult{
		Files:            make([]GeneratedFile, 0),
		SourceVersion:    parseResult.Version,
		SourceOASVersion: parseResult.OASVersion,
		SourceFormat:     parseResult.SourceFormat,
		PackageName:      g.PackageName,
		Issues:           make([]GenerateIssue, 0),
		LoadTime:         parseResult.LoadTime,
		SourceSize:       parseResult.SourceSize,
		Stats:            parseResult.Stats,
	}

	// Ensure package name
	if result.PackageName == "" {
		result.PackageName = "api"
	}

	// Create code generator based on OAS version
	var cg codeGenerator
	switch {
	case parseResult.OASVersion == parser.OASVersion20:
		doc, ok := parseResult.Document.(*parser.OAS2Document)
		if !ok {
			return nil, fmt.Errorf("generator: document type mismatch for OAS 2.0")
		}
		cg = newOAS2CodeGenerator(g, doc, result)
	case parseResult.OASVersion.IsValid():
		doc, ok := parseResult.Document.(*parser.OAS3Document)
		if !ok {
			return nil, fmt.Errorf("generator: document type mismatch for OAS 3.x")
		}
		cg = newOAS3CodeGenerator(g, doc, result)
	default:
		return nil, fmt.Errorf("generator: unsupported OAS version: %s", parseResult.Version)
	}

	// Generate types if enabled
	if g.GenerateTypes || g.GenerateClient || g.GenerateServer {
		if err := cg.generateTypes(); err != nil {
			return nil, fmt.Errorf("failed to generate types: %w", err)
		}
	}

	// Generate client if enabled
	if g.GenerateClient {
		if err := cg.generateClient(); err != nil {
			return nil, fmt.Errorf("failed to generate client: %w", err)
		}
	}

	// Generate server if enabled
	if g.GenerateServer {
		if err := cg.generateServer(); err != nil {
			return nil, fmt.Errorf("failed to generate server: %w", err)
		}
	}

	// Update counts and timing
	result.GenerateTime = time.Since(startTime)
	g.updateCounts(result)
	result.Success = result.CriticalCount == 0

	// In strict mode, fail on any issues
	if g.StrictMode && (result.CriticalCount > 0 || result.WarningCount > 0) {
		return result, fmt.Errorf("generation failed in strict mode: %d critical issue(s), %d warning(s)",
			result.CriticalCount, result.WarningCount)
	}

	// Filter info messages if not included
	if !g.IncludeInfo {
		filtered := make([]GenerateIssue, 0, len(result.Issues))
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
func (g *Generator) updateCounts(result *GenerateResult) {
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

// codeGenerator is the internal interface for version-specific code generation
type codeGenerator interface {
	generateTypes() error
	generateClient() error
	generateServer() error
}
