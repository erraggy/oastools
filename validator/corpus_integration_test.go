package validator

import (
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_Validate tests validation of all non-large corpus specifications.
func TestCorpus_Validate(t *testing.T) {
	specs := corpusutil.GetParseableSpecs(false) // Exclude large specs and specs with parsing issues

	for _, spec := range specs {
		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			v := New()
			v.IncludeWarnings = true

			result, err := v.Validate(spec.GetLocalPath())
			require.NoError(t, err, "Validation should complete for %s", spec.Name)
			require.NotNil(t, result)

			// Check expected validation outcome
			if spec.ExpectedValid {
				assert.True(t, result.Valid,
					"%s should be valid (got %d errors)", spec.Name, result.ErrorCount)
			} else {
				assert.False(t, result.Valid,
					"%s should be invalid", spec.Name)

				// Allow tolerance in error count (specs may change over time)
				// Check we have at least half the expected errors
				minErrors := spec.ExpectedErrors / 2
				if minErrors < 10 {
					minErrors = 1 // At least 1 error for invalid specs
				}
				assert.GreaterOrEqual(t, result.ErrorCount, minErrors,
					"%s should have approximately %d errors (got %d)",
					spec.Name, spec.ExpectedErrors, result.ErrorCount)
			}

			t.Logf("%s: Valid=%v, Errors=%d, Warnings=%d",
				spec.Name, result.Valid, result.ErrorCount, result.WarningCount)
		})
	}
}

// TestCorpus_ValidSpecs specifically tests specs expected to pass validation.
func TestCorpus_ValidSpecs(t *testing.T) {
	validSpecs := corpusutil.GetValidSpecs(false)

	for _, spec := range validSpecs {
		t.Run(spec.Name+"_Valid", func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)
			corpusutil.SkipIfHasParsingIssues(t, spec)

			result, err := ValidateWithOptions(
				WithFilePath(spec.GetLocalPath()),
				WithIncludeWarnings(true),
				WithStrictMode(false),
			)
			require.NoError(t, err)

			assert.True(t, result.Valid,
				"%s: Expected valid but got %d errors", spec.Name, result.ErrorCount)

			if !result.Valid && len(result.Errors) > 0 {
				// Log first 5 errors for debugging
				for i, e := range result.Errors {
					if i >= 5 {
						t.Logf("... and %d more errors", len(result.Errors)-5)
						break
					}
					t.Logf("  Error: %s", e.String())
				}
			}
		})
	}
}

// TestCorpus_InvalidSpecs specifically tests specs expected to fail validation.
func TestCorpus_InvalidSpecs(t *testing.T) {
	invalidSpecs := corpusutil.GetInvalidSpecs(false)

	for _, spec := range invalidSpecs {
		t.Run(spec.Name+"_Invalid", func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			result, err := ValidateWithOptions(
				WithFilePath(spec.GetLocalPath()),
				WithIncludeWarnings(false), // Only count errors
				WithStrictMode(true),
			)
			require.NoError(t, err)

			assert.False(t, result.Valid,
				"%s: Expected invalid but validation passed", spec.Name)
			assert.Greater(t, result.ErrorCount, 0,
				"%s: Should have at least 1 error", spec.Name)

			t.Logf("%s: %d errors (expected ~%d)",
				spec.Name, result.ErrorCount, spec.ExpectedErrors)
		})
	}
}

// TestCorpus_StrictMode tests strict validation mode on Petstore.
func TestCorpus_StrictMode(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec, "Petstore spec should exist in corpus")
	corpusutil.SkipIfNotCached(t, *spec)

	v := New()
	v.StrictMode = true
	v.IncludeWarnings = true

	result, err := v.Validate(spec.GetLocalPath())
	require.NoError(t, err)

	// Petstore should pass even in strict mode
	assert.True(t, result.Valid, "Petstore should pass strict validation")

	t.Logf("Petstore strict mode: Errors=%d, Warnings=%d",
		result.ErrorCount, result.WarningCount)
}

// TestCorpus_OAS31Validation tests validation of OAS 3.1.0 spec (Discord).
func TestCorpus_OAS31Validation(t *testing.T) {
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec, "Discord spec should exist in corpus")
	corpusutil.SkipIfNotCached(t, *spec)

	v := New()
	v.IncludeWarnings = true

	result, err := v.Validate(spec.GetLocalPath())
	require.NoError(t, err)

	// Discord should be valid
	assert.True(t, result.Valid, "Discord (OAS 3.1.0) should be valid")

	t.Logf("Discord OAS 3.1.0: Valid=%v, Errors=%d, Warnings=%d",
		result.Valid, result.ErrorCount, result.WarningCount)
}

// TestCorpus_OAS2Validation tests validation of OAS 2.0 spec (Petstore).
func TestCorpus_OAS2Validation(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec, "Petstore spec should exist in corpus")
	corpusutil.SkipIfNotCached(t, *spec)

	v := New()
	v.IncludeWarnings = true

	result, err := v.Validate(spec.GetLocalPath())
	require.NoError(t, err)

	assert.True(t, result.Valid, "Petstore (OAS 2.0) should be valid")

	t.Logf("Petstore OAS 2.0: Valid=%v, Errors=%d, Warnings=%d",
		result.Valid, result.ErrorCount, result.WarningCount)
}

// TestCorpus_ValidateParsed tests validation of pre-parsed documents.
func TestCorpus_ValidateParsed(t *testing.T) {
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	// First parse, then validate
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath(spec.GetLocalPath()),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	v := New()
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	assert.True(t, result.Valid, "Discord should be valid")
}

// Note: BenchmarkCorpus_Validate has been moved to corpus_bench_test.go
// Run with: go test -tags=corpus -bench=BenchmarkCorpus ./validator/...
