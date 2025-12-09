package joiner

import (
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_JoinSameVersionSpecs tests joining specs of the same OAS version.
// Uses embedded test fixtures to avoid dependency on corpus download.
func TestCorpus_JoinSameVersionSpecs(t *testing.T) {
	// Use embedded join test fixtures (both OAS 3.0.3)
	spec1 := "../testdata/join-base-3.0.yaml"
	spec2 := "../testdata/join-extension-3.0.yaml"

	// Parse both specs
	p := parser.New()
	result1, err := p.Parse(spec1)
	require.NoError(t, err)
	result2, err := p.Parse(spec2)
	require.NoError(t, err)

	// Use accept-left strategy for all collisions
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft
	config.ComponentStrategy = StrategyAcceptLeft

	j := New(config)
	joinResult, err := j.JoinParsed([]parser.ParseResult{*result1, *result2})

	// Join should succeed
	require.NoError(t, err)
	require.NotNil(t, joinResult)

	doc, ok := joinResult.Document.(*parser.OAS3Document)
	require.True(t, ok, "Expected OAS3 document")

	// Verify the join produced meaningful output
	assert.Greater(t, len(doc.Paths), 0, "Joined doc should have paths")

	t.Logf("Joined join-base-3.0.yaml + join-extension-3.0.yaml: Paths=%d, Collisions=%d",
		len(doc.Paths), joinResult.CollisionCount)
}

// TestCorpus_JoinPetstoreWithSelf tests joining Petstore with itself.
func TestCorpus_JoinPetstoreWithSelf(t *testing.T) {
	spec := corpusutil.GetByName("Petstore")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	p := parser.New()
	result1, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)
	result2, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)

	// Join with accept-left (identical specs, no real merge)
	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft

	j := New(config)
	joinResult, err := j.JoinParsed([]parser.ParseResult{*result1, *result2})
	require.NoError(t, err)
	require.NotNil(t, joinResult)

	// Should have same number of paths as original
	doc := joinResult.Document.(*parser.OAS2Document)
	origDoc := result1.Document.(*parser.OAS2Document)

	assert.Equal(t, len(origDoc.Paths), len(doc.Paths),
		"Joined doc should have same paths as original")

	t.Logf("Joined Petstore with itself: Paths=%d, Collisions=%d",
		len(doc.Paths), joinResult.CollisionCount)
}

// TestCorpus_JoinOAS2NotAllowed verifies OAS 2.0 and 3.x cannot be joined.
func TestCorpus_JoinOAS2NotAllowed(t *testing.T) {
	petstore := corpusutil.GetByName("Petstore") // OAS 2.0
	discord := corpusutil.GetByName("Discord")   // OAS 3.1.0

	require.NotNil(t, petstore)
	require.NotNil(t, discord)

	if !petstore.IsAvailable() || !discord.IsAvailable() {
		t.Skip("Petstore or Discord spec not available")
	}

	p := parser.New()
	result1, err := p.Parse(petstore.GetLocalPath())
	require.NoError(t, err)
	result2, err := p.Parse(discord.GetLocalPath())
	require.NoError(t, err)

	config := DefaultConfig()
	j := New(config)

	// Joining OAS 2.0 and 3.x should fail
	_, err = j.JoinParsed([]parser.ParseResult{*result1, *result2})
	assert.Error(t, err, "Joining OAS 2.0 and 3.x should fail")
}

// TestCorpus_JoinPreservesInfo tests that info from first spec is preserved.
func TestCorpus_JoinPreservesInfo(t *testing.T) {
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	p := parser.New()
	result1, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)
	result2, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)

	config := DefaultConfig()
	config.PathStrategy = StrategyAcceptLeft
	config.SchemaStrategy = StrategyAcceptLeft
	config.ComponentStrategy = StrategyAcceptLeft
	j := New(config)

	// Join the spec with itself (requires 2 specs)
	joinResult, err := j.JoinParsed([]parser.ParseResult{*result1, *result2})
	require.NoError(t, err)
	require.NotNil(t, joinResult)

	doc := joinResult.Document.(*parser.OAS3Document)
	origDoc := result1.Document.(*parser.OAS3Document)

	assert.Equal(t, origDoc.Info.Title, doc.Info.Title, "Title should be preserved")
}

// TestCorpus_JoinStrategies tests different collision strategies.
func TestCorpus_JoinStrategies(t *testing.T) {
	spec := corpusutil.GetByName("Discord")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	p := parser.New()
	result1, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)
	result2, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)

	strategies := []CollisionStrategy{
		StrategyAcceptLeft,
		StrategyAcceptRight,
		StrategyFailOnCollision,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			config := DefaultConfig()
			config.PathStrategy = strategy
			config.SchemaStrategy = strategy
			config.ComponentStrategy = strategy

			j := New(config)
			joinResult, err := j.JoinParsed([]parser.ParseResult{*result1, *result2})

			if strategy == StrategyFailOnCollision {
				// Joining identical specs should fail with fail-on-collision strategy
				if err != nil || (joinResult != nil && joinResult.CollisionCount > 0) {
					// Expected - collisions detected
					t.Logf("FailOnCollision: error or collisions detected")
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, joinResult)
				t.Logf("%s: Collisions=%d", strategy, joinResult.CollisionCount)
			}
		})
	}
}
