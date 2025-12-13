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

	// File splitting options for large APIs

	// MaxLinesPerFile is the maximum lines per generated file before splitting.
	// When exceeded, files are split by tag or path prefix.
	// Default: 2000, 0 = no limit
	MaxLinesPerFile int

	// MaxTypesPerFile is the maximum types per generated file before splitting.
	// Default: 200, 0 = no limit
	MaxTypesPerFile int

	// MaxOperationsPerFile is the maximum operations per generated file before splitting.
	// Default: 100, 0 = no limit
	MaxOperationsPerFile int

	// SplitByTag enables splitting files by operation tag.
	// Default: true
	SplitByTag bool

	// SplitByPathPrefix enables splitting files by path prefix as a fallback
	// when tags are not available.
	// Default: true
	SplitByPathPrefix bool

	// Security generation options

	// GenerateSecurity enables security helper generation.
	// When true, generates ClientOption functions for each security scheme.
	// Default: true when GenerateClient is true
	GenerateSecurity bool

	// GenerateOAuth2Flows enables OAuth2 token flow helper generation.
	// Generates token acquisition, refresh, and authorization code exchange.
	// Default: false
	GenerateOAuth2Flows bool

	// GenerateCredentialMgmt enables credential management interface generation.
	// Generates CredentialProvider interface and built-in implementations.
	// Default: false
	GenerateCredentialMgmt bool

	// GenerateSecurityEnforce enables security enforcement code generation.
	// Generates per-operation security requirements and validation middleware.
	// Default: false
	GenerateSecurityEnforce bool

	// GenerateOIDCDiscovery enables OpenID Connect discovery client generation.
	// Generates OIDC discovery client and auto-configuration helpers.
	// Default: false
	GenerateOIDCDiscovery bool

	// GenerateReadme enables README.md generation in the output directory.
	// The README includes regeneration commands, file listing, and usage examples.
	// Default: true
	GenerateReadme bool
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
		// File splitting defaults
		MaxLinesPerFile:      2000,
		MaxTypesPerFile:      200,
		MaxOperationsPerFile: 100,
		SplitByTag:           true,
		SplitByPathPrefix:    true,
		// Security defaults
		GenerateSecurity:        true, // Enabled by default when client is generated
		GenerateOAuth2Flows:     false,
		GenerateCredentialMgmt:  false,
		GenerateSecurityEnforce: false,
		GenerateOIDCDiscovery:   false,
		GenerateReadme:          true,
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

	// File splitting options
	maxLinesPerFile      int
	maxTypesPerFile      int
	maxOperationsPerFile int
	splitByTag           bool
	splitByPathPrefix    bool

	// Security generation options
	generateSecurity        bool
	generateOAuth2Flows     bool
	generateCredentialMgmt  bool
	generateSecurityEnforce bool
	generateOIDCDiscovery   bool
	generateReadme          bool
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
		// File splitting
		MaxLinesPerFile:      cfg.maxLinesPerFile,
		MaxTypesPerFile:      cfg.maxTypesPerFile,
		MaxOperationsPerFile: cfg.maxOperationsPerFile,
		SplitByTag:           cfg.splitByTag,
		SplitByPathPrefix:    cfg.splitByPathPrefix,
		// Security
		GenerateSecurity:        cfg.generateSecurity,
		GenerateOAuth2Flows:     cfg.generateOAuth2Flows,
		GenerateCredentialMgmt:  cfg.generateCredentialMgmt,
		GenerateSecurityEnforce: cfg.generateSecurityEnforce,
		GenerateOIDCDiscovery:   cfg.generateOIDCDiscovery,
		GenerateReadme:          cfg.generateReadme,
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
		// File splitting defaults
		maxLinesPerFile:      2000,
		maxTypesPerFile:      200,
		maxOperationsPerFile: 100,
		splitByTag:           true,
		splitByPathPrefix:    true,
		// Security defaults
		generateSecurity:        true,
		generateOAuth2Flows:     false,
		generateCredentialMgmt:  false,
		generateSecurityEnforce: false,
		generateOIDCDiscovery:   false,
		generateReadme:          true,
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
		return nil, fmt.Errorf("generator: must specify an input source (use WithFilePath or WithParsed)")
	}
	if sourceCount > 1 {
		return nil, fmt.Errorf("generator: must specify exactly one input source")
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
			return fmt.Errorf("generator: package name cannot be empty")
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

// File splitting options

// WithMaxLinesPerFile sets the maximum lines per generated file before splitting.
// When exceeded, files are split by tag or path prefix.
// Default: 2000, 0 = no limit
func WithMaxLinesPerFile(n int) Option {
	return func(cfg *generateConfig) error {
		if n < 0 {
			return fmt.Errorf("generator: max lines per file cannot be negative")
		}
		cfg.maxLinesPerFile = n
		return nil
	}
}

// WithMaxTypesPerFile sets the maximum types per generated file before splitting.
// Default: 200, 0 = no limit
func WithMaxTypesPerFile(n int) Option {
	return func(cfg *generateConfig) error {
		if n < 0 {
			return fmt.Errorf("generator: max types per file cannot be negative")
		}
		cfg.maxTypesPerFile = n
		return nil
	}
}

// WithMaxOperationsPerFile sets the maximum operations per generated file before splitting.
// Default: 100, 0 = no limit
func WithMaxOperationsPerFile(n int) Option {
	return func(cfg *generateConfig) error {
		if n < 0 {
			return fmt.Errorf("generator: max operations per file cannot be negative")
		}
		cfg.maxOperationsPerFile = n
		return nil
	}
}

// WithSplitByTag enables or disables splitting files by operation tag.
// Default: true
func WithSplitByTag(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.splitByTag = enabled
		return nil
	}
}

