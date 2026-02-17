//go:build integration

package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/differ"
	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
)

// executeFix executes a fix step.
func executeFix(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Fix requires a prior parse result
	if pc.ParseResult == nil {
		return fmt.Errorf("fix step requires a prior parse step")
	}

	// Build fixer options from step config
	opts := []fixer.Option{
		fixer.WithParsed(*pc.ParseResult),
	}

	// Handle enabled fixes configuration
	enabledFixes, hasExplicitConfig := getEnabledFixes(step.Config)
	if hasExplicitConfig {
		// Pass the explicit config (empty slice means all fixes enabled)
		opts = append(opts, fixer.WithEnabledFixes(enabledFixes...))
	}

	// Execute the fixer
	fixResult, err := fixer.FixWithOptions(opts...)
	if err != nil {
		return fmt.Errorf("fix failed: %w", err)
	}

	// Store the fix result
	pc.FixResult = fixResult
	result.Output.FixResult = fixResult

	// Update ParseResult with the fixed document for subsequent steps
	pc.ParseResult = fixResult.ToParseResult()

	return nil
}

// getEnabledFixes extracts enabled fix types from step config.
// Returns the fix types and a boolean indicating if explicit config was provided.
func getEnabledFixes(config map[string]any) ([]fixer.FixType, bool) {
	if config == nil {
		return nil, false
	}

	enabled, ok := config["enabled"]
	if !ok {
		return nil, false
	}

	// Handle "all" keyword
	if s, ok := enabled.(string); ok && s == "all" {
		return []fixer.FixType{}, true // empty slice enables all fixes
	}

	// Handle list of fix types
	if list, ok := enabled.([]any); ok {
		var fixes []fixer.FixType
		for _, item := range list {
			if s, ok := item.(string); ok {
				fixes = append(fixes, mapFixTypeName(s))
			}
		}
		return fixes, true
	}

	return nil, false
}

// mapFixTypeName maps scenario fix type names to fixer.FixType constants.
func mapFixTypeName(name string) fixer.FixType {
	switch strings.ToLower(name) {
	case "missing-path-params", "missing-path-parameter":
		return fixer.FixTypeMissingPathParameter
	case "generic-schemas", "renamed-generic-schema":
		return fixer.FixTypeRenamedGenericSchema
	case "duplicate-operationids", "duplicate-operation-id":
		return fixer.FixTypeDuplicateOperationId
	case "csv-enums", "enum-csv-expanded":
		return fixer.FixTypeEnumCSVExpanded
	case "unused-schemas", "pruned-unused-schema":
		return fixer.FixTypePrunedUnusedSchema
	case "empty-paths", "pruned-empty-path":
		return fixer.FixTypePrunedEmptyPath
	default:
		return fixer.FixType(name)
	}
}

// executeParseAll parses multiple input documents for multi-document scenarios.
func executeParseAll(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	if len(pc.Scenario.Inputs) == 0 {
		return fmt.Errorf("parse-all step requires inputs to be specified in the scenario")
	}

	pc.ParseResults = make([]*parser.ParseResult, 0, len(pc.Scenario.Inputs))

	for i, input := range pc.Scenario.Inputs {
		// Resolve base document path
		basePath := filepath.Join(pc.BasesDir, input.Base+".yaml")

		// Parse the document
		parseResult, err := parser.ParseWithOptions(
			parser.WithFilePath(basePath),
			parser.WithResolveRefs(false),
		)
		if err != nil {
			return fmt.Errorf("parse-all: failed to parse input %d (%s): %w", i, input.Base, err)
		}

		if len(parseResult.Errors) > 0 {
			return fmt.Errorf("parse-all: input %d (%s) has %d parse errors", i, input.Base, len(parseResult.Errors))
		}

		// Inject problems if specified for this input
		if hasProblems(&input.Problems) {
			if err := InjectProblems(parseResult, &input.Problems); err != nil {
				return fmt.Errorf("parse-all: problem injection failed for input %d (%s): %w", i, input.Base, err)
			}
		}

		// Set an alias for the source path if specified
		if input.As != "" {
			parseResult.SourcePath = input.As
		}

		pc.ParseResults = append(pc.ParseResults, parseResult)
	}

	// Also set the first parse result as the current one (for compatibility)
	if len(pc.ParseResults) > 0 {
		pc.ParseResult = pc.ParseResults[0]
		result.Output.ParseResult = pc.ParseResult
	}

	return nil
}

