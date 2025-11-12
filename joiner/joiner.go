package joiner

import (
	"fmt"
	"os"

	"github.com/erraggy/oastools/parser"
	"gopkg.in/yaml.v3"
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
	Document interface{}
	// Version is the OpenAPI version of the joined document
	Version string
	// OASVersion is the enumerated version
	OASVersion parser.OASVersion
	// Warnings contains non-fatal issues encountered during joining
	Warnings []string
	// CollisionCount tracks the number of collisions resolved
	CollisionCount int
	// firstFilePath stores the path of the first document for error reporting
	firstFilePath string
}

// documentContext tracks the source file and document for error reporting
type documentContext struct {
	filePath string
	docIndex int
	result   *parser.ParseResult
}

func (j *Joiner) JoinParsed(parsedDocs []*parser.ParseResult) (*JoinResult, error) {
	if len(parsedDocs) < 2 {
		return nil, fmt.Errorf("joiner: at least 2 specification documents are required for joining, got %d", len(parsedDocs))
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

	// Parse all documents
	p := parser.New()
	p.ValidateStructure = true
	var parsedDocs []*parser.ParseResult
	n := len(specPaths)
	for i, path := range specPaths {
		result, err := p.Parse(path)
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
		parsedDocs = append(parsedDocs, result)
	}
	return j.JoinParsed(parsedDocs)
}

// outputFileMode is the file permission mode for output files (owner read/write only)
const outputFileMode = 0600

// WriteResult writes a join result to a file
//
// The output file is written with restrictive permissions (0600 - owner read/write only)
// to protect potentially sensitive API specifications. If the file already exists, its
// permissions will be explicitly set to 0600 after writing.
func (j *Joiner) WriteResult(result *JoinResult, outputPath string) error {
	// Marshal to YAML
	data, err := yaml.Marshal(result.Document)
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
