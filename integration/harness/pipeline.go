//go:build integration

package harness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/differ"
	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/generator"
	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

// ExecuteStep executes a single pipeline step and returns the result.
func ExecuteStep(t *testing.T, pc *PipelineContext, step *Step) StepResult {
	t.Helper()

	start := time.Now()
	result := StepResult{
		StepName: step.Name,
		Success:  true,
	}

	var err error
	switch step.Name {
	case "parse":
		err = executeParse(t, pc, step, &result)
	case "parse-all":
		err = executeParseAll(t, pc, step, &result)
	case "validate":
		err = executeValidate(t, pc, step, &result)
	case "fix":
		err = executeFix(t, pc, step, &result)
	case "fix-all":
		err = executeFixAll(t, pc, step, &result)
	case "join":
		err = executeJoin(t, pc, step, &result)
	case "convert":
		err = executeConvert(t, pc, step, &result)
	case "convert-all":
		err = executeConvertAll(t, pc, step, &result)
	case "diff":
		err = executeDiff(t, pc, step, &result)
	case "generate":
		err = executeGenerate(t, pc, step, &result)
	case "build":
		err = executeBuild(t, pc, step, &result)
	case "overlay":
		err = executeOverlay(t, pc, step, &result)
	default:
		err = fmt.Errorf("unknown step type: %s", step.Name)
	}

	result.Duration = time.Since(start)

	// Handle expected errors
	if err != nil {
		if step.Expect == "error" {
			// Error was expected - check if error message matches
			if step.ErrorContains != "" {
				if containsSubstring(err.Error(), step.ErrorContains) {
					result.Success = true
					result.Error = nil
					return result
				}
				result.Success = false
				result.Error = fmt.Errorf("expected error containing %q, got: %v", step.ErrorContains, err)
				return result
			}
			// Any error is acceptable
			result.Success = true
			result.Error = nil
			return result
		}
		result.Success = false
		result.Error = err
		return result
	}

	// No error occurred - validate expectations
	if step.Expect == "error" {
		result.Success = false
		result.Error = fmt.Errorf("expected error but step succeeded")
		return result
	}

	// Check assertions
	result.AssertionResults = checkAssertions(t, pc, step, &result)
	for _, ar := range result.AssertionResults {
		if !ar.Passed {
			result.Success = false
			result.Error = fmt.Errorf("assertion failed: %s", ar.Message)
			break
		}
	}

	return result
}

// executeParse executes a parse step.
func executeParse(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	if pc.BasePath == "" {
		return fmt.Errorf("parse step requires a base document")
	}

	// Note: ResolveRefs is disabled to preserve $ref strings for fix operations
	// like prune-unused-schemas that need to track references
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(pc.BasePath),
		parser.WithResolveRefs(false),
	)
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}

	// Check for parse errors
	if len(parseResult.Errors) > 0 {
		return fmt.Errorf("parse produced %d errors: %v", len(parseResult.Errors), parseResult.Errors)
	}

	// Inject problems if specified in the scenario
	if pc.Scenario != nil && hasProblems(&pc.Scenario.Problems) {
		if err := InjectProblems(parseResult, &pc.Scenario.Problems); err != nil {
			return fmt.Errorf("problem injection failed: %w", err)
		}
	}

	// Store result in context for subsequent steps
	pc.ParseResult = parseResult
	result.Output.ParseResult = parseResult

	return nil
}

// hasProblems returns true if any problems are configured.
func hasProblems(p *Problems) bool {
	if p == nil {
		return false
	}
	return len(p.MissingPathParams) > 0 ||
		len(p.GenericSchemas) > 0 ||
		len(p.DuplicateOperationIDs) > 0 ||
		len(p.CSVEnums) > 0 ||
		len(p.UnusedSchemas) > 0 ||
		len(p.EmptyPaths) > 0 ||
		len(p.DuplicateSchemaIdentical) > 0 ||
		len(p.DuplicateSchemaDifferent) > 0 ||
		len(p.DuplicatePath) > 0 ||
		len(p.SemanticDuplicate) > 0 ||
		// Differ problems (Phase 6)
		len(p.RemoveEndpoint) > 0 ||
		len(p.RemoveOperation) > 0 ||
		len(p.AddRequiredParam) > 0 ||
		len(p.RemoveResponseCode) > 0 ||
		len(p.AddEndpoint) > 0 ||
		len(p.AddOptionalParam) > 0
}