// executeJoin joins multiple parsed documents into one.
func executeJoin(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Verify we have documents to join
	if len(pc.ParseResults) < 2 {
		return fmt.Errorf("join step requires at least 2 parsed documents (got %d)", len(pc.ParseResults))
	}

	// Build joiner options from step config
	opts := buildJoinerOptions(pc.ParseResults, step.Config)

	// Execute the join
	joinResult, err := joiner.JoinWithOptions(opts...)
	if err != nil {
		return fmt.Errorf("join failed: %w", err)
	}

	// Store the join result
	pc.JoinResult = joinResult
	result.Output.JoinResult = joinResult

	// Update ParseResult from the joined result for subsequent steps
	pc.ParseResult = joinResult.ToParseResult()
	result.Output.ParseResult = pc.ParseResult

	return nil
}

// buildJoinerOptions constructs joiner options from step config.
func buildJoinerOptions(parseResults []*parser.ParseResult, config map[string]any) []joiner.Option {
	// Convert []*parser.ParseResult to []parser.ParseResult for joiner
	docs := make([]parser.ParseResult, len(parseResults))
	for i, pr := range parseResults {
		docs[i] = *pr
	}

	opts := []joiner.Option{
		joiner.WithParsed(docs...),
	}

	if config == nil {
		return opts
	}

	// Handle strategy configuration
	if strategy, ok := config["strategy"].(string); ok {
		opts = append(opts, joiner.WithDefaultStrategy(joiner.CollisionStrategy(strategy)))
	}

	// Handle path-strategy
	if pathStrategy, ok := config["path-strategy"].(string); ok {
		opts = append(opts, joiner.WithPathStrategy(joiner.CollisionStrategy(pathStrategy)))
	}

	// Handle schema-strategy
	if schemaStrategy, ok := config["schema-strategy"].(string); ok {
		opts = append(opts, joiner.WithSchemaStrategy(joiner.CollisionStrategy(schemaStrategy)))
	}

	// Handle component-strategy
	if componentStrategy, ok := config["component-strategy"].(string); ok {
		opts = append(opts, joiner.WithComponentStrategy(joiner.CollisionStrategy(componentStrategy)))
	}

	// Handle semantic-deduplication
	if semanticDedup, ok := config["semantic-deduplication"].(bool); ok {
		opts = append(opts, joiner.WithSemanticDeduplication(semanticDedup))
	}

	// Handle collision-report (for debugging)
	if collisionReport, ok := config["collision-report"].(bool); ok {
		opts = append(opts, joiner.WithCollisionReport(collisionReport))
	}

	// Handle equivalence-mode
	if equivalenceMode, ok := config["equivalence-mode"].(string); ok {
		opts = append(opts, joiner.WithEquivalenceMode(equivalenceMode))
	}

	return opts
}

// executeConvert converts a document to a target version.
func executeConvert(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Convert requires a prior parse result
	if pc.ParseResult == nil {
		return fmt.Errorf("convert step requires a prior parse step")
	}

	// Get target version from config
	targetVersion, ok := step.Config["target-version"].(string)
	if !ok || targetVersion == "" {
		return fmt.Errorf("convert step requires 'target-version' config")
	}

	// Build converter options
	opts := []converter.Option{
		converter.WithParsed(*pc.ParseResult),
		converter.WithTargetVersion(targetVersion),
	}

	// Check for strict mode
	if strict, ok := step.Config["strict"].(bool); ok {
		opts = append(opts, converter.WithStrictMode(strict))
	}

	// Check for include-info
	if includeInfo, ok := step.Config["include-info"].(bool); ok {
		opts = append(opts, converter.WithIncludeInfo(includeInfo))
	}

	// Execute the conversion
	convertResult, err := converter.ConvertWithOptions(opts...)
	if err != nil {
		return fmt.Errorf("convert failed: %w", err)
	}

	// Store the convert result
	pc.ConvertResult = convertResult
	result.Output.ConvertResult = convertResult

	// Update ParseResult with the converted document for subsequent steps
	pc.ParseResult = convertResult.ToParseResult()
	result.Output.ParseResult = pc.ParseResult

	return nil
}

