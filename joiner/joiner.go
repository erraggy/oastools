package joiner

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/erraggy/oastools/parser"
	"go.yaml.in/yaml/v4"
)

// CollisionStrategy defines how to handle collisions when merging documents
type CollisionStrategy string

const (
	// StrategyAcceptLeft keeps values from the first document when collisions occur
	StrategyAcceptLeft CollisionStrategy = "accept-left"
	// StrategyAcceptRight keeps values from the last document when collisions occur (overwrites)
	StrategyAcceptRight CollisionStrategy = "accept-right"
	// StrategyFailOnCollision returns an error if any collision is detected
	StrategyFailOnCollision CollisionStrategy = "fail"
	// StrategyFailOnPaths fails only on path collisions, allows schema/component collisions
	StrategyFailOnPaths CollisionStrategy = "fail-on-paths"
)

// ValidStrategies returns all valid collision strategy strings
func ValidStrategies() []string {
	return []string{
		string(StrategyAcceptLeft),
		string(StrategyAcceptRight),
		string(StrategyFailOnCollision),
		string(StrategyFailOnPaths),
	}
}

// IsValidStrategy checks if a strategy string is valid
func IsValidStrategy(strategy string) bool {
	switch CollisionStrategy(strategy) {
	case StrategyAcceptLeft, StrategyAcceptRight, StrategyFailOnCollision, StrategyFailOnPaths:
		return true
	default:
		return false
	}
}

