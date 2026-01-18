//go:build integration

// Package harness provides the integration test framework for oastools.
// It enables declarative scenario-driven testing via YAML files.
package harness

import (
	"fmt"
	"os"
	"path/filepath"
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

// Scenario represents a complete integration test scenario.
type Scenario struct {
	// Name is a short, descriptive name for the scenario
	Name string `yaml:"name"`
	// Description provides additional context about what the scenario tests
	Description string `yaml:"description,omitempty"`
	// Base is the name of the base document from bases/ directory (without path)
	Base string `yaml:"base,omitempty"`
	// Inputs is used for multi-document scenarios (e.g., join tests)
	Inputs []Input `yaml:"inputs,omitempty"`
	// Problems defines issues to inject into the base document
	Problems Problems `yaml:"problems,omitempty"`
	// Pipeline is the sequence of steps to execute
	Pipeline []Step `yaml:"pipeline"`
	// Debug contains optional debug settings
	Debug DebugConfig `yaml:"debug,omitempty"`
	// Skip provides a reason to skip this scenario (if set, scenario is skipped)
	Skip string `yaml:"skip,omitempty"`
	// ExpectedFailure marks this scenario as a known failing case
	ExpectedFailure string `yaml:"expected-failure,omitempty"`

	// filePath is the path to the scenario file (set by loader)
	filePath string
}

// Input represents an input document for multi-document scenarios.
type Input struct {
	// Base is the name of the base document from bases/ directory
	Base string `yaml:"base"`
	// As provides an alias for this input (for error messages, reports)
	As string `yaml:"as,omitempty"`
	// Problems defines issues to inject into this specific input
	Problems Problems `yaml:"problems,omitempty"`
}

// Problems defines the issues to inject into a document.
type Problems struct {
	// Fixer problems (Phase 2)

	// MissingPathParams adds paths with missing parameter declarations
	MissingPathParams []MissingPathParam `yaml:"missing-path-params,omitempty"`
	// GenericSchemas adds schemas with bracket syntax (e.g., Response[Pet])
	GenericSchemas []string `yaml:"generic-schemas,omitempty"`
	// DuplicateOperationIDs creates operations with duplicate IDs
	DuplicateOperationIDs []DuplicateOperationID `yaml:"duplicate-operationids,omitempty"`
	// CSVEnums stores enum values as CSV strings
	CSVEnums []CSVEnum `yaml:"csv-enums,omitempty"`
	// UnusedSchemas adds schemas not referenced anywhere
	UnusedSchemas []string `yaml:"unused-schemas,omitempty"`
	// EmptyPaths adds paths with no operations
	EmptyPaths []string `yaml:"empty-paths,omitempty"`

	// Joiner problems (Phase 3)

	// DuplicateSchemaIdentical adds a schema with the same name and structure as an existing one
	DuplicateSchemaIdentical []DuplicateSchema `yaml:"duplicate-schema-identical,omitempty"`
	// DuplicateSchemaDifferent adds a schema with the same name but different structure
	DuplicateSchemaDifferent []DuplicateSchema `yaml:"duplicate-schema-different,omitempty"`
	// DuplicatePath adds a path that already exists in the document
	DuplicatePath []DuplicatePathConfig `yaml:"duplicate-path,omitempty"`
	// SemanticDuplicate adds a schema with different name but identical structure to another
	SemanticDuplicate []SemanticDuplicateConfig `yaml:"semantic-duplicate,omitempty"`

	// Differ problems (Phase 6) - Breaking API changes

	// RemoveEndpoint removes an existing path from the document
	RemoveEndpoint []string `yaml:"remove-endpoint,omitempty"`
	// RemoveOperation removes an existing operation from a path
	RemoveOperation []RemoveOperationConfig `yaml:"remove-operation,omitempty"`
	// AddRequiredParam adds a new required parameter to an operation
	AddRequiredParam []AddRequiredParamConfig `yaml:"add-required-param,omitempty"`
	// RemoveResponseCode removes a response code from an operation
	RemoveResponseCode []RemoveResponseCodeConfig `yaml:"remove-response-code,omitempty"`
	// AddEndpoint adds a new endpoint to the document (non-breaking)
	AddEndpoint []AddEndpointConfig `yaml:"add-endpoint,omitempty"`
	// AddOptionalParam adds a new optional parameter to an operation (non-breaking)
	AddOptionalParam []AddOptionalParamConfig `yaml:"add-optional-param,omitempty"`
}

// RemoveOperationConfig defines an operation to remove from a path.
type RemoveOperationConfig struct {
	// Path is the path to modify
	Path string `yaml:"path"`
	// Method is the HTTP method to remove
	Method string `yaml:"method"`
}

// AddRequiredParamConfig defines a required parameter to add.
type AddRequiredParamConfig struct {
	// Path is the path to modify
	Path string `yaml:"path"`
	// Method is the HTTP method to modify
	Method string `yaml:"method"`
	// ParamName is the name of the new required parameter
	ParamName string `yaml:"param-name"`
	// In is the location (query, header, path, cookie)
	In string `yaml:"in"`
}

// RemoveResponseCodeConfig defines a response code to remove.
type RemoveResponseCodeConfig struct {
	// Path is the path to modify
	Path string `yaml:"path"`
	// Method is the HTTP method to modify
	Method string `yaml:"method"`
	// Code is the response code to remove
	Code string `yaml:"code"`
}

// AddEndpointConfig defines a new endpoint to add.
type AddEndpointConfig struct {
	// Path is the new path to add
	Path string `yaml:"path"`
	// Method is the HTTP method
	Method string `yaml:"method"`
}

// AddOptionalParamConfig defines an optional parameter to add.
type AddOptionalParamConfig struct {
	// Path is the path to modify
	Path string `yaml:"path"`
	// Method is the HTTP method to modify
	Method string `yaml:"method"`
	// ParamName is the name of the new optional parameter
	ParamName string `yaml:"param-name"`
	// In is the location (query, header, path, cookie)
	In string `yaml:"in"`
}

// MissingPathParam defines a path with missing parameter declaration.
type MissingPathParam struct {
	Path   string `yaml:"path"`
	Method string `yaml:"method"`
}

// DuplicateOperationID defines an operation ID to duplicate.
type DuplicateOperationID struct {
	ID    string `yaml:"id"`
	Count int    `yaml:"count"`
}

// CSVEnum defines a schema with CSV enum values.
type CSVEnum struct {
	Schema string `yaml:"schema"`
	Values string `yaml:"values"`
}

// DuplicateSchema defines a schema to duplicate (for join testing).
type DuplicateSchema struct {
	// Name is the schema name to create/duplicate
	Name string `yaml:"name"`
	// CopyFrom specifies which existing schema to copy structure from (optional)
	CopyFrom string `yaml:"copy-from,omitempty"`
}

// DuplicatePathConfig defines a path to duplicate (for join testing).
type DuplicatePathConfig struct {
	// Path is the path to duplicate
	Path string `yaml:"path"`
	// Method is the HTTP method (defaults to GET)
	Method string `yaml:"method,omitempty"`
}

// SemanticDuplicateConfig defines a schema that's structurally identical to another.
type SemanticDuplicateConfig struct {
	// Original is the existing schema to copy structure from
	Original string `yaml:"original"`
	// DuplicateName is the new schema name with identical structure
	DuplicateName string `yaml:"duplicate-name"`
}

// Step represents a single step in the test pipeline.
type Step struct {
	// Name is the step type (parse, validate, fix, join, etc.)
	Name string `yaml:"step"`
	// Config contains step-specific configuration
	Config map[string]any `yaml:"config,omitempty"`
	// Expect defines the expected outcome (valid, invalid, error, success)
	Expect string `yaml:"expect,omitempty"`
	// Assertions are detailed checks to perform after the step
	Assertions []Assertion `yaml:"assertions,omitempty"`
	// ErrorContains checks that an error message contains this substring
	ErrorContains string `yaml:"error-contains,omitempty"`
}

// Assertion represents a validation check on a step result.
type Assertion struct {
	// Type is the assertion type (schema-count, error-count, etc.)
	Type string `yaml:"type,omitempty"`
	// Value is the expected value
	Value any `yaml:"value,omitempty"`
	// Detailed assertion fields (only one should be set)
	SchemaCount     *int           `yaml:"schema-count,omitempty"`
	SchemasExist    []string       `yaml:"schemas-exist,omitempty"`
	SchemasNotExist []string       `yaml:"schemas-not-exist,omitempty"`
	ErrorCount      *int           `yaml:"error-count,omitempty"`
	ErrorContains   string         `yaml:"error-contains,omitempty"`
	FixesApplied    map[string]int `yaml:"fixes-applied,omitempty"`
	NoFixesApplied  []string       `yaml:"no-fixes-applied,omitempty"`
	CollisionCount  *int           `yaml:"collision-count,omitempty"`
	// Converter-related assertions
	TargetVersion   string `yaml:"target-version,omitempty"`
	WarningCount    *int   `yaml:"warning-count,omitempty"`
	WarningContains string `yaml:"warning-contains,omitempty"`
	// Differ-related assertions
	BreakingChanges     *bool `yaml:"breaking-changes,omitempty"`
	BreakingChangeCount *int  `yaml:"breaking-change-count,omitempty"`
	ChangeCount         *int  `yaml:"change-count,omitempty"`
	// Overlay-related assertions
	ActionsApplied *int `yaml:"actions-applied,omitempty"`
	ActionsSkipped *int `yaml:"actions-skipped,omitempty"`
}

// DebugConfig contains debug settings for a scenario.
type DebugConfig struct {
	// DumpAfter specifies which steps should dump their output
	DumpAfter []string `yaml:"dump-after,omitempty"`
	// Verbose enables verbose logging
	Verbose bool `yaml:"verbose,omitempty"`
}

// StepResult contains the result of executing a single step.
type StepResult struct {
	// StepName is the name of the step that was executed
	StepName string
	// Success indicates whether the step completed without error
	Success bool
	// Error contains any error that occurred
	Error error
	// Duration is how long the step took to execute
	Duration time.Duration
	// Output contains step-specific output data
	Output StepOutput
	// AssertionResults contains results of any assertions
	AssertionResults []AssertionResult
}

// StepOutput contains the output data from a step.
type StepOutput struct {
	// ParseResult is set after a parse step
	ParseResult *parser.ParseResult
	// ValidationResult is set after a validate step
	ValidationResult *validator.ValidationResult
	// FixResult is set after a fix step
	FixResult *fixer.FixResult
	// JoinResult is set after a join step
	JoinResult *joiner.JoinResult
	// ConvertResult is set after a convert step
	ConvertResult *converter.ConversionResult
	// GenerateResult is set after a generate step
	GenerateResult *generator.GenerateResult
	// DiffResult is set after a diff step
	DiffResult *differ.DiffResult
	// OverlayResult is set after an overlay step
	OverlayResult *overlay.ApplyResult
	// Data contains arbitrary step output
	Data map[string]any
}

// AssertionResult contains the result of a single assertion.
type AssertionResult struct {
	// Assertion is the original assertion
	Assertion Assertion
	// Passed indicates whether the assertion passed
	Passed bool
	// Message provides details on failure
	Message string
	// Expected is the expected value
	Expected any
	// Actual is the actual value
	Actual any
}

// PipelineResult contains the result of running a complete pipeline.
type PipelineResult struct {
	// Scenario is the scenario that was executed
	Scenario *Scenario
	// StepResults contains results for each step
	StepResults []StepResult
	// Success indicates whether the entire pipeline passed
	Success bool
	// Duration is the total pipeline execution time
	Duration time.Duration
	// FailedStep is the name of the first step that failed (if any)
	FailedStep string
	// Error is the first error encountered
	Error error
}

// RunScenario executes a complete scenario and returns the result.
func RunScenario(t *testing.T, scenario *Scenario, basesDir string) *PipelineResult {
	t.Helper()

	start := time.Now()
	result := &PipelineResult{
		Scenario:    scenario,
		StepResults: make([]StepResult, 0, len(scenario.Pipeline)),
		Success:     true,
	}

	// Check if scenario should be skipped
	if scenario.Skip != "" {
		t.Skipf("Skipping: %s", scenario.Skip)
		return result
	}

	// Resolve base document path
	var basePath string
	if scenario.Base != "" {
		basePath = filepath.Join(basesDir, scenario.Base+".yaml")
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			// Try without .yaml extension (in case it was specified)
			basePath = filepath.Join(basesDir, scenario.Base)
			if _, err := os.Stat(basePath); os.IsNotExist(err) {
				result.Success = false
				result.Error = fmt.Errorf("base document not found: %s", scenario.Base)
				return result
			}
		}
	}

	// Create pipeline context
	pc := &PipelineContext{
		BasePath: basePath,
		BasesDir: basesDir,
		Scenario: scenario,
		Debug:    scenario.Debug.Verbose || os.Getenv("INTEGRATION_DEBUG") == "1",
		StepData: make(map[string]any),
	}

	// Ensure temporary directories are cleaned up when done
	defer func() {
		for _, dir := range pc.TempDirs {
			if err := os.RemoveAll(dir); err != nil {
				t.Logf("warning: failed to clean up temp directory %s: %v", dir, err)
			}
		}
	}()

	// Execute each step in order
	for i, step := range scenario.Pipeline {
		stepResult := ExecuteStep(t, pc, &step)
		result.StepResults = append(result.StepResults, stepResult)

		// Print step result
		PrintStepResult(t, &step, &stepResult, i+1, len(scenario.Pipeline))

		if !stepResult.Success {
			result.Success = false
			result.FailedStep = step.Name
			result.Error = stepResult.Error
			break // Fail-fast
		}
	}

	result.Duration = time.Since(start)
	return result
}

// PipelineContext holds state during pipeline execution.
type PipelineContext struct {
	// BasePath is the path to the base document
	BasePath string
	// BasesDir is the directory containing base documents
	BasesDir string
	// Scenario is the scenario being executed
	Scenario *Scenario
	// Debug enables debug output
	Debug bool
	// StepData contains data passed between steps
	StepData map[string]any
	// ParseResult is the most recent parse result
	ParseResult *parser.ParseResult
	// ParseResults holds all parsed documents for multi-document scenarios (e.g., join)
	ParseResults []*parser.ParseResult
	// ValidationResult is the most recent validation result
	ValidationResult *validator.ValidationResult
	// FixResult is the most recent fix result
	FixResult *fixer.FixResult
	// JoinResult is the most recent join result
	JoinResult *joiner.JoinResult
	// ConvertResult is the most recent convert result
	ConvertResult *converter.ConversionResult
	// GenerateResult is the most recent generate result
	GenerateResult *generator.GenerateResult
	// DiffResult is the most recent diff result
	DiffResult *differ.DiffResult
	// OverlayResult is the most recent overlay result
	OverlayResult *overlay.ApplyResult
	// TempDirs tracks temporary directories created during the test for cleanup
	TempDirs []string
}