// executeValidate executes a validate step.
func executeValidate(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Validate requires a prior parse result
	if pc.ParseResult == nil {
		return fmt.Errorf("validate step requires a prior parse step")
	}

	validationResult, err := validator.ValidateWithOptions(
		validator.WithParsed(*pc.ParseResult),
		validator.WithStrictMode(false),
	)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Store result in context
	pc.ValidationResult = validationResult
	result.Output.ValidationResult = validationResult

	// Check expectations
	switch step.Expect {
	case "valid":
		if !validationResult.Valid {
			var errMsgs []string
			for _, e := range validationResult.Errors {
				errMsgs = append(errMsgs, e.String())
			}
			return fmt.Errorf("expected valid but got %d errors: %v", validationResult.ErrorCount, errMsgs)
		}
	case "invalid":
		if validationResult.Valid {
			return fmt.Errorf("expected invalid but document is valid")
		}
	}

	return nil
}

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

// checkAssertions evaluates all assertions for a step.
func checkAssertions(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) []AssertionResult {
	t.Helper()

	results := make([]AssertionResult, 0, len(step.Assertions))
	for _, assertion := range step.Assertions {
		results = append(results, evaluateAssertion(t, pc, &assertion, result))
	}
	return results
}

// evaluateAssertion evaluates a single assertion.
func evaluateAssertion(t *testing.T, pc *PipelineContext, assertion *Assertion, result *StepResult) AssertionResult {
	t.Helper()

	ar := AssertionResult{
		Assertion: *assertion,
		Passed:    true,
	}

	// Schema count assertion
	if assertion.SchemaCount != nil {
		actual := 0
		if pc.ParseResult != nil {
			actual = pc.ParseResult.Stats.SchemaCount
		}
		expected := *assertion.SchemaCount
		ar.Expected = expected
		ar.Actual = actual
		if actual != expected {
			ar.Passed = false
			ar.Message = fmt.Sprintf("schema-count: expected %d, got %d", expected, actual)
		}
		return ar
	}

	// Error count assertion
	if assertion.ErrorCount != nil {
		actual := 0
		if pc.ValidationResult != nil {
			actual = pc.ValidationResult.ErrorCount
		}
		expected := *assertion.ErrorCount
		ar.Expected = expected
		ar.Actual = actual
		if actual != expected {
			ar.Passed = false
			ar.Message = fmt.Sprintf("error-count: expected %d, got %d", expected, actual)
		}
		return ar
	}

	// Error contains assertion
	if assertion.ErrorContains != "" {
		found := false
		if pc.ValidationResult != nil {
			for _, e := range pc.ValidationResult.Errors {
				if containsSubstring(e.String(), assertion.ErrorContains) {
					found = true
					break
				}
			}
		}
		ar.Expected = assertion.ErrorContains
		ar.Actual = found
		if !found {
			ar.Passed = false
			ar.Message = fmt.Sprintf("error-contains: no error containing %q found", assertion.ErrorContains)
		}
		return ar
	}

	// Schemas exist assertion
	if len(assertion.SchemasExist) > 0 {
		missing := checkSchemasExist(pc, assertion.SchemasExist)
		if len(missing) > 0 {
			ar.Passed = false
			ar.Expected = assertion.SchemasExist
			ar.Actual = missing
			ar.Message = fmt.Sprintf("schemas-exist: missing schemas: %v", missing)
		}
		return ar
	}

	// Schemas not exist assertion
	if len(assertion.SchemasNotExist) > 0 {
		found := checkSchemasNotExist(pc, assertion.SchemasNotExist)
		if len(found) > 0 {
			ar.Passed = false
			ar.Expected = "none of " + fmt.Sprint(assertion.SchemasNotExist)
			ar.Actual = found
			ar.Message = fmt.Sprintf("schemas-not-exist: unexpected schemas found: %v", found)
		}
		return ar
	}

	// Fixes applied assertion
	if len(assertion.FixesApplied) > 0 {
		ar = evaluateFixesApplied(pc, assertion.FixesApplied)
		return ar
	}

	// No fixes applied assertion
	if len(assertion.NoFixesApplied) > 0 {
		ar = evaluateNoFixesApplied(pc, assertion.NoFixesApplied)
		return ar
	}

	// Collision count assertion
	if assertion.CollisionCount != nil {
		actual := 0
		if pc.JoinResult != nil {
			actual = pc.JoinResult.CollisionCount
		}
		expected := *assertion.CollisionCount
		ar.Expected = expected
		ar.Actual = actual
		if actual != expected {
			ar.Passed = false
			ar.Message = fmt.Sprintf("collision-count: expected %d, got %d", expected, actual)
		}
		return ar
	}

	// Target version assertion (checks the converted document's version)
	if assertion.TargetVersion != "" {
		var actual string
		if pc.ConvertResult != nil {
			actual = pc.ConvertResult.TargetVersion
		} else if pc.ParseResult != nil {
			actual = pc.ParseResult.Version
		}
		expected := assertion.TargetVersion
		ar.Expected = expected
		ar.Actual = actual
		if actual != expected {
			ar.Passed = false
			ar.Message = fmt.Sprintf("target-version: expected %s, got %s", expected, actual)
		}
		return ar
	}

	// Warning count assertion (checks conversion warnings)
	if assertion.WarningCount != nil {
		actual := 0
		if pc.ConvertResult != nil {
			actual = pc.ConvertResult.WarningCount
		}
		expected := *assertion.WarningCount
		ar.Expected = expected
		ar.Actual = actual
		if actual != expected {
			ar.Passed = false
			ar.Message = fmt.Sprintf("warning-count: expected %d, got %d", expected, actual)
		}
		return ar
	}

	// Warning contains assertion (checks if any warning contains the substring)
	if assertion.WarningContains != "" {
		found := false
		if pc.ConvertResult != nil {
			for _, issue := range pc.ConvertResult.Issues {
				if issue.Severity == converter.SeverityWarning {
					if containsSubstring(issue.Message, assertion.WarningContains) ||
						containsSubstring(issue.Context, assertion.WarningContains) {
						found = true
						break
					}
				}
			}
		}
		ar.Expected = assertion.WarningContains
		ar.Actual = found
		if !found {
			ar.Passed = false
			ar.Message = fmt.Sprintf("warning-contains: no warning containing %q found", assertion.WarningContains)
		}
		return ar
	}

	// Breaking changes assertion (checks if any breaking changes were detected)
	if assertion.BreakingChanges != nil {
		actual := false
		if pc.DiffResult != nil {
			actual = pc.DiffResult.HasBreakingChanges
		}
		expected := *assertion.BreakingChanges
		ar.Expected = expected
		ar.Actual = actual
		if actual != expected {
			ar.Passed = false
			ar.Message = fmt.Sprintf("breaking-changes: expected %v, got %v", expected, actual)
		}
		return ar
	}

	// Breaking change count assertion
	if assertion.BreakingChangeCount != nil {
		actual := 0
		if pc.DiffResult != nil {
			actual = pc.DiffResult.BreakingCount
		}
		expected := *assertion.BreakingChangeCount
		ar.Expected = expected
		ar.Actual = actual
		if actual != expected {
			ar.Passed = false
			ar.Message = fmt.Sprintf("breaking-change-count: expected %d, got %d", expected, actual)
		}
		return ar
	}

	// Total change count assertion
	if assertion.ChangeCount != nil {
		actual := 0
		if pc.DiffResult != nil {
			actual = len(pc.DiffResult.Changes)
		}
		expected := *assertion.ChangeCount
		ar.Expected = expected
		ar.Actual = actual
		if actual != expected {
			ar.Passed = false
			ar.Message = fmt.Sprintf("change-count: expected %d, got %d", expected, actual)
		}
		return ar
	}

	// Actions applied assertion (overlay)
	if assertion.ActionsApplied != nil {
		actual := 0
		if pc.OverlayResult != nil {
			actual = pc.OverlayResult.ActionsApplied
		}
		expected := *assertion.ActionsApplied
		ar.Expected = expected
		ar.Actual = actual
		if actual != expected {
			ar.Passed = false
			ar.Message = fmt.Sprintf("actions-applied: expected %d, got %d", expected, actual)
		}
		return ar
	}

	// Actions skipped assertion (overlay)
	if assertion.ActionsSkipped != nil {
		actual := 0
		if pc.OverlayResult != nil {
			actual = pc.OverlayResult.ActionsSkipped
		}
		expected := *assertion.ActionsSkipped
		ar.Expected = expected
		ar.Actual = actual
		if actual != expected {
			ar.Passed = false
			ar.Message = fmt.Sprintf("actions-skipped: expected %d, got %d", expected, actual)
		}
		return ar
	}

	return ar
}

