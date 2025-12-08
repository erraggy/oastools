package differ

import (
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_DiffIdentical tests diffing a spec against itself.
func TestCorpus_DiffIdentical(t *testing.T) {
	specs := []string{"Petstore", "Discord"}

	for _, name := range specs {
		t.Run(name, func(t *testing.T) {
			spec := corpusutil.GetByName(name)
			require.NotNil(t, spec)
			corpusutil.SkipIfNotCached(t, *spec)

			d := New()
			d.Mode = ModeBreaking

			result, err := d.Diff(spec.GetLocalPath(), spec.GetLocalPath())
			require.NoError(t, err)

			assert.Empty(t, result.Changes, "Identical specs should have no changes")
			assert.False(t, result.HasBreakingChanges,
				"Identical specs should have no breaking changes")

			t.Logf("%s: No changes (as expected)", name)
		})
	}
}

// TestCorpus_DiffDifferentSpecs tests diffing two different specs.
func TestCorpus_DiffDifferentSpecs(t *testing.T) {
	// Use Petstore (OAS 2.0) and Discord (OAS 3.1)
	spec1 := corpusutil.GetByName("Petstore") // 2.0
	spec2 := corpusutil.GetByName("Discord")  // 3.1.0

	require.NotNil(t, spec1)
	require.NotNil(t, spec2)

	if !spec1.IsAvailable() || !spec2.IsAvailable() {
		t.Skip("Required specs not available")
	}

	d := New()
	d.Mode = ModeSimple
	d.IncludeInfo = true

	result, err := d.Diff(spec1.GetLocalPath(), spec2.GetLocalPath())
	require.NoError(t, err)

	// Different APIs will have many changes
	assert.True(t, len(result.Changes) > 0,
		"Different specs should have changes")

	t.Logf("Diff %s vs %s: Changes=%d, Breaking=%v",
		spec1.Name, spec2.Name, len(result.Changes), result.HasBreakingChanges)
}

// TestCorpus_DiffBreakingMode tests breaking change detection mode.
func TestCorpus_DiffBreakingMode(t *testing.T) {
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	d := New()
	d.Mode = ModeBreaking

	result, err := d.Diff(spec.GetLocalPath(), spec.GetLocalPath())
	require.NoError(t, err)

	// No breaking changes when diffing identical specs
	assert.False(t, result.HasBreakingChanges)
	assert.Equal(t, 0, result.BreakingCount)
}

// TestCorpus_DiffSimpleMode tests simple diff mode.
func TestCorpus_DiffSimpleMode(t *testing.T) {
	spec1 := corpusutil.GetByName("Petstore")
	spec2 := corpusutil.GetByName("Petstore") // Same spec for predictable results

	require.NotNil(t, spec1)
	require.NotNil(t, spec2)
	corpusutil.SkipIfNotCached(t, *spec1)

	d := New()
	d.Mode = ModeSimple
	d.IncludeInfo = true

	result, err := d.Diff(spec1.GetLocalPath(), spec2.GetLocalPath())
	require.NoError(t, err)

	// Identical specs should have no changes
	assert.Empty(t, result.Changes)
}

// TestCorpus_DiffOAS2 tests diffing OAS 2.0 specs.
func TestCorpus_DiffOAS2(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	d := New()
	d.Mode = ModeBreaking

	result, err := d.Diff(spec.GetLocalPath(), spec.GetLocalPath())
	require.NoError(t, err)

	assert.Empty(t, result.Changes)
	t.Logf("Petstore OAS 2.0 diff: No changes")
}

// TestCorpus_DiffOAS31 tests diffing OAS 3.1.0 specs.
func TestCorpus_DiffOAS31(t *testing.T) {
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	d := New()
	d.Mode = ModeBreaking

	result, err := d.Diff(spec.GetLocalPath(), spec.GetLocalPath())
	require.NoError(t, err)

	assert.Empty(t, result.Changes)
	t.Logf("Discord OAS 3.1.0 diff: No changes")
}

// TestCorpus_DiffVersionMismatch tests diffing specs of different OAS versions.
func TestCorpus_DiffVersionMismatch(t *testing.T) {
	petstore := corpusutil.GetByName("Petstore") // 2.0
	discord := corpusutil.GetByName("Discord")   // 3.1.0

	require.NotNil(t, petstore)
	require.NotNil(t, discord)

	if !petstore.IsAvailable() || !discord.IsAvailable() {
		t.Skip("Required specs not available")
	}

	d := New()
	d.Mode = ModeSimple

	// Diffing different OAS versions may work but produce many changes
	result, err := d.Diff(petstore.GetLocalPath(), discord.GetLocalPath())

	// This may error or produce changes - log either way
	if err != nil {
		t.Logf("Diff OAS 2.0 vs 3.1: Error (may be expected): %v", err)
	} else {
		t.Logf("Diff OAS 2.0 vs 3.1: Changes=%d", len(result.Changes))
	}
}

// TestCorpus_DiffParsed tests diffing pre-parsed documents.
func TestCorpus_DiffParsed(t *testing.T) {
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	// Parse first
	result1, err := parser.ParseWithOptions(
		parser.WithFilePath(spec.GetLocalPath()),
	)
	require.NoError(t, err)

	result2, err := parser.ParseWithOptions(
		parser.WithFilePath(spec.GetLocalPath()),
	)
	require.NoError(t, err)

	d := New()
	diffResult, err := d.DiffParsed(*result1, *result2)
	require.NoError(t, err)

	assert.Empty(t, diffResult.Changes)
}

// Note: BenchmarkCorpus_Diff has been moved to corpus_bench_test.go
// Run with: go test -tags=corpus -bench=BenchmarkCorpus ./differ/...
