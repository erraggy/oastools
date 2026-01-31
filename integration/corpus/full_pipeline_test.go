//go:build integration

package corpus

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/differ"
	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_FullPipeline_OverlayFixerDiffer tests the complete pipeline:
// overlay (remove operation) -> fixer (prune orphans) -> differ (detect changes).
// This exercises all three packages together in a realistic workflow.
func TestCorpus_FullPipeline_OverlayFixerDiffer(t *testing.T) {
	// Parse original Plaid spec
	original := parseCorpusSpec(t, "Plaid")

	// Step 1: Apply overlay to remove an operation
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Remove Signal Evaluate", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				Target: "$.paths['/signal/evaluate']",
				Remove: true,
			},
		},
	}

	applyResult, err := overlay.ApplyWithOptions(
		overlay.WithSpecParsed(*original),
		overlay.WithOverlayParsed(o),
	)
	require.NoError(t, err)

	if applyResult.ActionsApplied == 0 {
		t.Skip("Path not found in spec")
	}
	t.Logf("Step 1 - Overlay: removed %d paths", applyResult.ActionsApplied)

	// Step 2: Reparse to typed document
	reparsed, err := overlay.ReparseDocument(original, applyResult.Document)
	require.NoError(t, err)

	// Step 3: Run fixer to prune orphans
	fixResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*reparsed),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
			fixer.FixTypePrunedEmptyPath,
		),
	)
	require.NoError(t, err)

	t.Logf("Step 2 - Fixer: applied %d fixes", fixResult.FixCount)
	schemaCount := countFixesByType(fixResult, fixer.FixTypePrunedUnusedSchema)
	pathCount := countFixesByType(fixResult, fixer.FixTypePrunedEmptyPath)
	t.Logf("  - Unused schemas: %d", schemaCount)
	t.Logf("  - Empty paths: %d", pathCount)

	// Step 4: Run differ to detect all changes
	fixed := fixResult.ToParseResult()

	diffResult, err := differ.DiffWithOptions(
		differ.WithSourceParsed(*original),
		differ.WithTargetParsed(*fixed),
	)
	require.NoError(t, err)

	t.Logf("Step 3 - Differ: detected %d changes", len(diffResult.Changes))
	t.Logf("  - Breaking: %v", diffResult.HasBreakingChanges)

	// Assertions
	assert.True(t, len(diffResult.Changes) > 0, "Should detect changes")
	assert.True(t, diffResult.HasBreakingChanges, "Removed path is a breaking change")

	// Verify the removed path is in the diff
	pathRemoved := false
	for _, change := range diffResult.Changes {
		if strings.Contains(change.Path, "signal") &&
			strings.Contains(change.Path, "evaluate") &&
			change.Type == differ.ChangeTypeRemoved {
			pathRemoved = true
			t.Logf("  Found path removal: %s", change.Path)
			break
		}
	}
	assert.True(t, pathRemoved, "Differ should detect the removed path")
}

// TestCorpus_FullPipeline_CleanSpecNoChanges is a negative test verifying that
// a clean spec has no pruning fixes through the full pipeline.
func TestCorpus_FullPipeline_CleanSpecNoChanges(t *testing.T) {
	// Discord is a clean spec with no pruning needed
	original := parseCorpusSpec(t, "Discord")

	// Apply a no-op overlay (update that doesn't change anything meaningful)
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "No-op", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				// Add extension that doesn't affect validation/structure
				Target: "$.info",
				Update: map[string]any{
					"x-test-marker": "integration-test",
				},
			},
		},
	}

	applyResult, err := overlay.ApplyWithOptions(
		overlay.WithSpecParsed(*original),
		overlay.WithOverlayParsed(o),
	)
	require.NoError(t, err)
	require.Equal(t, 1, applyResult.ActionsApplied, "Expected 1 overlay action applied")

	reparsed, err := overlay.ReparseDocument(original, applyResult.Document)
	require.NoError(t, err)

	// Run fixer
	fixResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*reparsed),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
			fixer.FixTypePrunedEmptyPath,
		),
	)
	require.NoError(t, err)

	// No pruning fixes expected for clean spec
	schemaCount := countFixesByType(fixResult, fixer.FixTypePrunedUnusedSchema)
	pathCount := countFixesByType(fixResult, fixer.FixTypePrunedEmptyPath)

	assert.Equal(t, 0, schemaCount, "Clean spec should have no unused schemas")
	assert.Equal(t, 0, pathCount, "Clean spec should have no empty paths")

	t.Logf("Clean spec verification: 0 pruning fixes as expected")
}

// TestCorpus_FullPipeline_MultipleOverlayActions tests applying multiple overlay
// actions in sequence and verifying the fixer and differ handle them correctly.
func TestCorpus_FullPipeline_MultipleOverlayActions(t *testing.T) {
	original := parseCorpusSpec(t, "Petstore")

	// Apply multiple removals
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Remove Multiple", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				Target: "$.paths['/pet'].post",
				Remove: true,
			},
			{
				Target: "$.paths['/pet'].put",
				Remove: true,
			},
			{
				Target: "$.paths['/pet/findByStatus']",
				Remove: true,
			},
		},
	}

	applyResult, err := overlay.ApplyWithOptions(
		overlay.WithSpecParsed(*original),
		overlay.WithOverlayParsed(o),
	)
	require.NoError(t, err)
	require.Equal(t, 3, applyResult.ActionsApplied, "Expected 3 overlay actions applied")

	t.Logf("Applied %d overlay actions", applyResult.ActionsApplied)

	reparsed, err := overlay.ReparseDocument(original, applyResult.Document)
	require.NoError(t, err)

	fixResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*reparsed),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
			fixer.FixTypePrunedEmptyPath,
		),
	)
	require.NoError(t, err)

	t.Logf("Fixer applied %d fixes after multiple removals", fixResult.FixCount)

	// Run differ
	fixed := fixResult.ToParseResult()
	diffResult, err := differ.DiffWithOptions(
		differ.WithSourceParsed(*original),
		differ.WithTargetParsed(*fixed),
	)
	require.NoError(t, err)

	// Multiple operations removed = multiple breaking changes expected
	assert.True(t, diffResult.HasBreakingChanges)

	// Count how many path-related changes
	pathChanges := 0
	for _, change := range diffResult.Changes {
		if strings.HasPrefix(change.Path, "paths.") ||
			strings.Contains(change.Path, ".paths.") {
			pathChanges++
			t.Logf("  Path change: %s (%s)", change.Path, change.Type)
		}
	}

	t.Logf("Differ detected %d total changes, %d path-related",
		len(diffResult.Changes), pathChanges)

	// Expect at least 2 path-related changes (operations may be rolled up)
	// The overlay applied 3 actions but differ may report them differently
	assert.GreaterOrEqual(t, pathChanges, 2,
		"Should detect at least 2 path-related changes")
}
