//go:build integration

package corpus

import (
	"testing"

	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/overlay"
	"github.com/stretchr/testify/require"
)

// TestCorpus_OverlayFixer_RemoveOperation tests that removing an operation via
// overlay, then running the fixer, prunes orphaned schemas.
func TestCorpus_OverlayFixer_RemoveOperation(t *testing.T) {
	// Use Petstore - it's small and has well-defined operations
	original := parseCorpusSpec(t, "Petstore")

	// Create an overlay that removes the /pet POST operation
	// This operation likely references Pet schema which is also used elsewhere
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Remove Pet POST", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				Target: "$.paths['/pet'].post",
				Remove: true,
			},
		},
	}

	// Apply overlay
	applyResult, err := overlay.ApplyWithOptions(
		overlay.WithSpecParsed(*original),
		overlay.WithOverlayParsed(o),
	)
	require.NoError(t, err)
	require.Equal(t, 1, applyResult.ActionsApplied, "Expected 1 action applied")

	t.Logf("Overlay applied: %d actions", applyResult.ActionsApplied)

	// Reparse to get typed document for fixer
	reparsed, err := overlay.ReparseDocument(original, applyResult.Document)
	require.NoError(t, err)

	// Run fixer with pruning
	fixResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*reparsed),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
			fixer.FixTypePrunedEmptyPath,
		),
	)
	require.NoError(t, err)

	t.Logf("Fixer applied %d fixes after overlay", fixResult.FixCount)

	// The fixer may or may not find orphaned schemas depending on whether
	// the removed operation was the only user of certain schemas.
	// This is a smoke test to ensure the pipeline works.
	// Log the results for observability.
	for _, fix := range fixResult.Fixes {
		t.Logf("  Fix: %s - %s", fix.Type, fix.Description)
	}
}

// TestCorpus_OverlayFixer_RemovePath tests removing an entire path via overlay
// and verifying the fixer handles the resulting document correctly.
func TestCorpus_OverlayFixer_RemovePath(t *testing.T) {
	original := parseCorpusSpec(t, "Petstore")

	// Remove entire /store/inventory path
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Remove Store Inventory", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				Target: "$.paths['/store/inventory']",
				Remove: true,
			},
		},
	}

	applyResult, err := overlay.ApplyWithOptions(
		overlay.WithSpecParsed(*original),
		overlay.WithOverlayParsed(o),
	)
	require.NoError(t, err)
	require.Greater(t, applyResult.ActionsApplied, 0, "Expected overlay to apply at least 1 action")

	t.Logf("Overlay removed path: %d actions applied", applyResult.ActionsApplied)

	// Reparse
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

	t.Logf("After removing /store/inventory: %d fixes applied", fixResult.FixCount)

	// Check if any empty paths were created (unlikely for single path removal)
	emptyPaths := countFixesByType(fixResult, fixer.FixTypePrunedEmptyPath)
	t.Logf("  Empty paths pruned: %d", emptyPaths)

	// Check if any schemas became orphaned
	orphanedSchemas := countFixesByType(fixResult, fixer.FixTypePrunedUnusedSchema)
	t.Logf("  Orphaned schemas pruned: %d", orphanedSchemas)
}

// TestCorpus_OverlayFixer_PlaidPathRemoval tests on Plaid - since it already has
// empty paths, removing more should still work correctly.
func TestCorpus_OverlayFixer_PlaidPathRemoval(t *testing.T) {
	original := parseCorpusSpec(t, "Plaid")

	// Get baseline fix count
	baselineResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*original),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
			fixer.FixTypePrunedEmptyPath,
		),
	)
	require.NoError(t, err)
	baselineCount := baselineResult.FixCount
	t.Logf("Baseline fix count: %d", baselineCount)

	// Apply overlay to remove a populated path (not an empty one)
	// First, let's find a path that has operations
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Remove Credit Report Path", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				// Remove one of the signal paths which may have unique schemas
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
		t.Skip("Path /signal/evaluate not found in Plaid spec")
	}

	reparsed, err := overlay.ReparseDocument(original, applyResult.Document)
	require.NoError(t, err)

	// Run fixer on modified spec
	modifiedResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*reparsed),
		fixer.WithEnabledFixes(
			fixer.FixTypePrunedUnusedSchema,
			fixer.FixTypePrunedEmptyPath,
		),
	)
	require.NoError(t, err)

	t.Logf("After overlay: %d fixes (baseline was %d)", modifiedResult.FixCount, baselineCount)

	// The modified count might be >= baseline since removing a path could:
	// 1. Create new orphaned schemas
	// 2. Remove one of the empty paths that would have been pruned anyway
	// Just verify the pipeline works without errors
}