// executeConvertAll converts multiple documents to the same target version.
func executeConvertAll(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Verify we have documents to convert
	if len(pc.ParseResults) == 0 {
		return fmt.Errorf("convert-all step requires prior parse-all step")
	}

	// Get target version from config
	targetVersion, ok := step.Config["target-version"].(string)
	if !ok || targetVersion == "" {
		return fmt.Errorf("convert-all step requires 'target-version' config")
	}

	// Check for strict mode
	strict, _ := step.Config["strict"].(bool)

	// Check for include-info
	includeInfo := true
	if val, ok := step.Config["include-info"].(bool); ok {
		includeInfo = val
	}

	// Convert each document
	convertedResults := make([]*parser.ParseResult, 0, len(pc.ParseResults))
	for i, pr := range pc.ParseResults {
		// Build converter options for this document
		opts := []converter.Option{
			converter.WithParsed(*pr),
			converter.WithTargetVersion(targetVersion),
			converter.WithStrictMode(strict),
			converter.WithIncludeInfo(includeInfo),
		}

		convertResult, err := converter.ConvertWithOptions(opts...)
		if err != nil {
			return fmt.Errorf("convert-all: failed to convert document %d: %w", i, err)
		}

		// Store converted result as ParseResult
		convertedResults = append(convertedResults, convertResult.ToParseResult())
	}

	// Update ParseResults with converted documents
	pc.ParseResults = convertedResults

	// Also set the first result as current
	if len(pc.ParseResults) > 0 {
		pc.ParseResult = pc.ParseResults[0]
		result.Output.ParseResult = pc.ParseResult
	}

	return nil
}

// containsSubstring checks if s contains substr (case-insensitive would be nice but keeping simple).
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// executeFixAll applies fixes to all documents in ParseResults.
func executeFixAll(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Verify we have documents to fix
	if len(pc.ParseResults) == 0 {
		return fmt.Errorf("fix-all step requires prior parse-all step")
	}

	// Get enabled fixes configuration
	enabledFixes, hasExplicitConfig := getEnabledFixes(step.Config)

	// Fix each document
	fixedResults := make([]*parser.ParseResult, 0, len(pc.ParseResults))
	for i, pr := range pc.ParseResults {
		// Build fixer options for this document
		opts := []fixer.Option{
			fixer.WithParsed(*pr),
		}
		if hasExplicitConfig {
			opts = append(opts, fixer.WithEnabledFixes(enabledFixes...))
		}

		fixResult, err := fixer.FixWithOptions(opts...)
		if err != nil {
			return fmt.Errorf("fix-all: failed to fix document %d: %w", i, err)
		}

		// Store fixed result as ParseResult
		fixedResults = append(fixedResults, fixResult.ToParseResult())
	}

	// Update ParseResults with fixed documents
	pc.ParseResults = fixedResults

	// Also set the first result as current
	if len(pc.ParseResults) > 0 {
		pc.ParseResult = pc.ParseResults[0]
		result.Output.ParseResult = pc.ParseResult
	}

	return nil
}