// WithSplitByPathPrefix enables or disables splitting files by path prefix.
// This is used as a fallback when tags are not available.
// Default: true
func WithSplitByPathPrefix(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.splitByPathPrefix = enabled
		return nil
	}
}

// Security generation options

// WithGenerateSecurity enables or disables security helper generation.
// When true, generates ClientOption functions for each security scheme.
// Default: true
func WithGenerateSecurity(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.generateSecurity = enabled
		return nil
	}
}

// WithGenerateOAuth2Flows enables or disables OAuth2 token flow helper generation.
// Generates token acquisition, refresh, and authorization code exchange.
// Default: false
func WithGenerateOAuth2Flows(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.generateOAuth2Flows = enabled
		return nil
	}
}

// WithGenerateCredentialMgmt enables or disables credential management interface generation.
// Generates CredentialProvider interface and built-in implementations.
// Default: false
func WithGenerateCredentialMgmt(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.generateCredentialMgmt = enabled
		return nil
	}
}

// WithGenerateSecurityEnforce enables or disables security enforcement code generation.
// Generates per-operation security requirements and validation middleware.
// Default: false
func WithGenerateSecurityEnforce(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.generateSecurityEnforce = enabled
		return nil
	}
}

// WithGenerateOIDCDiscovery enables or disables OpenID Connect discovery client generation.
// Generates OIDC discovery client and auto-configuration helpers.
// Default: false
func WithGenerateOIDCDiscovery(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.generateOIDCDiscovery = enabled
		return nil
	}
}

// WithGenerateReadme enables or disables README.md generation in the output directory.
// The README includes regeneration commands, file listing, and usage examples.
// Default: true
func WithGenerateReadme(enabled bool) Option {
	return func(cfg *generateConfig) error {
		cfg.generateReadme = enabled
		return nil
	}
}

// Convenience aliases for security generation options

// WithSecurity is an alias for WithGenerateSecurity.
func WithSecurity(enabled bool) Option { return WithGenerateSecurity(enabled) }

// WithOAuth2Flows is an alias for WithGenerateOAuth2Flows.
func WithOAuth2Flows(enabled bool) Option { return WithGenerateOAuth2Flows(enabled) }

// WithCredentialMgmt is an alias for WithGenerateCredentialMgmt.
func WithCredentialMgmt(enabled bool) Option { return WithGenerateCredentialMgmt(enabled) }

// WithSecurityEnforce is an alias for WithGenerateSecurityEnforce.
func WithSecurityEnforce(enabled bool) Option { return WithGenerateSecurityEnforce(enabled) }

// WithOIDCDiscovery is an alias for WithGenerateOIDCDiscovery.
func WithOIDCDiscovery(enabled bool) Option { return WithGenerateOIDCDiscovery(enabled) }

// WithReadme is an alias for WithGenerateReadme.
func WithReadme(enabled bool) Option { return WithGenerateReadme(enabled) }

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
		return nil, fmt.Errorf("generator: failed to parse specification: %w", err)
	}

	// Check for parse errors
	if len(parseResult.Errors) > 0 {
		return nil, fmt.Errorf("generator: source document has %d parse error(s), cannot generate", len(parseResult.Errors))
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
	if doc, ok := parseResult.OAS2Document(); ok {
		cg = newOAS2CodeGenerator(g, doc, result)
	} else if doc, ok := parseResult.OAS3Document(); ok {
		cg = newOAS3CodeGenerator(g, doc, result)
	} else {
		return nil, fmt.Errorf("generator: unsupported OAS version: %s", parseResult.Version)
	}

	// Generate types if enabled
	if g.GenerateTypes || g.GenerateClient || g.GenerateServer {
		if err := cg.generateTypes(); err != nil {
			return nil, fmt.Errorf("generator: failed to generate types: %w", err)
		}
	}

	// Generate client if enabled
	if g.GenerateClient {
		if err := cg.generateClient(); err != nil {
			return nil, fmt.Errorf("generator: failed to generate client: %w", err)
		}
	}

	// Generate server if enabled
	if g.GenerateServer {
		if err := cg.generateServer(); err != nil {
			return nil, fmt.Errorf("generator: failed to generate server: %w", err)
		}
	}

	// Generate security helpers and related files
	if err := cg.generateSecurityHelpers(); err != nil {
		return nil, fmt.Errorf("generator: failed to generate security helpers: %w", err)
	}

	// Update counts and timing
	result.GenerateTime = time.Since(startTime)
	g.updateCounts(result)
	result.Success = result.CriticalCount == 0

	// In strict mode, fail on any issues
	if g.StrictMode && (result.CriticalCount > 0 || result.WarningCount > 0) {
		return result, fmt.Errorf("generator: generation failed in strict mode: %d critical issue(s), %d warning(s)",
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
	generateSecurityHelpers() error
}
