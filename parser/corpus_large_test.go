//go:build !short

package parser

import (
	"os"
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCorpus_LargeSpecs_Parse tests parsing large (>5MB) corpus specifications.
// This test is excluded when running with -short flag.
func TestCorpus_LargeSpecs_Parse(t *testing.T) {
	if os.Getenv("SKIP_LARGE_TESTS") == "1" {
		t.Skip("Skipping large spec tests (SKIP_LARGE_TESTS=1)")
	}

	for _, spec := range corpusutil.GetLargeSpecs() {
		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			p := New()
			p.ResolveRefs = false
			p.ValidateStructure = true

			result, err := p.Parse(spec.GetLocalPath())
			require.NoError(t, err, "Parser should handle large spec %s", spec.Name)
			require.NotNil(t, result)

			// Verify basic parsing succeeded
			assert.NotEmpty(t, result.Version)

			t.Logf("%s: Parsed %d byte file with version %s",
				spec.Name, result.SourceSize, result.Version)
		})
	}
}

// TestCorpus_Stripe tests parsing the Stripe specification specifically.
func TestCorpus_Stripe(t *testing.T) {
	if os.Getenv("SKIP_LARGE_TESTS") == "1" {
		t.Skip("Skipping large spec tests (SKIP_LARGE_TESTS=1)")
	}

	spec := corpusutil.GetByName("Stripe")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = true

	result, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)

	doc, ok := result.Document.(*OAS3Document)
	require.True(t, ok, "Stripe should parse as OAS3Document")

	assert.Equal(t, "3.0.0", doc.OpenAPI)
	assert.NotEmpty(t, doc.Paths, "Stripe should have paths")
	assert.NotNil(t, doc.Components, "Stripe should have components")
	assert.NotEmpty(t, doc.Components.Schemas, "Stripe should have schemas")

	t.Logf("Stripe: %d paths, %d schemas",
		len(doc.Paths), len(doc.Components.Schemas))
}

// TestCorpus_MicrosoftGraph tests parsing the Microsoft Graph specification.
func TestCorpus_MicrosoftGraph(t *testing.T) {
	if os.Getenv("SKIP_LARGE_TESTS") == "1" {
		t.Skip("Skipping large spec tests (SKIP_LARGE_TESTS=1)")
	}

	spec := corpusutil.GetByName("MicrosoftGraph")
	require.NotNil(t, spec)
	corpusutil.SkipIfNotCached(t, *spec)

	p := New()
	p.ResolveRefs = false
	p.ValidateStructure = true

	result, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)

	doc, ok := result.Document.(*OAS3Document)
	require.True(t, ok, "Microsoft Graph should parse as OAS3Document")

	// Microsoft Graph uses OAS 3.0.4
	assert.True(t, doc.OpenAPI == "3.0.4" || doc.OpenAPI == "3.0.3",
		"Microsoft Graph should be OAS 3.0.x (got %s)", doc.OpenAPI)
	assert.NotEmpty(t, doc.Paths, "Microsoft Graph should have paths")

	t.Logf("Microsoft Graph: %d paths, OAS version %s",
		len(doc.Paths), doc.OpenAPI)
}

// Note: BenchmarkCorpus_LargeSpecs has been moved to corpus_bench_test.go
// Run with: go test -tags=corpus -bench=BenchmarkCorpus ./parser/...
