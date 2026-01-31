//go:build integration

package corpus

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/differ"
	"github.com/erraggy/oastools/fixer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_FixerDiffer_Plaid verifies that the differ correctly detects
// schema and path removals applied by the fixer.
func TestCorpus_FixerDiffer_Plaid(t *testing.T) {
	// Parse original
	original := parseCorpusSpec(t, "Plaid")

	// Run fixer to get modified spec
	fixResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*original),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
			fixer.FixTypePrunedEmptyPath,
		),
	)
	require.NoError(t, err)
	require.True(t, fixResult.HasFixes(), "Expected fixer to apply fixes")

	t.Logf("Fixer applied %d fixes", fixResult.FixCount)

	// Convert to ParseResult for differ
	fixed := fixResult.ToParseResult()

	// Run differ(original, fixed)
	diffResult, err := differ.DiffWithOptions(
		differ.WithSourceParsed(*original),
		differ.WithTargetParsed(*fixed),
	)
	require.NoError(t, err)

	// Assert differ detects changes
	assert.True(t, len(diffResult.Changes) > 0, "Differ should detect changes")
	t.Logf("Differ detected %d changes", len(diffResult.Changes))

	// Removed paths are breaking changes
	assert.True(t, diffResult.HasBreakingChanges,
		"Removed paths should be detected as breaking changes")

	// Count schema removals in diff
	// Differ uses dot-notation paths like "document.components.schemas.X" or "document.paths./foo"
	schemaRemovals := 0
	pathRemovals := 0
	for _, change := range diffResult.Changes {
		if strings.Contains(change.Path, ".components.schemas.") &&
			change.Type == differ.ChangeTypeRemoved {
			schemaRemovals++
		}
		if strings.Contains(change.Path, ".paths.") &&
			change.Type == differ.ChangeTypeRemoved {
			pathRemovals++
		}
	}

	t.Logf("Differ found: %d schema removals, %d path removals", schemaRemovals, pathRemovals)

	// Verify differ detected at least some of the fixer's changes
	assert.Greater(t, schemaRemovals, 0, "Differ should detect schema removals")
	assert.Greater(t, pathRemovals, 0, "Differ should detect path removals")
}

// TestCorpus_FixerDiffer_NoChanges is a negative test verifying that the
// differ reports no changes for clean specs where fixer makes no modifications.
func TestCorpus_FixerDiffer_NoChanges(t *testing.T) {
	// Parse Discord (clean spec)
	original := parseCorpusSpec(t, "Discord")

	// Run fixer (should make no changes)
	fixResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*original),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
			fixer.FixTypePrunedEmptyPath,
		),
	)
	require.NoError(t, err)

	// No fixes expected for clean spec
	if fixResult.FixCount > 0 {
		t.Skipf("Discord now has %d fixes; spec may have changed", fixResult.FixCount)
	}

	// Convert to ParseResult
	fixed := fixResult.ToParseResult()

	// Run differ
	diffResult, err := differ.DiffWithOptions(
		differ.WithSourceParsed(*original),
		differ.WithTargetParsed(*fixed),
	)
	require.NoError(t, err)

	// No changes expected
	assert.False(t, diffResult.HasBreakingChanges,
		"No breaking changes expected for clean spec")

	// Filter out any info-level changes (if any)
	// Note: Using differ package constants which alias the severity package constants
	breakingChanges := 0
	for _, change := range diffResult.Changes {
		if change.Severity == differ.SeverityError || change.Severity == differ.SeverityCritical {
			breakingChanges++
		}
	}
	assert.Equal(t, 0, breakingChanges, "No breaking changes expected")
}

// TestCorpus_FixerDiffer_GoogleMaps is a minimal case testing that a single
// schema removal (ElevationResponse) is detected by the differ.
func TestCorpus_FixerDiffer_GoogleMaps(t *testing.T) {
	original := parseCorpusSpec(t, "GoogleMaps")

	fixResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*original),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
		),
	)
	require.NoError(t, err)

	schemaCount := countFixesByType(fixResult, fixer.FixTypePrunedUnusedSchema)
	require.Equal(t, 1, schemaCount, "Expected exactly 1 schema removal")

	fixed := fixResult.ToParseResult()

	diffResult, err := differ.DiffWithOptions(
		differ.WithSourceParsed(*original),
		differ.WithTargetParsed(*fixed),
	)
	require.NoError(t, err)

	// Find the schema removal change
	found := false
	for _, change := range diffResult.Changes {
		if strings.Contains(change.Path, "ElevationResponse") {
			found = true
			t.Logf("Found change: %s (%s)", change.Path, change.Type)
			break
		}
	}
	assert.True(t, found, "Differ should detect ElevationResponse schema removal")
}
