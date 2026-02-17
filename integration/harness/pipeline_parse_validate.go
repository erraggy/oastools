//go:build integration

package harness

import (
	"fmt"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

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
