//go:build integration

// Package integration provides integration tests for the oastools CLI.
// These tests exercise the full pipeline from parsing through generation
// using declarative YAML scenarios.
//
// Run with: go test -tags=integration ./integration/... -v
// Or: make integration-test
package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/integration/harness"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
)

// getIntegrationDir returns the absolute path to the integration directory.
func getIntegrationDir(t *testing.T) string {
	t.Helper()

	// Try to find the integration directory relative to the test file
	// This works whether running from repo root or integration directory
	wd, err := os.Getwd()
	require.NoError(t, err, "failed to get working directory")

	// Check if we're in the integration directory
	if filepath.Base(wd) == "integration" {
		return wd
	}

	// Check if integration directory exists relative to working directory
	integrationDir := filepath.Join(wd, "integration")
	if _, err := os.Stat(integrationDir); err == nil {
		return integrationDir
	}

	// Fall back to parent directory check
	integrationDir = filepath.Join(filepath.Dir(wd), "integration")
	if _, err := os.Stat(integrationDir); err == nil {
		return integrationDir
	}

	require.Failf(t, "could not find integration directory", "from %s", wd)
	return ""
}

// TestBasesAreValid verifies that all base fixtures are valid OAS documents.
func TestBasesAreValid(t *testing.T) {
	integrationDir := getIntegrationDir(t)
	basesDir := filepath.Join(integrationDir, "bases")

	bases := []struct {
		name            string
		expectedVersion string
	}{
		{"petstore-oas2.yaml", "2.0"},
		{"petstore-oas30.yaml", "3.0.3"},
		{"petstore-oas31.yaml", "3.1.0"},
		{"petstore-oas32.yaml", "3.2.0"},
	}

	for _, base := range bases {
		t.Run(base.name, func(t *testing.T) {
			basePath := filepath.Join(basesDir, base.name)

			// Parse the document
			parseResult, err := parser.ParseWithOptions(
				parser.WithFilePath(basePath),
				parser.WithResolveRefs(true),
			)
			require.NoError(t, err, "failed to parse %s", base.name)

			// Check for parse errors
			assert.Empty(t, parseResult.Errors, "parse errors in %s", base.name)

			// Verify version
			assert.Equal(t, base.expectedVersion, parseResult.Version)

			// Validate the document
			validationResult, err := validator.ValidateWithOptions(
				validator.WithParsed(*parseResult),
				validator.WithStrictMode(false),
			)
			require.NoError(t, err, "failed to validate %s", base.name)

			// Check validation result
			assert.True(t, validationResult.Valid, "base fixture %s is not valid", base.name)

			// Log stats for informational purposes
			t.Logf("  Version: %s", parseResult.Version)
			t.Logf("  Paths: %d", parseResult.Stats.PathCount)
			t.Logf("  Operations: %d", parseResult.Stats.OperationCount)
			t.Logf("  Schemas: %d", parseResult.Stats.SchemaCount)
		})
	}
}

// TestScenarios runs all scenarios from the scenarios directory.
func TestScenarios(t *testing.T) {
	integrationDir := getIntegrationDir(t)
	scenariosDir := filepath.Join(integrationDir, "scenarios")
	basesDir := filepath.Join(integrationDir, "bases")

	// Load all scenarios
	scenarios, err := harness.LoadAllScenarios(scenariosDir)
	require.NoError(t, err, "failed to load scenarios")

	if len(scenarios) == 0 {
		t.Skip("no scenarios found")
	}

	t.Logf("Found %d scenarios", len(scenarios))

	var results []*harness.PipelineResult
	start := time.Now()

	for _, scenario := range scenarios {
		testName := harness.ScenarioTestName(scenario, scenariosDir)
		t.Run(testName, func(t *testing.T) {
			harness.PrintScenarioHeader(t, scenario)
			result := harness.RunScenario(t, scenario, basesDir)
			results = append(results, result)
			harness.PrintPipelineResult(t, result)

			if scenario.ExpectedFailure == "" {
				assert.True(t, result.Success, "scenario failed: %v", result.Error)
			}
		})
	}

	// Print summary
	harness.PrintSummary(t, results, time.Since(start))
}

// TestParseAllVersions is a simple test that parses all OAS versions.
func TestParseAllVersions(t *testing.T) {
	integrationDir := getIntegrationDir(t)
	basesDir := filepath.Join(integrationDir, "bases")

	versions := []struct {
		file    string
		version parser.OASVersion
	}{
		{"petstore-oas2.yaml", parser.OASVersion20},
		{"petstore-oas30.yaml", parser.OASVersion303},
		{"petstore-oas31.yaml", parser.OASVersion310},
		{"petstore-oas32.yaml", parser.OASVersion320},
	}

	for _, v := range versions {
		t.Run(v.file, func(t *testing.T) {
			result, err := parser.ParseWithOptions(
				parser.WithFilePath(filepath.Join(basesDir, v.file)),
			)
			require.NoError(t, err, "parse failed")

			harness.AssertOASVersion(t, result, v.version)
			harness.AssertNoParseErrors(t, result)
		})
	}
}
