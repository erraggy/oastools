package joiner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"text/template"

	"github.com/erraggy/oastools/parser"
	"go.yaml.in/yaml/v4"
)

// joinerLogger is used for warnings in joiner functions.
// Tests can replace this with a discard logger to suppress expected warnings.
var joinerLogger = slog.Default()

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
	// StrategyRenameLeft keeps the right-side schema and renames the left-side schema
	StrategyRenameLeft CollisionStrategy = "rename-left"
	// StrategyRenameRight keeps the left-side schema and renames the right-side schema
	StrategyRenameRight CollisionStrategy = "rename-right"
	// StrategyDeduplicateEquivalent uses semantic comparison to deduplicate structurally identical schemas
	StrategyDeduplicateEquivalent CollisionStrategy = "deduplicate"
)

// ValidStrategies returns all valid collision strategy strings
func ValidStrategies() []string {
	return []string{
		string(StrategyAcceptLeft),
		string(StrategyAcceptRight),
		string(StrategyFailOnCollision),
		string(StrategyFailOnPaths),
		string(StrategyRenameLeft),
		string(StrategyRenameRight),
		string(StrategyDeduplicateEquivalent),
	}
}

// IsValidStrategy checks if a strategy string is valid
func IsValidStrategy(strategy string) bool {
	switch CollisionStrategy(strategy) {
	case StrategyAcceptLeft, StrategyAcceptRight, StrategyFailOnCollision, StrategyFailOnPaths,
		StrategyRenameLeft, StrategyRenameRight, StrategyDeduplicateEquivalent:
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

	// Advanced collision strategies configuration
	// RenameTemplate is a Go template for renamed schema names (default: "{{.Name}}_{{.Source}}")
	// Available variables: {{.Name}} (original name), {{.Source}} (source file), {{.Index}} (doc index)
	RenameTemplate string
	// NamespacePrefix maps source file paths to namespace prefixes for schema names
	// Example: {"users-api.yaml": "Users", "billing-api.yaml": "Billing"}
	// When a prefix is configured, schemas from that source get prefixed: User -> Users_User
	NamespacePrefix map[string]string
	// AlwaysApplyPrefix when true applies namespace prefix to all schemas from a source,
	// not just those that collide. When false (default), prefix is only applied on collision.
	AlwaysApplyPrefix bool
	// EquivalenceMode controls depth of schema comparison: "none", "shallow", or "deep"
	EquivalenceMode string
	// CollisionReport enables detailed collision analysis reporting
	CollisionReport bool
	// SemanticDeduplication enables cross-document schema deduplication after merging.
	// When enabled, semantically identical schemas are consolidated to a single
	// canonical schema (alphabetically first), and all references are rewritten.
	SemanticDeduplication bool

	// OperationContext enables operation-aware context in rename templates.
	// When true, builds a reference graph to populate Path, Method, OperationID,
	// Tags, and other operation-derived fields in the rename context.
	// Default: false.
	OperationContext bool

	// PrimaryOperationPolicy determines which operation provides primary context
	// when a schema is referenced by multiple operations.
	// Default: PolicyFirstEncountered.
	PrimaryOperationPolicy PrimaryOperationPolicy
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
		RenameTemplate:    "{{.Name}}_{{.Source}}",
		NamespacePrefix:   make(map[string]string),
		AlwaysApplyPrefix: false,
		EquivalenceMode:   "none",
		CollisionReport:   false,
	}
}

// Joiner handles joining of multiple OpenAPI specifications.
//
// Concurrency: Joiner instances are not safe for concurrent use.
// Create separate Joiner instances for concurrent operations.
type Joiner struct {
	config JoinerConfig
	// SourceMaps maps source file paths to their SourceMaps for location lookup.
	// When populated, collision errors and events include line/column information.
	SourceMaps map[string]*parser.SourceMap
	// collisionHandler is called when collisions are detected (nil if not configured).
	collisionHandler CollisionHandler
	// collisionHandlerTypes specifies which collision types invoke the handler.
	// Empty map means all types.
	collisionHandlerTypes map[CollisionType]bool
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
	// Warnings contains non-fatal issues encountered during joining (for backward compatibility)
	Warnings []string
	// StructuredWarnings contains detailed warning information with context
	StructuredWarnings JoinWarnings
	// CollisionCount tracks the number of collisions resolved
	CollisionCount int
	// Stats contains statistical information about the joined document
	Stats parser.DocumentStats
	// CollisionDetails contains detailed collision analysis (when CollisionReport is enabled)
	CollisionDetails *CollisionReport
	// firstFilePath stores the path of the first document for error reporting
	firstFilePath string
	// rewriter accumulates schema renames for reference rewriting
	rewriter *SchemaRewriter
}

