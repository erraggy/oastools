//go:build corpus

// Corpus benchmarks require the corpus build tag to run.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus ./differ/...
// Or use: make bench-corpus

package differ

import (
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
)

// BenchmarkCorpus_Diff benchmarks diffing of corpus specifications.
// This benchmark is excluded by default to prevent memory exhaustion.
// Run with: go test -tags=corpus -bench=BenchmarkCorpus_Diff ./differ/...
func BenchmarkCorpus_Diff(b *testing.B) {
	spec := corpusutil.GetByName("Petstore")
	if spec == nil || !spec.IsAvailable() {
		b.Skip("Petstore spec not available")
	}

	d := New()
	d.Mode = ModeBreaking

	for b.Loop() {
		_, err := d.Diff(spec.GetLocalPath(), spec.GetLocalPath())
		if err != nil {
			b.Fatal(err)
		}
	}
}