// executeDiff compares two parsed documents and detects changes.
func executeDiff(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Get source and target indices from config
	sourceIdx := 0
	targetIdx := 1

	if cfg := step.Config; cfg != nil {
		if src, ok := cfg["source"]; ok {
			switch v := src.(type) {
			case int:
				sourceIdx = v
			case float64:
				sourceIdx = int(v)
			case string:
				if v == "current" {
					sourceIdx = -1 // Use pc.ParseResult
				}
			}
		}
		if tgt, ok := cfg["target"]; ok {
			switch v := tgt.(type) {
			case int:
				targetIdx = v
			case float64:
				targetIdx = int(v)
			}
		}
	}

	// Get source document
	var source *parser.ParseResult
	if sourceIdx == -1 {
		if pc.ParseResult == nil {
			return fmt.Errorf("diff step: source='current' but no current ParseResult")
		}
		source = pc.ParseResult
	} else {
		if sourceIdx < 0 || sourceIdx >= len(pc.ParseResults) {
			return fmt.Errorf("diff step: source index %d out of range (have %d documents)", sourceIdx, len(pc.ParseResults))
		}
		source = pc.ParseResults[sourceIdx]
	}

	// Get target document
	if targetIdx < 0 || targetIdx >= len(pc.ParseResults) {
		return fmt.Errorf("diff step: target index %d out of range (have %d documents)", targetIdx, len(pc.ParseResults))
	}
	target := pc.ParseResults[targetIdx]

	// Build differ options
	opts := []differ.Option{
		differ.WithSourceParsed(*source),
		differ.WithTargetParsed(*target),
		differ.WithMode(differ.ModeBreaking),
	}

	// Check for include-info config
	if cfg := step.Config; cfg != nil {
		if includeInfo, ok := cfg["include-info"].(bool); ok {
			opts = append(opts, differ.WithIncludeInfo(includeInfo))
		}
	}

	// Execute the diff
	diffResult, err := differ.DiffWithOptions(opts...)
	if err != nil {
		return fmt.Errorf("diff failed: %w", err)
	}

	// Store the diff result
	pc.DiffResult = diffResult
	result.Output.DiffResult = diffResult

	if pc.Debug {
		t.Logf("  Diff: %d changes (%d breaking, %d warnings, %d info)",
			len(diffResult.Changes), diffResult.BreakingCount, diffResult.WarningCount, diffResult.InfoCount)
	}

	return nil
}

// executeOverlay applies an overlay specification to the current document.
func executeOverlay(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Overlay requires a prior parse result
	if pc.ParseResult == nil {
		return fmt.Errorf("overlay step requires a prior parse step")
	}

	// Get overlay file path from config
	overlayFile, ok := step.Config["overlay-file"].(string)
	if !ok || overlayFile == "" {
		return fmt.Errorf("overlay step requires 'overlay-file' config")
	}

	// Resolve overlay file path relative to scenario file or integration directory
	// If the path starts with ../../, it's relative to the scenario file
	var overlayPath string
	if filepath.IsAbs(overlayFile) {
		overlayPath = overlayFile
	} else if pc.Scenario != nil && pc.Scenario.filePath != "" {
		// Relative to scenario file
		scenarioDir := filepath.Dir(pc.Scenario.filePath)
		overlayPath = filepath.Join(scenarioDir, overlayFile)
	} else {
		// Fallback: relative to bases directory parent
		overlayPath = filepath.Join(filepath.Dir(pc.BasesDir), overlayFile)
	}

	// Check if overlay file exists
	if _, err := os.Stat(overlayPath); os.IsNotExist(err) {
		return fmt.Errorf("overlay file not found: %s", overlayPath)
	}

	// Build overlay options
	opts := []overlay.Option{
		overlay.WithSpecParsed(*pc.ParseResult),
		overlay.WithOverlayFilePath(overlayPath),
	}

	// Check for strict-targets config
	if cfg := step.Config; cfg != nil {
		if strictTargets, ok := cfg["strict-targets"].(bool); ok {
			opts = append(opts, overlay.WithStrictTargets(strictTargets))
		}
	}

	// Execute the overlay
	overlayResult, err := overlay.ApplyWithOptions(opts...)
	if err != nil {
		return fmt.Errorf("overlay failed: %w", err)
	}

	// Store the overlay result
	pc.OverlayResult = overlayResult
	result.Output.OverlayResult = overlayResult

	// Re-parse the overlaid document to restore typed structure for subsequent steps
	reparsed, err := overlay.ReparseDocument(pc.ParseResult, overlayResult.Document)
	if err != nil {
		return fmt.Errorf("overlay: failed to reparse overlaid document: %w", err)
	}

	// Update ParseResult with the overlaid document
	pc.ParseResult = reparsed
	result.Output.ParseResult = reparsed

	if pc.Debug {
		t.Logf("  Overlay: %d actions applied, %d skipped",
			overlayResult.ActionsApplied, overlayResult.ActionsSkipped)
	}

	return nil
}