// AddWarning adds a structured warning and populates the legacy Warnings slice.
func (r *JoinResult) AddWarning(w *JoinWarning) {
	r.StructuredWarnings = append(r.StructuredWarnings, w)
	r.Warnings = append(r.Warnings, w.String())
}

// WarningStrings returns warning messages for backward compatibility.
// Deprecated: Use StructuredWarnings directly for detailed information.
func (r *JoinResult) WarningStrings() []string {
	if len(r.StructuredWarnings) > 0 {
		return r.StructuredWarnings.Strings()
	}
	return r.Warnings
}

// ToParseResult converts the JoinResult to a ParseResult for use with
// other packages like validator, fixer, converter, and differ.
// The returned ParseResult has Document populated but Data is nil
// (consumers use Document, not Data).
func (r *JoinResult) ToParseResult() *parser.ParseResult {
	sourcePath := r.firstFilePath
	if sourcePath == "" {
		sourcePath = "joiner"
	}
	return &parser.ParseResult{
		SourcePath:   sourcePath,
		SourceFormat: r.SourceFormat,
		Version:      r.Version,
		OASVersion:   r.OASVersion,
		Document:     r.Document,
		Errors:       make([]error, 0),
		Warnings:     r.WarningStrings(),
		Stats:        r.Stats,
	}
}

// documentContext tracks the source file and document for error reporting
type documentContext struct {
	filePath string
	docIndex int
	result   *parser.ParseResult
}

