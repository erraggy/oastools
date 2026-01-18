//go:build integration

package harness

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// PrintStepResult prints the result of a single step to the test output.
func PrintStepResult(t *testing.T, step *Step, result *StepResult, stepNum, totalSteps int) {
	t.Helper()

	// Build status indicator
	var status string
	if result.Success {
		status = "PASS"
	} else {
		status = "FAIL"
	}

	// Build step info
	stepInfo := fmt.Sprintf("[%d/%d] %s", stepNum, totalSteps, step.Name)

	// Build duration string
	duration := formatDuration(result.Duration)

	// Build extra info based on step type and result
	var extra string
	if result.Output.ParseResult != nil {
		pr := result.Output.ParseResult
		extra = fmt.Sprintf(" - %s, %d paths, %d operations, %d schemas",
			pr.Version, pr.Stats.PathCount, pr.Stats.OperationCount, pr.Stats.SchemaCount)
	}
	if result.Output.ValidationResult != nil {
		vr := result.Output.ValidationResult
		if vr.Valid {
			extra += " - valid"
		} else {
			extra += fmt.Sprintf(" - %d errors, %d warnings", vr.ErrorCount, vr.WarningCount)
		}
	}

	// Print the result line
	t.Logf("    %s %s (%s)%s", status, stepInfo, duration, extra)

	// Print error details if failed
	if !result.Success && result.Error != nil {
		t.Logf("        Error: %v", result.Error)
	}

	// Print assertion failures
	for _, ar := range result.AssertionResults {
		if !ar.Passed {
			t.Logf("        Assertion failed: %s", ar.Message)
			if ar.Expected != nil {
				t.Logf("          Expected: %v", ar.Expected)
			}
			if ar.Actual != nil {
				t.Logf("          Actual:   %v", ar.Actual)
			}
		}
	}
}

// PrintPipelineResult prints a summary of the entire pipeline execution.
func PrintPipelineResult(t *testing.T, result *PipelineResult) {
	t.Helper()

	// Build summary header
	var status string
	if result.Success {
		status = "PASS"
	} else {
		status = "FAIL"
	}

	t.Logf("")
	t.Logf("  Pipeline: %s (%s)", status, formatDuration(result.Duration))

	if !result.Success && result.FailedStep != "" {
		t.Logf("  Failed at step: %s", result.FailedStep)
		if result.Error != nil {
			t.Logf("  Error: %v", result.Error)
		}
	}
}

// PrintScenarioHeader prints the header for a scenario.
func PrintScenarioHeader(t *testing.T, scenario *Scenario) {
	t.Helper()

	t.Logf("")
	t.Logf("Scenario: %s", scenario.Name)
	if scenario.Description != "" {
		t.Logf("  %s", scenario.Description)
	}
	if scenario.Base != "" {
		t.Logf("  Base: %s", scenario.Base)
	}
	t.Logf("")
}

// PrintSummary prints a summary of all scenario results.
func PrintSummary(t *testing.T, results []*PipelineResult, duration time.Duration) {
	t.Helper()

	passed := 0
	failed := 0
	skipped := 0

	for _, r := range results {
		if r.Scenario.Skip != "" {
			skipped++
		} else if r.Success {
			passed++
		} else {
			failed++
		}
	}

	t.Logf("")
	t.Logf("%s", strings.Repeat("=", 80))
	t.Logf("INTEGRATION TEST SUMMARY")
	t.Logf("%s", strings.Repeat("=", 80))
	t.Logf("Scenarios:  %d passed, %d failed, %d skipped", passed, failed, skipped)
	t.Logf("Duration:   %s", formatDuration(duration))
	t.Logf("%s", strings.Repeat("=", 80))

	// List failed scenarios
	if failed > 0 {
		t.Logf("")
		t.Logf("Failed scenarios:")
		for _, r := range results {
			if !r.Success && r.Scenario.Skip == "" {
				t.Logf("  - %s: %v", r.Scenario.Name, r.Error)
			}
		}
	}
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// ColorStatus returns a colored status string (for terminal output).
// This is a placeholder - actual coloring would require terminal detection.
func ColorStatus(success bool) string {
	if success {
		return "PASS"
	}
	return "FAIL"
}
