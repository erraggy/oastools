package corpus

import (
	"testing"

	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/require"
)

// parseCorpusSpec parses a corpus spec by name, skipping if not cached.
func parseCorpusSpec(t *testing.T, name string) *parser.ParseResult {
	t.Helper()
	spec := corpusutil.GetByName(name)
	require.NotNilf(t, spec, "Corpus spec %q not found", name)
	corpusutil.SkipIfNotCached(t, *spec)

	result, err := parser.ParseWithOptions(
		parser.WithFilePath(spec.GetLocalPath()),
	)
	require.NoError(t, err, "Failed to parse %s", name)
	return result
}

// assertFixCount asserts that a FixResult has the expected fix count within tolerance.
func assertFixCount(t *testing.T, result *fixer.FixResult, expected, tolerance int) {
	t.Helper()
	diff := abs(result.FixCount - expected)
	if diff > tolerance {
		t.Errorf("Fix count %d not within %d of expected %d",
			result.FixCount, tolerance, expected)
	}
}

// countFixesByType counts fixes of a specific type.
func countFixesByType(result *fixer.FixResult, fixType fixer.FixType) int {
	count := 0
	for _, fix := range result.Fixes {
		if fix.Type == fixType {
			count++
		}
	}
	return count
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