// JoinerConfig configures how documents are joined
type JoinerConfig struct {
	// DefaultStrategy is the global strategy for all collisions
	DefaultStrategy CollisionStrategy
	// PathStrategy defines strategy specifically for path collisions
	PathStrategy CollisionStrategy
	// SchemaStrategy defines strategy specifically for schema/definition collisions
	SchemaStrategy CollisionStrategy
	// ComponentStrategy defines strategy for other component collisions (parameters, responses, etc.)
	ComponentStrategy CollisionStrategy
	// DeduplicateTags removes duplicate tags by name
	DeduplicateTags bool
	// MergeArrays determines whether to merge array fields (servers, security, etc.)
	MergeArrays bool
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() JoinerConfig {
	return JoinerConfig{
		DefaultStrategy:   StrategyFailOnCollision,
		PathStrategy:      StrategyFailOnCollision,
		SchemaStrategy:    StrategyAcceptLeft,
		ComponentStrategy: StrategyAcceptLeft,
		DeduplicateTags:   true,
		MergeArrays:       true,
	}
}

// Joiner handles joining of multiple OpenAPI specifications.
//
// Concurrency: Joiner instances are not safe for concurrent use.
// Create separate Joiner instances for concurrent operations.
type Joiner struct {
	config JoinerConfig
}

// New creates a new Joiner instance with the provided configuration
func New(config JoinerConfig) *Joiner {
	return &Joiner{
		config: config,
	}
}

// JoinResult contains the joined OpenAPI specification and metadata
type JoinResult struct {
	// Document contains the joined document (*parser.OAS2Document or *parser.OAS3Document)
	Document any
	// Version is the OpenAPI version of the joined document
	Version string
	// OASVersion is the enumerated version
	OASVersion parser.OASVersion
	// SourceFormat is the format of the first source file (JSON or YAML)
	SourceFormat parser.SourceFormat
	// Warnings contains non-fatal issues encountered during joining
	Warnings []string
	// CollisionCount tracks the number of collisions resolved
	CollisionCount int
	// Stats contains statistical information about the joined document
	Stats parser.DocumentStats
	// firstFilePath stores the path of the first document for error reporting
	firstFilePath string
}

// documentContext tracks the source file and document for error reporting
type documentContext struct {
	filePath string
	docIndex int
	result   *parser.ParseResult
}

// Option is a function that configures a join operation
type Option func(*joinConfig) error

// joinConfig holds configuration for a join operation
type joinConfig struct {
	// Input sources (variadic, requires at least 2 total)
	filePaths  []string
	parsedDocs []parser.ParseResult

	// Configuration options (nil means use default from DefaultConfig)
	defaultStrategy   *CollisionStrategy
	pathStrategy      *CollisionStrategy
	schemaStrategy    *CollisionStrategy
	componentStrategy *CollisionStrategy
	deduplicateTags   *bool
	mergeArrays       *bool
}

// JoinWithOptions joins multiple OpenAPI specifications using functional options.
// This provides a flexible, extensible API that combines input source selection
// and configuration in a single function call.
//
// Example:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths("api1.yaml", "api2.yaml"),
//	    joiner.WithPathStrategy(joiner.StrategyAcceptLeft),
//	)
func JoinWithOptions(opts ...Option) (*JoinResult, error) {
	cfg, err := applyOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("joiner: invalid options: %w", err)
	}

	// Build JoinerConfig from options (use defaults for nil values)
	defaults := DefaultConfig()
	joinerCfg := JoinerConfig{
		DefaultStrategy:   valueOrDefault(cfg.defaultStrategy, defaults.DefaultStrategy),
		PathStrategy:      valueOrDefault(cfg.pathStrategy, defaults.PathStrategy),
		SchemaStrategy:    valueOrDefault(cfg.schemaStrategy, defaults.SchemaStrategy),
		ComponentStrategy: valueOrDefault(cfg.componentStrategy, defaults.ComponentStrategy),
		DeduplicateTags:   boolValueOrDefault(cfg.deduplicateTags, defaults.DeduplicateTags),
		MergeArrays:       boolValueOrDefault(cfg.mergeArrays, defaults.MergeArrays),
	}

	j := New(joinerCfg)

	// Route to appropriate join method based on input sources
	// Note: applyOptions ensures at least 2 total documents, so one branch must execute
	if len(cfg.filePaths) > 0 && len(cfg.parsedDocs) == 0 {
		// File paths only
		return j.Join(cfg.filePaths)
	}
	if len(cfg.parsedDocs) > 0 && len(cfg.filePaths) == 0 {
		// Parsed docs only
		return j.JoinParsed(cfg.parsedDocs)
	}
	// Mixed: parse file paths and append to parsed docs
	allDocs := make([]parser.ParseResult, 0, len(cfg.parsedDocs)+len(cfg.filePaths))
	allDocs = append(allDocs, cfg.parsedDocs...)

	p := parser.New()
	for _, path := range cfg.filePaths {
		result, err := p.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("joiner: failed to parse %s: %w", path, err)
		}
		if len(result.Errors) > 0 {
			return nil, fmt.Errorf("joiner: %s has %d parse error(s)", path, len(result.Errors))
		}
		allDocs = append(allDocs, *result)
	}
	return j.JoinParsed(allDocs)
}

