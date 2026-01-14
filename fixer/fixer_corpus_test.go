package fixer

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Corpus Integration Tests
// =============================================================================

// TestCorpus_FixerReducesErrors tests that the fixer reduces validation errors
// for corpus specs that have missing path parameter issues.
func TestCorpus_FixerReducesErrors(t *testing.T) {
	// Skip if corpus isn't downloaded
	spec := corpusutil.GetByName("DigitalOcean")
	if spec == nil {
		t.Skip("DigitalOcean spec not found in corpus")
	}
	corpusutil.SkipIfNotCached(t, *spec)
	corpusutil.SkipLargeInShortMode(t, *spec)

	// Parse the spec
	p := parser.New()
	parseResult, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err, "Failed to parse %s", spec.Name)

	// Validate before fixing
	v := validator.New()
	v.StrictMode = true
	beforeResult, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err, "Failed to validate before fixing")

	beforeErrors := beforeResult.ErrorCount

	// Apply fixes
	f := New()
	fixResult, err := f.FixParsed(*parseResult)
	require.NoError(t, err, "Failed to fix %s", spec.Name)

	t.Logf("%s: Applied %d fixes", spec.Name, fixResult.FixCount)

	// Validate after fixing - need to create a new ParseResult with fixed doc
	fixedParseResult := &parser.ParseResult{
		Document:     fixResult.Document,
		OASVersion:   fixResult.SourceOASVersion,
		Version:      fixResult.SourceVersion,
		SourceFormat: fixResult.SourceFormat,
	}

	afterResult, err := v.ValidateParsed(*fixedParseResult)
	require.NoError(t, err, "Failed to validate after fixing")

	afterErrors := afterResult.ErrorCount

	// The fixer should reduce errors (or at least not increase them)
	t.Logf("%s: Errors before: %d, after: %d, reduced by: %d",
		spec.Name, beforeErrors, afterErrors, beforeErrors-afterErrors)

	assert.LessOrEqual(t, afterErrors, beforeErrors,
		"Fixer should not increase errors for %s", spec.Name)
}

// TestCorpus_FixerAllInvalidSpecs tests the fixer on all invalid corpus specs
func TestCorpus_FixerAllInvalidSpecs(t *testing.T) {
	invalidSpecs := corpusutil.GetInvalidSpecs(false) // exclude large

	for _, spec := range invalidSpecs {
		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)
			corpusutil.SkipIfHasParsingIssues(t, spec)

			// Parse
			p := parser.New()
			parseResult, err := p.Parse(spec.GetLocalPath())
			require.NoError(t, err, "Failed to parse")

			// Validate before
			v := validator.New()
			v.StrictMode = true
			beforeResult, err := v.ValidateParsed(*parseResult)
			require.NoError(t, err, "Failed to validate before")

			// Fix
			f := New()
			fixResult, err := f.FixParsed(*parseResult)
			require.NoError(t, err, "Failed to fix")

			// Validate after
			fixedParseResult := &parser.ParseResult{
				Document:     fixResult.Document,
				OASVersion:   fixResult.SourceOASVersion,
				Version:      fixResult.SourceVersion,
				SourceFormat: fixResult.SourceFormat,
			}

			afterResult, err := v.ValidateParsed(*fixedParseResult)
			require.NoError(t, err, "Failed to validate after")

			t.Logf("Fixes: %d, Errors before: %d, after: %d, reduced: %d",
				fixResult.FixCount,
				beforeResult.ErrorCount,
				afterResult.ErrorCount,
				beforeResult.ErrorCount-afterResult.ErrorCount)

			// Fixer should not increase errors
			assert.LessOrEqual(t, afterResult.ErrorCount, beforeResult.ErrorCount)
		})
	}
}

// TestCorpus_FixerValidSpecs tests that fixer doesn't break valid specs
func TestCorpus_FixerValidSpecs(t *testing.T) {
	validSpecs := corpusutil.GetValidSpecs(false) // exclude large

	for _, spec := range validSpecs {
		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			// Parse
			p := parser.New()
			parseResult, err := p.Parse(spec.GetLocalPath())
			require.NoError(t, err, "Failed to parse")

			// Fix (should have no changes)
			f := New()
			fixResult, err := f.FixParsed(*parseResult)
			require.NoError(t, err, "Failed to fix")

			t.Logf("Fixes applied: %d", fixResult.FixCount)

			// Validate after - should still be valid
			v := validator.New()
			v.StrictMode = true

			fixedParseResult := &parser.ParseResult{
				Document:     fixResult.Document,
				OASVersion:   fixResult.SourceOASVersion,
				Version:      fixResult.SourceVersion,
				SourceFormat: fixResult.SourceFormat,
			}

			afterResult, err := v.ValidateParsed(*fixedParseResult)
			require.NoError(t, err, "Failed to validate after")

			assert.True(t, afterResult.Valid,
				"Valid spec should remain valid after fixing. Errors: %d",
				afterResult.ErrorCount)
		})
	}
}

// TestCorpus_FixerWithInferTypes tests type inference on real specs
func TestCorpus_FixerWithInferTypes(t *testing.T) {
	spec := corpusutil.GetByName("Asana")
	if spec == nil {
		t.Skip("Asana spec not found in corpus")
	}
	corpusutil.SkipIfNotCached(t, *spec)

	// Parse
	p := parser.New()
	parseResult, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)

	// Fix with type inference
	f := New()
	f.InferTypes = true
	fixResult, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	t.Logf("Applied %d fixes with type inference", fixResult.FixCount)

	// Check that some parameters were inferred as integers
	integerCount := 0
	stringCount := 0
	for _, fix := range fixResult.Fixes {
		if strings.Contains(fix.Description, "type: integer") {
			integerCount++
		} else if strings.Contains(fix.Description, "type: string") {
			stringCount++
		}
	}

	t.Logf("Integer params: %d, String params: %d", integerCount, stringCount)

	// With inference, we expect some integer types for ID parameters
	if fixResult.FixCount > 0 {
		assert.True(t, integerCount > 0 || stringCount > 0,
			"With --infer, should see typed parameters")
	}
}
