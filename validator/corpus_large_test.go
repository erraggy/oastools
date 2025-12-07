//go:build !short

package validator

import (
	"os"
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_LargeSpecs_Validate tests validation of large (>5MB) corpus specs.
// This test is excluded when running with -short flag.
func TestCorpus_LargeSpecs_Validate(t *testing.T) {
	if os.Getenv("SKIP_LARGE_TESTS") == "1" {
		t.Skip("Skipping large spec tests (SKIP_LARGE_TESTS=1)")
	}

	for _, spec := range corpusutil.GetLargeSpecs() {
		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			v := New()
			v.IncludeWarnings = false // Skip warnings for large specs

			result, err := v.Validate(spec.GetLocalPath())
			require.NoError(t, err, "Validation should complete for %s", spec.Name)
			require.NotNil(t, result)

			// Check expected validation outcome
			if spec.ExpectedValid {
				assert.True(t, result.Valid,
					"%s should be valid", spec.Name)
			} else {
				assert.False(t, result.Valid,
					"%s should be invalid", spec.Name)
			}

			t.Logf("%s: Valid=%v, Errors=%d",
				spec.Name, result.Valid, result.ErrorCount)
		})
	}
}

// TestCorpus_Stripe_Validate tests validation of the Stripe specification.
func TestCorpus_Stripe_Validate(t *testing.T) {
	if os.Getenv("SKIP_LARGE_TESTS") == "1" {
		t.Skip("Skipping large spec tests (SKIP_LARGE_TESTS=1)")
	}

	spec := corpusutil.GetByName("Stripe")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	v := New()
	v.IncludeWarnings = true

	result, err := v.Validate(spec.GetLocalPath())
	require.NoError(t, err)

	// Stripe should be valid
	assert.True(t, result.Valid, "Stripe should pass validation")

	t.Logf("Stripe: Valid=%v, Errors=%d, Warnings=%d",
		result.Valid, result.ErrorCount, result.WarningCount)
}

// TestCorpus_MicrosoftGraph_Validate tests validation of Microsoft Graph.
func TestCorpus_MicrosoftGraph_Validate(t *testing.T) {
	if os.Getenv("SKIP_LARGE_TESTS") == "1" {
		t.Skip("Skipping large spec tests (SKIP_LARGE_TESTS=1)")
	}

	spec := corpusutil.GetByName("MicrosoftGraph")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	v := New()
	v.IncludeWarnings = false // Too many warnings

	result, err := v.Validate(spec.GetLocalPath())
	require.NoError(t, err)

	// Microsoft Graph has many validation errors (path params)
	assert.False(t, result.Valid, "Microsoft Graph should have validation errors")
	assert.Greater(t, result.ErrorCount, 1000,
		"Microsoft Graph should have many errors")

	t.Logf("Microsoft Graph: Valid=%v, Errors=%d",
		result.Valid, result.ErrorCount)
}

// BenchmarkCorpus_LargeSpecs_Validate benchmarks validation of large specs.
func BenchmarkCorpus_LargeSpecs_Validate(b *testing.B) {
	if os.Getenv("SKIP_LARGE_TESTS") == "1" {
		b.Skip("Skipping large spec benchmarks (SKIP_LARGE_TESTS=1)")
	}

	spec := corpusutil.GetByName("Stripe")
	if spec == nil || !spec.IsAvailable() {
		b.Skip("Stripe spec not available")
	}

	b.Run("Stripe", func(b *testing.B) {
		v := New()
		v.IncludeWarnings = false

		for b.Loop() {
			_, err := v.Validate(spec.GetLocalPath())
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