// evaluateFixesApplied checks that the expected number of each fix type was applied.
func evaluateFixesApplied(pc *PipelineContext, expected map[string]int) AssertionResult {
	ar := AssertionResult{
		Assertion: Assertion{FixesApplied: expected},
		Passed:    true,
	}

	if pc.FixResult == nil {
		ar.Passed = false
		ar.Message = "fixes-applied: no fix result available"
		return ar
	}

	// Count fixes by type
	actual := countFixesByType(pc.FixResult)

	// Check each expected fix type
	for fixType, expectedCount := range expected {
		mappedType := mapFixTypeName(fixType)
		actualCount := actual[string(mappedType)]
		if actualCount != expectedCount {
			ar.Passed = false
			ar.Expected = expected
			ar.Actual = actual
			ar.Message = fmt.Sprintf("fixes-applied: expected %d %s fixes, got %d", expectedCount, fixType, actualCount)
			return ar
		}
	}

	ar.Expected = expected
	ar.Actual = actual
	return ar
}

// evaluateNoFixesApplied checks that none of the specified fix types were applied.
func evaluateNoFixesApplied(pc *PipelineContext, fixTypes []string) AssertionResult {
	ar := AssertionResult{
		Assertion: Assertion{NoFixesApplied: fixTypes},
		Passed:    true,
	}

	if pc.FixResult == nil {
		// No fixes applied means the assertion passes
		return ar
	}

	// Count fixes by type
	actual := countFixesByType(pc.FixResult)

	// Check that none of the specified types were applied
	var found []string
	for _, fixType := range fixTypes {
		mappedType := mapFixTypeName(fixType)
		if actual[string(mappedType)] > 0 {
			found = append(found, fixType)
		}
	}

	if len(found) > 0 {
		ar.Passed = false
		ar.Expected = "none of " + fmt.Sprint(fixTypes)
		ar.Actual = found
		ar.Message = fmt.Sprintf("no-fixes-applied: unexpected fixes found: %v", found)
	}

	return ar
}

