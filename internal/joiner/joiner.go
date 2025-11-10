package joiner

import (
	"fmt"
	"os"

	"github.com/erraggy/oastools/internal/parser"
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
	// PreserveFirstInfo keeps the info section from the first document
	PreserveFirstInfo bool
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
		PreserveFirstInfo: true,
	}
}

// Joiner handles joining of multiple OpenAPI specifications
type Joiner struct {
	config JoinerConfig
	parser *parser.Parser
}

// New creates a new Joiner instance with the provided configuration
func New(config JoinerConfig) *Joiner {
	p := parser.New()
	p.ValidateStructure = true
	return &Joiner{
		config: config,
		parser: p,
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

// Join joins multiple OpenAPI specifications into a single document
func (j *Joiner) Join(specPaths []string) (*JoinResult, error) {
	if len(specPaths) < 2 {
		return nil, fmt.Errorf("joiner: at least 2 specification files are required for joining, got %d", len(specPaths))
	}

	// Parse all documents
	var parsedDocs []*parser.ParseResult
	var docContexts []documentContext
	for i, path := range specPaths {
		result, err := j.parser.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("joiner: failed to parse %s: %w", path, err)
		}
		if len(result.Errors) > 0 {
			return nil, fmt.Errorf("joiner: validation errors in %s: %v", path, result.Errors[0])
		}
		parsedDocs = append(parsedDocs, result)
		docContexts = append(docContexts, documentContext{
			filePath: path,
			docIndex: i,
			result:   result,
		})
	}

	// Verify all documents are the same major version
	baseVersion := parsedDocs[0].OASVersion
	for i, doc := range parsedDocs[1:] {
		if !j.versionsCompatible(baseVersion, doc.OASVersion) {
			return nil, fmt.Errorf("joiner: incompatible versions: %s (%s) and %s (%s) cannot be joined",
				specPaths[0], parsedDocs[0].Version, specPaths[i+1], doc.Version)
		}
	}

	// Join based on version
	switch {
	case baseVersion == parser.OASVersion20:
		return j.joinOAS2Documents(parsedDocs, docContexts)
	case baseVersion.IsValid():
		return j.joinOAS3Documents(parsedDocs, docContexts)
	default:
		return nil, fmt.Errorf("joiner: unsupported OpenAPI version: %s", parsedDocs[0].Version)
	}
}

// WriteResult writes a join result to a file
func (j *Joiner) WriteResult(result *JoinResult, outputPath string) error {
	// Marshal to YAML
	data, err := yaml.Marshal(result.Document)
	if err != nil {
		return fmt.Errorf("joiner: failed to marshal joined document: %w", err)
	}

	// Write to file with restrictive permissions for potentially sensitive API specs
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("joiner: failed to write output file: %w", err)
	}

	return nil
}

// JoinToFile joins multiple specifications and writes the result to a file
// Deprecated: Use Join() followed by WriteResult() for better performance and control
func (j *Joiner) JoinToFile(specPaths []string, outputPath string) error {
	result, err := j.Join(specPaths)
	if err != nil {
		return err
	}

	return j.WriteResult(result, outputPath)
}

// versionsCompatible checks if two OAS versions can be joined
func (j *Joiner) versionsCompatible(v1, v2 parser.OASVersion) bool {
	// OAS 2.0 documents can only be joined with other 2.0 documents
	if v1 == parser.OASVersion20 || v2 == parser.OASVersion20 {
		return v1 == v2
	}

	// All OAS 3.x versions can be joined together
	// The result will use the version of the first document
	return v1.IsValid() && v2.IsValid()
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