// applyOptions applies option functions and validates configuration
func applyOptions(opts ...Option) (*joinConfig, error) {
	cfg := &joinConfig{
		filePaths:  make([]string, 0),
		parsedDocs: make([]parser.ParseResult, 0),
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate at least 2 documents total
	totalDocs := len(cfg.filePaths) + len(cfg.parsedDocs)
	if totalDocs < 2 {
		return nil, fmt.Errorf("joiner: at least 2 documents are required for joining, got %d", totalDocs)
	}

	return cfg, nil
}

// Helper functions for option defaults
func valueOrDefault(ptr *CollisionStrategy, defaultVal CollisionStrategy) CollisionStrategy {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

func boolValueOrDefault(ptr *bool, defaultVal bool) bool {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

// WithFilePaths specifies file paths as input sources
func WithFilePaths(paths ...string) Option {
	return func(cfg *joinConfig) error {
		cfg.filePaths = append(cfg.filePaths, paths...)
		return nil
	}
}

// WithParsed specifies parsed ParseResults as input sources
func WithParsed(docs ...parser.ParseResult) Option {
	return func(cfg *joinConfig) error {
		cfg.parsedDocs = append(cfg.parsedDocs, docs...)
		return nil
	}
}

// WithConfig applies an entire JoinerConfig struct
// This is useful for reusing existing configurations or loading from files
func WithConfig(config JoinerConfig) Option {
	return func(cfg *joinConfig) error {
		cfg.defaultStrategy = &config.DefaultStrategy
		cfg.pathStrategy = &config.PathStrategy
		cfg.schemaStrategy = &config.SchemaStrategy
		cfg.componentStrategy = &config.ComponentStrategy
		cfg.deduplicateTags = &config.DeduplicateTags
		cfg.mergeArrays = &config.MergeArrays
		return nil
	}
}

// WithDefaultStrategy sets the global collision strategy
func WithDefaultStrategy(strategy CollisionStrategy) Option {
	return func(cfg *joinConfig) error {
		cfg.defaultStrategy = &strategy
		return nil
	}
}

// WithPathStrategy sets the collision strategy for paths
func WithPathStrategy(strategy CollisionStrategy) Option {
	return func(cfg *joinConfig) error {
		cfg.pathStrategy = &strategy
		return nil
	}
}

// WithSchemaStrategy sets the collision strategy for schemas/definitions
func WithSchemaStrategy(strategy CollisionStrategy) Option {
	return func(cfg *joinConfig) error {
		cfg.schemaStrategy = &strategy
		return nil
	}
}

// WithComponentStrategy sets the collision strategy for components
func WithComponentStrategy(strategy CollisionStrategy) Option {
	return func(cfg *joinConfig) error {
		cfg.componentStrategy = &strategy
		return nil
	}
}

// WithDeduplicateTags enables or disables tag deduplication
// Default: true
func WithDeduplicateTags(enabled bool) Option {
	return func(cfg *joinConfig) error {
		cfg.deduplicateTags = &enabled
		return nil
	}
}

// WithMergeArrays enables or disables array merging (servers, security, etc.)
// Default: true
func WithMergeArrays(enabled bool) Option {
	return func(cfg *joinConfig) error {
		cfg.mergeArrays = &enabled
		return nil
	}
}

func (j *Joiner) JoinParsed(parsedDocs []parser.ParseResult) (*JoinResult, error) {
	if len(parsedDocs) < 2 {
		return nil, fmt.Errorf("joiner: at least 2 specification documents are required for joining, got %d", len(parsedDocs))
	}
	// Validate inputs
	for i, doc := range parsedDocs {
		if doc.Document == nil {
			return nil, fmt.Errorf("joiner: parsedDocs[%d].Document is nil", i)
		}
		if len(doc.Errors) > 0 {
			return nil, fmt.Errorf("joiner: parsedDocs[%d].Errors is not empty: %d errors found", i, len(doc.Errors))
		}
	}

	// Verify all documents are the same major version
	baseVersion := parsedDocs[0].OASVersion
	var warnings []string
	for i, doc := range parsedDocs[1:] {
		if !j.versionsCompatible(baseVersion, doc.OASVersion) {
			return nil, fmt.Errorf("joiner: incompatible versions: %s (%s) and %s (%s) cannot be joined",
				parsedDocs[0].SourcePath, parsedDocs[0].Version, parsedDocs[i+1].SourcePath, doc.Version)
		}

		// Warn about minor version mismatches (e.g., 3.0.x with 3.1.x)
		if baseVersion != doc.OASVersion && j.hasMinorVersionMismatch(baseVersion, doc.OASVersion) {
			warnings = append(warnings, fmt.Sprintf(
				"joining documents with different minor versions: %s (%s) and %s (%s). "+
					"This may result in an invalid specification if features from the later version are used. "+
					"Joined document will use version %s.",
				parsedDocs[0].SourcePath, parsedDocs[0].Version, parsedDocs[i+1].SourcePath, doc.Version, parsedDocs[0].Version))
		}
	}

	// Join based on version
	var result *JoinResult
	var err error
	switch {
	case baseVersion == parser.OASVersion20:
		result, err = j.joinOAS2Documents(parsedDocs)
	case baseVersion.IsValid():
		result, err = j.joinOAS3Documents(parsedDocs)
	default:
		return nil, fmt.Errorf("joiner: unsupported OpenAPI version: %s", parsedDocs[0].Version)
	}

	// Add version mismatch warnings to result
	if result != nil {
		result.Warnings = append(warnings, result.Warnings...)
	}

	return result, err
}

// Join joins multiple OpenAPI specifications into a single document
func (j *Joiner) Join(specPaths []string) (*JoinResult, error) {
	if len(specPaths) < 2 {
		return nil, fmt.Errorf("joiner: at least 2 specification files are required for joining, got %d", len(specPaths))
	}

	// Parse all documents using the parser
	parsedDocs := make([]parser.ParseResult, 0, len(specPaths))
	n := len(specPaths)
	for i, path := range specPaths {
		result, err := parser.ParseWithOptions(
			parser.WithFilePath(path),
			parser.WithValidateStructure(true),
		)
		if err != nil {
			return nil, fmt.Errorf("joiner: failed to parse %s (%d of %d): %w", path, i+1, n, err)
		}
		if len(result.Errors) > 0 {
			// Show all validation errors for better debugging
			errMsg := fmt.Sprintf("joiner: validation errors (%d error(s)) in %s (%d of %d):", len(result.Errors), path, i+1, n)
			for idx, e := range result.Errors {
				errMsg += fmt.Sprintf("\n  %d. %v", idx+1, e)
			}
			return nil, fmt.Errorf("%s", errMsg)
		}
		parsedDocs = append(parsedDocs, *result)
	}
	return j.JoinParsed(parsedDocs)
}

// outputFileMode is the file permission mode for output files (owner read/write only)
const outputFileMode = 0600

// marshalJSON marshals a document to JSON format with proper indentation
func marshalJSON(doc any) ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}

// WriteResult writes a join result to a file in YAML or JSON format (matching the source format)
//
// The output file is written with restrictive permissions (0600 - owner read/write only)
// to protect potentially sensitive API specifications. If the file already exists, its
// permissions will be explicitly set to 0600 after writing.
func (j *Joiner) WriteResult(result *JoinResult, outputPath string) error {
	var data []byte
	var err error

	// Marshal to the same format as the first input file
	if result.SourceFormat == parser.SourceFormatJSON {
		// Use encoding/json for JSON format with indentation
		data, err = marshalJSON(result.Document)
	} else {
		// Default to YAML
		data, err = yaml.Marshal(result.Document)
	}

	if err != nil {
		return fmt.Errorf("joiner: failed to marshal joined document: %w", err)
	}

	// Write to file with restrictive permissions for potentially sensitive API specs
	if err := os.WriteFile(outputPath, data, outputFileMode); err != nil {
		return fmt.Errorf("joiner: failed to write output file: %w", err)
	}

	// Explicitly set permissions to ensure they're correct even if file existed before
	// This handles the case where an existing file may have had different permissions
	if err := os.Chmod(outputPath, outputFileMode); err != nil {
		return fmt.Errorf("joiner: failed to set output file permissions: %w", err)
	}

	return nil
}

// versionsCompatible checks if two OAS versions can be joined
//
// Compatibility Rules:
//   - OAS 2.0 documents can only be joined with other 2.0 documents
//   - All OAS 3.x versions (3.0.x, 3.1.x, 3.2.x) can be joined together
//   - The joined document will use the OpenAPI version of the first input document
//
// Note: Joining documents with different minor versions (e.g., 3.0.3 + 3.1.0) is allowed
// but may result in a document that uses features from the later version while declaring
// an earlier version (or vice versa). Users should verify the joined document is valid
// for its declared version. Future OAS versions with breaking changes may require
// stricter compatibility checks.
func (j *Joiner) versionsCompatible(v1, v2 parser.OASVersion) bool {
	// OAS 2.0 documents can only be joined with other 2.0 documents
	if v1 == parser.OASVersion20 || v2 == parser.OASVersion20 {
		return v1 == v2
	}

	// All OAS 3.x versions can be joined together
	// The result will use the version of the first document
	return v1.IsValid() && v2.IsValid()
}

// hasMinorVersionMismatch detects if two OAS versions have different minor versions
// (e.g., 3.0.x vs 3.1.x). This is important because minor versions can introduce
// breaking changes like webhooks in 3.1.0 or schema changes.
func (j *Joiner) hasMinorVersionMismatch(v1, v2 parser.OASVersion) bool {
	// Not applicable to OAS 2.0
	if v1 == parser.OASVersion20 || v2 == parser.OASVersion20 {
		return false
	}

	// Detect minor version by grouping versions
	// 3.0.x: OASVersion300-304
	// 3.1.x: OASVersion310-312
	// 3.2.x: OASVersion320
	getMinorVersion := func(v parser.OASVersion) int {
		switch v {
		case parser.OASVersion300, parser.OASVersion301, parser.OASVersion302, parser.OASVersion303, parser.OASVersion304:
			return 0
		case parser.OASVersion310, parser.OASVersion311, parser.OASVersion312:
			return 1
		case parser.OASVersion320:
			return 2
		default:
			return -1
		}
	}

	return getMinorVersion(v1) != getMinorVersion(v2)
}

// getEffectiveStrategy determines which strategy to use for a specific type
func (j *Joiner) getEffectiveStrategy(specificStrategy CollisionStrategy) CollisionStrategy {
	if specificStrategy != "" {
		return specificStrategy
	}
	return j.config.DefaultStrategy
}

// CollisionError provides detailed information about a collision
type CollisionError struct {
	Section    string
	Key        string
	FirstFile  string
	FirstPath  string
	SecondFile string
	SecondPath string
	Strategy   CollisionStrategy
}

func (e *CollisionError) Error() string {
	return fmt.Sprintf("joiner: collision in %s: '%s'\n"+
		"  First defined in:  %s at %s\n"+
		"  Also defined in:   %s at %s\n"+
		"  Strategy: %s (set --%s-strategy to 'accept-left' or 'accept-right' to resolve)",
		e.Section, e.Key,
		e.FirstFile, e.FirstPath,
		e.SecondFile, e.SecondPath,
		e.Strategy, getSectionStrategyFlag(e.Section))
}

// getSectionStrategyFlag returns the CLI flag name for a given section
func getSectionStrategyFlag(section string) string {
	switch section {
	case "paths", "webhooks":
		return "path"
	case "definitions", "components.schemas":
		return "schema"
	default:
		return "component"
	}
}

// handleCollision processes a collision based on the strategy
func (j *Joiner) handleCollision(name, section string, strategy CollisionStrategy, firstFile, secondFile string) error {
	firstPath := section
	if name != "" {
		firstPath = fmt.Sprintf("%s.%s", section, name)
	}
	secondPath := firstPath

	switch strategy {
	case StrategyFailOnCollision:
		return &CollisionError{
			Section:    section,
			Key:        name,
			FirstFile:  firstFile,
			FirstPath:  firstPath,
			SecondFile: secondFile,
			SecondPath: secondPath,
			Strategy:   strategy,
		}
	case StrategyFailOnPaths:
		if section == "paths" || section == "webhooks" {
			return &CollisionError{
				Section:    section,
				Key:        name,
				FirstFile:  firstFile,
				FirstPath:  firstPath,
				SecondFile: secondFile,
				SecondPath: secondPath,
				Strategy:   strategy,
			}
		}
		return nil
	case StrategyAcceptLeft, StrategyAcceptRight:
		return nil
	default:
		return fmt.Errorf("joiner: unknown collision strategy: %s", strategy)
	}
}

// shouldOverwrite determines if a value should be overwritten based on strategy
func (j *Joiner) shouldOverwrite(strategy CollisionStrategy) bool {
	return strategy == StrategyAcceptRight
}