func (j *Joiner) JoinParsed(parsedDocs []parser.ParseResult) (*JoinResult, error) {
	if len(parsedDocs) < 2 {
		return nil, fmt.Errorf("joiner: at least 2 specification documents are required for joining, got %d", len(parsedDocs))
	}
	// Validate inputs and check for generic source names
	var genericNameWarnings JoinWarnings
	for i, doc := range parsedDocs {
		if doc.Document == nil {
			return nil, fmt.Errorf("joiner: parsedDocs[%d].Document is nil", i)
		}
		if len(doc.Errors) > 0 {
			return nil, fmt.Errorf("joiner: parsedDocs[%d].Errors is not empty: %d errors found", i, len(doc.Errors))
		}
		// Warn about generic source names that make collision reports less useful
		if IsGenericSourceName(doc.SourcePath) {
			genericNameWarnings = append(genericNameWarnings, NewGenericSourceNameWarning(doc.SourcePath, i))
		}
	}

	// Verify all documents are the same major version
	baseVersion := parsedDocs[0].OASVersion
	var versionWarnings JoinWarnings
	for i, doc := range parsedDocs[1:] {
		if !j.versionsCompatible(baseVersion, doc.OASVersion) {
			return nil, fmt.Errorf("joiner: incompatible versions: %s (%s) and %s (%s) cannot be joined",
				parsedDocs[0].SourcePath, parsedDocs[0].Version, parsedDocs[i+1].SourcePath, doc.Version)
		}

		// Warn about minor version mismatches (e.g., 3.0.x with 3.1.x)
		if baseVersion != doc.OASVersion && j.hasMinorVersionMismatch(baseVersion, doc.OASVersion) {
			versionWarnings = append(versionWarnings, NewVersionMismatchWarning(
				parsedDocs[0].SourcePath, parsedDocs[0].Version,
				parsedDocs[i+1].SourcePath, doc.Version,
				parsedDocs[0].Version))
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

	// Add early warnings to result (prepend so they appear first)
	if result != nil {
		var prependWarnings JoinWarnings
		prependWarnings = append(prependWarnings, genericNameWarnings...)
		prependWarnings = append(prependWarnings, versionWarnings...)
		if len(prependWarnings) > 0 {
			result.StructuredWarnings = append(prependWarnings, result.StructuredWarnings...)
			// Rebuild legacy Warnings slice from StructuredWarnings for consistency
			result.Warnings = result.StructuredWarnings.Strings()
		}
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
	Section      string
	Key          string
	FirstFile    string
	FirstPath    string
	FirstLine    int // 1-based line number in first file (0 if unknown)
	FirstColumn  int // 1-based column number in first file (0 if unknown)
	SecondFile   string
	SecondPath   string
	SecondLine   int // 1-based line number in second file (0 if unknown)
	SecondColumn int // 1-based column number in second file (0 if unknown)
	Strategy     CollisionStrategy
}

func (e *CollisionError) Error() string {
	firstLoc := ""
	if e.FirstLine > 0 {
		firstLoc = fmt.Sprintf(" (line %d)", e.FirstLine)
	}
	secondLoc := ""
	if e.SecondLine > 0 {
		secondLoc = fmt.Sprintf(" (line %d)", e.SecondLine)
	}
	return fmt.Sprintf("joiner: collision in %s: '%s'\n"+
		"  First defined in:  %s%s at %s\n"+
		"  Also defined in:   %s%s at %s\n"+
		"  Strategy: %s (set --%s-strategy to 'accept-left' or 'accept-right' to resolve)",
		e.Section, e.Key,
		e.FirstFile, firstLoc, e.FirstPath,
		e.SecondFile, secondLoc, e.SecondPath,
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

	// Look up line/column for both sides if SourceMaps are available
	var firstLine, firstCol, secondLine, secondCol int
	if j.SourceMaps != nil {
		jsonPath := "$." + firstPath
		firstLine, firstCol = j.getLocation(firstFile, jsonPath)
		secondLine, secondCol = j.getLocation(secondFile, jsonPath)
	}

	switch strategy {
	case StrategyFailOnCollision:
		return &CollisionError{
			Section:      section,
			Key:          name,
			FirstFile:    firstFile,
			FirstPath:    firstPath,
			FirstLine:    firstLine,
			FirstColumn:  firstCol,
			SecondFile:   secondFile,
			SecondPath:   secondPath,
			SecondLine:   secondLine,
			SecondColumn: secondCol,
			Strategy:     strategy,
		}
	case StrategyFailOnPaths:
		if section == "paths" || section == "webhooks" {
			return &CollisionError{
				Section:      section,
				Key:          name,
				FirstFile:    firstFile,
				FirstPath:    firstPath,
				FirstLine:    firstLine,
				FirstColumn:  firstCol,
				SecondFile:   secondFile,
				SecondPath:   secondPath,
				SecondLine:   secondLine,
				SecondColumn: secondCol,
				Strategy:     strategy,
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

// generateRenamedSchemaName generates a new name for a renamed schema based on the template
func (j *Joiner) generateRenamedSchemaName(originalName, sourcePath string, docIndex int, graph *RefGraph) string {
	// Build the rename context (handles both basic and operation-aware modes)
	ctx := buildRenameContext(originalName, sourcePath, docIndex, graph, j.config.PrimaryOperationPolicy)

	// Use template if configured
	tmplStr := j.config.RenameTemplate
	if tmplStr == "" {
		tmplStr = "{{.Name}}_{{.Source}}"
	}

	// Parse template with extended function map
	tmpl, err := template.New("rename").Funcs(renameFuncs()).Parse(tmplStr)
	if err != nil {
		// Fall back to default pattern on template parse error
		joinerLogger.Warn("joiner: template parse error", "schema", originalName, "template", tmplStr, "error", err)
		return fmt.Sprintf("%s_%s", originalName, ctx.Source)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		// Fall back to default pattern on template execution error
		joinerLogger.Warn("joiner: template execution error", "schema", originalName, "template", tmplStr, "error", err)
		return fmt.Sprintf("%s_%s", originalName, ctx.Source)
	}

	return buf.String()
}

// recordCollisionEvent records a collision event if reporting is enabled
func (j *Joiner) recordCollisionEvent(result *JoinResult, schemaName, leftSource, rightSource string, strategy CollisionStrategy, resolution, newName string) {
	if result.CollisionDetails == nil {
		return
	}

	// Look up line/column for both sides if SourceMaps are available
	// The JSON path for OAS 3.x schemas is $.components.schemas.<name>
	// The JSON path for OAS 2.0 definitions is $.definitions.<name>
	var leftLine, leftCol, rightLine, rightCol int
	if j.SourceMaps != nil {
		// Try OAS 3.x path first
		leftLine, leftCol = j.getLocation(leftSource, "$.components.schemas."+schemaName)
		if leftLine == 0 {
			// Fall back to OAS 2.0 path
			leftLine, leftCol = j.getLocation(leftSource, "$.definitions."+schemaName)
		}
		rightLine, rightCol = j.getLocation(rightSource, "$.components.schemas."+schemaName)
		if rightLine == 0 {
			rightLine, rightCol = j.getLocation(rightSource, "$.definitions."+schemaName)
		}
	}

	result.CollisionDetails.AddEvent(CollisionEvent{
		SchemaName:  schemaName,
		LeftSource:  leftSource,
		LeftLine:    leftLine,
		LeftColumn:  leftCol,
		RightSource: rightSource,
		RightLine:   rightLine,
		RightColumn: rightCol,
		Strategy:    strategy,
		Resolution:  resolution,
		NewName:     newName,
	})
}

// recordCollisionEventWithPath records a collision event using explicit JSON paths for location lookup.
// This is used for non-schema collisions (paths, webhooks, etc.) where the JSON path format differs.
// Note: NewName is always empty for these collisions since paths/webhooks don't support renaming.
func (j *Joiner) recordCollisionEventWithPath(result *JoinResult, name, jsonPath, leftSource, rightSource string, strategy CollisionStrategy, resolution string) {
	if result.CollisionDetails == nil {
		return
	}

	var leftLine, leftCol, rightLine, rightCol int
	if j.SourceMaps != nil {
		leftLine, leftCol = j.getLocation(leftSource, jsonPath)
		rightLine, rightCol = j.getLocation(rightSource, jsonPath)
	}

	result.CollisionDetails.AddEvent(CollisionEvent{
		SchemaName:  name, // Reusing SchemaName field for the collision item name
		LeftSource:  leftSource,
		LeftLine:    leftLine,
		LeftColumn:  leftCol,
		RightSource: rightSource,
		RightLine:   rightLine,
		RightColumn: rightCol,
		Strategy:    strategy,
		Resolution:  resolution,
		NewName:     "", // Paths don't support renaming
	})
}

// generatePrefixedSchemaName generates a schema name with a namespace prefix.
// The format is: Prefix_OriginalName (e.g., "Users_User", "Billing_Invoice")
func (j *Joiner) generatePrefixedSchemaName(originalName, prefix string) string {
	if prefix == "" {
		return originalName
	}
	return prefix + "_" + originalName
}

// getNamespacePrefix returns the namespace prefix configured for a source file path.
// Returns empty string if no prefix is configured for the source.
func (j *Joiner) getNamespacePrefix(sourcePath string) string {
	if j.config.NamespacePrefix == nil {
		return ""
	}
	return j.config.NamespacePrefix[sourcePath]
}

// shouldInvokeHandler checks if the handler wants this collision type.
func (j *Joiner) shouldInvokeHandler(collisionType CollisionType) bool {
	if j.collisionHandler == nil {
		return false
	}
	if len(j.collisionHandlerTypes) == 0 {
		return true // empty means all types
	}
	return j.collisionHandlerTypes[collisionType]
}

// getLocation looks up the source location for a JSON path in a specific file.
// Returns line and column (both 0 if no SourceMap is available or path not found).
// The jsonPath should use $ prefix (e.g., "$.components.schemas.Pet").
func (j *Joiner) getLocation(filePath, jsonPath string) (line, col int) {
	if j.SourceMaps == nil {
		return 0, 0
	}
	sm := j.SourceMaps[filePath]
	if sm == nil {
		return 0, 0
	}
	loc := sm.Get(jsonPath)
	return loc.Line, loc.Column
}

// getLocationPtr returns a *SourceLocation for the given file and JSON path.
// Returns nil if no SourceMap is available or path not found.
func (j *Joiner) getLocationPtr(filePath, jsonPath string) *SourceLocation {
	line, col := j.getLocation(filePath, jsonPath)
	if line == 0 {
		return nil
	}
	return &SourceLocation{Line: line, Column: col}
}
