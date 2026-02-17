//go:build integration

package harness

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/parser"
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
				if strings.Contains(err.Error(), step.ErrorContains) {
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
				if strings.Contains(e.String(), assertion.ErrorContains) {
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
					if strings.Contains(issue.Message, assertion.WarningContains) ||
						strings.Contains(issue.Context, assertion.WarningContains) {
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