// countFixesByType counts the number of fixes applied by type.
func countFixesByType(fr *fixer.FixResult) map[string]int {
	counts := make(map[string]int)
	for _, fix := range fr.Fixes {
		counts[string(fix.Type)]++
	}
	return counts
}

// checkSchemasExist returns a list of schemas that do not exist.
func checkSchemasExist(pc *PipelineContext, schemas []string) []string {
	if pc.ParseResult == nil {
		return schemas
	}

	existingSchemas := getSchemaNames(pc.ParseResult)
	var missing []string
	for _, s := range schemas {
		if !slices.Contains(existingSchemas, s) {
			missing = append(missing, s)
		}
	}
	return missing
}

// checkSchemasNotExist returns a list of schemas that exist but should not.
func checkSchemasNotExist(pc *PipelineContext, schemas []string) []string {
	if pc.ParseResult == nil {
		return nil
	}

	existingSchemas := getSchemaNames(pc.ParseResult)
	var found []string
	for _, s := range schemas {
		if slices.Contains(existingSchemas, s) {
			found = append(found, s)
		}
	}
	return found
}

// getSchemaNames returns the names of all schemas in the parsed document.
func getSchemaNames(pr *parser.ParseResult) []string {
	var names []string

	if doc, ok := pr.OAS2Document(); ok && doc.Definitions != nil {
		for name := range doc.Definitions {
			names = append(names, name)
		}
	}

	if doc, ok := pr.OAS3Document(); ok && doc.Components != nil && doc.Components.Schemas != nil {
		for name := range doc.Components.Schemas {
			names = append(names, name)
		}
	}

	return names
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

// executeGenerate generates code from the parsed document.
func executeGenerate(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Generate requires a prior parse result
	if pc.ParseResult == nil {
		return fmt.Errorf("generate step requires a prior parse step")
	}

	// Build generator options from step config
	opts := []generator.Option{
		generator.WithParsed(*pc.ParseResult),
	}

	// Handle package name config
	if packageName, ok := step.Config["package"].(string); ok && packageName != "" {
		opts = append(opts, generator.WithPackageName(packageName))
	} else {
		opts = append(opts, generator.WithPackageName("generated"))
	}

	// Handle client generation config
	if client, ok := step.Config["client"].(bool); ok {
		opts = append(opts, generator.WithClient(client))
	}

	// Handle server generation config
	if server, ok := step.Config["server"].(bool); ok {
		opts = append(opts, generator.WithServer(server))
	}

	// Handle types-only generation config
	if types, ok := step.Config["types"].(bool); ok {
		opts = append(opts, generator.WithTypes(types))
	}

	// Disable README generation for tests (reduces noise)
	opts = append(opts, generator.WithReadme(false))

	// Execute the generator
	genResult, err := generator.GenerateWithOptions(opts...)
	if err != nil {
		return fmt.Errorf("generate failed: %w", err)
	}

	// Check for critical issues
	if genResult.HasCriticalIssues() {
		return fmt.Errorf("generate produced %d critical issues", genResult.CriticalCount)
	}

	// Store the generate result
	pc.GenerateResult = genResult
	result.Output.GenerateResult = genResult

	// Create a temp directory for generated files
	outputDir, ok := step.Config["output-dir"].(string)
	if !ok || outputDir == "" {
		tempDir, err := os.MkdirTemp("", "oastools-generate-*")
		if err != nil {
			return fmt.Errorf("generate: failed to create temp directory: %w", err)
		}
		outputDir = tempDir
		pc.TempDirs = append(pc.TempDirs, tempDir)
	}

	// Write generated files to the output directory
	if err := genResult.WriteFiles(outputDir); err != nil {
		return fmt.Errorf("generate: failed to write files: %w", err)
	}

	// Store the output directory path for subsequent steps (e.g., build)
	pc.StepData["generate-output-dir"] = outputDir

	if pc.Debug {
		t.Logf("  Generated %d files to %s", len(genResult.Files), outputDir)
		for _, f := range genResult.Files {
			t.Logf("    - %s", f.Name)
		}
	}

	return nil
}

// executeBuild runs go build on generated code.
func executeBuild(t *testing.T, pc *PipelineContext, step *Step, result *StepResult) error {
	t.Helper()

	// Get output directory from generate step
	outputDir, ok := pc.StepData["generate-output-dir"].(string)
	if !ok || outputDir == "" {
		return fmt.Errorf("build step requires a prior generate step")
	}

	// Check that the directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return fmt.Errorf("build: output directory does not exist: %s", outputDir)
	}

	// Initialize go module in temp directory if needed
	goModPath := filepath.Join(outputDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		// Get package name from generate result or config
		pkgName := "generated"
		if pc.GenerateResult != nil && pc.GenerateResult.PackageName != "" {
			pkgName = pc.GenerateResult.PackageName
		}

		modCmd := exec.Command("go", "mod", "init", "test/"+pkgName)
		modCmd.Dir = outputDir
		if output, err := modCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build: failed to init go module: %s\n%s", err, output)
		}

		// Run go mod tidy to fetch dependencies
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = outputDir
		if output, err := tidyCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build: failed to tidy go module: %s\n%s", err, output)
		}
	}

	// Run go build
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = outputDir
	output, err := buildCmd.CombinedOutput()

	if err != nil {
		// Check if error was expected
		if step.Expect == "error" {
			if pc.Debug {
				t.Logf("  Build failed as expected: %s\n%s", err, output)
			}
			return nil // Expected failure
		}
		return fmt.Errorf("build failed: %s\n%s", err, output)
	}

	// Check if success was not expected
	if step.Expect == "error" {
		return fmt.Errorf("build succeeded but error was expected")
	}

	if pc.Debug {
		t.Logf("  Build succeeded in %s", outputDir)
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
