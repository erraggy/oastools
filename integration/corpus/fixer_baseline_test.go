//go:build integration

package corpus

import (
	"testing"

	"github.com/erraggy/oastools/fixer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_FixerBaseline_Plaid verifies fixer correctly identifies and removes
// unused schemas and empty paths in the Plaid specification.
func TestCorpus_FixerBaseline_Plaid(t *testing.T) {
	parseResult := parseCorpusSpec(t, "Plaid")

	// Run fixer with pruning enabled
	result, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
			fixer.FixTypePrunedEmptyPath,
		),
	)
	require.NoError(t, err)

	// Count fixes by type
	schemaCount := countFixesByType(result, fixer.FixTypePrunedUnusedSchema)
	pathCount := countFixesByType(result, fixer.FixTypePrunedEmptyPath)

	t.Logf("Plaid: %d unused schemas removed, %d empty paths removed", schemaCount, pathCount)

	// Assert expected counts (from exploration: 128 schemas, 10 paths)
	// Use tolerance of 10 to allow for upstream spec changes
	assertFixCount(t, result, 138, 10) // 128 + 10 with tolerance

	// Verify specific schemas are removed (spot check)
	assert.GreaterOrEqual(t, schemaCount, 100, "Expected at least 100 unused schemas")
	assert.GreaterOrEqual(t, pathCount, 5, "Expected at least 5 empty paths")
}

// TestCorpus_FixerBaseline_GoogleMaps verifies the minimal case with a single
// unused schema (ElevationResponse).
func TestCorpus_FixerBaseline_GoogleMaps(t *testing.T) {
	parseResult := parseCorpusSpec(t, "GoogleMaps")

	result, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
			fixer.FixTypePrunedEmptyPath,
		),
	)
	require.NoError(t, err)

	schemaCount := countFixesByType(result, fixer.FixTypePrunedUnusedSchema)
	pathCount := countFixesByType(result, fixer.FixTypePrunedEmptyPath)
	t.Logf("GoogleMaps: %d unused schemas removed, %d empty paths removed", schemaCount, pathCount)

	// Expect exactly 1 schema (ElevationResponse) and 0 empty paths
	assert.Equal(t, 1, schemaCount, "GoogleMaps should have exactly 1 unused schema")
	assert.Equal(t, 0, pathCount, "GoogleMaps should have no empty paths")
}

// TestCorpus_FixerBaseline_CleanSpecs is a negative test for specs that should
// have no pruning fixes applied.
func TestCorpus_FixerBaseline_CleanSpecs(t *testing.T) {
	cleanSpecs := []string{"Discord", "Stripe", "Petstore"}

	for _, name := range cleanSpecs {
		t.Run(name, func(t *testing.T) {
			parseResult := parseCorpusSpec(t, name)

			result, err := fixer.FixWithOptions(
				fixer.WithParsed(*parseResult),
				fixer.WithEnabledFixes(
					fixer.FixTypePrunedUnusedSchema,
					fixer.FixTypePrunedEmptyPath,
				),
			)
			require.NoError(t, err)

			// These specs should have 0 pruning fixes
			schemaCount := countFixesByType(result, fixer.FixTypePrunedUnusedSchema)
			pathCount := countFixesByType(result, fixer.FixTypePrunedEmptyPath)

			assert.Equal(t, 0, schemaCount, "%s should have no unused schemas", name)
			assert.Equal(t, 0, pathCount, "%s should have no empty paths", name)
		})
	}
}
